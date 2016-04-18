package main

import (
	"fmt"
	"net"
	"testing"
)

func respondWith(t *testing.T, server net.Listener, responseCode string) {
	for {
		conn, err := server.Accept()
		if err != nil {
			t.Fatalf("Error reading socket: %v", err)
		}
		fmt.Fprintln(conn, responseCode)
	}
}

func TestMakeTCPPackageIndexClient(t *testing.T) {
	client, err := MakeTCPPackageIndexClient("portisntopen", ":8089")

	if err == nil {
		t.Errorf("Expected connection to [8089] to raise error as there's no server, got %v", client)
	}
}

func TestSend(t *testing.T) {
	goodAddr := ":8080"
	goodServer, err := net.Listen("tcp", goodAddr)
	defer goodServer.Close()

	if err != nil {
		t.Fatalf("Error opening test server: %v", err)
	}

	go respondWith(t, goodServer, "OK")

	client, err := MakeTCPPackageIndexClient("goodAddr", goodAddr)
	if err != nil {
		t.Fatalf("Error connecting to server: %v", err)
	}

	responseCode, err := client.Send("A")

	if err != nil {
		t.Errorf("Error sending message to server: %v", err)
	}

	if responseCode == FAIL {
		t.Errorf("Expected responseCode to be 1, got %v", responseCode)
	}

	badAddr := ":8090"
	badServer, err := net.Listen("tcp", badAddr)
	defer badServer.Close()

	if err != nil {
		t.Fatalf("Error opening test server: %v", err)
	}

	go respondWith(t, badServer, "banana")

	client, err = MakeTCPPackageIndexClient("badAddr", badAddr)
	if err != nil {
		t.Fatalf("Error connecting to server: %v", err)
	}

	responseCode, err = client.Send("B")

	if err == nil {
		t.Errorf("No error returned for bad responseCode from server: %#v", responseCode)
	}
}
