package sharding

//TODO: convert test

//#[cfg(test)]
//mod test {
//    use super::*;
//
//    // See tests/sharding/partition_hash_test_setup.sql
//    // The output of those SELECT statements will match this test,
//    // confirming that we implemented Postgres BIGINT hashing correctly.
//    #[test]
//    fn test_pg_bigint_hash() {
//        let sharder = Sharder::new(5, ShardingFunction::PgBigintHash);
//
//        let shard_0 = vec![1, 4, 5, 14, 19, 39, 40, 46, 47, 53];
//
//        for v in shard_0 {
//            assert_eq!(sharder.shard(v), 0);
//        }
//
//        let shard_1 = vec![2, 3, 11, 17, 21, 23, 30, 49, 51, 54];
//
//        for v in shard_1 {
//            assert_eq!(sharder.shard(v), 1);
//        }
//
//        let shard_2 = vec![6, 7, 15, 16, 18, 20, 25, 28, 34, 35];
//
//        for v in shard_2 {
//            assert_eq!(sharder.shard(v), 2);
//        }
//
//        let shard_3 = vec![8, 12, 13, 22, 29, 31, 33, 36, 41, 43];
//
//        for v in shard_3 {
//            assert_eq!(sharder.shard(v), 3);
//        }
//
//        let shard_4 = vec![9, 10, 24, 26, 27, 32, 37, 38, 42, 45];
//
//        for v in shard_4 {
//            assert_eq!(sharder.shard(v), 4);
//        }
//    }
//
//    #[test]
//    fn test_sha1_hash() {
//        let sharder = Sharder::new(12, ShardingFunction::Sha1);
//        let ids = vec![
//            0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19,
//        ];
//        let shards = vec![
//            4, 7, 8, 3, 6, 0, 0, 10, 3, 11, 1, 7, 4, 4, 11, 2, 5, 0, 8, 3,
//        ];
//
//        for (i, id) in ids.iter().enumerate() {
//            assert_eq!(sharder.shard(*id), shards[i]);
//        }
//    }
//}
