package query_router

import (
	"errors"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/parse"
	"gfx.cafe/gfx/pggat/lib/util/cmux"
	"strconv"
	"unicode"
	"unicode/utf8"
)

type QueryRouter struct {
	router cmux.Mux[gat.Client, error]
}

var DefaultRouter = func() *QueryRouter {
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
	return &QueryRouter{
		router: r,
	}
}()

// Try to infer the server role to try to  connect to
// based on the contents of the query.
// note that the user needs to be checked to see if they are allowed to access.
// TODO: implement
func (r *QueryRouter) InferRole(query string) (config.ServerRole, error) {
	var active_role config.ServerRole
	// by default it will hit a primary (for now)
	active_role = config.SERVERROLE_PRIMARY
	//// ok now parse the query
	//wk := &walk.AstWalker{
	//	Fn: func(ctx, node any) (stop bool) {
	//		switch n := node.(type) {
	//		case *tree.Update, *tree.UpdateExpr,
	//			*tree.BeginTransaction, *tree.CommitTransaction, *tree.RollbackTransaction,
	//			*tree.SetTransaction, *tree.ShowTransactionStatus, *tree.Delete, *tree.Insert:
	//			//
	//			active_role = config.SERVERROLE_PRIMARY
	//			return true
	//		default:
	//			_ = n
	//		}
	//		return false
	//	},
	//}
	//stmts, err := parser.Parse(query)
	//if err != nil {
	//	log.Println("failed to parse (%query), assuming primary required", err)
	//	return config.SERVERROLE_PRIMARY, nil
	//}
	//_, err = wk.Walk(stmts, nil)
	//if err != nil {
	//	return config.SERVERROLE_PRIMARY, err
	//}
	return active_role, nil
}

func (r *QueryRouter) TryHandle(client gat.Client, query string) (handled bool, err error) {
	var parsed []parse.Command
	parsed, err = parse.Parse(query)
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
	err, handled = r.router.Call(client, append([]string{parsed[0].Command}, parsed[0].Arguments...))
	return
}
