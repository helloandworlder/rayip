package runtime_test

import (
	"context"
	"testing"
	"time"

	"github.com/rayip/rayip/services/node-agent/internal/runtime"
)

func TestMemoryCoreEnforcesConnectionLimitAndReleases(t *testing.T) {
	core := runtime.NewMemoryCore()
	account := runtime.Account{
		ProxyAccountID: "acct-1",
		RuntimeEmail:   "acct-1",
		Status:         runtime.AccountStatusEnabled,
		MaxConnections: 1,
	}
	if err := core.UpsertAccount(context.Background(), account); err != nil {
		t.Fatalf("UpsertAccount() error = %v", err)
	}

	release, err := core.AcquireConnection("acct-1")
	if err != nil {
		t.Fatalf("AcquireConnection() first error = %v", err)
	}
	if _, err := core.AcquireConnection("acct-1"); err == nil {
		t.Fatal("AcquireConnection() second error = nil, want limit error")
	}
	release()
	if _, err := core.AcquireConnection("acct-1"); err != nil {
		t.Fatalf("AcquireConnection() after release error = %v", err)
	}
}

func TestMemoryCoreFixedRateLimitAndFairShare(t *testing.T) {
	core := runtime.NewMemoryCore()
	core.SetFairPoolBPS(300)
	now := time.Unix(100, 0)
	_ = core.UpsertAccount(context.Background(), runtime.Account{
		ProxyAccountID: "low",
		RuntimeEmail:   "low",
		Status:         runtime.AccountStatusEnabled,
		Priority:       1,
	})
	_ = core.UpsertAccount(context.Background(), runtime.Account{
		ProxyAccountID:  "limited",
		RuntimeEmail:    "limited",
		Status:          runtime.AccountStatusEnabled,
		Priority:        3,
		EgressLimitBPS:  100,
		IngressLimitBPS: 200,
	})

	if allowed := core.AllowBytesAt("limited", runtime.DirectionEgress, 80, now); allowed != 80 {
		t.Fatalf("first fixed allowance = %d, want 80", allowed)
	}
	if allowed := core.AllowBytesAt("limited", runtime.DirectionEgress, 80, now); allowed != 20 {
		t.Fatalf("second fixed allowance = %d, want 20", allowed)
	}
	if share := core.FairShareBPS("low", now); share != 75 {
		t.Fatalf("low fair share = %d, want 75", share)
	}
}

func TestMemoryCoreAbuseDetectionDisablesAccount(t *testing.T) {
	core := runtime.NewMemoryCore()
	now := time.Unix(100, 0)
	_ = core.UpsertAccount(context.Background(), runtime.Account{
		ProxyAccountID:    "acct-1",
		RuntimeEmail:      "acct-1",
		Status:            runtime.AccountStatusEnabled,
		AbuseBytesPerMin:  1000,
		AbuseAction:       runtime.AbuseActionDisableAndReport,
		DesiredGeneration: 3,
	})

	event := core.RecordTrafficAt("acct-1", runtime.DirectionEgress, 1500, now)
	if event == nil || event.Action != runtime.AbuseActionDisableAndReport {
		t.Fatalf("abuse event = %#v, want disable and report", event)
	}
	account, ok := core.Account("acct-1")
	if !ok || account.Status != runtime.AccountStatusDisabled {
		t.Fatalf("account status = %#v ok=%v, want disabled", account, ok)
	}
}
