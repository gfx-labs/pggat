//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/jackc/pgx/v5"

	_ "gfx.cafe/gfx/pggat/lib/gat/gatcaddyfile"
	_ "gfx.cafe/gfx/pggat/lib/gat/standard"

	"gfx.cafe/gfx/pggat/test/pgtest"
)

const database = "testdb"

var (
	postgresUser     = pgtest.Username
	postgresPassword = pgtest.Password

	// addresses set by TestMain
	primaryAddr     string
	transactionAddr string
	sessionAddr     string
	hybridAddr      string
)

func connURL(addr string, query ...string) string {
	u := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", postgresUser, postgresPassword, addr, database)
	for _, q := range query {
		u += "&" + q
	}
	return u
}

// loadGatfile reads test/configs/<name>.Gatfile, points it at the embedded
// primary, and moves its listener to a free port. It returns the Gatfile and its address.
func loadGatfile(name, listen string) (string, string, error) {
	raw, err := os.ReadFile(filepath.Join("..", "configs", name+".Gatfile"))
	if err != nil {
		return "", "", err
	}
	port, err := pgtest.FreePort()
	if err != nil {
		return "", "", err
	}
	s := string(raw)
	if !strings.Contains(s, listen+" {") {
		return "", "", fmt.Errorf("%s.Gatfile: listener %s not found", name, listen)
	}
	s = strings.ReplaceAll(s, "postgres-primary:5432", primaryAddr)
	s = strings.Replace(s, listen+" {", fmt.Sprintf(":%d {", port), 1)
	return s, fmt.Sprintf("127.0.0.1:%d", port), nil
}

func startPggat() error {
	var gatfile strings.Builder
	for _, c := range []struct {
		name, listen string
		addr         *string
	}{
		{"transaction", ":6432", &transactionAddr},
		{"session", ":6433", &sessionAddr},
		{"hybrid", ":6434", &hybridAddr},
	} {
		s, addr, err := loadGatfile(c.name, c.listen)
		if err != nil {
			return err
		}
		*c.addr = addr
		gatfile.WriteString(s)
		gatfile.WriteString("\n")
	}

	adapter := caddyconfig.GetAdapter("gatfile")
	cfg, _, err := adapter.Adapt([]byte(gatfile.String()), nil)
	if err != nil {
		return fmt.Errorf("adapt gatfile: %w", err)
	}
	var config caddy.Config
	if err := json.Unmarshal(cfg, &config); err != nil {
		return err
	}
	config.Admin = &caddy.AdminConfig{Disabled: true}
	if err := caddy.Run(&config); err != nil {
		return fmt.Errorf("run pggat: %w", err)
	}
	return nil
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

	if err := startPggat(); err != nil {
		fmt.Fprintln(os.Stderr, "start pggat:", err)
		return 1
	}
	defer caddy.Stop()

	if err := waitReady(ctx, transactionAddr, sessionAddr, hybridAddr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	return m.Run()
}

func TestMain(m *testing.M) {
	os.Exit(run(m))
}
