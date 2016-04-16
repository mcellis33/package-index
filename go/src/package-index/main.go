package main

import (
	"flag"
	"log"
	"os"
	"package-index/index"
	"package-index/server"
	"time"
)

func main() {
	srv := server.Server{}
	flag.StringVar(&srv.Addr, "addr", ":8080", "TCP address to listen on")
	flag.IntVar(&srv.MaxConns, "max-conns", 300, "Maximum number of concurrent connections")
	flag.IntVar(&srv.MaxMessageSize, "max-message-size", 2048, "Maximum message size; server will respond with ERROR when exceeded")
	flag.DurationVar(&srv.ConnReadTimeout, "conn-read-timeout", 30*time.Second, "If the client does not send a message for longer than this the server will close the connection")
	flag.DurationVar(&srv.ConnWriteTimeout, "conn-write-timeout", 5*time.Second, "If the client does not accept a response for longer than this the server will close the connection")
	flag.DurationVar(&srv.AcceptDelay, "accept-delay", time.Second, "Time to wait before retrying Accept after a temporary network error.")
	flag.DurationVar(&srv.ConnReadDelay, "conn-read-delay", time.Second, "Time to wait before retrying Read after a temporary network error.")
	flag.Parse()
	srv.Index = index.NewIndex()
	// TODO: gracefully shut down (close listener and wait for outstanding
	// operations to complete) on os.Interrupt signal.
	err := srv.ListenAndServe()
	if err != nil {
		log.Printf("ListenAndServe: %v", err)
		os.Exit(1)
	}
}
