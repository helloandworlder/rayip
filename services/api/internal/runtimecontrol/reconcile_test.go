package runtimecontrol_test

import (
	"context"
	"testing"
	"time"

	"github.com/rayip/rayip/services/api/internal/runtimecontrol"
	"github.com/rayip/rayip/services/api/internal/runtimelab"
)

func TestBuildSnapshotApplyPaginatesDesiredResources(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	repo := runtimecontrol.NewMemoryRepository()
	svc := runtimecontrol.NewService(repo, func() time.Time { return now })
	for i := 0; i < 3; i++ {
		id := "acct-" + string(rune('a'+i))
		if _, err := svc.UpsertProxyAccount(context.Background(), runtimecontrol.ResourceInput{
			ProxyAccountID: id,
			NodeID:         "node-1",
			RuntimeEmail:   id,
			Protocol:       runtimecontrol.ProtocolSOCKS5,
			Port:           uint32(18080 + i),
			Username:       "u",
			Password:       "p",
		}); err != nil {
			t.Fatalf("upsert %s error = %v", id, err)
		}
	}

	planner := runtimecontrol.NewReconcilePlanner(svc, func() time.Time { return now })
	first, err := planner.BuildSnapshotApply(context.Background(), "node-1", 0, 2, 0, 3)
	if err != nil {
		t.Fatalf("BuildSnapshotApply first error = %v", err)
	}
	if first.Mode != runtimelab.ApplyModeSnapshot || first.BaseRevision != 0 || first.TargetRevision != 3 {
		t.Fatalf("first apply metadata = %#v", first)
	}
	if len(first.Resources) != 2 {
		t.Fatalf("first resources = %#v", first.Resources)
	}

	second, err := planner.BuildSnapshotApply(context.Background(), "node-1", 2, 2, 0, 3)
	if err != nil {
		t.Fatalf("BuildSnapshotApply second error = %v", err)
	}
	if len(second.Resources) != 1 || second.Resources[0].Name != "proxy/acct-c" {
		t.Fatalf("second resources = %#v", second.Resources)
	}
}
