package node_test

import (
	"context"
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
