package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v3"
)

type PoolMode string

const (
	POOLMODE_SESSION PoolMode = "session"
	POOLMODE_TXN     PoolMode = "transaction"
)

type ServerRole string

const (
	SERVERROLE_PRIMARY ServerRole = "primary"
	SERVERROLE_REPLICA ServerRole = "replica"
	SERVERROLE_NONE    ServerRole = "NONE"
)

type UserRole string

const (
	USERROLE_ADMIN  UserRole = "admin"
	USERROLE_WRITER UserRole = "writer"
	USERROLE_READER UserRole = "reader"
)

func Load(path string) (*Global, error) {
	var g Global
	ext := filepath.Ext(path)
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	switch ext {
	case "toml":
		err := toml.Unmarshal(file, &g)
		if err != nil {
			return nil, err
		}
	case "yml", "yaml", "json":
		fallthrough
	default:
		err := yaml.Unmarshal(file, &g)
		if err != nil {
			return nil, err
		}
	}

	for _, g := range g.Pools {
		for _, shard := range g.Shards {
			for _, srv := range shard.Servers {
				if strings.HasPrefix(srv.Host, "ENV$") {
					srv.Host = os.Getenv(strings.TrimPrefix(srv.Host, "ENV$"))
				}
				if strings.HasPrefix(srv.Username, "ENV$") {
					srv.Username = os.Getenv(strings.TrimPrefix(srv.Username, "ENV$"))
				}
				if strings.HasPrefix(srv.Password, "ENV$") {
					srv.Password = os.Getenv(strings.TrimPrefix(srv.Password, "ENV$"))
				}
			}
			for _, srv := range g.Users {
				if strings.HasPrefix(srv.Name, "ENV$") {
					srv.Name = os.Getenv(strings.TrimPrefix(srv.Name, "ENV$"))
				}
				if strings.HasPrefix(srv.Password, "ENV$") {
					srv.Password = os.Getenv(strings.TrimPrefix(srv.Password, "ENV$"))
				}
			}
		}
	}
	spew.Println(g)
	return &g, nil
}

type Global struct {
	General General          `toml:"general" yaml:"general" json:"general"`
	Pools   map[string]*Pool `toml:"pools" yaml:"pools" json:"pools"`
}

type General struct {
	Host string `toml:"host" yaml:"host" json:"host"`
	Port uint16 `toml:"port" yaml:"port" json:"port"`

	AdminOnly     bool   `toml:"admin_only" yaml:"admin_only" json:"admin_only"`
	AdminUsername string `toml:"admin_username" yaml:"admin_username" json:"admin_username"`
	AdminPassword string `toml:"admin_password" yaml:"admin_password" json:"admin_password"`

	EnableMetrics bool   `toml:"enable_prometheus_exporter" yaml:"enable_prometheus_exporter" json:"enable_prometheus_exporter"`
	MetricsPort   uint16 `toml:"prometheus_exporter_port" yaml:"prometheus_exporter_port" json:"prometheus_exporter_port"`

	PoolSize int      `toml:"pool_size" yaml:"pool_size" json:"pool_size"`
	PoolMode PoolMode `toml:"pool_mode" yaml:"pool_mode" json:"pool_mode"`

	ConnectTimeout     int `toml:"connect_timeout" yaml:"connect_timeout" json:"connect_timeout"`
	HealthCheckTimeout int `toml:"healthcheck_timeout" yaml:"healthcheck_timeout" json:"healthcheck_timeout"`
	ShutdownTimeout    int `toml:"shutdown_timeout" yaml:"shutdown_timeout" json:"shutdown_timeout"`

	BanTime int `toml:"ban_time" yaml:"ban_time" json:"ban_time"`

	TlsCertificate string `toml:"tls_certificate" yaml:"tls_certificate" json:"tls_certificate"`
	TlsPrivateKey  string `toml:"tls_private_key" yaml:"tls_private_key" json:"tls_private_key"`

	AutoReload bool `toml:"autoreload" yaml:"autoreload" json:"autoreload"`
}

type Pool struct {
	PoolMode            PoolMode `toml:"pool_mode" yaml:"pool_mode" json:"pool_mode"`
	DefaultRole         string   `toml:"default_role" yaml:"default_role" json:"default_role"`
	QueryParserEnabled  bool     `toml:"query_parser_enabled" yaml:"query_parser_enabled" json:"query_parser_enabled"`
	PrimaryReadsEnabled bool     `toml:"primary_reads_enabled" yaml:"primary_reads_enabled" json:"primary_reads_enabled"`
	ShardingFunction    string   `toml:"sharding_function" yaml:"sharding_function" json:"sharding_function"`

	Shards []*Shard `toml:"shards" yaml:"shards" json:"shards"`
	Users  []*User  `toml:"users" yaml:"users" json:"users"`
}

type User struct {
	Name     string `toml:"username" yaml:"username" json:"username"`
	Password string `toml:"password" yaml:"password" json:"password"`

	Role             UserRole `toml:"role" yaml:"role" json:"role"`
	PoolSize         int      `toml:"pool_size" yaml:"pool_size" json:"pool_size"`
	StatementTimeout int      `toml:"statement_timeout" yaml:"statement_timeout" json:"statement_timeout"`
}

type Shard struct {
	Database string    `toml:"database" yaml:"database" json:"database"`
	Servers  []*Server `toml:"servers" yaml:"servers" json:"servers"`
}

type Server struct {
	Host string     `toml:"host" yaml:"host" json:"host"`
	Port uint16     `toml:"port" yaml:"port" json:"port"`
	Role ServerRole `toml:"role" yaml:"role" json:"role"`

	Username string `toml:"username" yaml:"username" json:"username"`
	Password string `toml:"password" yaml:"password" json:"password"`
}
