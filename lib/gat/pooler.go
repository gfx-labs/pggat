package gat

import (
	"net"

	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/unterminate"
	"pggat2/lib/util/maps"
	"pggat2/lib/zap"
)

var DefaultParameterStatus = map[string]string{
	// TODO(garet) we should just get these from the first server connection
	"DateStyle":                     "ISO, MDY",
	"IntervalStyle":                 "postgres",
	"TimeZone":                      "America/Chicago",
	"application_name":              "",
	"client_encoding":               "UTF8",
	"default_transaction_read_only": "off",
	"in_hot_standby":                "off",
	"integer_datetimes":             "on",
	"is_superuser":                  "on",
	"server_encoding":               "UTF8",
	"server_version":                "14.5",
	"session_authorization":         "postgres",
	"standard_conforming_strings":   "on",
}

type Pooler struct {
	users maps.RWLocked[string, *User]
}

func NewPooler() *Pooler {
	return &Pooler{}
}

func (T *Pooler) AddUser(name string, user *User) {
	T.users.Store(name, user)
}

func (T *Pooler) RemoveUser(name string) {
	T.users.Delete(name)
}

func (T *Pooler) GetUser(name string) *User {
	user, _ := T.users.Load(name)
	return user
}

func (T *Pooler) Serve(client zap.ReadWriter) {
	client = interceptor.NewInterceptor(
		client,
		unterminate.Unterminate,
	)

	username, database, ok := frontends.Accept(client, func(username string) string {
		user := T.GetUser(username)
		if user == nil {
			return ""
		}
		return user.GetPassword()
	}, DefaultParameterStatus)
	if !ok {
		return
	}

	user := T.GetUser(username)
	if user == nil {
		return
	}

	pool := user.GetPool(database)
	if pool == nil {
		return
	}

	pool.Serve(client)
}

func (T *Pooler) ListenAndServe(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}
		go T.Serve(zap.CombinedReadWriter{
			Reader: zap.IOReader{Reader: conn},
			Writer: zap.IOWriter{Writer: conn},
		})
	}
}
