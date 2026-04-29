package runtimelab_test

import (
	"context"
	"testing"
	"time"

	"github.com/rayip/rayip/services/api/internal/runtimelab"
)

func TestCreateAccountDispatchesRuntimeCommandAndStoresResult(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &recordingDispatcher{
		result: runtimelab.ApplyResult{
			CommandID:         "cmd-1",
			Status:            runtimelab.ApplyStatusSuccess,
			AppliedGeneration: 1,
			Usage: runtimelab.Usage{
				RxBytes: 100,
				TxBytes: 200,
			},
			Digest: runtimelab.Digest{
				AccountCount: 1,
				Hash:         "digest-1",
			},
		},
	}
	svc := runtimelab.NewService(repo, dispatcher, func() time.Time {
		return time.Date(2026, 4, 30, 10, 0, 0, 0, time.UTC)
	})

	account, result, err := svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:            "node-1",
		Protocol:          runtimelab.ProtocolSOCKS5,
		ListenIP:          "127.0.0.1",
		Port:              18080,
		Username:          "u1",
		Password:          "p1",
		EgressLimitBPS:    1024,
		IngressLimitBPS:   2048,
		MaxConnections:    2,
		DesiredGeneration: 1,
	})
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}

	if account.RuntimeEmail != account.ProxyAccountID {
		t.Fatalf("runtime email = %q, want proxy account id %q", account.RuntimeEmail, account.ProxyAccountID)
	}
	if result.Status != runtimelab.ApplyStatusSuccess || result.AppliedGeneration != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(dispatcher.commands) != 1 {
		t.Fatalf("dispatch count = %d, want 1", len(dispatcher.commands))
	}
	got := dispatcher.commands[0]
	if got.Operation != runtimelab.OperationUpsert || got.Account.ProxyAccountID != account.ProxyAccountID {
		t.Fatalf("unexpected command: %#v", got)
	}
	if got.Account.EgressLimitBPS != 1024 || got.Account.MaxConnections != 2 {
		t.Fatalf("policy was not copied into command: %#v", got.Account)
	}
}

func TestCreateAccountSkipsDuplicateGenerationWithoutDispatch(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &recordingDispatcher{result: runtimelab.ApplyResult{Status: runtimelab.ApplyStatusSuccess, AppliedGeneration: 7}}
	svc := runtimelab.NewService(repo, dispatcher, time.Now)

	input := runtimelab.CreateAccountInput{
		NodeID:            "node-1",
		Protocol:          runtimelab.ProtocolHTTP,
		ListenIP:          "127.0.0.1",
		Port:              18081,
		Username:          "u1",
		Password:          "p1",
		DesiredGeneration: 7,
	}
	account, _, err := svc.CreateAccount(context.Background(), input)
	if err != nil {
		t.Fatalf("CreateAccount() first error = %v", err)
	}
	if len(dispatcher.commands) != 1 {
		t.Fatalf("first dispatch count = %d, want 1", len(dispatcher.commands))
	}

	_, duplicate, err := svc.UpsertAccountPolicy(context.Background(), account.ProxyAccountID, runtimelab.PolicyInput{
		EgressLimitBPS:    account.EgressLimitBPS,
		IngressLimitBPS:   account.IngressLimitBPS,
		MaxConnections:    account.MaxConnections,
		DesiredGeneration: 7,
	})
	if err != nil {
		t.Fatalf("UpsertAccountPolicy() error = %v", err)
	}
	if duplicate.Status != runtimelab.ApplyStatusDuplicate || duplicate.AppliedGeneration != 7 {
		t.Fatalf("duplicate result = %#v, want DUPLICATE generation 7", duplicate)
	}
	if len(dispatcher.commands) != 1 {
		t.Fatalf("dispatch count after duplicate = %d, want still 1", len(dispatcher.commands))
	}
}

type recordingDispatcher struct {
	commands []runtimelab.RuntimeCommand
	result   runtimelab.ApplyResult
}

func (d *recordingDispatcher) DispatchRuntimeCommand(_ context.Context, cmd runtimelab.RuntimeCommand) (runtimelab.ApplyResult, error) {
	d.commands = append(d.commands, cmd)
	d.result.CommandID = cmd.CommandID
	return d.result, nil
}
