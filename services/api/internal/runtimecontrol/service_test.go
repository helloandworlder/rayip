package runtimecontrol_test

import (
	"context"
	"testing"
	"time"

	"github.com/rayip/rayip/services/api/internal/runtimecontrol"
)

func TestUpsertProxyAccountCreatesDesiredStateChangeAndOutbox(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	repo := runtimecontrol.NewMemoryRepository()
	svc := runtimecontrol.NewService(repo, func() time.Time { return now })

	result, err := svc.UpsertProxyAccount(context.Background(), runtimecontrol.ResourceInput{
		ProxyAccountID:  "acct-1",
		NodeID:          "node-1",
		RuntimeEmail:    "acct-1",
		Protocol:        runtimecontrol.ProtocolSOCKS5,
		ListenIP:        "127.0.0.1",
		Port:            18080,
		Username:        "u1",
		Password:        "p1",
		EgressLimitBPS:  1024,
		IngressLimitBPS: 2048,
		MaxConnections:  2,
	})
	if err != nil {
		t.Fatalf("UpsertProxyAccount() error = %v", err)
	}

	if result.State.ResourceName != "proxy/acct-1" || result.State.DesiredRevision != 1 || result.State.Removed {
		t.Fatalf("state = %#v", result.State)
	}
	if result.Change.NodeID != "node-1" || result.Change.Seq != 1 || result.Change.Action != runtimecontrol.ChangeActionUpsert {
		t.Fatalf("change = %#v", result.Change)
	}
	if result.Outbox.Topic != "rayip.runtime.apply.v1" || result.Outbox.AggregateID != result.Change.ID {
		t.Fatalf("outbox = %#v", result.Outbox)
	}
}

func TestRepeatedUpsertIncrementsResourceRevisionAndNodeSeq(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	repo := runtimecontrol.NewMemoryRepository()
	svc := runtimecontrol.NewService(repo, func() time.Time { return now })
	input := runtimecontrol.ResourceInput{
		ProxyAccountID: "acct-1",
		NodeID:         "node-1",
		RuntimeEmail:   "acct-1",
		Protocol:       runtimecontrol.ProtocolHTTP,
		Port:           18081,
		Username:       "u1",
		Password:       "p1",
	}
	if _, err := svc.UpsertProxyAccount(context.Background(), input); err != nil {
		t.Fatalf("first upsert error = %v", err)
	}
	input.EgressLimitBPS = 4096
	second, err := svc.UpsertProxyAccount(context.Background(), input)
	if err != nil {
		t.Fatalf("second upsert error = %v", err)
	}
	if second.State.DesiredRevision != 2 || second.Change.Seq != 2 {
		t.Fatalf("second result = %#v", second)
	}
	changes, err := svc.ListChanges(context.Background(), "node-1", 0, 10)
	if err != nil {
		t.Fatalf("ListChanges() error = %v", err)
	}
	if len(changes) != 2 || changes[0].Seq != 1 || changes[1].Seq != 2 {
		t.Fatalf("changes = %#v", changes)
	}
}

func TestRemoveProxyAccountMarksRemovedAndWritesRemoveChange(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	repo := runtimecontrol.NewMemoryRepository()
	svc := runtimecontrol.NewService(repo, func() time.Time { return now })
	if _, err := svc.UpsertProxyAccount(context.Background(), runtimecontrol.ResourceInput{
		ProxyAccountID: "acct-1",
		NodeID:         "node-1",
		RuntimeEmail:   "acct-1",
		Protocol:       runtimecontrol.ProtocolSOCKS5,
		Port:           18080,
		Username:       "u1",
		Password:       "p1",
	}); err != nil {
		t.Fatalf("upsert error = %v", err)
	}

	result, err := svc.RemoveProxyAccount(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("RemoveProxyAccount() error = %v", err)
	}
	if !result.State.Removed || result.State.DesiredRevision != 2 {
		t.Fatalf("removed state = %#v", result.State)
	}
	if result.Change.Action != runtimecontrol.ChangeActionRemove || result.Change.Seq != 2 {
		t.Fatalf("remove change = %#v", result.Change)
	}
	outbox, err := svc.ListOutbox(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListOutbox() error = %v", err)
	}
	if len(outbox) != 2 || outbox[1].Payload["action"] != string(runtimecontrol.ChangeActionRemove) {
		t.Fatalf("outbox = %#v", outbox)
	}
}
