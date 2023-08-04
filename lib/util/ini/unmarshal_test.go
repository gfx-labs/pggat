package ini

import (
	"reflect"
	"testing"
)

type Database struct {
	Host           string `ini:"host"`
	Port           string `ini:"port"`
	User           string `ini:"user"`
	Password       string `ini:"password"`
	ClientEncoding string `ini:"client_encoding"`
	Datestyle      string `ini:"datestyle"`
	DBName         string `ini:"dbname"`
	AuthUser       string `ini:"auth_user"`
}

type Peer struct {
	Host string `ini:"host"`
}

type PoolMode string

const (
	PoolModeSession     PoolMode = "session"
	PoolModeTransaction PoolMode = "transaction"
)

type PgBouncer struct {
	PoolMode      PoolMode `ini:"pool_mode"`
	ListenPort    string   `ini:"listen_port"`
	ListenAddr    string   `ini:"listen_addr"`
	AuthType      string   `ini:"auth_type"`
	AuthFile      string   `ini:"auth_file"`
	Logfile       string   `ini:"logfile"`
	Pidfile       string   `ini:"pidfile"`
	AdminUsers    string   `ini:"admin_users"`
	StatsUsers    string   `ini:"stats_users"`
	SoReuseport   string   `ini:"so_reuseport"`
	UnixSocketDir string   `ini:"unix_socket_dir"`
	PeerId        string   `ini:"peer_id"`
}

type Root struct {
	Databases map[string]Database `ini:"databases"`
	Peers     map[string]Peer     `ini:"peers"`
	PgBouncer PgBouncer           `ini:"pgbouncer"`
}

type Case struct {
	Value    string
	Expected Root
}

var Cases = []Case{
	{
		Value: `[databases]
postgres = host=localhost dbname=postgres

[peers]
1 = host=/tmp/pgbouncer1
2 = host=/tmp/pgbouncer2

[pgbouncer]
listen_addr=127.0.0.1
auth_file=auth_file.conf
so_reuseport=1
; only unix_socket_dir and peer_id are different
unix_socket_dir=/tmp/pgbouncer2
peer_id=2
`,
		Expected: Root{
			Databases: map[string]Database{
				"postgres": {
					Host:   "localhost",
					DBName: "postgres",
				},
			},
			Peers: map[string]Peer{
				"1": {
					Host: "/tmp/pgbouncer1",
				},
				"2": {
					Host: "/tmp/pgbouncer2",
				},
			},
			PgBouncer: PgBouncer{
				ListenAddr:    "127.0.0.1",
				AuthFile:      "auth_file.conf",
				SoReuseport:   "1",
				UnixSocketDir: "/tmp/pgbouncer2",
				PeerId:        "2",
			},
		},
	},
	{
		Value: `[databases]
template1 = host=localhost dbname=template1 auth_user=someuser

[pgbouncer]
pool_mode = session
listen_port = 6432
listen_addr = localhost
auth_type = md5
auth_file = users.txt
logfile = pgbouncer.log
pidfile = pgbouncer.pid
admin_users = someuser
stats_users = stat_collector`,
		Expected: Root{
			Databases: map[string]Database{
				"template1": {
					Host:     "localhost",
					DBName:   "template1",
					AuthUser: "someuser",
				},
			},
			PgBouncer: PgBouncer{
				PoolMode:   PoolModeSession,
				ListenPort: "6432",
				ListenAddr: "localhost",
				AuthType:   "md5",
				AuthFile:   "users.txt",
				Logfile:    "pgbouncer.log",
				Pidfile:    "pgbouncer.pid",
				AdminUsers: "someuser",
				StatsUsers: "stat_collector",
			},
		},
	},
	{
		Value: `[databases]

; foodb over Unix socket
foodb =

; redirect bardb to bazdb on localhost
bardb = host=localhost dbname=bazdb

; access to destination database will go with single user
forcedb = host=localhost port=300 user=baz password=foo client_encoding=UNICODE datestyle=ISO`,
		Expected: Root{
			Databases: map[string]Database{
				"foodb": {},
				"bardb": {
					Host:   "localhost",
					DBName: "bazdb",
				},
				"forcedb": {
					Host:           "localhost",
					Port:           "300",
					User:           "baz",
					Password:       "foo",
					ClientEncoding: "UNICODE",
					Datestyle:      "ISO",
				},
			},
		},
	},
}

func TestUnmarshal(t *testing.T) {
	for _, cas := range Cases {
		var result Root
		err := Unmarshal([]byte(cas.Value), &result)
		if err != nil {
			t.Error(err)
			continue
		}
		if !reflect.DeepEqual(result, cas.Expected) {
			t.Error("result != expected")
			continue
		}
	}
}
