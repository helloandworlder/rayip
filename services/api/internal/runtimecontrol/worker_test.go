package runtimecontrol_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rayip/rayip/services/api/internal/runtimecontrol"
	"github.com/rayip/rayip/services/api/internal/runtimelab"
)

type recordingPublisher struct {
	events []runtimecontrol.OutboxEvent
}

func (p *recordingPublisher) PublishRuntimeApply(ctx context.Context, event runtimecontrol.OutboxEvent) error {
	p.events = append(p.events, event)
	return nil
}

type recordingDispatcher struct {
	applies []runtimelab.RuntimeApply
	result  runtimelab.ApplyResult
	err     error
}

func (d *recordingDispatcher) DispatchRuntimeApply(ctx context.Context, apply runtimelab.RuntimeApply) (runtimelab.ApplyResult, error) {
	d.applies = append(d.applies, apply)
	if d.result.ApplyID == "" {
		d.result.ApplyID = apply.ApplyID
	}
	if d.result.NodeID == "" {
		d.result.NodeID = apply.NodeID
	}
	if d.result.VersionInfo == "" {
		d.result.VersionInfo = apply.VersionInfo
	}
	if d.result.Nonce == "" {
		d.result.Nonce = apply.Nonce
	}
	return d.result, d.err
}

func TestPublishPendingOutboxMarksEventsPublishedIdempotently(t *testing.T) {
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

	publisher := &recordingPublisher{}
	published, err := svc.PublishPendingOutbox(context.Background(), publisher, 10)
	if err != nil {
		t.Fatalf("PublishPendingOutbox() error = %v", err)
	}
	if published != 1 || len(publisher.events) != 1 {
		t.Fatalf("published=%d events=%#v", published, publisher.events)
	}

	published, err = svc.PublishPendingOutbox(context.Background(), publisher, 10)
	if err != nil {
		t.Fatalf("second PublishPendingOutbox() error = %v", err)
	}
	if published != 0 || len(publisher.events) != 1 {
		t.Fatalf("second publish should be idempotent, published=%d events=%#v", published, publisher.events)
	}
}

func TestRuntimeWorkerRereadsDesiredStateAndDispatchesDeltaBatch(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	repo := runtimecontrol.NewMemoryRepository()
	svc := runtimecontrol.NewService(repo, func() time.Time { return now })
	if _, err := svc.UpsertProxyAccount(context.Background(), runtimecontrol.ResourceInput{
		ProxyAccountID:  "acct-1",
		NodeID:          "node-1",
		RuntimeEmail:    "acct-1",
		Protocol:        runtimecontrol.ProtocolHTTP,
		Port:            18081,
		Username:        "u1",
		Password:        "p1",
		EgressLimitBPS:  1024,
		IngressLimitBPS: 2048,
		MaxConnections:  2,
	}); err != nil {
		t.Fatalf("upsert error = %v", err)
	}
	if _, err := svc.UpsertProxyAccount(context.Background(), runtimecontrol.ResourceInput{
		ProxyAccountID: "acct-2",
		NodeID:         "node-1",
		RuntimeEmail:   "acct-2",
		Protocol:       runtimecontrol.ProtocolSOCKS5,
		Port:           18082,
		Username:       "u2",
		Password:       "p2",
	}); err != nil {
		t.Fatalf("second upsert error = %v", err)
	}

	dispatcher := &recordingDispatcher{result: runtimelab.ApplyResult{
		Status:           runtimelab.ApplyStatusACK,
		AppliedRevision:  2,
		LastGoodRevision: 2,
	}}
	worker := runtimecontrol.NewWorker(svc, dispatcher, func() time.Time { return now })
	result, err := worker.ProcessNodeChanges(context.Background(), "node-1", 0, 100)
	if err != nil {
		t.Fatalf("ProcessNodeChanges() error = %v", err)
	}

	if result.Status != runtimecontrol.JobStatusSucceeded || result.TargetRevision != 2 || result.LastGoodRevision != 2 {
		t.Fatalf("job result = %#v", result)
	}
	if len(dispatcher.applies) != 1 {
		t.Fatalf("dispatches = %#v", dispatcher.applies)
	}
	apply := dispatcher.applies[0]
	if apply.Mode != runtimelab.ApplyModeDelta || apply.BaseRevision != 0 || apply.TargetRevision != 2 {
		t.Fatalf("apply metadata = %#v", apply)
	}
	if len(apply.Resources) != 2 || len(apply.RemovedResourceNames) != 0 {
		t.Fatalf("apply resources=%#v removed=%#v", apply.Resources, apply.RemovedResourceNames)
	}
	if apply.Resources[0].Name != "proxy/acct-1" || apply.Resources[1].Name != "proxy/acct-2" {
		t.Fatalf("resource names = %#v", apply.Resources)
	}
}

func TestRuntimeWorkerNACKDoesNotAdvanceLastGoodRevision(t *testing.T) {
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

	dispatcher := &recordingDispatcher{result: runtimelab.ApplyResult{
		Status:           runtimelab.ApplyStatusNACK,
		AppliedRevision:  0,
		LastGoodRevision: 0,
		ErrorDetail:      "base revision mismatch",
	}}
	worker := runtimecontrol.NewWorker(svc, dispatcher, func() time.Time { return now })
	result, err := worker.ProcessNodeChanges(context.Background(), "node-1", 0, 100)
	if err != nil {
		t.Fatalf("ProcessNodeChanges() error = %v", err)
	}
	if result.Status != runtimecontrol.JobStatusFailed || result.AcceptedRevision != 0 || result.LastGoodRevision != 0 {
		t.Fatalf("nack job result = %#v", result)
	}
	if result.ErrorDetail != "base revision mismatch" {
		t.Fatalf("error detail = %q", result.ErrorDetail)
	}
}

func TestRuntimeWorkerDispatcherErrorCreatesRetryableAttempt(t *testing.T) {
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

	worker := runtimecontrol.NewWorker(svc, &recordingDispatcher{err: errors.New("node is not connected")}, func() time.Time { return now })
	result, err := worker.ProcessNodeChanges(context.Background(), "node-1", 0, 100)
	if err == nil {
		t.Fatalf("ProcessNodeChanges() expected dispatcher error")
	}
	if result.Status != runtimecontrol.JobStatusRetryable || result.ErrorDetail != "node is not connected" {
		t.Fatalf("retryable result = %#v", result)
	}
}
