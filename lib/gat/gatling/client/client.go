package client

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/parse"
	"io"
	"math/big"
	"net"
	"reflect"
	"strings"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/messages"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/gat/protocol/pg_error"
	"git.tuxpa.in/a/zlog"
	"git.tuxpa.in/a/zlog/log"
	"github.com/ethereum/go-ethereum/common/math"
)

// / client state, one per client
type Client struct {
	conn net.Conn
	r    *bufio.Reader
	wr   io.Writer

	bufwr *bufio.Writer
	recv  chan protocol.Packet

	buf bytes.Buffer

	addr net.Addr

	cancel_mode bool
	txn_mode    bool

	pid        int32
	secret_key int32

	parameters map[string]string
	stats      any // TODO: Reporter
	admin      bool

	server gat.ConnectionPool

	last_addr_id int
	last_srv_id  int

	pool_name string
	username  string

	gatling gat.Gat
	conf    *config.Global

	log zlog.Logger
}

func NewClient(
	gatling gat.Gat,
	conf *config.Global,
	conn net.Conn,
	admin_only bool,
) *Client {
	c := &Client{
		conn:    conn,
		r:       bufio.NewReader(conn),
		wr:      conn,
		bufwr:   bufio.NewWriter(conn),
		recv:    make(chan protocol.Packet),
		addr:    conn.RemoteAddr(),
		gatling: gatling,
		conf:    conf,
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
			c.bufwr.Reset(c.wr)
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
		return &pg_error.Error{
			Severity: pg_error.Fatal,
			Code:     pg_error.InvalidAuthorizationSpecification,
			Message:  "param database required",
		}
	}

	c.username, ok = params["user"]
	if !ok {
		return &pg_error.Error{
			Severity: pg_error.Fatal,
			Code:     pg_error.InvalidAuthorizationSpecification,
			Message:  "param user required",
		}
	}

	c.admin = (c.pool_name == "pgcat" || c.pool_name == "pgbouncer")

	if c.conf.General.AdminOnly && !c.admin {
		c.log.Debug().Msg("rejected non admin, since admin only mode")
		return &pg_error.Error{
			Severity: pg_error.Fatal,
			Code:     pg_error.InvalidAuthorizationSpecification,
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
	pkt, salt, err := messages.CreateMd5Challenge()
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
		return &pg_error.Error{
			Severity: pg_error.Fatal,
			Code:     pg_error.InvalidAuthorizationSpecification,
			Message:  fmt.Sprintf("wanted AuthenticationResponse packet, got '%+v'", rsp),
		}
	}

	var pool gat.Pool
	pool, err = c.gatling.GetPool(c.pool_name)
	if err != nil {
		return err
	}

	// get user
	var user *config.User
	user, err = pool.GetUser(c.username)
	if err != nil {
		return err
	}

	// Authenticate admin user.
	if c.admin {
		pw_hash := messages.Md5HashPassword(c.conf.General.AdminUsername, c.conf.General.AdminPassword, salt[:])
		if !reflect.DeepEqual(pw_hash, passwordResponse) {
			return &pg_error.Error{
				Severity: pg_error.Fatal,
				Code:     pg_error.InvalidPassword,
				Message:  "invalid password",
			}
		}
	} else {
		pw_hash := messages.Md5HashPassword(c.username, user.Password, salt[:])
		if !reflect.DeepEqual(pw_hash, passwordResponse) {
			return &pg_error.Error{
				Severity: pg_error.Fatal,
				Code:     pg_error.InvalidPassword,
				Message:  "invalid password",
			}
		}
	}

	c.server, err = pool.WithUser(c.username)
	if err != nil {
		return err
	}

	authOk := new(protocol.Authentication)
	authOk.Fields.Code = 0
	_, err = authOk.Write(c.wr)
	if err != nil {
		return err
	}

	//
	info := c.server.GetServerInfo()
	for _, inf := range info {
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
	go c.recvLoop()
	open := true
	for open {
		open, err = c.tick(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) recvLoop() {
	for {
		recv, err := protocol.ReadFrontend(c.r)
		if err != nil {
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				log.Err(err)
			}
			break
		}
		//log.Printf("got packet(%s) %+v", reflect.TypeOf(recv), recv)
		c.recv <- recv
	}
}

// TODO: we need to keep track of queries so we can handle cancels
func (c *Client) handle_cancel(ctx context.Context, p *protocol.StartupMessage) error {
	log.Println("cancel msg", p)
	return nil
}

// reads a packet from stream and handles it
func (c *Client) tick(ctx context.Context) (bool, error) {
	var rsp protocol.Packet
	select {
	case rsp = <-c.Recv():
	case <-ctx.Done():
		return false, ctx.Err()
	}
	switch cast := rsp.(type) {
	case *protocol.Query:
		return true, c.handle_query(ctx, cast)
	case *protocol.FunctionCall:
		return true, c.handle_function(ctx, cast)
	case *protocol.Terminate:
		return false, nil
	default:
	}
	return true, nil
}

func (c *Client) handle_query(ctx context.Context, q *protocol.Query) error {
	parsed, err := parse.Parse(q.Fields.Query)
	if err != nil {
		return err
	}

	// we can handle empty queries here
	if len(parsed) == 0 {
		err = c.Send(&protocol.EmptyQueryResponse{})
		if err != nil {
			return err
		}
		ready := new(protocol.ReadyForQuery)
		ready.Fields.Status = 'I'
		return c.Send(ready)
	}

	prev := 0
	transaction := false
	for idx, cmd := range parsed {
		switch strings.ToUpper(cmd.Command) {
		case "START":
			if len(cmd.Arguments) < 1 || strings.ToUpper(cmd.Arguments[0]) != "TRANSACTION" {
				break
			}
			fallthrough
		case "BEGIN":
			// begin transaction
			if prev != cmd.Index {
				query := q.Fields.Query[prev:cmd.Index]
				err = c.handle_simple_query(ctx, query)
				prev = cmd.Index
				if err != nil {
					return err
				}
			}
			transaction = true
		case "END":
			// end transaction block
			var query string
			if idx+1 >= len(parsed) {
				query = q.Fields.Query[prev:]
			} else {
				query = q.Fields.Query[prev:parsed[idx+1].Index]
			}
			if query != "" {
				err = c.handle_transaction(ctx, query)
				prev = cmd.Index
				if err != nil {
					return err
				}
			}
			transaction = false

		}
	}
	query := q.Fields.Query[prev:]
	if transaction {
		err = c.handle_transaction(ctx, query)
	} else {
		err = c.handle_simple_query(ctx, query)
	}
	return err
}

func (c *Client) handle_simple_query(ctx context.Context, q string) error {
	log.Println("query", q)
	//log.Println("query: ", q.Fields.Query)
	return c.server.SimpleQuery(ctx, c, q)
}

func (c *Client) handle_transaction(ctx context.Context, q string) error {
	log.Println("transaction", q)
	return c.server.Transaction(ctx, c, q)
}

func (c *Client) handle_function(ctx context.Context, f *protocol.FunctionCall) error {
	err := c.server.CallFunction(ctx, c, f)
	if err != nil {
		return err
	}
	return err
}

func (c *Client) Send(pkt protocol.Packet) error {
	//log.Printf("sent packet(%s) %+v", reflect.TypeOf(pkt), pkt)
	_, err := pkt.Write(c.bufwr)
	if err != nil {
		c.bufwr.Reset(c.wr)
		return err
	}
	return c.bufwr.Flush()
}

func (c *Client) Recv() <-chan protocol.Packet {
	return c.recv
}

var _ gat.Client = (*Client)(nil)

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
	//    ) -> Result<Client<S, T>, Err> {
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
	//    pub async fn handle(&mut self) -> Result<(), Err> {
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
	//                    return Err(Err::ClientError);
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
	//    ) -> Result<(), Err> {
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
	//    ) -> Result<(), Err> {
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
	//    ) -> Result<BytesMut, Err> {
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
	//                    Err(Err::StatementTimeout)
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
