package admin

import (
	"context"
	"errors"
	"gfx.cafe/gfx/pggat/lib/util/gatutil"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/parse"
	"gfx.cafe/gfx/pggat/lib/util/cmux"
)

// The admin database, implemented through the gat.Database interface, allowing it to be added to any existing Gat

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
				Value:     g.GetVersion(),
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
	conf := g.GetConfig()
	return &config.User{
		Name:     conf.General.AdminUsername,
		Password: conf.General.AdminPassword,

		Role:             config.USERROLE_ADMIN,
		PoolSize:         1,
		StatementTimeout: 0,
	}
}

type Database struct {
	gat      gat.Gat
	connPool *Pool

	r cmux.Mux[gat.Client, error]
}

func New(g gat.Gat) *Database {
	out := &Database{
		gat: g,
	}
	out.connPool = &Pool{
		database: out,
	}
	out.r = cmux.NewMapMux[gat.Client, error]()

	out.r.Register([]string{"show", "servers"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "clients"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "pools"}, func(c gat.Client, _ []string) error {
		table := gatutil.Table{
			Header: gatutil.TableHeader{
				Columns: []gatutil.TableHeaderColumn{
					{
						Name: "Table name",
						Type: gatutil.Text{},
					},
					{
						Name: "Min latency",
						Type: gatutil.Float64{},
					},
					{
						Name: "Max latency",
						Type: gatutil.Float64{},
					},
					{
						Name: "Avg latency",
						Type: gatutil.Float64{},
					},
					{
						Name: "Request Count",
						Type: gatutil.Int64{},
					},
				},
			},
			Rows: []gatutil.TableRow{
				{
					Columns: []any{
						"Test",
						float64(1.0),
						float64(2.0),
						float64(3.0),
						int64(123),
					},
				},
			},
		}
		return table.Send(c)
	})
	out.r.Register([]string{"show", "lists"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "users"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "databases"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "fds"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "sockets"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "active_sockets"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "config"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "mem"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "dns_hosts"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "dns_zones"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "version"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"pause"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"disable"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"enable"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"reconnect"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"kill"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"suspend"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"resume"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"shutdown"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"reload"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"wait_close"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"set"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	return out
}

func (p *Database) GetUser(name string) *config.User {
	u := getAdminUser(p.gat)
	if name != u.Name {
		return nil
	}
	return u
}

func (p *Database) GetRouter() gat.QueryRouter {
	return nil
}

func (p *Database) GetName() string {
	return "pggat"
}

func (p *Database) WithUser(name string) gat.Pool {
	conf := p.gat.GetConfig()
	if name != conf.General.AdminUsername {
		return nil
	}
	return p.connPool
}

func (p *Database) GetPools() []gat.Pool {
	return []gat.Pool{
		p.connPool,
	}
}

func (p *Database) EnsureConfig(c *config.Pool) {
	// TODO
}

var _ gat.Database = (*Database)(nil)

type Pool struct {
	database *Database
}

func (c *Pool) GetUser() *config.User {
	return getAdminUser(c.database.gat)
}

func (c *Pool) GetServerInfo(_ gat.Client) []*protocol.ParameterStatus {
	return getServerInfo(c.database.gat)
}

func (c *Pool) GetDatabase() gat.Database {
	return c.database
}

func (c *Pool) EnsureConfig(conf *config.Pool, u *config.User) {
	// TODO
}

func (c *Pool) OnDisconnect(_ gat.Client) {}

func (c *Pool) Describe(ctx context.Context, client gat.Client, describe *protocol.Describe) error {
	return errors.New("describe not implemented")
}

func (c *Pool) Execute(ctx context.Context, client gat.Client, execute *protocol.Execute) error {
	return errors.New("execute not implemented")
}

func (c *Pool) SimpleQuery(ctx context.Context, client gat.Client, query string) error {
	parsed, err := parse.Parse(query)
	if err != nil {
		return err
	}
	if len(parsed) == 0 {
		return client.Send(new(protocol.EmptyQueryResponse))
	}
	for _, cmd := range parsed {
		var matched bool
		err, matched = c.database.r.Call(client, append([]string{cmd.Command}, cmd.Arguments...))
		if !matched {
			return errors.New("unknown command")
		}
		if err != nil {
			return err
		}
		done := new(protocol.CommandComplete)
		done.Fields.Data = cmd.Command
		err = client.Send(done)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Pool) Transaction(ctx context.Context, client gat.Client, query string) error {
	return errors.New("transactions not implemented")
}

func (c *Pool) CallFunction(ctx context.Context, client gat.Client, payload *protocol.FunctionCall) error {
	return errors.New("functions not implemented")
}

var _ gat.Pool = (*Pool)(nil)
