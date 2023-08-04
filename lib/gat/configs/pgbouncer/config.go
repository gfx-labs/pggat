package pgbouncer

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

type SSLMode string

const (
	SSLModeDisable    SSLMode = "disable"
	SSLModeAllow      SSLMode = "allow"
	SSLModePrefer     SSLMode = "prefer"
	SSLModeRequire    SSLMode = "require"
	SSLModeVerifyCa   SSLMode = "verify-ca"
	SSLModeVerifyFull SSLMode = "verify-full"
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

type PgBouncer struct {
	LogFile                 string        `ini:"logfile"`
	PidFile                 string        `ini:"pidfile"`
	ListenAddr              string        `ini:"listen_addr"`
	ListenPort              string        `ini:"listen_port"`
	UnixSocketDir           string        `ini:"unix_socket_dir"`
	UnixSocketMode          string        `ini:"unix_socket_mode"`
	UnixSocketGroup         string        `ini:"unix_socket_group"`
	User                    string        `ini:"user"`
	PoolMode                PoolMode      `ini:"pool_mode"`
	MaxClientConn           int           `ini:"max_client_conn"`
	DefaultPoolSize         int           `ini:"default_pool_size"`
	MinPoolSize             int           `ini:"min_pool_size"`
	ReservePoolSize         int           `ini:"reserve_pool_size"`
	ReservePoolTimeout      float64       `ini:"reserve_pool_timeout"`
	MaxDBConnections        int           `ini:"max_db_connections"`
	MaxUserConnections      int           `ini:"max_user_connections"`
	ServerRoundRobin        int           `ini:"server_round_robin"`
	TrackExtraParameters    []string      `ini:"track_extra_parameters"`
	IgnoreStartupParameters []string      `ini:"ignore_startup_parameters"`
	PeerID                  int           `ini:"peer_id"`
	DisablePQExec           int           `ini:"disable_pqexec"`
	ApplicationNameAddHost  int           `ini:"application_name_add_host"`
	ConfFile                string        `ini:"conffile"`
	ServiceName             string        `ini:"service_name"`
	StatsPeriod             int           `ini:"stats_period"`
	AuthType                string        `ini:"auth_type"`
	AuthHbaFile             string        `ini:"auth_hba_file"`
	AuthFile                string        `ini:"auth_file"`
	AuthUser                string        `ini:"auth_user"`
	AuthQuery               string        `ini:"auth_query"`
	AuthDbname              string        `ini:"auth_dbname"`
	Syslog                  string        `ini:"syslog"`
	SyslogIdent             string        `ini:"syslog_ident"`
	SyslogFacility          string        `ini:"syslog_facility"`
	LogConnections          int           `ini:"log_connections"`
	LogDisconnections       int           `ini:"log_disconnections"`
	LogPoolerErrors         int           `ini:"log_pooler_errors"`
	LogStats                int           `ini:"log_stats"`
	Verbose                 int           `ini:"verbose"`
	AdminUsers              []string      `ini:"auth_users"`
	StatsUsers              []string      `ini:"stats_users"`
	ServerResetQuery        string        `ini:"server_reset_query"`
	ServerResetQueryAlways  int           `ini:"server_reset_query_always"`
	ServerCheckDelay        float64       `ini:"server_check_delay"`
	ServerCheckQuery        string        `ini:"server_check_query"`
	ServerFastClose         int           `ini:"server_fast_close"`
	ServerLifetime          float64       `ini:"server_lifetime"`
	ServerIdleTimeout       float64       `ini:"server_idle_timeout"`
	ServerConnectTimeout    float64       `ini:"server_connect_timeout"`
	ServerLoginRetry        float64       `ini:"server_login_retry"`
	ClientLoginTimeout      float64       `ini:"client_login_timeout"`
	AutodbIdleTimeout       float64       `ini:"autodb_idle_timeout"`
	DnsMaxTtl               float64       `ini:"dns_max_ttl"`
	DnsNxdomainTtl          float64       `ini:"dns_nxdomain_ttl"`
	DnsZoneCheckPeriod      float64       `ini:"dns_zone_check_period"`
	ResolvConf              string        `ini:"resolv.conf"`
	ClientTLSSSLMode        SSLMode       `ini:"client_tls_sslmode"`
	ClientTLSKeyFile        string        `ini:"client_tls_key_file"`
	ClientTLSCertFile       string        `ini:"client_tls_cert_file"`
	ClientTLSCaFile         string        `ini:"client_tls_ca_file"`
	ClientTLSProtocols      []TLSProtocol `ini:"client_tls_protocols"`
}

type Config struct {
	PgBouncer PgBouncer         `ini:"pgbouncer"`
	Databases map[string]string `ini:"databases"`
	Users     map[string]string `ini:"users"`
}

func Test() {

}
