//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	_ "github.com/caddyserver/caddy/v2/modules/standard"
	"github.com/jackc/pgx/v5"

	"gfx.cafe/gfx/pggat/lib/gat"
	_ "gfx.cafe/gfx/pggat/lib/gat/pgcaddyfile"
	_ "gfx.cafe/gfx/pggat/lib/gat/standard"

	"gfx.cafe/gfx/pggat/test/pgtest"
)

const database = "testdb"

var (
	postgresUser     = pgtest.Username
	postgresPassword = pgtest.Password

	// addresses set by TestMain
	primaryAddr      string
	httpAddr         string
	transactionAddr  string
	sessionAddr      string
	hybridAddr       string
	singleServerAddr string
	// one-backend servers, see addSingleServers
	sessionSingleAddr string
	hybridSingleAddr  string
	// one-backend transaction and hybrid pools whose backend is reached through cancels
	cancelSingleAddr string
	cancelHybridAddr string
	cancels          *cancelProxy
)

func connURL(addr string, query ...string) string {
	u := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", postgresUser, postgresPassword, addr, database)
	for _, q := range query {
		u += "&" + q
	}
	return u
}

// loadCaddyfile reads test/configs/<name>.Caddyfile, points it at the embedded
// primary, and moves its PostgreSQL listener to a free port.
func loadCaddyfile(name, listen string) (string, string, error) {
	raw, err := os.ReadFile(filepath.Join("..", "configs", name+".Caddyfile"))
	if err != nil {
		return "", "", err
	}
	port, err := pgtest.FreePort()
	if err != nil {
		return "", "", err
	}
	s := string(raw)
	oldListener := "[pg] " + listen + " {"
	if !strings.Contains(s, oldListener) {
		return "", "", fmt.Errorf("%s.Caddyfile: listener %s not found", name, listen)
	}
	s = strings.ReplaceAll(s, "postgres-primary:5432", primaryAddr)
	s = strings.Replace(s, oldListener, fmt.Sprintf("[pg] :%d {", port), 1)
	return s, fmt.Sprintf("127.0.0.1:%d", port), nil
}

func startPggat() error {
	var caddyfile strings.Builder
	caddyfile.WriteString("[global] {\n\tadmin off\n}\n\n[pg] {\n\tstat_log_period 17m\n}\n\n")
	for _, c := range []struct {
		name, listen string
		addr         *string
	}{
		{"transaction", ":6432", &transactionAddr},
		{"session", ":6433", &sessionAddr},
		{"hybrid", ":6434", &hybridAddr},
	} {
		s, addr, err := loadCaddyfile(c.name, c.listen)
		if err != nil {
			return err
		}
		*c.addr = addr
		caddyfile.WriteString(s)
		caddyfile.WriteString("\n")
	}

	httpPort, err := pgtest.FreePort()
	if err != nil {
		return err
	}
	httpAddr = fmt.Sprintf("127.0.0.1:%d", httpPort)
	if _, err := fmt.Fprintf(&caddyfile, "[http.server] http://%s {\n\trespond \"pggat-http-ok\"\n}\n", httpAddr); err != nil {
		return err
	}

	adapter := caddyconfig.GetAdapter("caddyfile")
	cfg, _, err := adapter.Adapt([]byte(caddyfile.String()), nil)
	if err != nil {
		return fmt.Errorf("adapt Caddyfile: %w", err)
	}
	var config caddy.Config
	if err := json.Unmarshal(cfg, &config); err != nil {
		return err
	}
	if config.Admin == nil || !config.Admin.Disabled {
		return fmt.Errorf("Caddyfile [global] admin off was not applied")
	}
	var pgApp gat.Config
	if err := json.Unmarshal(config.AppsRaw["pggat"], &pgApp); err != nil {
		return err
	}
	if pgApp.StatLogPeriod != caddy.Duration(17*time.Minute) {
		return fmt.Errorf("Caddyfile [pg] stat_log_period = %s, want 17m", time.Duration(pgApp.StatLogPeriod))
	}
	if err := addSingleServers(&config); err != nil {
		return err
	}
	if err := caddy.Run(&config); err != nil {
		return fmt.Errorf("run pggat: %w", err)
	}
	return nil
}

// Recipe limits are exposed by native JSON, but not by the static pool Caddyfile directive.
func addSingleServers(config *caddy.Config) error {
	var app gat.Config
	if err := json.Unmarshal(config.AppsRaw["pggat"], &app); err != nil {
		return err
	}
	for _, c := range []struct {
		file, listen string
		addr         *string
		backend      string
	}{
		{"single_server.json", ":6435", &singleServerAddr, primaryAddr},
		{"session_single.json", ":6436", &sessionSingleAddr, primaryAddr},
		{"hybrid_single.json", ":6437", &hybridSingleAddr, primaryAddr},
		{"single_server.json", ":6435", &cancelSingleAddr, cancels.Addr()},
		{"hybrid_single.json", ":6437", &cancelHybridAddr, cancels.Addr()},
	} {
		raw, err := os.ReadFile(filepath.Join("..", "configs", c.file))
		if err != nil {
			return err
		}
		port, err := pgtest.FreePort()
		if err != nil {
			return err
		}
		*c.addr = fmt.Sprintf("127.0.0.1:%d", port)
		s := strings.ReplaceAll(string(raw), "postgres-primary:5432", c.backend)
		s = strings.ReplaceAll(s, c.listen, fmt.Sprintf(":%d", port))
		var server gat.ServerConfig
		if err := json.Unmarshal([]byte(s), &server); err != nil {
			return err
		}
		app.Servers = append(app.Servers, server)
	}
	var err error
	config.AppsRaw["pggat"], err = json.Marshal(app)
	return err
}

func seed(ctx context.Context, pg *pgtest.Server) error {
	sql, err := os.ReadFile(filepath.Join("..", "fixtures", "init-primary.sql"))
	if err != nil {
		return err
	}
	return pg.Exec(ctx, string(sql))
}

func waitReady(ctx context.Context, addrs ...string) error {
	deadline := time.Now().Add(30 * time.Second)
	for _, addr := range addrs {
		for {
			conn, err := pgx.Connect(ctx, connURL(addr))
			if err == nil {
				_ = conn.Close(ctx)
				break
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("pggat at %s not ready: %w", addr, err)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	return nil
}

func TestCombinedHTTPAndPostgresCaddyfile(t *testing.T) {
	t.Parallel()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://" + httpAddr)
	if err != nil {
		t.Fatalf("GET from configured HTTP server: %v", err)
	}
	body, readErr := io.ReadAll(resp.Body)
	closeErr := resp.Body.Close()
	if readErr != nil {
		t.Fatalf("read HTTP response: %v", readErr)
	}
	if closeErr != nil {
		t.Fatalf("close HTTP response: %v", closeErr)
	}
	if resp.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != "pggat-http-ok" {
		t.Fatalf("HTTP response = %d %q, want 200 %q", resp.StatusCode, body, "pggat-http-ok")
	}

	ctx := tctx(t)
	conn := lifecycleConnect(ctx, t, transactionAddr)
	var result int
	if err := conn.QueryRow(ctx, "SELECT 42").Scan(&result); err != nil {
		t.Fatalf("query PostgreSQL app from same Caddyfile: %v", err)
	}
	if result != 42 {
		t.Fatalf("PostgreSQL result = %d, want 42", result)
	}
}

func run(m *testing.M) int {
	ctx := context.Background()

	dir, err := os.MkdirTemp("", "pggat-integration-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(dir)

	logger := io.Discard
	if os.Getenv("PGTEST_LOG") != "" {
		logger = os.Stderr
	}
	pg, err := pgtest.Start(database, filepath.Join(dir, "pg"), logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start postgres:", err)
		return 1
	}
	defer pg.Stop()
	primaryAddr = pg.Addr()

	if err := seed(ctx, pg); err != nil {
		fmt.Fprintln(os.Stderr, "seed:", err)
		return 1
	}

	cancels, err = startCancelProxy(primaryAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start cancel proxy:", err)
		return 1
	}
	defer cancels.Close()

	if err := startPggat(); err != nil {
		fmt.Fprintln(os.Stderr, "start pggat:", err)
		return 1
	}
	defer caddy.Stop()

	if err := waitReady(ctx, transactionAddr, sessionAddr, hybridAddr, singleServerAddr, sessionSingleAddr, hybridSingleAddr, cancelSingleAddr, cancelHybridAddr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(run(m))
}
