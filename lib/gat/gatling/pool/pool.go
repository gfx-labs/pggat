package pool

import (
	"fmt"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/pool/conn_pool"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/pool/query_router"
	"sync"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
)

type Pool struct {
	c         *config.Pool
	users     map[string]config.User
	connPools map[string]gat.ConnectionPool

	stats *Stats

	router query_router.QueryRouter

	mu sync.RWMutex
}

func NewPool(conf *config.Pool) *Pool {
	pool := &Pool{
		connPools: make(map[string]gat.ConnectionPool),
		stats:     newStats(),
	}
	pool.EnsureConfig(conf)
	return pool
}

func (p *Pool) EnsureConfig(conf *config.Pool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.c = conf
	p.users = make(map[string]config.User)
	for _, user := range conf.Users {
		p.users[user.Name] = *user
	}
	// ensure conn pools
	for name, user := range p.users {
		if existing, ok := p.connPools[name]; ok {
			existing.EnsureConfig(conf)
		} else {
			u := user
			p.connPools[name] = conn_pool.NewConnectionPool(p, conf, &u)
		}
	}
}

func (p *Pool) GetUser(name string) (*config.User, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	user, ok := p.users[name]
	if !ok {
		return nil, fmt.Errorf("user '%s' not found", name)
	}
	return &user, nil
}

func (p *Pool) GetRouter() gat.QueryRouter {
	return &p.router
}

func (p *Pool) WithUser(name string) (gat.ConnectionPool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	pool, ok := p.connPools[name]
	if !ok {
		return nil, fmt.Errorf("no pool for '%s'", name)
	}
	return pool, nil
}

func (p *Pool) ConnectionPools() []gat.ConnectionPool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	out := make([]gat.ConnectionPool, len(p.connPools))
	idx := 0
	for _, v := range p.connPools {
		out[idx] = v
		idx += 1
	}
	return out
}

func (p *Pool) Stats() gat.PoolStats {
	return p.stats
}

var _ gat.Pool = (*Pool)(nil)

//TODO: implement server pool
//#[async_trait]
//impl ManageConnection for ServerPool {
//    type Connection = Server;
//    type Err = Err;
//
//    /// Attempts to create a new connection.
//    async fn connect(&self) -> Result<Self::Connection, Self::Err> {
//        info!(
//            "Creating a new connection to {:?} using user {:?}",
//            self.address.name(),
//            self.user.username
//        );
//
//        // Put a temporary process_id into the stats
//        // for server login.
//        let process_id = rand::random::<i32>();
//        self.stats.server_login(process_id, self.address.id);
//
//        // Connect to the PostgreSQL server.
//        match Server::startup(
//            &self.address,
//            &self.user,
//            &self.database,
//            self.client_server_map.clone(),
//            self.stats.clone(),
//        )
//        .await
//        {
//            Ok(conn) => {
//                // Remove the temporary process_id from the stats.
//                self.stats.server_disconnecting(process_id, self.address.id);
//                Ok(conn)
//            }
//            Err(err) => {
//                // Remove the temporary process_id from the stats.
//                self.stats.server_disconnecting(process_id, self.address.id);
//                Err(err)
//            }
//        }
//    }
//
//    /// Determines if the connection is still connected to the database.
//    async fn is_valid(&self, _conn: &mut PooledConnection<'_, Self>) -> Result<(), Self::Err> {
//        Ok(())
//    }
//
//    /// Synchronously determine if the connection is no longer usable, if possible.
//    fn has_broken(&self, conn: &mut Self::Connection) -> bool {
//        conn.is_bad()
//    }
//}
//
///// Get the connection pool
//pub fn get_pool(db: String, user: String) -> Option<ConnectionPool> {
//    match get_all_pools().get(&(db, user)) {
//        Some(pool) => Some(pool.clone()),
//        None => None,
//    }
//}
//
///// How many total servers we have in the config.
//pub fn get_number_of_addresses() -> usize {
//    get_all_pools()
//        .iter()
//        .map(|(_, pool)| pool.databases())
//        .sum()
//}
//
///// Get a pointer to all configured pools.
//pub fn get_all_pools() -> HashMap<(String, String), ConnectionPool> {
//    return (*(*POOLS.load())).clone();
//}

//TODO: implement this
//    /// Construct the connection pool from the configuration.
//    func (c *ConnectionPool) from_config(client_server_map: ClientServerMap)  Result<(), Err> {
//        let config = get_config()
//
//         new_pools = HashMap::new()
//         address_id = 0
//
//        for (pool_name, pool_config) in &config.pools {
//            // There is one pool per database/user pair.
//            for (_, user) in &pool_config.users {
//                 shards = Vec::new()
//                 addresses = Vec::new()
//                 banlist = Vec::new()
//                 shard_ids = pool_config
//                    .shards
//                    .clone()
//                    .into_keys()
//                    .map(|x| x.to_string())
//                    .collect::<Vec<string>>()
//
//                // Sort by shard number to ensure consistency.
//                shard_ids.sort_by_key(|k| k.parse::<i64>().unwrap())
//
//                for shard_idx in &shard_ids {
//                    let shard = &pool_config.shards[shard_idx]
//                     pools = Vec::new()
//                     servers = Vec::new()
//                     address_index = 0
//                     replica_number = 0
//
//                    for server in shard.servers.iter() {
//                        let role = match server.2.as_ref() {
//                            "primary" => Role::Primary,
//                            "replica" => Role::Replica,
//                            _ => {
//                                error!("Config error: server role can be 'primary' or 'replica', have: '{}'. Defaulting to 'replica'.", server.2)
//                                Role::Replica
//                            }
//                        }
//
//                        let address = Address {
//                            id: address_id,
//                            database: shard.database.clone(),
//                            host: server.0.clone(),
//                            port: server.1 as u16,
//                            role: role,
//                            address_index,
//                            replica_number,
//                            shard: shard_idx.parse::<usize>().unwrap(),
//                            username: user.username.clone(),
//                            pool_name: pool_name.clone(),
//                        }
//
//                        address_id += 1
//                        address_index += 1
//
//                        if role == Role::Replica {
//                            replica_number += 1
//                        }
//
//                        let manager = ServerPool::new(
//                            address.clone(),
//                            user.clone(),
//                            &shard.database,
//                            client_server_map.clone(),
//                            get_reporter(),
//                        )
//
//                        let pool = Pool::builder()
//                            .max_size(user.pool_size)
//                            .connection_timeout(std::time::Duration::from_millis(
//                                config.general.connect_timeout,
//                            ))
//                            .test_on_check_out(false)
//                            .build(manager)
//                            .await
//                            .unwrap()
//
//                        pools.push(pool)
//                        servers.push(address)
//                    }
//
//                    shards.push(pools)
//                    addresses.push(servers)
//                    banlist.push(HashMap::new())
//                }
//
//                assert_eq!(shards.len(), addresses.len())
//
//                 pool = ConnectionPool {
//                    databases: shards,
//                    addresses: addresses,
//                    banlist: Arc::new(RwLock::new(banlist)),
//                    stats: get_reporter(),
//                    server_info: BytesMut::new(),
//                    settings: PoolSettings {
//                        pool_mode: match pool_config.pool_mode.as_str() {
//                            "transaction" => PoolMode::Transaction,
//                            "session" => PoolMode::Session,
//                            _ => unreachable!(),
//                        },
//                        // shards: pool_config.shards.clone(),
//                        shards: shard_ids.len(),
//                        user: user.clone(),
//                        default_role: match pool_config.default_role.as_str() {
//                            "any" => None,
//                            "replica" => Some(Role::Replica),
//                            "primary" => Some(Role::Primary),
//                            _ => unreachable!(),
//                        },
//                        query_parser_enabled: pool_config.query_parser_enabled.clone(),
//                        primary_reads_enabled: pool_config.primary_reads_enabled,
//                        sharding_function: match pool_config.sharding_function.as_str() {
//                            "pg_bigint_hash" => ShardingFunction::PgBigintHash,
//                            "sha1" => ShardingFunction::Sha1,
//                            _ => unreachable!(),
//                        },
//                    },
//                }
//
//                // Connect to the servers to make sure pool configuration is valid
//                // before setting it globally.
//                match pool.validate().await {
//                    Ok(_) => (),
//                    Err(err) => {
//                        error!("Could not validate connection pool: {:?}", err)
//                        return Err(err)
//                    }
//                }
//
//                // There is one pool per database/user pair.
//                new_pools.insert((pool_name.clone(), user.username.clone()), pool)
//            }
//        }
//
//        POOLS.store(Arc::new(new_pools.clone()))
//
//        Ok(())
//    }
//
//    /// Connect to all shards and grab server information.
//    /// Return server information we will pass to the clients
//    /// when they connect.
//    /// This also warms up the pool for clients that connect when
//    /// the pooler starts up.
//    async fn validate(&mut self)  Result<(), Err> {
//         server_infos = Vec::new()
//        for shard in 0..self.shards() {
//            for server in 0..self.servers(shard) {
//                let connection = match self.databases[shard][server].get().await {
//                    Ok(conn) => conn,
//                    Err(err) => {
//                        error!("Shard {} down or misconfigured: {:?}", shard, err)
//                        continue
//                    }
//                }
//
//                let proxy = connection
//                let server = &*proxy
//                let server_info = server.server_info()
//
//                if server_infos.len() > 0 {
//                    // Compare against the last server checked.
//                    if server_info != server_infos[server_infos.len() - 1] {
//                        warn!(
//                            "{:?} has different server configuration than the last server",
//                            proxy.address()
//                        )
//                    }
//                }
//
//                server_infos.push(server_info)
//            }
//        }
//
//        // TODO: compare server information to make sure
//        // all shards are running identical configurations.
//        if server_infos.len() == 0 {
//            return Err(Err::AllServersDown)
//        }
//
//        // We're assuming all servers are identical.
//        // TODO: not true.
//        self.server_info = server_infos[0].clone()
//
//        Ok(())
//    }
//
//    /// Get a connection from the pool.
//    func (c *ConnectionPool) get(
//        &self,
//        shard: usize,       // shard number
//        role: Option<Role>, // primary or replica
//        process_id: i32,    // client id
//    )  Result<(PooledConnection<'_, ServerPool>, Address), Err> {
//        let now = Instant::now()
//         candidates: Vec<&Address> = self.addresses[shard]
//            .iter()
//            .filter(|address| address.role == role)
//            .collect()
//
//        // Random load balancing
//        candidates.shuffle(&mut thread_rng())
//
//        let healthcheck_timeout = get_config().general.healthcheck_timeout
//        let healthcheck_delay = get_config().general.healthcheck_delay as u128
//
//        while !candidates.is_empty() {
//            // Get the next candidate
//            let address = match candidates.pop() {
//                Some(address) => address,
//                None => break,
//            }
//
//            if self.is_banned(&address, role) {
//                debug!("Address {:?} is banned", address)
//                continue
//            }
//
//            // Indicate we're waiting on a server connection from a pool.
//            self.stats.client_waiting(process_id, address.id)
//
//            // Check if we can connect
//             conn = match self.databases[address.shard][address.address_index]
//                .get()
//                .await
//            {
//                Ok(conn) => conn,
//                Err(err) => {
//                    error!("Banning instance {:?}, error: {:?}", address, err)
//                    self.ban(&address, process_id)
//                    self.stats
//                        .checkout_time(now.elapsed().as_micros(), process_id, address.id)
//                    continue
//                }
//            }
//
//            // // Check if this server is alive with a health check.
//            let server = &mut *conn
//
//            // Will return error if timestamp is greater than current system time, which it should never be set to
//            let require_healthcheck =
//                server.last_activity().elapsed().unwrap().as_millis() > healthcheck_delay
//
//            // Do not issue a health check unless it's been a little while
//            // since we last checked the server is ok.
//            // Health checks are pretty expensive.
//            if !require_healthcheck {
//                self.stats
//                    .checkout_time(now.elapsed().as_micros(), process_id, address.id)
//                self.stats.server_active(conn.process_id(), address.id)
//                return Ok((conn, address.clone()))
//            }
//
//            debug!("Running health check on server {:?}", address)
//
//            self.stats.server_tested(server.process_id(), address.id)
//
//            match tokio::time::timeout(
//                tokio::time::Duration::from_millis(healthcheck_timeout),
//                server.query(""), // Cheap query (query parser not used in PG)
//            )
//            .await
//            {
//                // Check if health check succeeded.
//                Ok(res) => match res {
//                    Ok(_) => {
//                        self.stats
//                            .checkout_time(now.elapsed().as_micros(), process_id, address.id)
//                        self.stats.server_active(conn.process_id(), address.id)
//                        return Ok((conn, address.clone()))
//                    }
//
//                    // Health check failed.
//                    Err(err) => {
//                        error!(
//                            "Banning instance {:?} because of failed health check, {:?}",
//                            address, err
//                        )
//
//                        // Don't leave a bad connection in the pool.
//                        server.mark_bad()
//
//                        self.ban(&address, process_id)
//                        continue
//                    }
//                },
//
//                // Health check timed out.
//                Err(err) => {
//                    error!(
//                        "Banning instance {:?} because of health check timeout, {:?}",
//                        address, err
//                    )
//                    // Don't leave a bad connection in the pool.
//                    server.mark_bad()
//
//                    self.ban(&address, process_id)
//                    continue
//                }
//            }
//        }
//
//        Err(Err::AllServersDown)
//    }
//
//    /// Ban an address (i.e. replica). It no longer will serve
//    /// traffic for any new transactions. Existing transactions on that replica
//    /// will finish successfully or error out to the clients.
//    func (c *ConnectionPool) ban(&self, address: &Address, process_id: i32) {
//        self.stats.client_disconnecting(process_id, address.id)
//
//        error!("Banning {:?}", address)
//
//        let now = chrono::offset::Utc::now().naive_utc()
//         guard = self.banlist.write()
//        guard[address.shard].insert(address.clone(), now)
//    }
//
//    /// Clear the replica to receive traffic again. Takes effect immediately
//    /// for all new transactions.
//    func (c *ConnectionPool) _unban(&self, address: &Address) {
//         guard = self.banlist.write()
//        guard[address.shard].remove(address)
//    }
//
//    /// Check if a replica can serve traffic. If all replicas are banned,
//    /// we unban all of them. Better to try then not to.
//    func (c *ConnectionPool) is_banned(&self, address: &Address, role: Option<Role>)  bool {
//        let replicas_available = match role {
//            Some(Role::Replica) => self.addresses[address.shard]
//                .iter()
//                .filter(|addr| addr.role == Role::Replica)
//                .count(),
//            None => self.addresses[address.shard].len(),
//            Some(Role::Primary) => return false, // Primary cannot be banned.
//        }
//
//        debug!("Available targets for {:?}: {}", role, replicas_available)
//
//        let guard = self.banlist.read()
//
//        // Everything is banned = nothing is banned.
//        if guard[address.shard].len() == replicas_available {
//            drop(guard)
//             guard = self.banlist.write()
//            guard[address.shard].clear()
//            drop(guard)
//            warn!("Unbanning all replicas.")
//            return false
//        }
//
//        // I expect this to miss 99.9999% of the time.
//        match guard[address.shard].get(address) {
//            Some(timestamp) => {
//                let now = chrono::offset::Utc::now().naive_utc()
//                let config = get_config()
//
//                // Ban expired.
//                if now.timestamp() - timestamp.timestamp() > config.general.ban_time {
//                    drop(guard)
//                    warn!("Unbanning {:?}", address)
//                     guard = self.banlist.write()
//                    guard[address.shard].remove(address)
//                    false
//                } else {
//                    debug!("{:?} is banned", address)
//                    true
//                }
//            }
//
//            None => {
//                debug!("{:?} is ok", address)
//                false
//            }
//        }
//    }
//
//    /// Get the number of configured shards.
//    func (c *ConnectionPool) shards(&self)  usize {
//        self.databases.len()
//    }
//
//    /// Get the number of servers (primary and replicas)
//    /// configured for a shard.
//    func (c *ConnectionPool) servers(&self, shard: usize)  usize {
//        self.addresses[shard].len()
//    }
//
//    /// Get the total number of servers (databases) we are connected to.
//    func (c *ConnectionPool) databases(&self)  usize {
//         databases = 0
//        for shard in 0..self.shards() {
//            databases += self.servers(shard)
//        }
//        databases
//    }
//
//    /// Get pool state for a particular shard server as reported by bb8.
//    func (c *ConnectionPool) pool_state(&self, shard: usize, server: usize)  bb8::State {
//        self.databases[shard][server].state()
//    }
//
//    /// Get the address information for a shard server.
//    func (c *ConnectionPool) address(&self, shard: usize, server: usize)  &Address {
//        &self.addresses[shard][server]
//    }
//
//    func (c *ConnectionPool) server_info(&self)  BytesMut {
//        self.server_info.clone()
//    }
