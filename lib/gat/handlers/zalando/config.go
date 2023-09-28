package zalando

type Config struct {
	PGHost              string `json:"pg_host"`
	PGPort              int    `json:"pg_port"`
	PGUser              string `json:"pg_user"`
	PGSchema            string `json:"pg_schema"`
	PGPassword          string `json:"pg_password"`
	PoolerPort          int    `json:"pooler_port"`
	PoolerMode          string `json:"pooler_mode"`
	PoolerDefaultSize   int    `json:"pooler_default_size"`
	PoolerMinSize       int    `json:"pooler_min_size"`
	PoolerReserveSize   int    `json:"pooler_reserve_size"`
	PoolerMaxClientConn int    `json:"pooler_max_client_conn"`
	PoolerMaxDBConn     int    `json:"pooler_max_db_conn"`
}
