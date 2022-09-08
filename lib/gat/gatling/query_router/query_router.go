package query_router

import (
	"regexp"

	"git.tuxpa.in/a/zlog/log"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/util/go/lambda"

	"github.com/auxten/postgresql-parser/pkg/sql/parser"
	"github.com/auxten/postgresql-parser/pkg/sql/sem/tree"
	"github.com/auxten/postgresql-parser/pkg/walk"
)

var CustomSqlRegex = lambda.MapV([]string{
	"(?i)^ *SET SHARDING KEY TO '?([0-9]+)'? *;? *$",
	"(?i)^ *SET SHARD TO '?([0-9]+|ANY)'? *;? *$",
	"(?i)^ *SHOW SHARD *;? *$",
	"(?i)^ *SET SERVER ROLE TO '(PRIMARY|REPLICA|ANY|AUTO|DEFAULT)' *;? *$",
	"(?i)^ *SHOW SERVER ROLE *;? *$",
	"(?i)^ *SET PRIMARY READS TO '?(on|off|default)'? *;? *$",
	"(?i)^ *SHOW PRIMARY READS *;? *$",
}, regexp.MustCompile)

type Command interface {
}

var _ []Command = []Command{
	&CommandSetShardingKey{},
	&CommandSetShard{},
	&CommandShowShard{},
	&CommandSetServerRole{},
	&CommandShowServerRole{},
	&CommandSetPrimaryReads{},
	&CommandShowPrimaryReads{},
}

type CommandSetShardingKey struct{}
type CommandSetShard struct{}
type CommandShowShard struct{}
type CommandSetServerRole struct{}
type CommandShowServerRole struct{}
type CommandSetPrimaryReads struct{}
type CommandShowPrimaryReads struct{}

type QueryRouter struct {
	active_shard          int
	primary_reads_enabled bool
	//pool_settings         pool.PoolSettings
}

/* TODO
// / Pool settings can change because of a config reload.
func (r *QueryRouter) UpdatePoolSettings(pool_settings pool.PoolSettings) {
	r.pool_settings = pool_settings
}

*/

// / Try to parse a command and execute it.
// TODO: needs to just provide the execution function and so gatling can then plug in the client, server, etc
func (r *QueryRouter) try_execute_command(pkt *protocol.Query) (Command, string) {
	// Only simple protocol supported for commands.
	// TODO: read msg len
	// msglen := buf.get_i32()
	custom := false
	for _, v := range CustomSqlRegex {
		if v.MatchString(pkt.Fields.Query) {
			custom = true
			break
		}
	}
	// This is not a custom query, try to infer which
	// server it'll go to if the query parser is enabled.
	if !custom {
		log.Println("regular query, not a command")
		return nil, ""
	}

	// TODO: command matching
	//command := switch matches[0] {
	//	0 => Command::SetShardingKey,
	//	1 => Command::SetShard,
	//	2 => Command::ShowShard,
	//	3 => Command::SetServerRole,
	//	4 => Command::ShowServerRole,
	//	5 => Command::SetPrimaryReads,
	//	6 => Command::ShowPrimaryReads,
	//	_ => unreachable!(),
	//}

	//mut value := switch command {
	//	Command::SetShardingKey
	//	| Command::SetShard
	//	| Command::SetServerRole
	//	| Command::SetPrimaryReads => {
	//		// Capture value. I know this re-runs the regex engine, but I haven't
	//		// figured out a better way just yet. I think I can write a single Regex
	//		// that switches all 5 custom SQL patterns, but maybe that's not very legible?
	//		//
	//		// I think this is faster than running the Regex engine 5 times.
	//		switch regex_list[matches[0]].captures(&query) {
	//			Some(captures) => switch captures.get(1) {
	//				Some(value) => value.as_str().to_string(),
	//				None => return None,
	//			},
	//			None => return None,
	//		}
	//	}

	//	Command::ShowShard => self.shard().to_string(),
	//	Command::ShowServerRole => switch self.active_role {
	//		Some(Role::Primary) => string("primary"),
	//		Some(Role::Replica) => string("replica"),
	//		None => {
	//			if self.query_parser_enabled {
	//				string("auto")
	//			} else {
	//				string("any")
	//			}
	//		}
	//	},

	//	Command::ShowPrimaryReads => switch self.primary_reads_enabled {
	//		true => string("on"),
	//		false => string("off"),
	//	},
	//}

	//switch command {
	//	Command::SetShardingKey => {
	//		sharder := Sharder::new(
	//			self.pool_settings.shards,
	//			self.pool_settings.sharding_function,
	//		)
	//		shard := sharder.shard(value.parse::<i64>().unwrap())
	//		self.active_shard := Some(shard)
	//		value := shard.to_string()
	//	}

	//	Command::SetShard => {
	//		self.active_shard := switch value.to_ascii_uppercase().as_ref() {
	//			"ANY" => Some(rand::random::<usize>() % self.pool_settings.shards),
	//			_ => Some(value.parse::<usize>().unwrap()),
	//		}
	//	}

	//	Command::SetServerRole => {
	//		self.active_role := switch value.to_ascii_lowercase().as_ref() {
	//			"primary" => {
	//				self.query_parser_enabled := false
	//				Some(Role::Primary)
	//			}

	//			"replica" => {
	//				self.query_parser_enabled := false
	//				Some(Role::Replica)
	//			}

	//			"any" => {
	//				self.query_parser_enabled := false
	//				None
	//			}

	//			"auto" => {
	//				self.query_parser_enabled := true
	//				None
	//			}

	//			"default" => {
	//				self.active_role := self.pool_settings.default_role
	//				self.query_parser_enabled := self.query_parser_enabled
	//				self.active_role
	//			}

	//			_ => unreachable!(),
	//		}
	//	}

	//	Command::SetPrimaryReads => {
	//		if value == "on" {
	//			log.Println("Setting primary reads to on")
	//			self.primary_reads_enabled := true
	//		} else if value == "off" {
	//			log.Println("Setting primary reads to off")
	//			self.primary_reads_enabled := false
	//		} else if value == "default" {
	//			log.Println("Setting primary reads to default")
	//			self.primary_reads_enabled := self.pool_settings.primary_reads_enabled
	//		}
	//	}

	//	_ => (),
	//}

	//Some((command, value))
	return nil, ""
}

// Try to infer the server role to try to  connect to
// based on the contents of the query.
// note that the user needs to be checked to see if they are allowed to access.
// TODO: implement
func (r *QueryRouter) InferRole(query string) (config.ServerRole, error) {
	var active_role config.ServerRole
	// by default it will hit a replica
	active_role = config.SERVERROLE_REPLICA
	// ok now parse the query
	wk := &walk.AstWalker{
		Fn: func(ctx, node any) (stop bool) {
			switch n := node.(type) {
			case *tree.Update, *tree.UpdateExpr,
				*tree.BeginTransaction, *tree.CommitTransaction, *tree.RollbackTransaction,
				*tree.SetTransaction, *tree.ShowTransactionStatus, *tree.Delete, *tree.Insert:
				//
				active_role = config.SERVERROLE_PRIMARY
				return true
			default:
				_ = n
			}
			return false
		},
	}
	stmts, err := parser.Parse(query)
	if err != nil {
		log.Println("failed to parse (%query), assuming primary required", err)
		return config.SERVERROLE_PRIMARY, nil
	}
	_, err = wk.Walk(stmts, nil)
	if err != nil {
		return config.SERVERROLE_PRIMARY, err
	}
	return active_role, nil
}

// / Get desired shard we should be talking to.
func (r *QueryRouter) Shard() int {
	return r.active_shard
}

func (r *QueryRouter) SetShard(shard int) {
	r.active_shard = shard
}
