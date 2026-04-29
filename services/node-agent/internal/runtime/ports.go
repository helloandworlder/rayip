package runtime

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

func ResolveXrayAPIAddr(raw string) (string, error) {
	addr := strings.TrimSpace(raw)
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	if strings.EqualFold(addr, "auto") {
		addr = "127.0.0.1:0"
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}
	if host == "" {
		host = "127.0.0.1"
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", fmt.Errorf("xray gRPC address must listen on loopback, got %s", host)
	}
	if port == "0" {
		listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
		if err != nil {
			return "", fmt.Errorf("allocate xray gRPC port failed: %w", err)
		}
		resolved := listener.Addr().String()
		_ = listener.Close()
		return resolved, nil
	}
	if _, err := strconv.Atoi(port); err != nil {
		return "", err
	}
	if err := EnsureListenAddrAvailable(net.JoinHostPort(host, port)); err != nil {
		return "", fmt.Errorf("xray gRPC address unavailable %s: %w", net.JoinHostPort(host, port), err)
	}
	return net.JoinHostPort(host, port), nil
}

func EnsureListenAddrAvailable(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return listener.Close()
}
