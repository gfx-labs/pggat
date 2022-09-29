package query_router

import (
	"errors"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/util/cmux"
	"gfx.cafe/ghalliday1/pg3p"
	"gfx.cafe/ghalliday1/pg3p/lex"
	"strconv"
	"unicode"
	"unicode/utf8"
)

type QueryRouter struct {
	router cmux.Mux[gat.Client, error]
	parser *pg3p.Parser
	c      *config.Pool
}

var defaultMux = func() *cmux.MapMux[gat.Client, error] {
	r := cmux.NewMapMux[gat.Client, error]()
	r.Register([]string{"set", "sharding", "key", "to"}, func(_ gat.Client, args []string) error {
		return nil
	})
	r.Register([]string{"set", "shard", "to"}, func(client gat.Client, args []string) error {
		if len(args) == 0 {
			return errors.New("expected at least one argument")
		}

		v := args[0]
		r, l := utf8.DecodeRuneInString(v)
		if !unicode.IsNumber(r) {
			if len(v)-l <= l {
				return errors.New("malformed input")
			}
			v = v[l : len(v)-l]
		}

		if v == "any" {
			client.UnsetRequestedShard()
			return nil
		}

		num, err := strconv.Atoi(v)
		if err != nil {
			return err
		}

		client.SetRequestedShard(num)

		return nil
	})
	r.Register([]string{"show", "shard"}, func(_ gat.Client, args []string) error {
		return nil
	})
	r.Register([]string{"set", "server", "role", "to"}, func(_ gat.Client, args []string) error {
		return nil
	})
	r.Register([]string{"show", "server", "role"}, func(_ gat.Client, args []string) error {
		return nil
	})
	r.Register([]string{"set", "primary", "reads", "to"}, func(_ gat.Client, args []string) error {
		return nil
	})
	r.Register([]string{"show", "primary", "reads"}, func(_ gat.Client, args []string) error {
		return nil
	})
	return r
}()

func DefaultRouter(c *config.Pool) *QueryRouter {
	return &QueryRouter{
		router: defaultMux,
		parser: pg3p.NewParser(),
		c:      c,
	}
}

// Try to infer the server role to try to  connect to
// based on the contents of the query.
// note that the user needs to be checked to see if they are allowed to access.
func (r *QueryRouter) InferRole(query string) (config.ServerRole, error) {
	// if we don't want to parse queries, route them to primary
	if !r.c.QueryParserEnabled {
		return config.SERVERROLE_PRIMARY, nil
	}
	// parse the query
	tokens := r.parser.Lex(query)
	depth := 0
	for _, token := range tokens {
		switch token.Token {
		case lex.KeywordUpdate,
			lex.KeywordDelete,
			lex.KeywordInsert,
			lex.KeywordDrop,
			lex.KeywordCreate,
			lex.KeywordTruncate,
			lex.KeywordVacuum,
			lex.KeywordAnalyze:
			return config.SERVERROLE_PRIMARY, nil
		case lex.KeywordBegin:
			depth += 1
		case lex.KeywordCase:
			if depth > 0 {
				depth += 1
			}
		case lex.KeywordEnd:
			if depth > 0 {
				depth -= 1
			}
		}
	}
	if depth > 0 {
		return config.SERVERROLE_PRIMARY, nil
	}
	return config.SERVERROLE_REPLICA, nil
}

func (r *QueryRouter) TryHandle(client gat.Client, query string) (handled bool, err error) {
	if !r.c.QueryParserEnabled {
		return
	}
	/*var parsed parser.Statements TODO
	parsed, err = parser.Parse(query)
	if err != nil {
		return
	}
	if len(parsed) == 0 {
		// send empty query
		err = client.Send(new(protocol.EmptyQueryResponse))
		return true, err
	}
	if len(parsed) != 1 {
		return
	}
	err, handled = r.router.Call(client, append([]string{parsed[0].Cmd}, parsed[0].Arguments...))
	*/
	return
}
