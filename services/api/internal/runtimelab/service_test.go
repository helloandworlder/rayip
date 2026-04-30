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

func TestCreateAccountUsesLatestNodeRevisionForSecondResource(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &recordingDispatcher{ack: runtimelab.ApplyResult{Status: runtimelab.ApplyStatusACK, AppliedRevision: 1, LastGoodRevision: 1}}
	svc := runtimelab.NewService(repo, dispatcher, time.Now)

	_, _, err := svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:   "node-1",
		Protocol: runtimelab.ProtocolSOCKS5,
		ListenIP: "0.0.0.0",
		Port:     18080,
		Username: "u1",
		Password: "p1",
	})
	if err != nil {
		t.Fatalf("CreateAccount() first error = %v", err)
	}
	dispatcher.ack = runtimelab.ApplyResult{Status: runtimelab.ApplyStatusACK, AppliedRevision: 2, LastGoodRevision: 2}
	_, _, err = svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:   "node-1",
		Protocol: runtimelab.ProtocolHTTP,
		ListenIP: "0.0.0.0",
		Port:     18081,
		Username: "u2",
		Password: "p2",
	})
	if err != nil {
		t.Fatalf("CreateAccount() second error = %v", err)
	}
	if len(dispatcher.applies) != 2 {
		t.Fatalf("dispatch count = %d, want 2", len(dispatcher.applies))
	}
	second := dispatcher.applies[1]
	if second.BaseRevision != 1 || second.TargetRevision != 2 {
		t.Fatalf("second apply revisions = base %d target %d, want 1 -> 2", second.BaseRevision, second.TargetRevision)
	}
}

func TestCreateAccountUsesLatestNACKRevisionAfterNodeRestart(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &recordingDispatcher{ack: runtimelab.ApplyResult{Status: runtimelab.ApplyStatusACK, AppliedRevision: 1, LastGoodRevision: 1}}
	svc := runtimelab.NewService(repo, dispatcher, time.Now)

	_, _, err := svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:   "node-1",
		Protocol: runtimelab.ProtocolSOCKS5,
		ListenIP: "0.0.0.0",
		Port:     18080,
		Username: "u1",
		Password: "p1",
	})
	if err != nil {
		t.Fatalf("CreateAccount() first error = %v", err)
	}
	if err := svc.SaveApplyResult(context.Background(), runtimelab.ApplyResult{
		ApplyID:          "restart-nack",
		NodeID:           "node-1",
		Operation:        runtimelab.OperationUpsert,
		Status:           runtimelab.ApplyStatusNACK,
		AppliedRevision:  0,
		LastGoodRevision: 0,
		CreatedAt:        time.Now().Add(time.Second),
	}); err != nil {
		t.Fatalf("SaveApplyResult() error = %v", err)
	}
	dispatcher.ack = runtimelab.ApplyResult{Status: runtimelab.ApplyStatusACK, AppliedRevision: 1, LastGoodRevision: 1}
	_, _, err = svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:   "node-1",
		Protocol: runtimelab.ProtocolHTTP,
		ListenIP: "0.0.0.0",
		Port:     18081,
		Username: "u2",
		Password: "p2",
	})
	if err != nil {
		t.Fatalf("CreateAccount() second error = %v", err)
	}
	second := dispatcher.applies[1]
	if second.BaseRevision != 0 || second.TargetRevision != 1 {
		t.Fatalf("second apply revisions after restart = base %d target %d, want 0 -> 1", second.BaseRevision, second.TargetRevision)
	}
}

func TestCreateAccountRetriesBaseRevisionMismatch(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &sequenceDispatcher{acks: []runtimelab.ApplyResult{
		{
			Status:           runtimelab.ApplyStatusNACK,
			AppliedRevision:  0,
			LastGoodRevision: 0,
			ErrorDetail:      "base revision does not match last good revision",
		},
		{
			Status:           runtimelab.ApplyStatusACK,
			AppliedRevision:  1,
			LastGoodRevision: 1,
		},
	}, returnNACKError: true}
	svc := runtimelab.NewService(repo, dispatcher, time.Now)
	if err := svc.SaveApplyResult(context.Background(), runtimelab.ApplyResult{
		ApplyID:          "old-ack",
		NodeID:           "node-1",
		Operation:        runtimelab.OperationUpsert,
		Status:           runtimelab.ApplyStatusACK,
		AppliedRevision:  1,
		LastGoodRevision: 1,
		CreatedAt:        time.Now(),
	}); err != nil {
		t.Fatalf("SaveApplyResult() error = %v", err)
	}

	_, result, err := svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:   "node-1",
		Protocol: runtimelab.ProtocolHTTP,
		ListenIP: "0.0.0.0",
		Port:     18081,
		Username: "u1",
		Password: "p1",
	})
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}
	if result.Status != runtimelab.ApplyStatusACK || len(dispatcher.applies) != 2 {
		t.Fatalf("result = %#v dispatch count = %d, want ACK after retry with 2 dispatches", result, len(dispatcher.applies))
	}
	if dispatcher.applies[0].BaseRevision != 1 || dispatcher.applies[1].BaseRevision != 0 || dispatcher.applies[1].TargetRevision != 1 {
		t.Fatalf("retry revisions = first %d second %d -> %d", dispatcher.applies[0].BaseRevision, dispatcher.applies[1].BaseRevision, dispatcher.applies[1].TargetRevision)
	}
}

func TestCreateAccountRetriesBaseRevisionMismatchWhenDispatcherReturnsNACKWithoutError(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &sequenceDispatcher{acks: []runtimelab.ApplyResult{
		{
			Status:           runtimelab.ApplyStatusNACK,
			AppliedRevision:  0,
			LastGoodRevision: 0,
			ErrorDetail:      "base revision does not match last good revision",
		},
		{
			Status:           runtimelab.ApplyStatusACK,
			AppliedRevision:  1,
			LastGoodRevision: 1,
		},
	}}
	svc := runtimelab.NewService(repo, dispatcher, time.Now)
	if err := svc.SaveApplyResult(context.Background(), runtimelab.ApplyResult{
		ApplyID:          "old-ack",
		NodeID:           "node-1",
		Operation:        runtimelab.OperationUpsert,
		Status:           runtimelab.ApplyStatusACK,
		AppliedRevision:  1,
		LastGoodRevision: 1,
		CreatedAt:        time.Now(),
	}); err != nil {
		t.Fatalf("SaveApplyResult() error = %v", err)
	}

	_, result, err := svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:   "node-1",
		Protocol: runtimelab.ProtocolMixed,
		ListenIP: "0.0.0.0",
		Port:     18080,
		Username: "u1",
		Password: "p1",
	})
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}
	if result.Status != runtimelab.ApplyStatusACK || len(dispatcher.applies) != 2 {
		t.Fatalf("result = %#v dispatch count = %d, want ACK after retry with 2 dispatches", result, len(dispatcher.applies))
	}
	if dispatcher.applies[1].BaseRevision != 0 || dispatcher.applies[1].TargetRevision != 1 {
		t.Fatalf("retry revisions = second %d -> %d, want 0 -> 1", dispatcher.applies[1].BaseRevision, dispatcher.applies[1].TargetRevision)
	}
}

func TestCreateAccountIgnoresQueryResultsWhenComputingNodeRevision(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &recordingDispatcher{ack: runtimelab.ApplyResult{Status: runtimelab.ApplyStatusACK, AppliedRevision: 1, LastGoodRevision: 1}}
	svc := runtimelab.NewService(repo, dispatcher, time.Now)

	_, _, err := svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:   "node-1",
		Protocol: runtimelab.ProtocolHTTP,
		ListenIP: "0.0.0.0",
		Port:     18081,
		Username: "u1",
		Password: "p1",
	})
	if err != nil {
		t.Fatalf("CreateAccount() first error = %v", err)
	}
	if err := svc.SaveApplyResult(context.Background(), runtimelab.ApplyResult{
		ApplyID:          "digest-1",
		NodeID:           "node-1",
		Operation:        runtimelab.OperationGetDigest,
		Status:           runtimelab.ApplyStatusACK,
		AppliedRevision:  0,
		LastGoodRevision: 0,
		CreatedAt:        time.Now().Add(time.Second),
	}); err != nil {
		t.Fatalf("SaveApplyResult() error = %v", err)
	}
	dispatcher.ack = runtimelab.ApplyResult{Status: runtimelab.ApplyStatusACK, AppliedRevision: 2, LastGoodRevision: 2}
	_, _, err = svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:   "node-1",
		Protocol: runtimelab.ProtocolSOCKS5,
		ListenIP: "0.0.0.0",
		Port:     18080,
		Username: "u2",
		Password: "p2",
	})
	if err != nil {
		t.Fatalf("CreateAccount() second error = %v", err)
	}
	second := dispatcher.applies[1]
	if second.BaseRevision != 1 || second.TargetRevision != 2 {
		t.Fatalf("second apply revisions after query result = base %d target %d, want 1 -> 2", second.BaseRevision, second.TargetRevision)
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

func TestGetUsageDispatchesRuntimeQueryAndStoresUsage(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &recordingDispatcher{ack: runtimelab.ApplyResult{
		Status:           runtimelab.ApplyStatusACK,
		AppliedRevision:  1,
		LastGoodRevision: 1,
		Usage: runtimelab.Usage{
			RuntimeEmail:      "email-1",
			RxBytes:           120,
			TxBytes:           340,
			ActiveConnections: 2,
		},
	}}
	svc := runtimelab.NewService(repo, dispatcher, time.Now)
	account, _, err := svc.CreateAccount(context.Background(), runtimelab.CreateAccountInput{
		NodeID:   "node-1",
		Protocol: runtimelab.ProtocolSOCKS5,
		ListenIP: "127.0.0.1",
		Port:     18080,
		Username: "u1",
		Password: "p1",
	})
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}

	result, err := svc.GetUsage(context.Background(), account.ProxyAccountID)
	if err != nil {
		t.Fatalf("GetUsage() error = %v", err)
	}
	if result.Status != runtimelab.ApplyStatusACK || result.Usage.TxBytes != 340 || result.Usage.ActiveConnections != 2 {
		t.Fatalf("usage result = %#v", result)
	}
	if result.Usage.ProxyAccountID != account.ProxyAccountID {
		t.Fatalf("usage proxy account = %q, want %q", result.Usage.ProxyAccountID, account.ProxyAccountID)
	}
	query := dispatcher.applies[len(dispatcher.applies)-1]
	if query.QueryOperation != runtimelab.OperationGetUsage || query.QueryResourceName != "proxy/"+account.RuntimeEmail {
		t.Fatalf("query apply = %#v, want GET_USAGE for resource", query)
	}
}

func TestSetFairnessStateDispatchesRuntimeControl(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	dispatcher := &recordingDispatcher{ack: runtimelab.ApplyResult{
		Status:          runtimelab.ApplyStatusACK,
		AppliedRevision: 1,
		Digest:          runtimelab.Digest{AccountCount: 2, Hash: "digest-1"},
	}}
	svc := runtimelab.NewService(repo, dispatcher, time.Now)

	result, err := svc.SetFairnessState(context.Background(), "node-1", runtimelab.FairnessState{
		EgressPoolBPS:       300000,
		IngressPoolBPS:      300000,
		WindowSeconds:       300,
		LossRatePPM:         20000,
		RetransmitRatePPM:   15000,
		TargetLossPPM:       5000,
		TargetRetransmitPPM: 10000,
		MinCongestionBPS:    100000,
		RTTMillis:           80,
	})
	if err != nil {
		t.Fatalf("SetFairnessState() error = %v", err)
	}
	if result.Status != runtimelab.ApplyStatusACK || result.Operation != runtimelab.OperationSetFairness {
		t.Fatalf("fairness result = %#v", result)
	}
	if len(dispatcher.applies) != 1 {
		t.Fatalf("dispatch count = %d, want 1", len(dispatcher.applies))
	}
	apply := dispatcher.applies[0]
	if apply.NodeID != "node-1" || apply.FairnessState.EgressPoolBPS != 300000 || apply.FairnessState.LossRatePPM != 20000 {
		t.Fatalf("fairness apply = %#v", apply)
	}
}

func TestGetDigestReturnsLatestNodeDigest(t *testing.T) {
	repo := runtimelab.NewMemoryRepository()
	svc := runtimelab.NewService(repo, &recordingDispatcher{}, time.Now)
	if err := svc.SaveApplyResult(context.Background(), runtimelab.ApplyResult{
		ApplyID: "digest-1",
		NodeID:  "node-1",
		Status:  runtimelab.ApplyStatusACK,
		Digest: runtimelab.Digest{
			AccountCount:  3,
			EnabledCount:  2,
			DisabledCount: 1,
			MaxGeneration: 9,
			Hash:          "abc123",
		},
	}); err != nil {
		t.Fatalf("SaveApplyResult() error = %v", err)
	}

	result, err := svc.GetDigest(context.Background(), "node-1")
	if err != nil {
		t.Fatalf("GetDigest() error = %v", err)
	}
	if result.Status != runtimelab.ApplyStatusACK || result.Digest.Hash != "abc123" || result.Digest.MaxGeneration != 9 {
		t.Fatalf("digest result = %#v", result)
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

type sequenceDispatcher struct {
	applies         []runtimelab.RuntimeApply
	acks            []runtimelab.ApplyResult
	returnNACKError bool
}

func (d *sequenceDispatcher) DispatchRuntimeApply(_ context.Context, apply runtimelab.RuntimeApply) (runtimelab.ApplyResult, error) {
	d.applies = append(d.applies, apply)
	ack := d.acks[0]
	d.acks = d.acks[1:]
	ack.ApplyID = apply.ApplyID
	ack.NodeID = apply.NodeID
	ack.VersionInfo = apply.VersionInfo
	ack.Nonce = apply.Nonce
	if ack.Status == runtimelab.ApplyStatusNACK && d.returnNACKError {
		return ack, errBaseRevisionMismatch
	}
	return ack, nil
}

var errBaseRevisionMismatch = errString("base revision does not match last good revision")

type errString string

func (e errString) Error() string { return string(e) }
