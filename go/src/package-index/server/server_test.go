package server

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net"
	"strings"
	"testing"
	"time"

	"package-index/index"
)

func newTestServer(t *testing.T) (*net.TCPListener, Server) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	tl := l.(*net.TCPListener)
	srv := Server{
		Index:            index.NewIndex(),
		MaxConns:         4,
		MaxMessageSize:   16,
		ConnReadTimeout:  1 * time.Second,
		ConnWriteTimeout: 1 * time.Second,
		AcceptDelay:      time.Second,
		ConnReadDelay:    time.Second,
	}
	go func() {
		log.Println(srv.Serve(tl))
	}()
	return tl, srv
}

func TestMaxConns(t *testing.T) {
	l, srv := newTestServer(t)
	defer l.Close()
	testMaxConns(t, l.Addr().String(), srv.MaxConns)
}

func TestMaxMessageSize(t *testing.T) {
	l, srv := newTestServer(t)
	defer l.Close()
	testMaxMessageSize(t, l.Addr().String(), srv.MaxMessageSize)
}

func TestConnReadTimeout(t *testing.T) {
	l, _ := newTestServer(t)
	defer l.Close()
	testConnReadTimeout(t, l.Addr().String())
}

func testMaxConns(t *testing.T, addr string, maxConns int) {
	log.Println("testMaxConns")
	var err error
	conns := make([]net.Conn, maxConns+2)
	defer func() {
		for i := range conns {
			if conns[i] != nil {
				conns[i].Close()
			}
		}
	}()
	for i := range conns {
		conns[i], err = net.Dial("tcp", addr)
		if err != nil {
			t.Fatal(err)
		}
		_, err = conns[i].Write([]byte("INDEX|aoeu|\n"))
		if err != nil {
			t.Fatal(err)
		}
		r := bufio.NewReader(conns[i])
		resp, err := r.ReadSlice('\n')
		if i < maxConns {
			if err != nil {
				t.Fatal(err)
			}
			if string(resp) != "OK\n" {
				t.Fatalf("unexpected resp: %s", resp)
			}
		} else {
			if err == nil {
				t.Fatalf("expected err from conn %d", i)
			}
			if !strings.Contains(err.Error(), "connection reset by peer") {
				t.Fatal("expected err to be 'connection reset by peer'")
			}
			netErr, ok := err.(net.Error)
			if !ok {
				t.Fatal("expected net.Error")
			}
			if !netErr.Temporary() {
				t.Fatal("expected net.Error to indicate Temporary() to the client")
			}
			if netErr.Timeout() {
				t.Fatal("server should not indicate Timeout() to the client")
			}
		}
	}
}

func testMaxMessageSize(t *testing.T, addr string, maxMessageSize int) {
	log.Println("testMaxMessageSize")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	try := func(size int) string {
		cruftSize := len([]byte("INDEX||\n"))
		pkg := genPkg(size - cruftSize)
		message := []byte(fmt.Sprintf("INDEX|%s|\n", pkg))
		if len(message) != size {
			t.Fatalf("tried to produce message of len %d but got one of len %d: %q", size, len(message), message)
		}
		log.Printf("sending %q", message)
		_, err = conn.Write(message)
		if err != nil {
			t.Fatal(err)
		}
		b, err := r.ReadSlice('\n')
		if err != nil {
			t.Fatal(err)
		}
		resp := string(b)
		log.Printf("response %q", resp)
		return resp
	}
	if resp := try(maxMessageSize); resp != "OK\n" {
		t.Fatal(resp)
	}
	if resp := try(maxMessageSize + 1); resp != "ERROR\n" {
		t.Fatal(resp)
	}
	if resp := try(maxMessageSize); resp != "OK\n" {
		t.Fatal(resp)
	}
	if resp := try(maxMessageSize * 4); resp != "ERROR\n" {
		t.Fatal(resp)
	}
	if resp := try(maxMessageSize - 1); resp != "OK\n" {
		t.Fatal(resp)
	}
}

var genPkgRunes = []rune("aoeusnth")

func genPkg(size int) string {
	r := make([]rune, size)
	for i := range r {
		r[i] = genPkgRunes[rand.Intn(len(genPkgRunes))]
	}
	return string(r)
}

func testConnReadTimeout(t *testing.T, addr string) {
	log.Println("testConnReadTimeout")
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	b := make([]byte, 1)
	// The server will close the connection after ConnReadTimeout because no
	// data has been written to conn. Read should return an error when the
	// server closes the connection.
	_, err = conn.Read(b)
	if err == nil {
		t.Fatal()
	}
}
