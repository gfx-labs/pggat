package admin

import (
	"errors"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/parse"
	"gfx.cafe/gfx/pggat/lib/util/cmux"
	"time"
)

// The admin database, implemented through the gat.Database interface, allowing it to be added to any existing Gat

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
	out.r.Register([]string{"show", "stats_totals"}, func(client gat.Client, _ []string) error {
		return out.showStats(client, true, false)
	})
	out.r.Register([]string{"show", "stats_averages"}, func(client gat.Client, _ []string) error {
		return out.showStats(client, false, true)
	})
	out.r.Register([]string{"show", "stats"}, func(client gat.Client, _ []string) error {
		return out.showStats(client, true, true)
	})
	out.r.Register([]string{"show", "totals"}, func(client gat.Client, _ []string) error {
		return out.showTotals(client)
	})
	out.r.Register([]string{"show", "servers"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "clients"}, func(_ gat.Client, _ []string) error {
		return nil
	})
	out.r.Register([]string{"show", "pools"}, func(_ gat.Client, _ []string) error {
		return nil
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

func (p *Database) showStats(client gat.Client, totals, averages bool) error {
	rowDesc := new(protocol.RowDescription)
	rowDesc.Fields.Fields = []protocol.FieldsRowDescriptionFields{
		{
			Name:         "database",
			DataType:     DataType_String,
			DataTypeSize: -1,
			TypeModifier: -1,
		},
	}
	if totals {
		rowDesc.Fields.Fields = append(rowDesc.Fields.Fields,
			protocol.FieldsRowDescriptionFields{
				Name:         "total_xact_count",
				DataType:     DataType_Int64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "total_query_count",
				DataType:     DataType_Int64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "total_received",
				DataType:     DataType_Int64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "total_sent",
				DataType:     DataType_Int64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "total_xact_time",
				DataType:     DataType_Int64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "total_query_time",
				DataType:     DataType_Int64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "total_wait_time",
				DataType:     DataType_Int64,
				DataTypeSize: 8,
				TypeModifier: -1,
			})
	}
	if averages {
		rowDesc.Fields.Fields = append(rowDesc.Fields.Fields,
			protocol.FieldsRowDescriptionFields{
				Name:         "avg_xact_count",
				DataType:     DataType_Float64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "avg_query_count",
				DataType:     DataType_Float64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "avg_recv",
				DataType:     DataType_Float64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "avg_sent",
				DataType:     DataType_Float64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "avg_xact_time",
				DataType:     DataType_Float64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "avg_query_time",
				DataType:     DataType_Float64,
				DataTypeSize: 8,
				TypeModifier: -1,
			},
			protocol.FieldsRowDescriptionFields{
				Name:         "avg_wait_time",
				DataType:     DataType_Float64,
				DataTypeSize: 8,
				TypeModifier: -1,
			})
	}
	err := client.Send(rowDesc)
	if err != nil {
		return err
	}
	for name, pl := range p.gat.GetDatabases() {
		stats := pl.GetStats()
		if stats == nil {
			continue
		}
		row := new(protocol.DataRow)
		row.Fields.Columns = []protocol.FieldsDataRowColumns{
			{
				[]byte(name),
			},
		}
		if totals {
			row.Fields.Columns = append(row.Fields.Columns,
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%d", stats.TotalXactCount())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%d", stats.TotalQueryCount())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%d", stats.TotalReceived())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%d", stats.TotalSent())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%d", stats.TotalXactTime())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%d", stats.TotalQueryTime())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%d", stats.TotalWaitTime())),
				})
		}
		if averages {
			row.Fields.Columns = append(row.Fields.Columns,
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%f", stats.AvgXactCount())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%f", stats.AvgQueryCount())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%f", stats.AvgRecv())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%f", stats.AvgSent())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%f", stats.AvgXactTime())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%f", stats.AvgQueryTime())),
				},
				protocol.FieldsDataRowColumns{
					[]byte(fmt.Sprintf("%f", stats.AvgWaitTime())),
				})
		}
		err = client.Send(row)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Database) showTotals(client gat.Client) error {
	rowDesc := new(protocol.RowDescription)
	rowDesc.Fields.Fields = []protocol.FieldsRowDescriptionFields{
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
	err := client.Send(rowDesc)
	if err != nil {
		return err
	}

	var totalXactCount, totalQueryCount, totalWaitCount, totalReceived, totalSent, totalXactTime, totalQueryTime, totalWaitTime int64
	var alive time.Duration

	for _, pl := range p.gat.GetDatabases() {
		stats := pl.GetStats()
		if stats == nil {
			continue
		}
		totalXactCount += stats.TotalXactCount()
		totalQueryCount += stats.TotalQueryCount()
		totalWaitCount += stats.TotalWaitCount()
		totalReceived += stats.TotalReceived()
		totalSent += stats.TotalSent()
		totalXactTime += stats.TotalXactTime()
		totalQueryTime += stats.TotalQueryTime()
		totalWaitTime += stats.TotalWaitTime()

		active := stats.TimeActive()
		if active > alive {
			alive = active
		}
	}

	avgXactCount := float64(totalXactCount) / alive.Seconds()
	avgQueryCount := float64(totalQueryCount) / alive.Seconds()
	avgReceive := float64(totalReceived) / alive.Seconds()
	avgSent := float64(totalSent) / alive.Seconds()
	avgXactTime := float64(totalXactTime) / float64(totalXactCount)
	avgQueryTime := float64(totalQueryTime) / float64(totalQueryCount)
	avgWaitTime := float64(totalWaitTime) / float64(totalWaitCount)

	row := new(protocol.DataRow)
	row.Fields.Columns = []protocol.FieldsDataRowColumns{
		{
			[]byte(fmt.Sprintf("%d", totalXactCount)),
		},
		{
			[]byte(fmt.Sprintf("%d", totalQueryCount)),
		},
		{
			[]byte(fmt.Sprintf("%d", totalReceived)),
		},
		{
			[]byte(fmt.Sprintf("%d", totalSent)),
		},
		{
			[]byte(fmt.Sprintf("%d", totalXactTime)),
		},
		{
			[]byte(fmt.Sprintf("%d", totalQueryTime)),
		},
		{
			[]byte(fmt.Sprintf("%d", totalWaitTime)),
		},
		{
			[]byte(fmt.Sprintf("%f", avgXactCount)),
		},
		{
			[]byte(fmt.Sprintf("%f", avgQueryCount)),
		},
		{
			[]byte(fmt.Sprintf("%f", avgReceive)),
		},
		{
			[]byte(fmt.Sprintf("%f", avgSent)),
		},
		{
			[]byte(fmt.Sprintf("%f", avgXactTime)),
		},
		{
			[]byte(fmt.Sprintf("%f", avgQueryTime)),
		},
		{
			[]byte(fmt.Sprintf("%f", avgWaitTime)),
		},
	}

	return client.Send(row)
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

func (p *Database) GetStats() *gat.PoolStats {
	return nil
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

func (c *Pool) EnsureConfig(conf *config.Pool) {
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
