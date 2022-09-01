package gat

import (
	"log"
	"regexp"

	"gfx.cafe/gfx/pggat/lib/config"
)

var compiler = regexp.MustCompile

var CustomSqlRegex = []*regexp.Regexp{
	compiler("(?i)^ *SET SHARDING KEY TO '?([0-9]+)'? *;? *$"),
	compiler("(?i)^ *SET SHARD TO '?([0-9]+|ANY)'? *;? *$"),
	compiler("(?i)^ *SHOW SHARD *;? *$"),
	compiler("(?i)^ *SET SERVER ROLE TO '(PRIMARY|REPLICA|ANY|AUTO|DEFAULT)' *;? *$"),
	compiler("(?i)^ *SHOW SERVER ROLE *;? *$"),
	compiler("(?i)^ *SET PRIMARY READS TO '?(on|off|default)'? *;? *$"),
	compiler("(?i)^ *SHOW PRIMARY READS *;? *$"),
}

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
	active_role           config.ServerRole
	query_parser_enabled  bool
	primary_reads_enabled bool
	pool_settings         PoolSettings
}

// / Pool settings can change because of a config reload.
func (r *QueryRouter) UpdatePoolSettings(pool_settings PoolSettings) {
	r.pool_settings = pool_settings
}

// / Try to parse a command and execute it.
func (r *QueryRouter) try_execute_command(buf []byte) (Command, string) {
	// Only simple protocol supported for commands.
	if buf[0] != 'Q' {
		return nil, ""
	}
	msglen := 0
	// TODO: read msg len
	// msglen := buf.get_i32()
	custom := false
	for _, v := range CustomSqlRegex {
		if v.Match(buf[:msglen-5]) {
			custom = true
			break
		}
	}
	// This is not a custom query, try to infer which
	// server it'll go to if the query parser is enabled.
	if !custom {
		log.Println("Regular query, not a command")
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

// / Try to infer which server to connect to based on the contents of the query.
// TODO: implement
func (r *QueryRouter) InferRole(buf []byte) bool {
	log.Println("Inferring role")

	//code := buf.get_u8() as char
	//len := buf.get_i32() as usize

	//query := switch code {
	//	// Query
	//	'Q' => {
	//		query := string(&buf[:len - 5]).to_string()
	//		log.Println("Query: '%v'", query)
	//		query
	//	}

	//	// Parse (prepared statement)
	//	'P' => {
	//		mut start := 0
	//		mut end

	//		// Skip the name of the prepared statement.
	//		while buf[start] != 0 && start < buf.len() {
	//			start += 1
	//		}
	//		start += 1 // Skip terminating null

	//		// Find the end of the prepared stmt (\0)
	//		end := start
	//		while buf[end] != 0 && end < buf.len() {
	//			end += 1
	//		}

	//		query := string(&buf[start:end]).to_string()

	//		log.Println("Prepared statement: '%v'", query)

	//		query.replace("$", "") // Remove placeholders turning them into "values"
	//	}

	//	_ => return false,
	//}

	//ast := switch Parser::parse_sql(&PostgreSqlDialect %v, &query) {
	//	Ok(ast) => ast,
	//	Err(err) => {
	//		log.Println("%v", err.to_string())
	//		return false
	//	}
	//}

	//if ast.len() == 0 {
	//	return false
	//}

	//switch ast[0] {
	//	// All transactions go to the primary, probably a write.
	//	StartTransaction { : } => {
	//		self.active_role := Some(Role::Primary)
	//	}

	//	// Likely a read-only query
	//	Query { : } => {
	//		self.active_role := switch self.primary_reads_enabled {
	//			false => Some(Role::Replica), // If primary should not be receiving reads, use a replica.
	//			true => None,                 // Any server role is fine in this case.
	//		}
	//	}

	//	// Likely a write
	//	_ => {
	//		self.active_role := Some(Role::Primary)
	//	}
	//}

	return true
}

// / Get the current desired server role we should be talking to.
func (r *QueryRouter) Role() config.ServerRole {
	return r.active_role
}

// / Get desired shard we should be talking to.
func (r *QueryRouter) Shard() int {
	return r.active_shard
}

func (r *QueryRouter) SetShard(shard int) {
	r.active_shard = shard
}

func (r *QueryRouter) QueryParserEnabled() bool {
	return r.query_parser_enabled
}
