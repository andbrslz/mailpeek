package app

import (
	"net"
	"strconv"
	"testing"
)

func TestListenLocalhostAcceptsBothLoopbacks(t *testing.T) {
	l, err := listen("localhost", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	port := strconv.Itoa(l.Addr().(*net.TCPAddr).Port)

	targets := []string{net.JoinHostPort("127.0.0.1", port)}
	if probe, err := net.Listen("tcp6", "[::1]:0"); err == nil {
		probe.Close()
		targets = append(targets, net.JoinHostPort("::1", port))
	}
	for _, target := range targets {
		conn, err := net.Dial("tcp", target)
		if err != nil {
			t.Fatalf("dial %s: %v", target, err)
		}
		accepted, err := l.Accept()
		if err != nil {
			t.Fatalf("accept from %s: %v", target, err)
		}
		accepted.Close()
		conn.Close()
	}

	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := l.Accept(); err == nil {
		t.Fatal("Accept after Close should fail")
	}
}

func TestListenLocalhostIsNotReachableFromOtherInterfaces(t *testing.T) {
	l, err := listen("localhost", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if ip := l.Addr().(*net.TCPAddr).IP; !ip.IsLoopback() {
		t.Fatalf("bound to %s, want a loopback address", ip)
	}
}

func TestLoopback(t *testing.T) {
	for host, want := range map[string]bool{
		"": true, "localhost": true, "127.0.0.1": true, "127.0.0.2": true, "::1": true,
		"0.0.0.0": false, "::": false, "192.168.1.20": false, "mailpeek": false,
	} {
		if got := loopback(host); got != want {
			t.Errorf("loopback(%q) = %v, want %v", host, got, want)
		}
	}
}
