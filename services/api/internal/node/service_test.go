package node_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/rayip/rayip/services/api/internal/node"
)

func TestRegisterLeaseCreatesOnlineNode(t *testing.T) {
	now := time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)
	repo := node.NewMemoryRepository()
	leases := node.NewMemoryLeaseStore()
	svc := node.NewService(repo, leases, func() time.Time { return now })

	_, err := svc.RegisterLease(context.Background(), node.LeaseInput{
		NodeCode:        "nyc-home-001",
		SessionID:       "session-1",
		APIInstanceID:   "api-1",
		BundleVersion:   "bundle-0.1.0",
		AgentVersion:    "agent-0.1.0",
		XrayVersion:     "xray-0.1.0",
		Capabilities:    []string{"socks5", "http"},
		CandidatePublicIPs: []string{
			"198.51.100.10",
			"198.51.100.11",
		},
		ScanHost:        "node.example.net",
		ProbePort:       18080,
		ProbeProtocols:  []string{"SOCKS5"},
		ProbeCheckedAt:  now.Add(-time.Minute),
		Sequence:        7,
		LeaseTTLSeconds: 45,
	})
	if err != nil {
		t.Fatalf("RegisterLease() error = %v", err)
	}

	nodes, err := svc.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("node count = %d, want 1", len(nodes))
	}

	got := nodes[0]
	if got.Code != "nyc-home-001" || got.Status != node.StatusOnline {
		t.Fatalf("unexpected node summary: %#v", got)
	}
	if got.BundleVersion != "bundle-0.1.0" || got.APIInstanceID != "api-1" {
		t.Fatalf("lease metadata was not retained: %#v", got)
	}
	if got.PublicIP != "" || got.ScanHost != "node.example.net" || got.ProbePort != 18080 || got.ProbeCheckedAt.IsZero() {
		t.Fatalf("probe metadata was not retained: %#v", got)
	}
	if len(got.CandidatePublicIPs) != 2 || got.CandidatePublicIPs[0] != "198.51.100.10" || got.CandidatePublicIPs[1] != "198.51.100.11" {
		t.Fatalf("candidate public ips were not retained: %#v", got.CandidatePublicIPs)
	}
	if len(got.ProbeProtocols) != 1 || got.ProbeProtocols[0] != "SOCKS5" {
		t.Fatalf("probe protocols = %#v", got.ProbeProtocols)
	}
}

func TestScanNodeScansEachCandidatePublicIP(t *testing.T) {
	now := time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)
	repo := node.NewMemoryRepository()
	leases := node.NewMemoryLeaseStore()
	svc := node.NewService(repo, leases, func() time.Time { return now })

	summary, err := svc.RegisterLease(context.Background(), node.LeaseInput{
		NodeCode:           "scan-home-ips",
		SessionID:          "session-1",
		CandidatePublicIPs: []string{"204.42.251.2", "204.42.251.3"},
		ProbePort:          9878,
		LeaseTTLSeconds:    45,
	})
	if err != nil {
		t.Fatalf("RegisterLease() error = %v", err)
	}

	var targets []string
	svc.SetDialerForTest(func(_ context.Context, _, address string) (net.Conn, error) {
		targets = append(targets, address)
		return nil, &net.DNSError{Err: "expected test dial failure", Name: address}
	})

	result, err := svc.ScanNode(context.Background(), summary.ID)
	if err != nil {
		t.Fatalf("ScanNode() error = %v", err)
	}
	if result.Status != "UNREACHABLE" {
		t.Fatalf("scan result = %#v", result)
	}
	if len(targets) != 2 || targets[0] != "204.42.251.2:9878" || targets[1] != "204.42.251.3:9878" {
		t.Fatalf("scanned targets = %#v", targets)
	}
}

func TestScanNodePersistsReachableResult(t *testing.T) {
	now := time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)
	repo := node.NewMemoryRepository()
	leases := node.NewMemoryLeaseStore()
	svc := node.NewService(repo, leases, func() time.Time { return now })

	summary, err := svc.RegisterLease(context.Background(), node.LeaseInput{
		NodeCode:        "scan-home-001",
		SessionID:       "session-1",
		ScanHost:        "127.0.0.1",
		ProbePort:       18080,
		LeaseTTLSeconds: 45,
	})
	if err != nil {
		t.Fatalf("RegisterLease() error = %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	_, _ = repo.UpsertLease(context.Background(), node.LeaseInput{
		NodeCode:        "scan-home-001",
		SessionID:       "session-1",
		ScanHost:        "127.0.0.1",
		ProbePort:       uint32(port),
		LeaseTTLSeconds: 45,
	}, now)
	go func() {
		conn, acceptErr := ln.Accept()
		if acceptErr == nil {
			_ = conn.Close()
		}
	}()

	result, err := svc.ScanNode(context.Background(), summary.ID)
	if err != nil {
		t.Fatalf("ScanNode() error = %v", err)
	}
	if result.Status != "REACHABLE" || result.Target == "" {
		t.Fatalf("scan result = %#v", result)
	}
	nodes, err := svc.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if nodes[0].LastScanStatus != "REACHABLE" || nodes[0].LastScanAt.IsZero() {
		t.Fatalf("scan result was not retained: %#v", nodes[0])
	}
}

func TestListNodesMarksExpiredLeaseOffline(t *testing.T) {
	now := time.Date(2026, 4, 30, 9, 0, 0, 0, time.UTC)
	repo := node.NewMemoryRepository()
	leases := node.NewMemoryLeaseStore()
	svc := node.NewService(repo, leases, func() time.Time { return now })

	_, err := svc.RegisterLease(context.Background(), node.LeaseInput{
		NodeCode:        "la-home-001",
		SessionID:       "session-1",
		APIInstanceID:   "api-1",
		BundleVersion:   "bundle-0.1.0",
		LeaseTTLSeconds: 30,
	})
	if err != nil {
		t.Fatalf("RegisterLease() error = %v", err)
	}

	later := node.NewService(repo, leases, func() time.Time { return now.Add(2 * time.Minute) })
	nodes, err := later.ListNodes(context.Background())
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if nodes[0].Status != node.StatusOffline {
		t.Fatalf("status = %s, want OFFLINE", nodes[0].Status)
	}
}
