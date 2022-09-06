package gat

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"

	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

// TODO: decide which of these we need and don't need.
// impelement the ones we need

// / Generate md5 password challenge.
func CreateMd5Challenge() (*protocol.Authentication, [4]byte, error) {
	salt := [4]byte{}
	_, err := rand.Read(salt[:])
	if err != nil {
		return nil, salt, err
	}

	pkt := new(protocol.Authentication)
	pkt.Fields.Code = 5
	pkt.Fields.Salt = salt[:]

	return pkt, salt, nil
}

// Create md5 password hash given a salt.
func Md5HashPassword(user string, password string, salt []byte) []byte {
	hsh1, hsh2 := md5.New(), md5.New()
	hsh1.Write([]byte(user + password))
	hsh2.Write(
		[]byte(hex.EncodeToString(hsh1.Sum(nil))),
	)
	hsh2.Write(salt)
	sum := hsh2.Sum(nil)
	return append([]byte("md5"+hex.EncodeToString(sum)), 0)
}

// /// Implements a response to our custom `SET SHARDING KEY`
// /// and `SET SERVER ROLE` commands.
// /// This tells the client we're ready for the next query.
// func custom_protocol_response_ok(w io.Writer, message: &str)  error
// where
//
//	S: tokio::io::AsyncWrite + std::marker::Unpin,
//
//	{
//	    res = BytesMut::with_capacity(25)
//
//	   let set_complete = BytesMut::from(&format!("{}\0", message)[..])
//	   let len = (set_complete.len() + 4) as i32
//
//	   // CommandComplete
//	   res.put_u8(b'C')
//	   res.put_i32(len)
//	   res.put_slice(&set_complete[..])
//
//	   write_all_half(stream, res).await?
//	   ready_for_query(stream).await
//	}
//
// /// Send a custom error message to the client.
// /// Tell the client we are ready for the next query and no rollback is necessary.
// /// Docs on error codes: <https://www.postgresql.org/docs/12/errcodes-appendix.html>.
// func error_response(w io.Writer, message: &str)  error
// where
//
//	S: tokio::io::AsyncWrite + std::marker::Unpin,
//
//	{
//	   error_response_terminal(stream, message).await?
//	   ready_for_query(stream).await
//	}
//
// /// Send a custom error message to the client.
// /// Tell the client we are ready for the next query and no rollback is necessary.
// /// Docs on error codes: <https://www.postgresql.org/docs/12/errcodes-appendix.html>.
// func error_response_terminal(w io.Writer, message: &str)  error
// where
//
//	S: tokio::io::AsyncWrite + std::marker::Unpin,
//
//	{
//	    error = BytesMut::new()
//
//	   // error level
//	   error.put_u8(b'S')
//	   error.put_slice(&b"FATAL\0"[..])
//
//	   // error level (non-translatable)
//	   error.put_u8(b'V')
//	   error.put_slice(&b"FATAL\0"[..])
//
//	   // error code: not sure how much this matters.
//	   error.put_u8(b'C')
//	   error.put_slice(&b"58000\0"[..]) // system_error, see Appendix A.
//
//	   // The short error message.
//	   error.put_u8(b'M')
//	   error.put_slice(&format!("{}\0", message).as_bytes())
//
//	   // No more fields follow.
//	   error.put_u8(0)
//
//	   // Compose the two message reply.
//	    res = BytesMut::with_capacity(error.len() + 5)
//
//	   res.put_u8(b'E')
//	   res.put_i32(error.len() as i32 + 4)
//	   res.put(error)
//
//	   Ok(write_all_half(stream, res).await?)
//	}
//
// func wrong_password(w io.Writer, user: &str)  error
// where
//
//	S: tokio::io::AsyncWrite + std::marker::Unpin,
//
//	{
//	    error = BytesMut::new()
//
//	   // error level
//	   error.put_u8(b'S')
//	   error.put_slice(&b"FATAL\0"[..])
//
//	   // error level (non-translatable)
//	   error.put_u8(b'V')
//	   error.put_slice(&b"FATAL\0"[..])
//
//	   // error code: not sure how much this matters.
//	   error.put_u8(b'C')
//	   error.put_slice(&b"28P01\0"[..]) // system_error, see Appendix A.
//
//	   // The short error message.
//	   error.put_u8(b'M')
//	   error.put_slice(&format!("password authentication failed for user \"{}\"\0", user).as_bytes())
//
//	   // No more fields follow.
//	   error.put_u8(0)
//
//	   // Compose the two message reply.
//	    res = BytesMut::new()
//
//	   res.put_u8(b'E')
//	   res.put_i32(error.len() as i32 + 4)
//
//	   res.put(error)
//
//	   write_all(stream, res).await
//	}
//
// /// Respond to a SHOW SHARD command.
// func show_response(w io.Writer, name: &str, value: &str)  error
// where
//
//	S: tokio::io::AsyncWrite + std::marker::Unpin,
//
//	{
//	   // A SELECT response consists of:
//	   // 1. RowDescription
//	   // 2. One or more DataRow
//	   // 3. CommandComplete
//	   // 4. ReadyForQuery
//
//	   // The final messages sent to the client
//	    res = BytesMut::new()
//
//	   // RowDescription
//	   res.put(row_description(&vec![(name, DataType::Text)]))
//
//	   // DataRow
//	   res.put(data_row(&vec![value.to_string()]))
//
//	   // CommandComplete
//	   res.put(command_complete("SELECT 1"))
//
//	   write_all_half(stream, res).await?
//	   ready_for_query(stream).await
//	}
//
//	pub fn row_description(columns: &Vec<(&str, DataType)>)  BytesMut {
//	    res = BytesMut::new()
//	    row_desc = BytesMut::new()
//
//	   // how many colums we are storing
//	   row_desc.put_i16(columns.len() as i16)
//
//	   for (name, data_type) in columns {
//	       // Column name
//	       row_desc.put_slice(&format!("{}\0", name).as_bytes())
//
//	       // Doesn't belong to any table
//	       row_desc.put_i32(0)
//
//	       // Doesn't belong to any table
//	       row_desc.put_i16(0)
//
//	       // Text
//	       row_desc.put_i32(data_type.into())
//
//	       // Text size = variable (-1)
//	       let type_size = match data_type {
//	           DataType::Text => -1,
//	           DataType::Int4 => 4,
//	           DataType::Numeric => -1,
//	       }
//
//	       row_desc.put_i16(type_size)
//
//	       // Type modifier: none that I know
//	       row_desc.put_i32(-1)
//
//	       // Format being used: text (0), binary (1)
//	       row_desc.put_i16(0)
//	   }
//
//	   res.put_u8(b'T')
//	   res.put_i32(row_desc.len() as i32 + 4)
//	   res.put(row_desc)
//
//	   res
//	}
//
// /// Create a DataRow message.
//
//	pub fn data_row(row: &Vec<String>)  BytesMut {
//	    res = BytesMut::new()
//	    data_row = BytesMut::new()
//
//	   data_row.put_i16(row.len() as i16)
//
//	   for column in row {
//	       let column = column.as_bytes()
//	       data_row.put_i32(column.len() as i32)
//	       data_row.put_slice(&column)
//	   }
//
//	   res.put_u8(b'D')
//	   res.put_i32(data_row.len() as i32 + 4)
//	   res.put(data_row)
//
//	   res
//	}
//
// /// Create a CommandComplete message.
//
//	pub fn command_complete(command: &str)  BytesMut {
//	   let cmd = BytesMut::from(format!("{}\0", command).as_bytes())
//	    res = BytesMut::new()
//	   res.put_u8(b'C')
//	   res.put_i32(cmd.len() as i32 + 4)
//	   res.put(cmd)
//	   res
//	}
//
// /// Write all data in the buffer to the TcpStream.
// func write_all(w io.Writer, buf: BytesMut)  error
// where
//
//	S: tokio::io::AsyncWrite + std::marker::Unpin,
//
//	{
//	   match stream.write_all(&buf).await {
//	       Ok(_) => Ok(()),
//	       Err(_) => return Err(error::Socketerror),
//	   }
//	}
//
// /// Write all the data in the buffer to the TcpStream, write owned half (see mpsc).
// func write_all_half(w io.Writer, buf: BytesMut)  error
// where
//
//	S: tokio::io::AsyncWrite + std::marker::Unpin,
//
//	{
//	   match stream.write_all(&buf).await {
//	       Ok(_) => Ok(()),
//	       Err(_) => return Err(error::Socketerror),
//	   }
//	}
//
// /// Read a complete message from the socket.
// func read_message(w io.Writer)  Result<BytesMut, error
// where
//
//	S: tokio::io::AsyncRead + std::marker::Unpin,
//
//	{
//	   let code = match stream.read_u8().await {
//	       Ok(code) => code,
//	       Err(_) => return Err(error::Socketerror),
//	   }
//
//	   let len = match stream.read_i32().await {
//	       Ok(len) => len,
//	       Err(_) => return Err(error::Socketerror),
//	   }
//
//	    buf = vec![0u8 len as usize - 4]
//
//	   match stream.read_exact(&mut buf).await {
//	       Ok(_) => (),
//	       Err(_) => return Err(error::Socketerror),
//	   }
//
//	    bytes = BytesMut::with_capacity(len as usize + 1)
//
//	   bytes.put_u8(code)
//	   bytes.put_i32(len)
//	   bytes.put_slice(&buf)
//
//	   Ok(bytes)
//	}
func ServerParameterMessage(key, value string) []byte {
	buf := new(bytes.Buffer)
	pkt := new(protocol.ParameterStatus)
	pkt.Fields.Parameter = key
	pkt.Fields.Value = value
	_, _ = pkt.Write(buf)

	return buf.Bytes()
}
