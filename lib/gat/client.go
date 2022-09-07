package gat

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"math/big"
	"net"
	"reflect"

	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/util/maps"

	"gfx.cafe/gfx/pggat/lib/config"
	"git.tuxpa.in/a/zlog"
	"git.tuxpa.in/a/zlog/log"
	"github.com/ethereum/go-ethereum/common/math"
)

type ClientKey [2]int

type ClientInfo struct {
	A int
	B int
	C string
	D uint16
}

// / client state, one per client
type Client struct {
	conn net.Conn
	r    *bufio.Reader
	wr   io.Writer

	buf bytes.Buffer

	addr net.Addr

	cancel_mode bool
	txn_mode    bool

	pid        int32
	secret_key int32

	parameters map[string]string
	stats      any // TODO: Reporter
	admin      bool

	server *Server

	last_addr_id int
	last_srv_id  int

	pool_name string
	username  string

	conf *config.Global

	log zlog.Logger
}

func NewClient(
	conf *config.Global,
	conn net.Conn,
	admin_only bool,
) *Client {
	c := &Client{
		conn: conn,
		r:    bufio.NewReader(conn),
		wr:   conn,
		addr: conn.RemoteAddr(),
		conf: conf,
	}
	c.log = log.With().
		Stringer("clientaddr", c.addr).Logger()
	return c
}

func (c *Client) Accept(ctx context.Context) error {
	// read a packet
	startup := new(protocol.StartupMessage)
	err := startup.Read(c.r)
	if err != nil {
		return err
	}
	switch startup.Fields.ProtocolVersionNumber {
	case 196608:
	case 80877102:
		return c.handle_cancel(ctx, startup)
	case 80877103:
		// ssl stuff now
		useSsl := (c.conf.General.TlsCertificate != "")
		if !useSsl {
			_, err = protocol.WriteByte(c.wr, 'N')
			if err != nil {
				return err
			}
			startup = new(protocol.StartupMessage)
			err = startup.Read(c.r)
			if err != nil {
				return err
			}
		} else {
			_, err = protocol.WriteByte(c.wr, 'S')
			if err != nil {
				return err
			}
			//TODO: we need to do an ssl handshake here.
			var cert tls.Certificate
			cert, err = tls.LoadX509KeyPair(c.conf.General.TlsCertificate, c.conf.General.TlsPrivateKey)
			if err != nil {
				return err
			}
			cfg := &tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true,
			}
			c.conn = tls.Server(c.conn, cfg)
			c.r = bufio.NewReader(c.conn)
			c.wr = c.conn
			err = startup.Read(c.r)
			if err != nil {
				return err
			}
		}
	}
	params := make(map[string]string)
	for _, v := range startup.Fields.Parameters {
		params[v.Name] = v.Value
	}

	var ok bool
	c.pool_name, ok = params["database"]
	if !ok {
		return &PostgresError{
			Severity: Fatal,
			Code:     InvalidAuthorizationSpecification,
			Message:  "param database required",
		}
	}

	c.username, ok = params["user"]
	if !ok {
		return &PostgresError{
			Severity: Fatal,
			Code:     InvalidAuthorizationSpecification,
			Message:  "param user required",
		}
	}

	c.admin = (c.pool_name == "pgcat" || c.pool_name == "pgbouncer")

	if c.conf.General.AdminOnly && !c.admin {
		c.log.Debug().Msg("rejected non admin, since admin only mode")
		return &PostgresError{
			Severity: Fatal,
			Code:     InvalidAuthorizationSpecification,
			Message:  "rejected non admin",
		}
	}

	pid, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	if err != nil {
		return err
	}
	c.pid = int32(pid.Int64())
	skey, err := rand.Int(rand.Reader, big.NewInt(math.MaxInt32))
	if err != nil {
		return err
	}

	c.secret_key = int32(skey.Int64())
	// TODO: Add SASL support.

	// Perform MD5 authentication.
	pkt, salt, err := CreateMd5Challenge()
	if err != nil {
		return err
	}
	_, err = pkt.Write(c.wr)
	if err != nil {
		return err
	}

	var rsp protocol.Packet
	rsp, err = protocol.ReadFrontend(c.r)
	if err != nil {
		return err
	}
	var passwordResponse []byte
	switch r := rsp.(type) {
	case *protocol.AuthenticationResponse:
		passwordResponse = r.Fields.Data
	default:
		return &PostgresError{
			Severity: Fatal,
			Code:     InvalidAuthorizationSpecification,
			Message:  fmt.Sprintf("wanted AuthenticationResponse packet, got '%+v'", rsp),
		}
	}

	pool, ok := c.conf.Pools[c.pool_name]
	if !ok {
		return &PostgresError{
			Severity: Fatal,
			Code:     InvalidAuthorizationSpecification,
			Message:  "no such pool",
		}
	}
	_, user, ok := maps.FirstWhere(pool.Users, func(_ string, user config.User) bool {
		return user.Name == c.username
	})
	if !ok {
		return &PostgresError{
			Severity: Fatal,
			Code:     InvalidPassword,
			Message:  "user not found",
		}
	}

	// Authenticate admin user.
	if c.admin {
		pw_hash := Md5HashPassword(c.conf.General.AdminUsername, c.conf.General.AdminPassword, salt[:])
		if !reflect.DeepEqual(pw_hash, passwordResponse) {
			return &PostgresError{
				Severity: Fatal,
				Code:     InvalidPassword,
				Message:  "invalid password",
			}
		}
	} else {
		pw_hash := Md5HashPassword(c.username, user.Password, salt[:])
		if !reflect.DeepEqual(pw_hash, passwordResponse) {
			return &PostgresError{
				Severity: Fatal,
				Code:     InvalidPassword,
				Message:  "invalid password",
			}
		}
	}

	shard := pool.Shards["0"]
	serv := shard.Servers[0]
	c.server, err = DialServer(context.TODO(), fmt.Sprintf("%s:%d", serv.Host(), serv.Port()), &user, shard.Database, nil)
	if err != nil {
		return err
	}

	c.log.Debug().Msg("Password authentication successful")
	authOk := new(protocol.Authentication)
	authOk.Fields.Code = 0
	_, err = authOk.Write(c.wr)
	if err != nil {
		return err
	}

	//
	for _, inf := range c.server.server_info {
		_, err = inf.Write(c.wr)
		if err != nil {
			return err
		}
	}
	backendKeyData := new(protocol.BackendKeyData)
	backendKeyData.Fields.ProcessID = c.pid
	backendKeyData.Fields.SecretKey = c.secret_key
	_, err = backendKeyData.Write(c.wr)
	if err != nil {
		return err
	}
	readyForQuery := new(protocol.ReadyForQuery)
	readyForQuery.Fields.Status = byte('I')
	_, err = readyForQuery.Write(c.wr)
	if err != nil {
		return err
	}
	c.log.Debug().Msg("Ready for Query")
	open := true
	for open {
		open, err = c.tick(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

// TODO: we need to keep track of queries so we can handle cancels
func (c *Client) handle_cancel(ctx context.Context, p *protocol.StartupMessage) error {
	log.Println("cancel msg", p)
	return nil
}

// reads a packet from stream and handles it
func (c *Client) tick(ctx context.Context) (bool, error) {
	rsp, err := protocol.ReadFrontend(c.r)
	if err != nil {
		return true, err
	}
	switch cast := rsp.(type) {
	case *protocol.Describe:
	case *protocol.FunctionCall:
		return true, c.handle_function(ctx, cast)
	case *protocol.Query:
		return true, c.handle_query(ctx, cast)
	case *protocol.Terminate:
		return false, nil
	default:
	}
	return true, nil
}

func (c *Client) handle_query(ctx context.Context, q *protocol.Query) error {
	// TODO extract query and do stuff based on it
	_, err := q.Write(c.server.wr)
	if err != nil {
		return err
	}
	for {
		var rsp protocol.Packet
		rsp, err = protocol.ReadBackend(c.server.r)
		if err != nil {
			return err
		}
		switch r := rsp.(type) {
		case *protocol.ReadyForQuery:
			if r.Fields.Status == 'I' {
				_, err = r.Write(c.wr)
				if err != nil {
					return err
				}
				return nil
			}
		case *protocol.CopyInResponse, *protocol.CopyOutResponse, *protocol.CopyBothResponse:
			err = c.handle_copy(ctx, rsp)
			if err != nil {
				return err
			}
			continue
		}
		_, err = rsp.Write(c.wr)
		if err != nil {
			return err
		}
	}
}

func (c *Client) handle_function(ctx context.Context, f *protocol.FunctionCall) error {
	_, err := f.Write(c.wr)
	if err != nil {
		return err
	}
	for {
		var rsp protocol.Packet
		rsp, err = protocol.ReadBackend(c.server.r)
		if err != nil {
			return err
		}
		_, err = rsp.Write(c.wr)
		if err != nil {
			return err
		}
		if r, ok := rsp.(*protocol.ReadyForQuery); ok {
			if r.Fields.Status == 'I' {
				break
			}
		}
	}

	return nil
}

func (c *Client) handle_copy(ctx context.Context, p protocol.Packet) error {
	_, err := p.Write(c.wr)
	if err != nil {
		return err
	}
	switch p.(type) {
	case *protocol.CopyInResponse:
	outer:
		for {
			var rsp protocol.Packet
			rsp, err = protocol.ReadFrontend(c.r)
			if err != nil {
				return err
			}
			// forward packet
			_, err = rsp.Write(c.server.wr)
			if err != nil {
				return err
			}

			switch rsp.(type) {
			case *protocol.CopyDone, *protocol.CopyFail:
				break outer
			}
		}
		return nil
	case *protocol.CopyOutResponse:
		for {
			var rsp protocol.Packet
			rsp, err = protocol.ReadBackend(c.server.r)
			if err != nil {
				return err
			}
			// forward packet
			_, err = rsp.Write(c.wr)
			if err != nil {
				return err
			}

			switch r := rsp.(type) {
			case *protocol.CopyDone:
				return nil
			case *protocol.ErrorResponse:
				e := new(PostgresError)
				e.Read(r)
				return e
			}
		}
	case *protocol.CopyBothResponse:
		// TODO fix this filthy hack, instead of going in parallel (like normal), read fields serially
		err = c.handle_copy(ctx, new(protocol.CopyInResponse))
		if err != nil {
			return err
		}
		err = c.handle_copy(ctx, new(protocol.CopyOutResponse))
		if err != nil {
			return err
		}
		return nil
	default:
		panic("unreachable")
	}
}

func todo() {
	//
	//    /// Handle cancel request.
	//    pub async fn cancel(
	//        read: S,
	//        write: T,
	//        addr: std::net::SocketAddr,
	//        mut bytes: BytesMut, // The rest of the startup message.
	//        client_server_map: ClientServerMap,
	//        shutdown: Receiver<()>,
	//    ) -> Result<Client<S, T>, Error> {
	//        let process_id = bytes.get_i32();
	//        let secret_key = bytes.get_i32();
	//        return Ok(Client {
	//            read: BufReader::new(read),
	//            write: write,
	//            addr,
	//            buffer: BytesMut::with_capacity(8196),
	//            cancel_mode: true,
	//            transaction_mode: false,
	//            process_id,
	//            secret_key,
	//            client_server_map,
	//            parameters: HashMap::new(),
	//            stats: get_reporter(),
	//            admin: false,
	//            last_address_id: None,
	//            last_server_id: None,
	//            pool_name: String::from("undefined"),
	//            username: String::from("undefined"),
	//            shutdown,
	//            connected_to_server: false,
	//        });
	//    }
	//
	//    /// Handle a connected and authenticated client.
	//    pub async fn handle(&mut self) -> Result<(), Error> {
	//        // The client wants to cancel a query it has issued previously.
	//        if self.cancel_mode {
	//            trace!("Sending CancelRequest");
	//
	//            let (process_id, secret_key, address, port) = {
	//                let guard = self.client_server_map.lock();
	//
	//                match guard.get(&(self.process_id, self.secret_key)) {
	//                    // Drop the mutex as soon as possible.
	//                    // We found the server the client is using for its query
	//                    // that it wants to cancel.
	//                    Some((process_id, secret_key, address, port)) => (
	//                        process_id.clone(),
	//                        secret_key.clone(),
	//                        address.clone(),
	//                        *port,
	//                    ),
	//
	//                    // The client doesn't know / got the wrong server,
	//                    // we're closing the connection for security reasons.
	//                    None => return Ok(()),
	//                }
	//            };
	//
	//            // Opens a new separate connection to the server, sends the backend_id
	//            // and secret_key and then closes it for security reasons. No other interactions
	//            // take place.
	//            return Ok(Server::cancel(&address, port, process_id, secret_key).await?);
	//        }
	//
	//        // The query router determines where the query is going to go,
	//        // e.g. primary, replica, which shard.
	//        let mut query_router = QueryRouter::new();
	//
	//        // Our custom protocol loop.
	//        // We expect the client to either start a transaction with regular queries
	//        // or issue commands for our sharding and server selection protocol.
	//        loop {
	//            trace!(
	//                "Client idle, waiting for message, transaction mode: {}",
	//                self.transaction_mode
	//            );
	//
	//            // Read a complete message from the client, which normally would be
	//            // either a `Q` (query) or `P` (prepare, extended protocol).
	//            // We can parse it here before grabbing a server from the pool,
	//            // in case the client is sending some custom protocol messages, e.g.
	//            // SET SHARDING KEY TO 'bigint';
	//
	//            let mut message = tokio::select! {
	//                _ = self.shutdown.recv() => {
	//                    if !self.admin {
	//                        error_response_terminal(
	//                            &mut self.write,
	//                            &format!("terminating connection due to administrator command")
	//                        ).await?;
	//                        return Ok(())
	//                    }
	//
	//                    // Admin clients ignore shutdown.
	//                    else {
	//                        read_message(&mut self.read).await?
	//                    }
	//                },
	//                message_result = read_message(&mut self.read) => message_result?
	//            };
	//
	//            // Avoid taking a server if the client just wants to disconnect.
	//            if message[0] as char == 'X' {
	//                debug!("Client disconnecting");
	//                return Ok(());
	//            }
	//
	//            // Handle admin database queries.
	//            if self.admin {
	//                debug!("Handling admin command");
	//                handle_admin(&mut self.write, message, self.client_server_map.clone()).await?;
	//                continue;
	//            }
	//
	//            // Get a pool instance referenced by the most up-to-date
	//            // pointer. This ensures we always read the latest config
	//            // when starting a query.
	//            let pool = match get_pool(self.pool_name.clone(), self.username.clone()) {
	//                Some(pool) => pool,
	//                None => {
	//                    error_response(
	//                        &mut self.write,
	//                        &format!(
	//                            "No pool configured for database: {:?}, user: {:?}",
	//                            self.pool_name, self.username
	//                        ),
	//                    )
	//                    .await?;
	//                    return Err(Error::ClientError);
	//                }
	//            };
	//            query_router.update_pool_settings(pool.settings.clone());
	//            let current_shard = query_router.shard();
	//
	//            // Handle all custom protocol commands, if any.
	//            match query_router.try_execute_command(message.clone()) {
	//                // Normal query, not a custom command.
	//                None => {
	//                    if query_router.query_parser_enabled() {
	//                        query_router.infer_role(message.clone());
	//                    }
	//                }
	//
	//                // SET SHARD TO
	//                Some((Command::SetShard, _)) => {
	//                    // Selected shard is not configured.
	//                    if query_router.shard() >= pool.shards() {
	//                        // Set the shard back to what it was.
	//                        query_router.set_shard(current_shard);
	//
	//                        error_response(
	//                            &mut self.write,
	//                            &format!(
	//                                "shard {} is more than configured {}, staying on shard {}",
	//                                query_router.shard(),
	//                                pool.shards(),
	//                                current_shard,
	//                            ),
	//                        )
	//                        .await?;
	//                    } else {
	//                        custom_protocol_response_ok(&mut self.write, "SET SHARD").await?;
	//                    }
	//                    continue;
	//                }
	//
	//                // SET PRIMARY READS TO
	//                Some((Command::SetPrimaryReads, _)) => {
	//                    custom_protocol_response_ok(&mut self.write, "SET PRIMARY READS").await?;
	//                    continue;
	//                }
	//
	//                // SET SHARDING KEY TO
	//                Some((Command::SetShardingKey, _)) => {
	//                    custom_protocol_response_ok(&mut self.write, "SET SHARDING KEY").await?;
	//                    continue;
	//                }
	//
	//                // SET SERVER ROLE TO
	//                Some((Command::SetServerRole, _)) => {
	//                    custom_protocol_response_ok(&mut self.write, "SET SERVER ROLE").await?;
	//                    continue;
	//                }
	//
	//                // SHOW SERVER ROLE
	//                Some((Command::ShowServerRole, value)) => {
	//                    show_response(&mut self.write, "server role", &value).await?;
	//                    continue;
	//                }
	//
	//                // SHOW SHARD
	//                Some((Command::ShowShard, value)) => {
	//                    show_response(&mut self.write, "shard", &value).await?;
	//                    continue;
	//                }
	//
	//                // SHOW PRIMARY READS
	//                Some((Command::ShowPrimaryReads, value)) => {
	//                    show_response(&mut self.write, "primary reads", &value).await?;
	//                    continue;
	//                }
	//            };
	//
	//            debug!("Waiting for connection from pool");
	//
	//            // Grab a server from the pool.
	//            let connection = match pool
	//                .get(query_router.shard(), query_router.role(), self.process_id)
	//                .await
	//            {
	//                Ok(conn) => {
	//                    debug!("Got connection from pool");
	//                    conn
	//                }
	//                Err(err) => {
	//                    // Clients do not expect to get SystemError followed by ReadyForQuery in the middle
	//                    // of extended protocol submission. So we will hold off on sending the actual error
	//                    // message to the client until we get 'S' message
	//                    match message[0] as char {
	//                        'P' | 'B' | 'E' | 'D' => (),
	//                        _ => {
	//                            error_response(
	//                                &mut self.write,
	//                                "could not get connection from the pool",
	//                            )
	//                            .await?;
	//                        }
	//                    };
	//
	//                    error!("Could not get connection from pool: {:?}", err);
	//
	//                    continue;
	//                }
	//            };
	//
	//            let mut reference = connection.0;
	//            let address = connection.1;
	//            let server = &mut *reference;
	//
	//            // Server is assigned to the client in case the client wants to
	//            // cancel a query later.
	//            server.claim(self.process_id, self.secret_key);
	//            self.connected_to_server = true;
	//
	//            // Update statistics.
	//            if let Some(last_address_id) = self.last_address_id {
	//                self.stats
	//                    .client_disconnecting(self.process_id, last_address_id);
	//            }
	//            self.stats.client_active(self.process_id, address.id);
	//
	//            self.last_address_id = Some(address.id);
	//            self.last_server_id = Some(server.process_id());
	//
	//            debug!(
	//                "Client {:?} talking to server {:?}",
	//                self.addr,
	//                server.address()
	//            );
	//
	//            // Set application_name if any.
	//            // TODO: investigate other parameters and set them too.
	//            if self.parameters.contains_key("application_name") {
	//                server
	//                    .set_name(&self.parameters["application_name"])
	//                    .await?;
	//            }
	//
	//            // Transaction loop. Multiple queries can be issued by the client here.
	//            // The connection belongs to the client until the transaction is over,
	//            // or until the client disconnects if we are in session mode.
	//            //
	//            // If the client is in session mode, no more custom protocol
	//            // commands will be accepted.
	//            loop {
	//                let mut message = if message.len() == 0 {
	//                    trace!("Waiting for message inside transaction or in session mode");
	//
	//                    match read_message(&mut self.read).await {
	//                        Ok(message) => message,
	//                        Err(err) => {
	//                            // Client disconnected inside a transaction.
	//                            // Clean up the server and re-use it.
	//                            // This prevents connection thrashing by bad clients.
	//                            if server.in_transaction() {
	//                                server.query("ROLLBACK").await?;
	//                                server.query("DISCARD ALL").await?;
	//                                server.set_name("pgcat").await?;
	//                            }
	//
	//                            return Err(err);
	//                        }
	//                    }
	//                } else {
	//                    let msg = message.clone();
	//                    message.clear();
	//                    msg
	//                };
	//
	//                // The message will be forwarded to the server intact. We still would like to
	//                // parse it below to figure out what to do with it.
	//                let original = message.clone();
	//
	//                let code = message.get_u8() as char;
	//                let _len = message.get_i32() as usize;
	//
	//                trace!("Message: {}", code);
	//
	//                match code {
	//                    // ReadyForQuery
	//                    'Q' => {
	//                        debug!("Sending query to server");
	//
	//                        self.send_and_receive_loop(code, original, server, &address, &pool)
	//                            .await?;
	//
	//                        if !server.in_transaction() {
	//                            // Report transaction executed statistics.
	//                            self.stats.transaction(self.process_id, address.id);
	//
	//                            // Release server back to the pool if we are in transaction mode.
	//                            // If we are in session mode, we keep the server until the client disconnects.
	//                            if self.transaction_mode {
	//                                break;
	//                            }
	//                        }
	//                    }
	//
	//                    // Terminate
	//                    'X' => {
	//                        // Client closing. Rollback and clean up
	//                        // connection before releasing into the pool.
	//                        // Pgbouncer closes the connection which leads to
	//                        // connection thrashing when clients misbehave.
	//                        if server.in_transaction() {
	//                            server.query("ROLLBACK").await?;
	//                            server.query("DISCARD ALL").await?;
	//                            server.set_name("pgcat").await?;
	//                        }
	//
	//                        self.release();
	//
	//                        return Ok(());
	//                    }
	//
	//                    // Parse
	//                    // The query with placeholders is here, e.g. `SELECT * FROM users WHERE email = $1 AND active = $2`.
	//                    'P' => {
	//                        self.buffer.put(&original[..]);
	//                    }
	//
	//                    // Bind
	//                    // The placeholder's replacements are here, e.g. 'user@email.com' and 'true'
	//                    'B' => {
	//                        self.buffer.put(&original[..]);
	//                    }
	//
	//                    // Describe
	//                    // Command a client can issue to describe a previously prepared named statement.
	//                    'D' => {
	//                        self.buffer.put(&original[..]);
	//                    }
	//
	//                    // Execute
	//                    // Execute a prepared statement prepared in `P` and bound in `B`.
	//                    'E' => {
	//                        self.buffer.put(&original[..]);
	//                    }
	//
	//                    // Sync
	//                    // Frontend (client) is asking for the query result now.
	//                    'S' => {
	//                        debug!("Sending query to server");
	//
	//                        self.buffer.put(&original[..]);
	//
	//                        self.send_and_receive_loop(
	//                            code,
	//                            self.buffer.clone(),
	//                            server,
	//                            &address,
	//                            &pool,
	//                        )
	//                        .await?;
	//
	//                        self.buffer.clear();
	//
	//                        if !server.in_transaction() {
	//                            self.stats.transaction(self.process_id, address.id);
	//
	//                            // Release server back to the pool if we are in transaction mode.
	//                            // If we are in session mode, we keep the server until the client disconnects.
	//                            if self.transaction_mode {
	//                                break;
	//                            }
	//                        }
	//                    }
	//
	//                    // CopyData
	//                    'd' => {
	//                        // Forward the data to the server,
	//                        // don't buffer it since it can be rather large.
	//                        self.send_server_message(server, original, &address, &pool)
	//                            .await?;
	//                    }
	//
	//                    // CopyDone or CopyFail
	//                    // Copy is done, successfully or not.
	//                    'c' | 'f' => {
	//                        self.send_server_message(server, original, &address, &pool)
	//                            .await?;
	//
	//                        let response = self.receive_server_message(server, &address, &pool).await?;
	//
	//                        match write_all_half(&mut self.write, response).await {
	//                            Ok(_) => (),
	//                            Err(err) => {
	//                                server.mark_bad();
	//                                return Err(err);
	//                            }
	//                        };
	//
	//                        if !server.in_transaction() {
	//                            self.stats.transaction(self.process_id, address.id);
	//
	//                            // Release server back to the pool if we are in transaction mode.
	//                            // If we are in session mode, we keep the server until the client disconnects.
	//                            if self.transaction_mode {
	//                                break;
	//                            }
	//                        }
	//                    }
	//
	//                    // Some unexpected message. We either did not implement the protocol correctly
	//                    // or this is not a Postgres client we're talking to.
	//                    _ => {
	//                        error!("Unexpected code: {}", code);
	//                    }
	//                }
	//            }
	//
	//            // The server is no longer bound to us, we can't cancel it's queries anymore.
	//            debug!("Releasing server back into the pool");
	//            self.stats.server_idle(server.process_id(), address.id);
	//            self.connected_to_server = false;
	//            self.release();
	//            self.stats.client_idle(self.process_id, address.id);
	//        }
	//    }
	//
	//    /// Release the server from the client: it can't cancel its queries anymore.
	//    pub fn release(&self) {
	//        let mut guard = self.client_server_map.lock();
	//        guard.remove(&(self.process_id, self.secret_key));
	//    }
	//
	//    async fn send_and_receive_loop(
	//        &mut self,
	//        code: char,
	//        message: BytesMut,
	//        server: &mut Server,
	//        address: &Address,
	//        pool: &ConnectionPool,
	//    ) -> Result<(), Error> {
	//        debug!("Sending {} to server", code);
	//
	//        self.send_server_message(server, message, &address, &pool)
	//            .await?;
	//
	//        // Read all data the server has to offer, which can be multiple messages
	//        // buffered in 8196 bytes chunks.
	//        loop {
	//            let response = self.receive_server_message(server, &address, &pool).await?;
	//
	//            match write_all_half(&mut self.write, response).await {
	//                Ok(_) => (),
	//                Err(err) => {
	//                    server.mark_bad();
	//                    return Err(err);
	//                }
	//            };
	//
	//            if !server.is_data_available() {
	//                break;
	//            }
	//        }
	//
	//        // Report query executed statistics.
	//        self.stats.query(self.process_id, address.id);
	//
	//        Ok(())
	//    }
	//
	//    async fn send_server_message(
	//        &self,
	//        server: &mut Server,
	//        message: BytesMut,
	//        address: &Address,
	//        pool: &ConnectionPool,
	//    ) -> Result<(), Error> {
	//        match server.send(message).await {
	//            Ok(_) => Ok(()),
	//            Err(err) => {
	//                pool.ban(address, self.process_id);
	//                Err(err)
	//            }
	//        }
	//    }
	//
	//    async fn receive_server_message(
	//        &mut self,
	//        server: &mut Server,
	//        address: &Address,
	//        pool: &ConnectionPool,
	//    ) -> Result<BytesMut, Error> {
	//        if pool.settings.user.statement_timeout > 0 {
	//            match tokio::time::timeout(
	//                tokio::time::Duration::from_millis(pool.settings.user.statement_timeout),
	//                server.recv(),
	//            )
	//            .await
	//            {
	//                Ok(result) => match result {
	//                    Ok(message) => Ok(message),
	//                    Err(err) => {
	//                        pool.ban(address, self.process_id);
	//                        error_response_terminal(
	//                            &mut self.write,
	//                            &format!("error receiving data from server: {:?}", err),
	//                        )
	//                        .await?;
	//                        Err(err)
	//                    }
	//                },
	//                Err(_) => {
	//                    error!(
	//                        "Statement timeout while talking to {:?} with user {}",
	//                        address, pool.settings.user.username
	//                    );
	//                    server.mark_bad();
	//                    pool.ban(address, self.process_id);
	//                    error_response_terminal(&mut self.write, "pool statement timeout").await?;
	//                    Err(Error::StatementTimeout)
	//                }
	//            }
	//        } else {
	//            match server.recv().await {
	//                Ok(message) => Ok(message),
	//                Err(err) => {
	//                    pool.ban(address, self.process_id);
	//                    error_response_terminal(
	//                        &mut self.write,
	//                        &format!("error receiving data from server: {:?}", err),
	//                    )
	//                    .await?;
	//                    Err(err)
	//                }
	//            }
	//        }
	//    }
	//}
	//
	//impl<S, T> Drop for Client<S, T> {
	//    fn drop(&mut self) {
	//        let mut guard = self.client_server_map.lock();
	//        guard.remove(&(self.process_id, self.secret_key));
	//
	//        // Dirty shutdown
	//        // TODO: refactor, this is not the best way to handle state management.
	//        if let Some(address_id) = self.last_address_id {
	//            self.stats.client_disconnecting(self.process_id, address_id);
	//
	//            if self.connected_to_server {
	//                if let Some(process_id) = self.last_server_id {
	//                    self.stats.server_idle(process_id, address_id);
	//                }
	//            }
	//        }
	//    }
	//}

}
