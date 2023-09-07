package metrics

import "github.com/google/uuid"

type Pool struct {
	Servers map[uuid.UUID]Conn
	Clients map[uuid.UUID]Conn
}
