package runtime_test

import (
	"context"
	"testing"

	"github.com/rayip/rayip/services/node-agent/internal/runtime"
)

func testResource(name string, revision uint64) runtime.Resource {
	return runtime.Resource{
		Name:              name,
		Kind:              runtime.ResourceKindProxyAccount,
		ResourceVersion:   revision,
		RuntimeEmail:      name[len("proxy/"):],
		Protocol:          runtime.ProtocolSOCKS5,
		ListenIP:          "127.0.0.1",
		Port:              18080,
		Username:          "u1",
		Password:          "p1",
		EgressLimitBPS:    1024,
		IngressLimitBPS:   2048,
		MaxConnections:    2,
		Priority:          1,
		AbuseReportPolicy: "REPORT_ONLY",
	}
}

func TestManagerAppliesDeltaAndTracksRevisionNonce(t *testing.T) {
	core := runtime.NewMemoryCore()
	manager := runtime.NewManager(core)

	ack, err := manager.Apply(context.Background(), runtime.Apply{
		ApplyID:        "apply-1",
		Mode:           runtime.ApplyModeDelta,
		VersionInfo:    "rv-1",
		Nonce:          "nonce-1",
		BaseRevision:   0,
		TargetRevision: 1,
		Resources:      []runtime.Resource{testResource("proxy/acct-1", 1)},
	})
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if ack.Status != runtime.AckStatusACK || ack.AppliedRevision != 1 || ack.LastGoodRevision != 1 {
		t.Fatalf("ack = %#v", ack)
	}
	if ack.VersionInfo != "rv-1" || ack.Nonce != "nonce-1" {
		t.Fatalf("version/nonce not echoed: %#v", ack)
	}
	if _, ok := core.Account("acct-1"); !ok {
		t.Fatal("resource was not upserted into core")
	}
}

func TestManagerDeltaBaseRevisionMismatchReturnsNACK(t *testing.T) {
	manager := runtime.NewManager(runtime.NewMemoryCore())

	ack, err := manager.Apply(context.Background(), runtime.Apply{
		ApplyID:        "apply-2",
		Mode:           runtime.ApplyModeDelta,
		VersionInfo:    "rv-2",
		Nonce:          "nonce-2",
		BaseRevision:   7,
		TargetRevision: 8,
		Resources:      []runtime.Resource{testResource("proxy/acct-1", 8)},
	})
	if err == nil {
		t.Fatal("Apply() error = nil, want base revision mismatch")
	}
	if ack.Status != runtime.AckStatusNACK || ack.AppliedRevision != 0 || ack.LastGoodRevision != 0 {
		t.Fatalf("ack = %#v", ack)
	}
}

func TestManagerDuplicateVersionNonceIsIdempotent(t *testing.T) {
	manager := runtime.NewManager(runtime.NewMemoryCore())
	apply := runtime.Apply{
		ApplyID:        "apply-1",
		Mode:           runtime.ApplyModeDelta,
		VersionInfo:    "rv-1",
		Nonce:          "nonce-1",
		BaseRevision:   0,
		TargetRevision: 1,
		Resources:      []runtime.Resource{testResource("proxy/acct-1", 1)},
	}
	if _, err := manager.Apply(context.Background(), apply); err != nil {
		t.Fatalf("first Apply() error = %v", err)
	}

	ack, err := manager.Apply(context.Background(), apply)
	if err != nil {
		t.Fatalf("duplicate Apply() error = %v", err)
	}
	if ack.Status != runtime.AckStatusACK || ack.AppliedRevision != 1 || ack.LastGoodRevision != 1 {
		t.Fatalf("duplicate ack = %#v", ack)
	}
}

func TestManagerGetUsageQueryDoesNotAdvanceRevision(t *testing.T) {
	core := runtime.NewMemoryCore()
	manager := runtime.NewManager(core)
	if _, err := manager.Apply(context.Background(), runtime.Apply{
		ApplyID:        "apply-1",
		Mode:           runtime.ApplyModeDelta,
		VersionInfo:    "rv-1",
		Nonce:          "nonce-1",
		BaseRevision:   0,
		TargetRevision: 1,
		Resources:      []runtime.Resource{testResource("proxy/acct-1", 1)},
	}); err != nil {
		t.Fatalf("upsert Apply() error = %v", err)
	}

	ack, err := manager.Apply(context.Background(), runtime.Apply{
		ApplyID:           "query-1",
		NodeID:            "node-1",
		QueryOperation:    "GET_USAGE",
		QueryResourceName: "proxy/acct-1",
	})
	if err != nil {
		t.Fatalf("query Apply() error = %v", err)
	}
	if ack.Status != runtime.AckStatusACK || ack.AppliedRevision != 1 || ack.LastGoodRevision != 1 {
		t.Fatalf("query ack revisions = %#v, want revision unchanged at 1", ack)
	}
	if ack.Usage.ProxyAccountID != "acct-1" || ack.Usage.RuntimeEmail == "" {
		t.Fatalf("query usage = %#v, want account usage", ack.Usage)
	}
}

func TestManagerFairnessApplyDoesNotAdvanceRevision(t *testing.T) {
	core := runtime.NewMemoryCore()
	manager := runtime.NewManager(core)
	if _, err := manager.Apply(context.Background(), runtime.Apply{
		ApplyID:        "apply-1",
		Mode:           runtime.ApplyModeDelta,
		VersionInfo:    "rv-1",
		Nonce:          "nonce-1",
		BaseRevision:   0,
		TargetRevision: 1,
		Resources:      []runtime.Resource{testResource("proxy/acct-1", 1)},
	}); err != nil {
		t.Fatalf("upsert Apply() error = %v", err)
	}

	ack, err := manager.Apply(context.Background(), runtime.Apply{
		ApplyID: "fair-1",
		NodeID:  "node-1",
		FairnessState: runtime.FairnessState{
			EgressPoolBPS:       300,
			IngressPoolBPS:      300,
			WindowSeconds:       300,
			LossRatePPM:         20000,
			TargetLossPPM:       5000,
			TargetRetransmitPPM: 10000,
			MinCongestionBPS:    100,
		},
	})
	if err != nil {
		t.Fatalf("fairness Apply() error = %v", err)
	}
	if ack.Status != runtime.AckStatusACK || ack.AppliedRevision != 1 || ack.LastGoodRevision != 1 {
		t.Fatalf("fairness ack revisions = %#v, want revision unchanged at 1", ack)
	}
}

func TestManagerRemovedResourceDeletesAccount(t *testing.T) {
	core := runtime.NewMemoryCore()
	manager := runtime.NewManager(core)
	if _, err := manager.Apply(context.Background(), runtime.Apply{
		ApplyID:        "apply-1",
		Mode:           runtime.ApplyModeDelta,
		VersionInfo:    "rv-1",
		Nonce:          "nonce-1",
		BaseRevision:   0,
		TargetRevision: 1,
		Resources:      []runtime.Resource{testResource("proxy/acct-1", 1)},
	}); err != nil {
		t.Fatalf("upsert error = %v", err)
	}

	ack, err := manager.Apply(context.Background(), runtime.Apply{
		ApplyID:              "apply-2",
		Mode:                 runtime.ApplyModeDelta,
		VersionInfo:          "rv-2",
		Nonce:                "nonce-2",
		BaseRevision:         1,
		TargetRevision:       2,
		RemovedResourceNames: []string{"proxy/acct-1"},
	})
	if err != nil {
		t.Fatalf("remove error = %v", err)
	}
	if ack.Status != runtime.AckStatusACK || ack.AppliedRevision != 2 {
		t.Fatalf("remove ack = %#v", ack)
	}
	if _, ok := core.Account("acct-1"); ok {
		t.Fatal("removed resource still exists")
	}
}

func TestManagerSnapshotIsAcceptedWithoutPriorRevision(t *testing.T) {
	core := runtime.NewMemoryCore()
	manager := runtime.NewManager(core)

	ack, err := manager.Apply(context.Background(), runtime.Apply{
		ApplyID:        "snapshot-1",
		Mode:           runtime.ApplyModeSnapshot,
		VersionInfo:    "rv-10",
		Nonce:          "nonce-10",
		BaseRevision:   0,
		TargetRevision: 10,
		Resources: []runtime.Resource{
			testResource("proxy/acct-1", 10),
			testResource("proxy/acct-2", 10),
		},
	})
	if err != nil {
		t.Fatalf("snapshot error = %v", err)
	}
	if ack.Status != runtime.AckStatusACK || ack.AppliedRevision != 10 || ack.Digest.AccountCount != 2 {
		t.Fatalf("snapshot ack = %#v", ack)
	}
}
