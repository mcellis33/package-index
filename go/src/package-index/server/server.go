package server

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"package-index/index"
)

type Server struct {
	Index index.Index

	// TCP address to listen on.
	Addr string

	// Maximum number of concurrent connections that this server can accept.
	MaxConns int

	// Maximum size of messages that this server can accept. If a client sends
	// a message that is too large, the server will send an error response.
	MaxMessageSize int

	// The protocol spec omits heartbeating. The server sets a read and write
	// deadline on each TCP connection so that we do not block forever waiting
	// for a dead client.
	//
	// TODO: We could use TCP keepalive to support clients that send messages
	// far apart in time. I decided not to do this because the interface to
	// TCP keepalives in Go is incomplete - it yields platform-dependent
	// results. See this blog post for more info:
	//
	// http://felixge.de/2014/08/26/tcp-keepalive-with-golang.html

	// If the client does not send a message for longer than this the server
	// will close the connection.
	ConnReadTimeout time.Duration

	// If the client does not accept a response for longer than this the
	// server will close the connection.
	ConnWriteTimeout time.Duration

	// Time to wait before retrying Accept after a temporary network error.
	AcceptDelay time.Duration
	// Time to wait before retrying Read after a temporary network error.
	ConnReadDelay time.Duration
}

func (s *Server) ListenAndServe() error {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("Listen: %v", err)
	}
	return s.Serve(l.(*net.TCPListener))
}

func (s *Server) Serve(l *net.TCPListener) error {
	// Limit the number of concurrent connections the server tries to handle.
	// This limits the server-side memory allocations a client can trigger.
	// However, a client can still attack the server by running it out of file
	// descriptors. In production it would probably be most practical to
	// handle this scenario in a load balancer, for example:
	// https://www.nginx.com/resources/admin-guide/restricting-access-tcp/
	outstanding := make(chan struct{}, s.MaxConns)
	// bufPool is used to pool message read buffers. This minimizes the impact
	// of buffer allocation on response latency.
	bufPool := &bufioReaderPool{BufSize: s.MaxMessageSize}
	for {
		conn, err := l.AcceptTCP()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				// TODO: implement backoff like net/http/server.go:2123.
				// For rationale see https://www.awsarchitectureblog.com/2015/03/backoff.html
				log.Printf("AcceptTCP: temporary error, sleeping for %v: %v", s.AcceptDelay, err)
				time.Sleep(s.AcceptDelay)
				continue
			}
			return fmt.Errorf("Serve: %v", err)
		}
		select {
		case outstanding <- struct{}{}:
			go s.serve(conn, outstanding, bufPool)
		default:
			log.Printf("too many connections, closing")
			conn.Close()
		}
	}
}

func (s *Server) serve(conn *net.TCPConn, outstanding <-chan struct{}, bufPool *bufioReaderPool) {
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Printf("Conn.Close: %v", err)
		}
		<-outstanding
	}()
	for {
		err := conn.SetReadDeadline(time.Now().Add(s.ConnReadTimeout))
		if err != nil {
			log.Printf("Conn.Read deadline set err, closing: %v", err)
			return
		}
		message, err := readMessage(conn, bufPool)
		if err != nil {
			if err == io.EOF {
				// The client closed the connection gracefully.
				return
			} else if netErr, ok := err.(net.Error); ok {
				if netErr.Timeout() {
					log.Printf("dead client %v, closing connection", conn.RemoteAddr())
					return
				}
				if netErr.Temporary() {
					// TODO: implement backoff
					log.Printf("Conn.Read temporary network error, sleeping for %v: %v", s.ConnReadDelay, netErr)
					time.Sleep(s.ConnReadDelay)
					continue
				}
			}
			// NB this should be hit in the netErr.Timeout() case.
			log.Printf("readMessage: %v", err)
			respond(ErrorResponse, conn, s.ConnWriteTimeout)
			continue
		}
		var ok bool
		switch message.Command {
		case "INDEX":
			ok = s.Index.Index(message.Package, message.Dependencies)
		case "REMOVE":
			ok = s.Index.Remove(message.Package)
		case "QUERY":
			ok = s.Index.Query(message.Package)
		default:
			// Command not recognized
			respond(ErrorResponse, conn, s.ConnWriteTimeout)
			continue
		}
		if ok {
			respond(OKResponse, conn, s.ConnWriteTimeout)
		} else {
			respond(FailResponse, conn, s.ConnWriteTimeout)
		}
	}
}

func readMessage(conn *net.TCPConn, bufPool *bufioReaderPool) (Message, error) {
	// Maintainability note: do not let messageBytes or buf escape this scope.
	buf := bufPool.Get(conn)
	defer bufPool.Put(buf)
	messageBytes, err := buf.ReadSlice('\n')
	// If the message is too large, discard the rest of the message and return
	// bufio.ErrBufferFull. Exception: if we hit a different error while
	// discarding the rest of the message, return that error.
	if err == bufio.ErrBufferFull {
		for err == bufio.ErrBufferFull {
			_, err = buf.ReadSlice('\n')
		}
		if err != nil {
			return Message{}, err
		}
		return Message{}, bufio.ErrBufferFull
	}
	if err != nil {
		return Message{}, err
	}
	return parseMessage(messageBytes)
}

var (
	ErrorResponse = []byte("ERROR\n")
	OKResponse    = []byte("OK\n")
	FailResponse  = []byte("FAIL\n")
)

func respond(resp []byte, conn *net.TCPConn, timeout time.Duration) {
	err := conn.SetWriteDeadline(time.Now().Add(timeout))
	if err != nil {
		log.Printf("Conn.Write deadline set err: %v", err)
		// If we can't set a deadline don't block.
		return
	}
	_, err = conn.Write(resp)
	if err != nil {
		// TODO: backoff on temporary net errors?
		log.Printf("Conn.Write: %v", err)
	}
}
