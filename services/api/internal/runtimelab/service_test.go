package runtimelab_test

import (
	"context"
	"testing"
	"time"

	"github.com/rayip/rayip/services/api/internal/runtimelab"
)

func TestCreateAccountDispatchesRuntimeDeltaAndStoresAck(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &recordingDispatcher{
		ack: runtimelab.ApplyResult{
			ApplyID:          "apply-1",
			Status:           runtimelab.ApplyStatusACK,
			AppliedRevision:  1,
			LastGoodRevision: 1,
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
	if result.Status != runtimelab.ApplyStatusACK || result.AppliedRevision != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(dispatcher.applies) != 1 {
		t.Fatalf("dispatch count = %d, want 1", len(dispatcher.applies))
	}
	got := dispatcher.applies[0]
	if got.Mode != runtimelab.ApplyModeDelta || got.BaseRevision != 0 || got.TargetRevision != 1 {
		t.Fatalf("unexpected apply metadata: %#v", got)
	}
	if len(got.Resources) != 1 || got.Resources[0].Name != "proxy/"+account.RuntimeEmail {
		t.Fatalf("unexpected resources: %#v", got.Resources)
	}
	if got.Resources[0].EgressLimitBPS != 1024 || got.Resources[0].MaxConnections != 2 {
		t.Fatalf("policy was not copied into resource: %#v", got.Resources[0])
	}
}

func TestDisableAccountDispatchesRemovedResourceNotBusinessDisable(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &recordingDispatcher{ack: runtimelab.ApplyResult{Status: runtimelab.ApplyStatusACK, AppliedRevision: 1, LastGoodRevision: 1}}
	svc := runtimelab.NewService(repo, dispatcher, time.Now)
	account, _, err := svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:   "node-1",
		Protocol: runtimelab.ProtocolHTTP,
		ListenIP: "127.0.0.1",
		Port:     18081,
		Username: "u1",
		Password: "p1",
	})
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}
	dispatcher.ack = runtimelab.ApplyResult{Status: runtimelab.ApplyStatusACK, AppliedRevision: 2, LastGoodRevision: 2}

	_, result, err := svc.DisableAccount(context.Background(), account.ProxyAccountID)
	if err != nil {
		t.Fatalf("DisableAccount() error = %v", err)
	}
	if result.Status != runtimelab.ApplyStatusACK || result.AppliedRevision != 2 {
		t.Fatalf("disable result = %#v", result)
	}
	if len(dispatcher.applies) != 2 {
		t.Fatalf("dispatch count = %d, want 2", len(dispatcher.applies))
	}
	got := dispatcher.applies[1]
	if len(got.Resources) != 0 {
		t.Fatalf("disable apply should not send resources: %#v", got.Resources)
	}
	if len(got.RemovedResourceNames) != 1 || got.RemovedResourceNames[0] != "proxy/"+account.RuntimeEmail {
		t.Fatalf("removed resources = %#v", got.RemovedResourceNames)
	}
}

func TestCreateAccountSkipsDuplicateRevisionWithoutDispatch(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &recordingDispatcher{ack: runtimelab.ApplyResult{Status: runtimelab.ApplyStatusACK, AppliedRevision: 7, LastGoodRevision: 7}}
	svc := runtimelab.NewService(repo, dispatcher, time.Now)

	account, _, err := svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:            "node-1",
		Protocol:          runtimelab.ProtocolHTTP,
		ListenIP:          "127.0.0.1",
		Port:              18081,
		Username:          "u1",
		Password:          "p1",
		DesiredGeneration: 7,
	})
	if err != nil {
		t.Fatalf("CreateAccount() first error = %v", err)
	}
	if len(dispatcher.applies) != 1 {
		t.Fatalf("first dispatch count = %d, want 1", len(dispatcher.applies))
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
	if duplicate.Status != runtimelab.ApplyStatusDuplicate || duplicate.AppliedRevision != 7 {
		t.Fatalf("duplicate result = %#v, want DUPLICATE revision 7", duplicate)
	}
	if len(dispatcher.applies) != 1 {
		t.Fatalf("dispatch count after duplicate = %d, want still 1", len(dispatcher.applies))
	}
}

type recordingDispatcher struct {
	applies []runtimelab.RuntimeApply
	ack     runtimelab.ApplyResult
}

func (d *recordingDispatcher) DispatchRuntimeApply(_ context.Context, apply runtimelab.RuntimeApply) (runtimelab.ApplyResult, error) {
	d.applies = append(d.applies, apply)
	d.ack.ApplyID = apply.ApplyID
	d.ack.NodeID = apply.NodeID
	d.ack.VersionInfo = apply.VersionInfo
	d.ack.Nonce = apply.Nonce
	return d.ack, nil
}
