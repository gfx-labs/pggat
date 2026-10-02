// Package pgtest starts an embedded PostgreSQL server for tests.
package pgtest

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jackc/pgx/v5"
)

const (
	Username = "postgres"
	Password = "postgres"

	// PostgreSQL 18 reports search_path in ParameterStatus, which pggat relies on to track it.
	version = embeddedpostgres.V18

	pgHBA = `local all         all              scram-sha-256
host  all         all 127.0.0.1/32 scram-sha-256
host  all         all ::1/128      scram-sha-256
local replication all              scram-sha-256
host  replication all 127.0.0.1/32 scram-sha-256
host  replication all ::1/128      scram-sha-256
`
)

// Server is a running embedded PostgreSQL instance.
type Server struct {
	Host     string
	Port     int
	Database string

	pg *embeddedpostgres.EmbeddedPostgres
}

// Addr returns host:port.
func (s *Server) Addr() string {
	return net.JoinHostPort(s.Host, fmt.Sprint(s.Port))
}

// URL returns a connection URL for database. An empty database uses the default one.
func (s *Server) URL(database string) string {
	if database == "" {
		database = s.Database
	}
	return fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", Username, Password, s.Addr(), database)
}

// Exec runs sql against the server's default database.
func (s *Server) Exec(ctx context.Context, sql string) error {
	conn, err := pgx.Connect(ctx, s.URL(""))
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, sql)
	return err
}

// Stop shuts the server down.
func (s *Server) Stop() error {
	return s.pg.Stop()
}

// FreePort returns a TCP port that is currently free on localhost.
func FreePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func cacheDir() string {
	if dir := os.Getenv("PGTEST_CACHE"); dir != "" {
		return dir
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".embedded-postgres-go")
	}
	return filepath.Join(os.TempDir(), "embedded-postgres-go")
}

// prepare extracts postgres and runs initdb once into cache, so each start
// only copies a template data directory. It returns the binaries and template paths.
func prepare(cache string) (string, string, error) {
	bin := filepath.Join(cache, "bin-"+string(version))
	tmpl := filepath.Join(cache, "template-"+string(version))
	ready := func() bool {
		_, err1 := os.Stat(filepath.Join(bin, "bin", "pg_ctl"))
		_, err2 := os.Stat(filepath.Join(tmpl, "PG_VERSION"))
		return err1 == nil && err2 == nil
	}
	if ready() {
		return bin, tmpl, nil
	}
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return "", "", err
	}

	lock, err := os.OpenFile(filepath.Join(cache, "pgtest-"+string(version)+".lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return "", "", err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return "", "", err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	if ready() {
		return bin, tmpl, nil
	}
	if err := os.RemoveAll(bin); err != nil {
		return "", "", err
	}
	if err := os.RemoveAll(tmpl); err != nil {
		return "", "", err
	}

	// Let the library download, extract and initdb into a staging dir, then move both into place.
	staging, err := os.MkdirTemp(cache, "staging-")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(staging)
	port, err := FreePort()
	if err != nil {
		return "", "", err
	}
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Version(version).
		Port(uint32(port)).
		Username(Username).
		Password(Password).
		CachePath(cache).
		RuntimePath(filepath.Join(staging, "rt")).
		DataPath(filepath.Join(staging, "data")).
		BinariesPath(filepath.Join(staging, "bin")).
		StartTimeout(60 * time.Second).
		Logger(io.Discard))
	if err := pg.Start(); err != nil {
		return "", "", err
	}
	if err := pg.Stop(); err != nil {
		return "", "", err
	}
	if err := os.Rename(filepath.Join(staging, "bin"), bin); err != nil {
		return "", "", err
	}
	if err := os.Rename(filepath.Join(staging, "data"), tmpl); err != nil {
		return "", "", err
	}
	return bin, tmpl, nil
}

// Start launches a server with the given database. runtimeDir must be unique per server.
// Binaries are cached in ~/.embedded-postgres-go (or $PGTEST_CACHE).
func Start(database, runtimeDir string, logger io.Writer) (*Server, error) {
	cache := cacheDir()
	binaries, tmpl, err := prepare(cache)
	if err != nil {
		return nil, fmt.Errorf("prepare postgres binaries: %w", err)
	}

	// The library reuses a data dir whose PG_VERSION matches, skipping initdb.
	data := filepath.Join(runtimeDir, "data")
	if err := os.MkdirAll(runtimeDir, 0o755); err != nil {
		return nil, err
	}
	if out, err := exec.Command("cp", "-a", tmpl, data).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("copy template: %w: %s", err, out)
	}
	// Require SCRAM so tests exercise pggat's SASL client against a real server.
	if err := os.WriteFile(filepath.Join(data, "pg_hba.conf"), []byte(pgHBA), 0o600); err != nil {
		return nil, err
	}

	port, err := FreePort()
	if err != nil {
		return nil, err
	}

	config := embeddedpostgres.DefaultConfig().
		Version(version).
		Port(uint32(port)).
		Database("postgres").
		Username(Username).
		Password(Password).
		CachePath(cache).
		RuntimePath(filepath.Join(runtimeDir, "rt")).
		DataPath(data).
		BinariesPath(binaries).
		StartTimeout(60 * time.Second).
		StartParameters(map[string]string{
			"max_connections":    "300",
			"fsync":              "off",
			"synchronous_commit": "off",
			"full_page_writes":   "off",
		}).
		Logger(logger)

	pg := embeddedpostgres.NewDatabase(config)
	if err := pg.Start(); err != nil {
		return nil, err
	}

	s := &Server{
		Host:     "127.0.0.1",
		Port:     port,
		Database: database,
		pg:       pg,
	}
	// The data dir is reused, so the library does not create the database.
	if database != "postgres" {
		if err := s.createDatabase(database); err != nil {
			_ = pg.Stop()
			return nil, err
		}
	}
	return s, nil
}

func (s *Server) createDatabase(name string) error {
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, s.URL("postgres"))
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	return err
}

// StartT launches a server scoped to a test and stops it on cleanup.
func StartT(t testing.TB, database string) *Server {
	t.Helper()
	s, err := Start(database, filepath.Join(t.TempDir(), "pg"), io.Discard)
	if err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := s.Stop(); err != nil {
			t.Errorf("stop embedded postgres: %v", err)
		}
	})
	return s
}
