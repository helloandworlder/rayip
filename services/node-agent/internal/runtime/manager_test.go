package runtime_test

import (
	"context"
	"testing"

	"github.com/rayip/rayip/services/node-agent/internal/runtime"
)

func TestManagerReturnsDuplicateForSameGeneration(t *testing.T) {
	manager := runtime.NewManager(runtime.NewMemoryCore())
	account := runtime.Account{
		ProxyAccountID:    "acct-1",
		RuntimeEmail:      "acct-1",
		Protocol:          runtime.ProtocolSOCKS5,
		ListenIP:          "127.0.0.1",
		Port:              18080,
		Username:          "u1",
		Password:          "p1",
		EgressLimitBPS:    1024,
		IngressLimitBPS:   2048,
		MaxConnections:    2,
		DesiredGeneration: 3,
		Status:            runtime.AccountStatusEnabled,
	}

	first, err := manager.Apply(context.Background(), runtime.Command{
		CommandID:         "cmd-1",
		Operation:         runtime.OperationUpsert,
		Account:           account,
		DesiredGeneration: 3,
	})
	if err != nil {
		t.Fatalf("Apply() first error = %v", err)
	}
	if first.Status != runtime.ResultSuccess {
		t.Fatalf("first status = %s, want SUCCESS", first.Status)
	}

	second, err := manager.Apply(context.Background(), runtime.Command{
		CommandID:         "cmd-2",
		Operation:         runtime.OperationUpsert,
		Account:           account,
		DesiredGeneration: 3,
	})
	if err != nil {
		t.Fatalf("Apply() second error = %v", err)
	}
	if second.Status != runtime.ResultDuplicate || second.AppliedGeneration != 3 {
		t.Fatalf("duplicate result = %#v", second)
	}
}

func TestManagerUpdatesPolicyForNewGenerationAndDigest(t *testing.T) {
	core := runtime.NewMemoryCore()
	manager := runtime.NewManager(core)
	account := runtime.Account{
		ProxyAccountID:    "acct-1",
		RuntimeEmail:      "acct-1",
		Protocol:          runtime.ProtocolHTTP,
		ListenIP:          "127.0.0.1",
		Port:              18081,
		Username:          "u1",
		Password:          "p1",
		EgressLimitBPS:    1024,
		IngressLimitBPS:   2048,
		MaxConnections:    2,
		DesiredGeneration: 1,
		Status:            runtime.AccountStatusEnabled,
	}
	if _, err := manager.Apply(context.Background(), runtime.Command{
		CommandID:         "cmd-1",
		Operation:         runtime.OperationUpsert,
		Account:           account,
		DesiredGeneration: 1,
	}); err != nil {
		t.Fatalf("Apply() generation 1 error = %v", err)
	}

	account.EgressLimitBPS = 4096
	account.MaxConnections = 4
	account.DesiredGeneration = 2
	result, err := manager.Apply(context.Background(), runtime.Command{
		CommandID:         "cmd-2",
		Operation:         runtime.OperationUpdatePolicy,
		Account:           account,
		DesiredGeneration: 2,
	})
	if err != nil {
		t.Fatalf("Apply() generation 2 error = %v", err)
	}
	if result.Status != runtime.ResultSuccess || result.AppliedGeneration != 2 {
		t.Fatalf("update result = %#v", result)
	}
	if result.Digest.AccountCount != 1 || result.Digest.MaxGeneration != 2 || result.Digest.Hash == "" {
		t.Fatalf("digest not updated: %#v", result.Digest)
	}

	stored, ok := core.Account("acct-1")
	if !ok {
		t.Fatal("account missing from core")
	}
	if stored.EgressLimitBPS != 4096 || stored.MaxConnections != 4 {
		t.Fatalf("policy not updated: %#v", stored)
	}
}

func TestManagerDisableAndDelete(t *testing.T) {
	core := runtime.NewMemoryCore()
	manager := runtime.NewManager(core)
	account := runtime.Account{
		ProxyAccountID:    "acct-1",
		RuntimeEmail:      "acct-1",
		Protocol:          runtime.ProtocolSOCKS5,
		ListenIP:          "127.0.0.1",
		Port:              18080,
		Username:          "u1",
		Password:          "p1",
		DesiredGeneration: 1,
		Status:            runtime.AccountStatusEnabled,
	}
	if _, err := manager.Apply(context.Background(), runtime.Command{CommandID: "cmd-1", Operation: runtime.OperationUpsert, Account: account, DesiredGeneration: 1}); err != nil {
		t.Fatalf("upsert error = %v", err)
	}

	account.DesiredGeneration = 2
	disabled, err := manager.Apply(context.Background(), runtime.Command{CommandID: "cmd-2", Operation: runtime.OperationDisable, Account: account, DesiredGeneration: 2})
	if err != nil {
		t.Fatalf("disable error = %v", err)
	}
	if disabled.Status != runtime.ResultSuccess {
		t.Fatalf("disable status = %s", disabled.Status)
	}
	stored, _ := core.Account("acct-1")
	if stored.Status != runtime.AccountStatusDisabled {
		t.Fatalf("stored status = %s, want DISABLED", stored.Status)
	}

	deleted, err := manager.Apply(context.Background(), runtime.Command{CommandID: "cmd-3", Operation: runtime.OperationDelete, Account: account, DesiredGeneration: 3})
	if err != nil {
		t.Fatalf("delete error = %v", err)
	}
	if deleted.Digest.AccountCount != 0 {
		t.Fatalf("account count after delete = %d, want 0", deleted.Digest.AccountCount)
	}
}
