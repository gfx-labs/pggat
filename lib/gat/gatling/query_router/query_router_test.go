package query_router

import (
	"testing"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

// TODO: adapt tests
func TestQueryRouterInterRoleReplica(t *testing.T) {
	qr := &QueryRouter{}
	pkt := &protocol.Parse{}
	pkt.Fields.Query = `UPDATE items SET name = 'pumpkin' WHERE id = 5`
	role, err := qr.InferRole(pkt)
	if err != nil {
		t.Fatal(err)
	}
	if role != config.SERVERROLE_PRIMARY {
		t.Error("expect primary")
	}
	pkt.Fields.Query = `select * from items WHERE id = 5`
	role, err = qr.InferRole(pkt)
	if err != nil {
		t.Fatal(err)
	}
	if role != config.SERVERROLE_REPLICA {
		t.Error("expect replica")
	}

}

//      assert!(qr.try_execute_command(simple_query("SET SERVER ROLE TO 'auto'")) != None);
//      assert_eq!(qr.query_parser_enabled(), true);

//      assert!(qr.try_execute_command(simple_query("SET PRIMARY READS TO off")) != None);

//      let queries = vec![
//          simple_query("SELECT * FROM items WHERE id = 5"),
//          simple_query(
//              "SELECT id, name, value FROM items INNER JOIN prices ON item.id = prices.item_id",
//          ),
//          simple_query("WITH t AS (SELECT * FROM items) SELECT * FROM t"),
//      ];

//      for query in queries {
//          // It's a recognized query
//          assert!(qr.infer_role(query));
//          assert_eq!(qr.role(), Some(Role::Replica));
//      }
//  }
//
//    #[test]
//    fn test_infer_role_primary() {
//        QueryRouter::setup();
//        let mut qr = QueryRouter::new();
//
//        let queries = vec![
//            simple_query("UPDATE items SET name = 'pumpkin' WHERE id = 5"),
//            simple_query("INSERT INTO items (id, name) VALUES (5, 'pumpkin')"),
//            simple_query("DELETE FROM items WHERE id = 5"),
//            simple_query("BEGIN"), // Transaction start
//        ];
//
//        for query in queries {
//            // It's a recognized query
//            assert!(qr.infer_role(query));
//            assert_eq!(qr.role(), Some(Role::Primary));
//        }
//    }
//
//    #[test]
//    fn test_infer_role_primary_reads_enabled() {
//        QueryRouter::setup();
//        let mut qr = QueryRouter::new();
//        let query = simple_query("SELECT * FROM items WHERE id = 5");
//        assert!(qr.try_execute_command(simple_query("SET PRIMARY READS TO on")) != None);
//
//        assert!(qr.infer_role(query));
//        assert_eq!(qr.role(), None);
//    }
//
//    #[test]
//    fn test_infer_role_parse_prepared() {
//        QueryRouter::setup();
//        let mut qr = QueryRouter::new();
//        qr.try_execute_command(simple_query("SET SERVER ROLE TO 'auto'"));
//        assert!(qr.try_execute_command(simple_query("SET PRIMARY READS TO off")) != None);
//
//        let prepared_stmt = BytesMut::from(
//            &b"WITH t AS (SELECT * FROM items WHERE name = $1) SELECT * FROM t WHERE id = $2\0"[..],
//        );
//        let mut res = BytesMut::from(&b"P"[..]);
//        res.put_i32(prepared_stmt.len() as i32 + 4 + 1 + 2);
//        res.put_u8(0);
//        res.put(prepared_stmt);
//        res.put_i16(0);
//
//        assert!(qr.infer_role(res));
//        assert_eq!(qr.role(), Some(Role::Replica));
//    }
//
//    #[test]
//    fn test_regex_set() {
//        QueryRouter::setup();
//
//        let tests = [
//            // Upper case
//            "SET SHARDING KEY TO '1'",
//            "SET SHARD TO '1'",
//            "SHOW SHARD",
//            "SET SERVER ROLE TO 'replica'",
//            "SET SERVER ROLE TO 'primary'",
//            "SET SERVER ROLE TO 'any'",
//            "SET SERVER ROLE TO 'auto'",
//            "SHOW SERVER ROLE",
//            "SET PRIMARY READS TO 'on'",
//            "SET PRIMARY READS TO 'off'",
//            "SET PRIMARY READS TO 'default'",
//            "SHOW PRIMARY READS",
//            // Lower case
//            "set sharding key to '1'",
//            "set shard to '1'",
//            "show shard",
//            "set server role to 'replica'",
//            "set server role to 'primary'",
//            "set server role to 'any'",
//            "set server role to 'auto'",
//            "show server role",
//            "set primary reads to 'on'",
//            "set primary reads to 'OFF'",
//            "set primary reads to 'deFaUlt'",
//            // No quotes
//            "SET SHARDING KEY TO 11235",
//            "SET SHARD TO 15",
//            "SET PRIMARY READS TO off",
//            // Spaces and semicolon
//            "  SET SHARDING KEY TO 11235  ; ",
//            "  SET SHARD TO 15;   ",
//            "  SET SHARDING KEY TO 11235  ;",
//            " SET SERVER ROLE TO 'primary';   ",
//            "    SET SERVER ROLE TO 'primary'  ; ",
//            "  SET SERVER ROLE TO 'primary'  ;",
//            "  SET PRIMARY READS TO 'off'    ;",
//        ];
//
//        // Which regexes it'll match to in the list
//        let matches = [
//            0, 1, 2, 3, 3, 3, 3, 4, 5, 5, 5, 6, 0, 1, 2, 3, 3, 3, 3, 4, 5, 5, 5, 0, 1, 5, 0, 1, 0,
//            3, 3, 3, 5,
//        ];
//
//        let list = CUSTOM_SQL_REGEX_LIST.get().unwrap();
//        let set = CUSTOM_SQL_REGEX_SET.get().unwrap();
//
//        for (i, test) in tests.iter().enumerate() {
//            if !list[matches[i]].is_match(test) {
//                println!("{} does not match {}", test, list[matches[i]]);
//                assert!(false);
//            }
//            assert_eq!(set.matches(test).into_iter().collect::<Vec<_>>().len(), 1);
//        }
//
//        let bad = [
//            "SELECT * FROM table",
//            "SELECT * FROM table WHERE value = 'set sharding key to 5'", // Don't capture things in the middle of the query
//        ];
//
//        for query in &bad {
//            assert_eq!(set.matches(query).into_iter().collect::<Vec<_>>().len(), 0);
//        }
//    }
//
//    #[test]
//    fn test_try_execute_command() {
//        QueryRouter::setup();
//        let mut qr = QueryRouter::new();
//
//        // SetShardingKey
//        let query = simple_query("SET SHARDING KEY TO 13");
//        assert_eq!(
//            qr.try_execute_command(query),
//            Some((Command::SetShardingKey, String::from("0")))
//        );
//        assert_eq!(qr.shard(), 0);
//
//        // SetShard
//        let query = simple_query("SET SHARD TO '1'");
//        assert_eq!(
//            qr.try_execute_command(query),
//            Some((Command::SetShard, String::from("1")))
//        );
//        assert_eq!(qr.shard(), 1);
//
//        // ShowShard
//        let query = simple_query("SHOW SHARD");
//        assert_eq!(
//            qr.try_execute_command(query),
//            Some((Command::ShowShard, String::from("1")))
//        );
//
//        // SetServerRole
//        let roles = ["primary", "replica", "any", "auto", "primary"];
//        let verify_roles = [
//            Some(Role::Primary),
//            Some(Role::Replica),
//            None,
//            None,
//            Some(Role::Primary),
//        ];
//        let query_parser_enabled = [false, false, false, true, false];
//
//        for (idx, role) in roles.iter().enumerate() {
//            let query = simple_query(&format!("SET SERVER ROLE TO '{}'", role));
//            assert_eq!(
//                qr.try_execute_command(query),
//                Some((Command::SetServerRole, String::from(*role)))
//            );
//            assert_eq!(qr.role(), verify_roles[idx],);
//            assert_eq!(qr.query_parser_enabled(), query_parser_enabled[idx],);
//
//            // ShowServerRole
//            let query = simple_query("SHOW SERVER ROLE");
//            assert_eq!(
//                qr.try_execute_command(query),
//                Some((Command::ShowServerRole, String::from(*role)))
//            );
//        }
//
//        let primary_reads = ["on", "off", "default"];
//        let primary_reads_enabled = ["on", "off", "on"];
//
//        for (idx, primary_reads) in primary_reads.iter().enumerate() {
//            assert_eq!(
//                qr.try_execute_command(simple_query(&format!(
//                    "SET PRIMARY READS TO {}",
//                    primary_reads
//                ))),
//                Some((Command::SetPrimaryReads, String::from(*primary_reads)))
//            );
//            assert_eq!(
//                qr.try_execute_command(simple_query("SHOW PRIMARY READS")),
//                Some((
//                    Command::ShowPrimaryReads,
//                    String::from(primary_reads_enabled[idx])
//                ))
//            );
//        }
//    }
//
//    #[test]
//    fn test_enable_query_parser() {
//        QueryRouter::setup();
//        let mut qr = QueryRouter::new();
//        let query = simple_query("SET SERVER ROLE TO 'auto'");
//        assert!(qr.try_execute_command(simple_query("SET PRIMARY READS TO off")) != None);
//
//        assert!(qr.try_execute_command(query) != None);
//        assert!(qr.query_parser_enabled());
//        assert_eq!(qr.role(), None);
//
//        let query = simple_query("INSERT INTO test_table VALUES (1)");
//        assert_eq!(qr.infer_role(query), true);
//        assert_eq!(qr.role(), Some(Role::Primary));
//
//        let query = simple_query("SELECT * FROM test_table");
//        assert_eq!(qr.infer_role(query), true);
//        assert_eq!(qr.role(), Some(Role::Replica));
//
//        assert!(qr.query_parser_enabled());
//        let query = simple_query("SET SERVER ROLE TO 'default'");
//        assert!(qr.try_execute_command(query) != None);
//        assert!(qr.query_parser_enabled());
//    }
//
//    #[test]
//    fn test_update_from_pool_settings() {
//        QueryRouter::setup();
//
//        let pool_settings = PoolSettings {
//            pool_mode: PoolMode::Transaction,
//            shards: 0,
//            user: crate::config::User::default(),
//            default_role: Some(Role::Replica),
//            query_parser_enabled: true,
//            primary_reads_enabled: false,
//            sharding_function: ShardingFunction::PgBigintHash,
//        };
//        let mut qr = QueryRouter::new();
//        assert_eq!(qr.active_role, None);
//        assert_eq!(qr.active_shard, None);
//        assert_eq!(qr.query_parser_enabled, false);
//        assert_eq!(qr.primary_reads_enabled, false);
//
//        // Internal state must not be changed due to this, only defaults
//        qr.update_pool_settings(pool_settings.clone());
//
//        assert_eq!(qr.active_role, None);
//        assert_eq!(qr.active_shard, None);
//        assert_eq!(qr.query_parser_enabled, false);
//        assert_eq!(qr.primary_reads_enabled, false);
//
//        let q1 = simple_query("SET SERVER ROLE TO 'primary'");
//        assert!(qr.try_execute_command(q1) != None);
//        assert_eq!(qr.active_role.unwrap(), Role::Primary);
//
//        let q2 = simple_query("SET SERVER ROLE TO 'default'");
//        assert!(qr.try_execute_command(q2) != None);
//        assert_eq!(qr.active_role.unwrap(), pool_settings.clone().default_role);
//    }
//}
