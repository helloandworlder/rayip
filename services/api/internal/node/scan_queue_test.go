package node_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/rayip/rayip/services/api/internal/node"
)

type recordingScanPublisher struct {
	jobs []node.ScanJob
}

func (p *recordingScanPublisher) PublishNodeScan(ctx context.Context, job node.ScanJob) error {
	p.jobs = append(p.jobs, job)
	return nil
}

func TestScanSchedulerEnqueuesOnlyDueNodesWithStableJobID(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	repo := node.NewMemoryRepository()
	leases := node.NewMemoryLeaseStore()
	svc := node.NewService(repo, leases, func() time.Time { return now })

	due, err := svc.RegisterLease(context.Background(), node.LeaseInput{
		NodeCode:           "due-node",
		SessionID:          "session-due",
		CandidatePublicIPs: []string{"204.42.251.2"},
		ProbePort:          9878,
		LeaseTTLSeconds:    45,
	})
	if err != nil {
		t.Fatalf("RegisterLease(due) error = %v", err)
	}
	recent, err := svc.RegisterLease(context.Background(), node.LeaseInput{
		NodeCode:           "recent-node",
		SessionID:          "session-recent",
		CandidatePublicIPs: []string{"204.42.251.3"},
		ProbePort:          9878,
		LeaseTTLSeconds:    45,
	})
	if err != nil {
		t.Fatalf("RegisterLease(recent) error = %v", err)
	}
	if err := repo.SaveScanResult(context.Background(), recent.ID, node.ScanResult{
		NodeID:    recent.ID,
		Target:    "204.42.251.3:9878",
		Status:    "REACHABLE",
		ScannedAt: now.Add(-1 * time.Minute),
	}); err != nil {
		t.Fatalf("SaveScanResult() error = %v", err)
	}

	publisher := &recordingScanPublisher{}
	scheduler := node.NewScanScheduler(svc, func() time.Time { return now })
	count, err := scheduler.EnqueueDueScans(context.Background(), publisher, node.ScanScheduleOptions{
		Interval: 10 * time.Minute,
		Limit:    100,
	})
	if err != nil {
		t.Fatalf("EnqueueDueScans() error = %v", err)
	}
	if count != 1 || len(publisher.jobs) != 1 {
		t.Fatalf("count=%d jobs=%#v", count, publisher.jobs)
	}
	job := publisher.jobs[0]
	if job.NodeID != due.ID || job.Attempt != 1 {
		t.Fatalf("job = %#v", job)
	}
	wantJobID := "scan:" + due.ID + ":2962584"
	if job.JobID != wantJobID {
		t.Fatalf("job id = %q", job.JobID)
	}
}

func TestScanWorkerRetriesUnreachableWithExponentialBackoff(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	repo := node.NewMemoryRepository()
	leases := node.NewMemoryLeaseStore()
	svc := node.NewService(repo, leases, func() time.Time { return now })
	summary, err := svc.RegisterLease(context.Background(), node.LeaseInput{
		NodeCode:           "scan-worker-node",
		SessionID:          "session-1",
		CandidatePublicIPs: []string{"204.42.251.2"},
		ProbePort:          9878,
		LeaseTTLSeconds:    45,
	})
	if err != nil {
		t.Fatalf("RegisterLease() error = %v", err)
	}
	svc.SetDialerForTest(func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("connection refused")
	})

	worker := node.NewScanWorker(svc, func() time.Time { return now })
	decision, err := worker.ProcessScanJob(context.Background(), node.ScanJob{
		JobID:   "job-1",
		NodeID:  summary.ID,
		Attempt: 2,
	}, node.ScanWorkerOptions{
		BaseBackoff: 5 * time.Second,
		MaxBackoff:  1 * time.Minute,
		MaxAttempts: 4,
	})
	if err != nil {
		t.Fatalf("ProcessScanJob() error = %v", err)
	}
	if !decision.Retry || decision.NextAttempt != 3 || decision.RetryDelay != 10*time.Second {
		t.Fatalf("decision = %#v", decision)
	}
	if decision.Scan.Status != "UNREACHABLE" {
		t.Fatalf("scan = %#v", decision.Scan)
	}
}

func TestScanWorkerStopsAfterMaxAttempts(t *testing.T) {
	decision := node.ScanRetryDecision(node.ScanResult{Status: "UNREACHABLE"}, 4, node.ScanWorkerOptions{
		BaseBackoff: 5 * time.Second,
		MaxBackoff:  1 * time.Minute,
		MaxAttempts: 4,
	})
	if decision.Retry {
		t.Fatalf("decision = %#v, want no retry after max attempts", decision)
	}
}
