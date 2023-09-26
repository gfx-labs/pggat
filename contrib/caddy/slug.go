package caddy

import (
	"strconv"
	"strings"
)

type ServerSlug struct {
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Database string `json:"database,omitempty"`
}

func (T *ServerSlug) FromString(str string) error {
	userPassword, hostPortDatabase, ok := strings.Cut(str, "@")
	if !ok {
		hostPortDatabase = userPassword
		userPassword = ""
	}
	T.User, T.Password, ok = strings.Cut(userPassword, ":")
	var hostPort string
	hostPort, T.Database, ok = strings.Cut(hostPortDatabase, "/")
	var port string
	T.Host, port, ok = strings.Cut(hostPort, ":")
	var err error
	T.Port, err = strconv.Atoi(port)
	return err
}
