package admin

import (
	"errors"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/parse"
	"strings"
)

// The admin database, implemented through the gat.Pool interface, allowing it to be added to any existing Gat

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

const DataType_String = 25
const DataType_Int64 = 20
const DataType_Float64 = 701

func getServerInfo(g gat.Gat) []*protocol.ParameterStatus {
	return []*protocol.ParameterStatus{
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "application_name",
				Value:     "",
			},
		},
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "client_encoding",
				Value:     "UTF8",
			},
		},
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "server_encoding",
				Value:     "UTF8",
			},
		},
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "server_encoding",
				Value:     "UTF8",
			},
		},
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "server_version",
				Value:     g.Version(),
			},
		},
		{
			Fields: protocol.FieldsParameterStatus{
				Parameter: "DataStyle",
				Value:     "ISO, MDY",
			},
		},
	}
}

func getAdminUser(g gat.Gat) *config.User {
	conf := g.Config()
	return &config.User{
		Name:     conf.General.AdminUsername,
		Password: conf.General.AdminPassword,

		Role:             config.USERROLE_ADMIN,
		PoolSize:         1,
		StatementTimeout: 0,
	}
}

type Pool struct {
	gat      gat.Gat
	connPool *ConnectionPool
}

func NewPool(g gat.Gat) *Pool {
	out := &Pool{
		gat: g,
	}
	out.connPool = &ConnectionPool{
		pool: out,
	}
	return out
}

func (p *Pool) GetUser(name string) (*config.User, error) {
	u := getAdminUser(p.gat)
	if name != u.Name {
		return nil, fmt.Errorf("%w: %s", gat.UserNotFound, name)
	}
	return u, nil
}

func (p *Pool) GetRouter() gat.QueryRouter {
	return nil
}

func (p *Pool) WithUser(name string) (gat.ConnectionPool, error) {
	conf := p.gat.Config()
	if name != conf.General.AdminUsername {
		return nil, fmt.Errorf("%w: %s", gat.UserNotFound, name)
	}
	return p.connPool, nil
}

func (p *Pool) ConnectionPools() []gat.ConnectionPool {
	return []gat.ConnectionPool{
		p.connPool,
	}
}

func (p *Pool) Stats() *gat.PoolStats {
	return nil // TODO
}

func (p *Pool) EnsureConfig(c *config.Pool) {
	// TODO
}

var _ gat.Pool = (*Pool)(nil)

type ConnectionPool struct {
	pool *Pool
}

func (c *ConnectionPool) GetUser() *config.User {
	return getAdminUser(c.pool.gat)
}

func (c *ConnectionPool) GetServerInfo() []*protocol.ParameterStatus {
	return getServerInfo(c.pool.gat)
}

func (c *ConnectionPool) Shards() []gat.Shard {
	// this db is within gat, there are no shards
	return nil
}

func (c *ConnectionPool) EnsureConfig(conf *config.Pool) {
	// TODO
}

func (c *ConnectionPool) Describe(ctx context.Context, client gat.Client, describe *protocol.Describe) error {
	return errors.New("describe not implemented")
}

func (c *ConnectionPool) Execute(ctx context.Context, client gat.Client, execute *protocol.Execute) error {
	return errors.New("execute not implemented")
}

func (c *ConnectionPool) SimpleQuery(ctx context.Context, client gat.Client, query string) error {
	parsed, err := parse.Parse(query)
	if err != nil {
		return err
	}
	for _, cmd := range parsed {
		switch strings.ToLower(cmd.Command) {
		case "show":
			if len(cmd.Arguments) < 1 {
				return errors.New("usage: show [item]")
			}

			switch strings.ToLower(cmd.Arguments[0]) {
			case "stats":
				rowDesc := new(protocol.RowDescription)
				rowDesc.Fields.Fields = []protocol.FieldsRowDescriptionFields{
					{
						Name:         "database",
						DataType:     DataType_String,
						DataTypeSize: -1,
						TypeModifier: -1,
					},
					{
						Name:         "total_xact_count",
						DataType:     DataType_Int64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "total_query_count",
						DataType:     DataType_Int64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "total_received",
						DataType:     DataType_Int64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "total_sent",
						DataType:     DataType_Int64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "total_xact_time",
						DataType:     DataType_Int64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "total_query_time",
						DataType:     DataType_Int64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "total_wait_time",
						DataType:     DataType_Int64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "avg_xact_count",
						DataType:     DataType_Float64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "avg_query_count",
						DataType:     DataType_Float64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "avg_recv",
						DataType:     DataType_Float64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "avg_sent",
						DataType:     DataType_Float64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "avg_xact_time",
						DataType:     DataType_Float64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "avg_query_time",
						DataType:     DataType_Float64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
					{
						Name:         "avg_wait_time",
						DataType:     DataType_Float64,
						DataTypeSize: 8,
						TypeModifier: -1,
					},
				}
				err = client.Send(rowDesc)
				if err != nil {
					return err
				}
				for name, pool := range c.pool.gat.Pools() {
					stats := pool.Stats()
					if stats == nil {
						continue
					}
					row := new(protocol.DataRow)
					row.Fields.Columns = []protocol.FieldsDataRowColumns{
						{
							[]byte(name),
						},
						{
							[]byte(fmt.Sprintf("%d", stats.TotalXactCount())),
						},
						{
							[]byte(fmt.Sprintf("%d", stats.TotalQueryCount())),
						},
						{
							[]byte(fmt.Sprintf("%d", stats.TotalReceived())),
						},
						{
							[]byte(fmt.Sprintf("%d", stats.TotalSent())),
						},
						{
							[]byte(fmt.Sprintf("%d", stats.TotalXactTime())),
						},
						{
							[]byte(fmt.Sprintf("%d", stats.TotalQueryTime())),
						},
						{
							[]byte(fmt.Sprintf("%d", stats.TotalWaitTime())),
						},
						{
							[]byte(fmt.Sprintf("%f", stats.AvgXactCount())),
						},
						{
							[]byte(fmt.Sprintf("%f", stats.AvgQueryCount())),
						},
						{
							[]byte(fmt.Sprintf("%f", stats.AvgRecv())),
						},
						{
							[]byte(fmt.Sprintf("%f", stats.AvgSent())),
						},
						{
							[]byte(fmt.Sprintf("%f", stats.AvgXactTime())),
						},
						{
							[]byte(fmt.Sprintf("%f", stats.AvgQueryTime())),
						},
						{
							[]byte(fmt.Sprintf("%f", stats.AvgWaitTime())),
						},
					}
					err = client.Send(row)
					if err != nil {
						return err
					}
				}
				done := new(protocol.CommandComplete)
				done.Fields.Data = cmd.Command
				err = client.Send(done)
				if err != nil {
					return err
				}
			default:
				return errors.New("unknown command")
			}
		case "pause":
		case "disable":
		case "enable":
		case "reconnect":
		case "kill":
		case "suspend":
		case "resume":
		case "shutdown":
		case "reload":
		case "wait_close":
		case "set":
		default:
			return errors.New("unknown command")
		}
	}
	return nil
}

func (c *ConnectionPool) Transaction(ctx context.Context, client gat.Client, query string) error {
	return errors.New("transactions not implemented")
}

func (c *ConnectionPool) CallFunction(ctx context.Context, client gat.Client, payload *protocol.FunctionCall) error {
	return errors.New("functions not implemented")
}

var _ gat.ConnectionPool = (*ConnectionPool)(nil)
