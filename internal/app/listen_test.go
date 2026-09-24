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
