package runtime_test

import (
	"net"
	"testing"

	"github.com/rayip/rayip/services/node-agent/internal/runtime"
)

func TestResolveXrayAPIAddrAutoAllocatesLoopbackPort(t *testing.T) {
	addr, err := runtime.ResolveXrayAPIAddr("auto")
	if err != nil {
		t.Fatalf("ResolveXrayAPIAddr() error = %v", err)
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("SplitHostPort() error = %v", err)
	}
	if net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() || port == "0" {
		t.Fatalf("resolved addr = %q, want loopback concrete port", addr)
	}
}

func TestResolveXrayAPIAddrRejectsOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer listener.Close()

	if _, err := runtime.ResolveXrayAPIAddr(listener.Addr().String()); err == nil {
		t.Fatal("ResolveXrayAPIAddr() error = nil, want occupied port error")
	}
}

func TestResolveXrayAPIAddrRejectsNonLoopback(t *testing.T) {
	if _, err := runtime.ResolveXrayAPIAddr("0.0.0.0:10085"); err == nil {
		t.Fatal("ResolveXrayAPIAddr() error = nil, want non-loopback rejection")
	}
}
