package pgbouncer

import (
	"crypto/tls"
	"net"
	"strconv"
	"strings"

	"tuxpa.in/a/zlog/log"

	"pggat2/lib/bouncer"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/gat"
	"pggat2/lib/util/encoding/ini"
	"pggat2/lib/util/flip"
	"pggat2/lib/util/strutil"
)

type PoolMode string

const (
	PoolModeSession     PoolMode = "session"
	PoolModeTransaction PoolMode = "transaction"
	PoolModeStatement   PoolMode = "statement"
)

type AuthType string

const (
	AuthTypeCert        AuthType = "cert"
	AuthTypeMd5         AuthType = "md5"
	AuthTypeScramSha256 AuthType = "scram-sha-256"
	AuthTypePlain       AuthType = "plain"
	AuthTypeTrust       AuthType = "trust"
	AuthTypeAny         AuthType = "any"
	AuthTypeHba         AuthType = "hba"
	AuthTypePam         AuthType = "pam"
)

type TLSProtocol string

const (
	TLSProtocolV1_0   TLSProtocol = "tlsv1.0"
	TLSProtocolV1_1   TLSProtocol = "tlsv1.1"
	TLSProtocolV1_2   TLSProtocol = "tlsv1.2"
	TLSProtocolV1_3   TLSProtocol = "tlsv1.3"
	TLSProtocolAll    TLSProtocol = "all"
	TLSProtocolSecure TLSProtocol = "secure"
	TLSProtocolLegacy TLSProtocol = "legacy"
)

type TLSCipher string

type TLSECDHCurve string

type TLSDHEParams string

type PgBouncer struct {
	LogFile                 string             `ini:"logfile"`
	PidFile                 string             `ini:"pidfile"`
	ListenAddr              string             `ini:"listen_addr"`
	ListenPort              int                `ini:"listen_port"`
	UnixSocketDir           string             `ini:"unix_socket_dir"`
	UnixSocketMode          string             `ini:"unix_socket_mode"`
	UnixSocketGroup         string             `ini:"unix_socket_group"`
	User                    string             `ini:"user"`
	PoolMode                PoolMode           `ini:"pool_mode"`
	MaxClientConn           int                `ini:"max_client_conn"`
	DefaultPoolSize         int                `ini:"default_pool_size"`
	MinPoolSize             int                `ini:"min_pool_size"`
	ReservePoolSize         int                `ini:"reserve_pool_size"`
	ReservePoolTimeout      float64            `ini:"reserve_pool_timeout"`
	MaxDBConnections        int                `ini:"max_db_connections"`
	MaxUserConnections      int                `ini:"max_user_connections"`
	ServerRoundRobin        int                `ini:"server_round_robin"`
	TrackExtraParameters    []strutil.CIString `ini:"track_extra_parameters"`
	IgnoreStartupParameters []strutil.CIString `ini:"ignore_startup_parameters"`
	PeerID                  int                `ini:"peer_id"`
	DisablePQExec           int                `ini:"disable_pqexec"`
	ApplicationNameAddHost  int                `ini:"application_name_add_host"`
	ConfFile                string             `ini:"conffile"`
	ServiceName             string             `ini:"service_name"`
	StatsPeriod             int                `ini:"stats_period"`
	AuthType                string             `ini:"auth_type"`
	AuthHbaFile             string             `ini:"auth_hba_file"`
	AuthFile                AuthFile           `ini:"auth_file"`
	AuthUser                string             `ini:"auth_user"`
	AuthQuery               string             `ini:"auth_query"`
	AuthDbname              string             `ini:"auth_dbname"`
	Syslog                  string             `ini:"syslog"`
	SyslogIdent             string             `ini:"syslog_ident"`
	SyslogFacility          string             `ini:"syslog_facility"`
	LogConnections          int                `ini:"log_connections"`
	LogDisconnections       int                `ini:"log_disconnections"`
	LogPoolerErrors         int                `ini:"log_pooler_errors"`
	LogStats                int                `ini:"log_stats"`
	Verbose                 int                `ini:"verbose"`
	AdminUsers              []string           `ini:"auth_users"`
	StatsUsers              []string           `ini:"stats_users"`
	ServerResetQuery        string             `ini:"server_reset_query"`
	ServerResetQueryAlways  int                `ini:"server_reset_query_always"`
	ServerCheckDelay        float64            `ini:"server_check_delay"`
	ServerCheckQuery        string             `ini:"server_check_query"`
	ServerFastClose         int                `ini:"server_fast_close"`
	ServerLifetime          float64            `ini:"server_lifetime"`
	ServerIdleTimeout       float64            `ini:"server_idle_timeout"`
	ServerConnectTimeout    float64            `ini:"server_connect_timeout"`
	ServerLoginRetry        float64            `ini:"server_login_retry"`
	ClientLoginTimeout      float64            `ini:"client_login_timeout"`
	AutodbIdleTimeout       float64            `ini:"autodb_idle_timeout"`
	DnsMaxTtl               float64            `ini:"dns_max_ttl"`
	DnsNxdomainTtl          float64            `ini:"dns_nxdomain_ttl"`
	DnsZoneCheckPeriod      float64            `ini:"dns_zone_check_period"`
	ResolvConf              string             `ini:"resolv.conf"`
	ClientTLSSSLMode        bouncer.SSLMode    `ini:"client_tls_sslmode"`
	ClientTLSKeyFile        string             `ini:"client_tls_key_file"`
	ClientTLSCertFile       string             `ini:"client_tls_cert_file"`
	ClientTLSCaFile         string             `ini:"client_tls_ca_file"`
	ClientTLSProtocols      []TLSProtocol      `ini:"client_tls_protocols"`
	ClientTLSCiphers        []TLSCipher        `ini:"client_tls_ciphers"`
	ClientTLSECDHCurve      TLSECDHCurve       `ini:"client_tls_ecdhcurve"`
	ClientTLSDHEParams      TLSDHEParams       `ini:"client_tls_dheparams"`
	ServerTLSSSLMode        bouncer.SSLMode    `ini:"server_tls_sslmode"`
	ServerTLSCaFile         string             `ini:"server_tls_ca_file"`
	ServerTLSKeyFile        string             `ini:"server_tls_key_file"`
	ServerTLSCertFile       string             `ini:"server_tls_cert_file"`
	ServerTLSProtocols      []TLSProtocol      `ini:"server_tls_protocols"`
	ServerTLSCiphers        []TLSCipher        `ini:"server_tls_ciphers"`
	QueryTimeout            float64            `ini:"query_timeout"`
	QueryWaitTimeout        float64            `ini:"query_wait_timeout"`
	CancelWaitTimeout       float64            `ini:"cancel_wait_timeout"`
	ClientIdleTimeout       float64            `ini:"client_idle_timeout"`
	IdleTransactionTimeout  float64            `ini:"idle_transaction_timeout"`
	SuspendTimeout          float64            `ini:"suspend_timeout"`
	PktBuf                  int                `ini:"pkt_buf"`
	MaxPacketSize           int                `ini:"max_packet_size"`
	ListenBacklog           int                `ini:"listen_backlog"`
	SbufLoopcnt             int                `ini:"sbuf_loopcnt"`
	SoReuseport             int                `ini:"so_reuseport"`
	TcpDeferAccept          int                `ini:"tcp_defer_accept"`
	TcpSocketBuffer         int                `ini:"tcp_socket_buffer"`
	TcpKeepalive            int                `ini:"tcp_keepalive"`
	TcpKeepidle             int                `ini:"tcp_keepidle"`
	TcpKeepintvl            int                `ini:"tcp_keepintvl"`
	TcpUserTimeout          int                `ini:"tcp_user_timeout"`
}

type Database struct {
	DBName            string                      `ini:"dbname"`
	Host              string                      `ini:"host"`
	Port              int                         `ini:"port"`
	User              string                      `ini:"user"`
	Password          string                      `ini:"password"`
	AuthUser          string                      `ini:"auth_user"`
	PoolSize          int                         `ini:"pool_size"`
	MinPoolSize       int                         `ini:"min_pool_size"`
	ReservePool       int                         `ini:"reserve_pool"`
	ConnectQuery      string                      `ini:"connect_query"`
	PoolMode          PoolMode                    `ini:"pool_mode"`
	MaxDBConnections  int                         `ini:"max_db_connections"`
	AuthDBName        string                      `ini:"auth_dbname"`
	StartupParameters map[strutil.CIString]string `ini:"*"`
}

type User struct {
	PoolMode           PoolMode `ini:"pool_mode"`
	MaxUserConnections int      `ini:"max_user_connections"`
}

type Peer struct {
	Host     string `ini:"host"`
	Port     int    `ini:"port"`
	PoolSize int    `ini:"pool_size"`
}

type Config struct {
	PgBouncer PgBouncer           `ini:"pgbouncer"`
	Databases map[string]Database `ini:"databases"`
	Users     map[string]User     `ini:"users"`
	Peers     map[string]Peer     `ini:"peers"`
}

var Default = Config{
	PgBouncer: PgBouncer{
		ListenPort:         6432,
		UnixSocketDir:      "/tmp",
		UnixSocketMode:     "0777",
		PoolMode:           PoolModeSession,
		MaxClientConn:      100,
		DefaultPoolSize:    20,
		ReservePoolTimeout: 5.0,
		TrackExtraParameters: []strutil.CIString{
			strutil.MakeCIString("IntervalStyle"),
		},
		ServiceName:          "pgbouncer",
		StatsPeriod:          60,
		AuthQuery:            "SELECT usename, passwd FROM pg_shadow WHERE usename=$1",
		SyslogIdent:          "pgbouncer",
		SyslogFacility:       "daemon",
		LogConnections:       1,
		LogDisconnections:    1,
		LogPoolerErrors:      1,
		LogStats:             1,
		ServerResetQuery:     "DISCARD ALL",
		ServerCheckDelay:     30.0,
		ServerCheckQuery:     "select 1",
		ServerLifetime:       3600.0,
		ServerIdleTimeout:    600.0,
		ServerConnectTimeout: 15.0,
		ServerLoginRetry:     15.0,
		ClientLoginTimeout:   60.0,
		AutodbIdleTimeout:    3600.0,
		DnsMaxTtl:            15.0,
		DnsNxdomainTtl:       15.0,
		ClientTLSSSLMode:     bouncer.SSLModeDisable,
		ClientTLSProtocols: []TLSProtocol{
			TLSProtocolSecure,
		},
		ClientTLSCiphers: []TLSCipher{
			"fast",
		},
		ClientTLSECDHCurve: "auto",
		ServerTLSSSLMode:   bouncer.SSLModePrefer,
		ServerTLSProtocols: []TLSProtocol{
			TLSProtocolSecure,
		},
		ServerTLSCiphers: []TLSCipher{
			"fast",
		},
		QueryWaitTimeout:  120.0,
		CancelWaitTimeout: 10.0,
		SuspendTimeout:    10.0,
		PktBuf:            4096,
		MaxPacketSize:     2147483647,
		ListenBacklog:     128,
		SbufLoopcnt:       5,
		TcpDeferAccept:    1,
		TcpKeepalive:      1,
	},
}

func Load(config string) (Config, error) {
	conf, err := ini.ReadFile(config)
	if err != nil {
		return Config{}, err
	}

	var c = Default
	err = ini.Unmarshal(conf, &c)
	return c, err
}

func (T *Config) ListenAndServe() error {
	trackedParameters := append([]strutil.CIString{
		strutil.MakeCIString("client_encoding"),
		strutil.MakeCIString("datestyle"),
		strutil.MakeCIString("timezone"),
		strutil.MakeCIString("standard_conforming_strings"),
		strutil.MakeCIString("application_name"),
	}, T.PgBouncer.TrackExtraParameters...)

	allowedStartupParameters := append(trackedParameters, T.PgBouncer.IgnoreStartupParameters...)

	var sslConfig *tls.Config
	if T.PgBouncer.ClientTLSCertFile != "" && T.PgBouncer.ClientTLSKeyFile != "" {
		certificate, err := tls.LoadX509KeyPair(T.PgBouncer.ClientTLSCertFile, T.PgBouncer.ClientTLSKeyFile)
		if err != nil {
			return err
		}
		sslConfig = &tls.Config{
			Certificates: []tls.Certificate{
				certificate,
			},
		}
	}

	acceptOptions := frontends.AcceptOptions{
		SSLRequired:           T.PgBouncer.ClientTLSSSLMode.IsRequired(),
		SSLConfig:             sslConfig,
		AllowedStartupOptions: allowedStartupParameters,
	}

	pools, err := NewPools(T)
	if err != nil {
		return err
	}

	var bank flip.Bank

	if T.PgBouncer.ListenAddr != "" {
		bank.Queue(func() error {
			listenAddr := T.PgBouncer.ListenAddr
			if listenAddr == "*" {
				listenAddr = ""
			}

			listen := net.JoinHostPort(listenAddr, strconv.Itoa(T.PgBouncer.ListenPort))

			log.Printf("listening on %s", listen)

			return gat.ListenAndServe("tcp", listen, acceptOptions, pools)
		})
	}

	// listen on unix socket
	bank.Queue(func() error {
		dir := T.PgBouncer.UnixSocketDir
		port := T.PgBouncer.ListenPort

		if !strings.HasSuffix(dir, "/") {
			dir = dir + "/"
		}
		dir = dir + ".s.PGSQL." + strconv.Itoa(port)

		log.Printf("listening on unix:%s", dir)

		return gat.ListenAndServe("unix", dir, acceptOptions, pools)
	})

	return bank.Wait()
}
