package pgbouncer_spilo

type Config struct {
	Host          string `json:"host"`
	Port          int    `json:"port"`
	User          string `json:"user"`
	Schema        string `json:"schema"`
	Password      string `json:"password"`
	Mode          string `json:"mode"`
	DefaultSize   int    `json:"default_size"`
	MinSize       int    `json:"min_size"`
	ReserveSize   int    `json:"reserve_size"`
	MaxClientConn int    `json:"max_client_conn"`
	MaxDBConn     int    `json:"max_db_conn"`
}
