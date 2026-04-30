package node

import (
	"context"
	"fmt"
	"time"
)

type ScanJob struct {
	JobID      string    `json:"job_id"`
	NodeID     string    `json:"node_id"`
	Attempt    int       `json:"attempt"`
	EnqueuedAt time.Time `json:"enqueued_at"`
}

type ScanScheduleOptions struct {
	Interval time.Duration
	Limit    int
}

type ScanWorkerOptions struct {
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
	MaxAttempts int
}

type ScanDecision struct {
	Scan        ScanResult
	Retry       bool
	NextAttempt int
	RetryDelay  time.Duration
}

type ScanPublisher interface {
	PublishNodeScan(ctx context.Context, job ScanJob) error
}

type ScanScheduler struct {
	nodes *Service
	now   func() time.Time
}

func NewScanScheduler(nodes *Service, now func() time.Time) *ScanScheduler {
	if now == nil {
		now = time.Now
	}
	return &ScanScheduler{nodes: nodes, now: now}
}

func (s *ScanScheduler) EnqueueDueScans(ctx context.Context, publisher ScanPublisher, opts ScanScheduleOptions) (int, error) {
	if opts.Interval <= 0 {
		opts.Interval = 5 * time.Minute
	}
	if opts.Limit <= 0 {
		opts.Limit = 100
	}
	items, err := s.nodes.ListNodes(ctx)
	if err != nil {
		return 0, err
	}
	now := s.now().UTC()
	bucket := now.Unix() / int64(opts.Interval.Seconds())
	count := 0
	for _, item := range items {
		if count >= opts.Limit {
			break
		}
		if item.Status != StatusOnline {
			continue
		}
		if !item.LastScanAt.IsZero() && now.Sub(item.LastScanAt) < opts.Interval {
			continue
		}
		job := ScanJob{
			JobID:      fmt.Sprintf("scan:%s:%d", item.ID, bucket),
			NodeID:     item.ID,
			Attempt:    1,
			EnqueuedAt: now,
		}
		if err := publisher.PublishNodeScan(ctx, job); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

type ScanWorker struct {
	nodes *Service
	now   func() time.Time
}

func NewScanWorker(nodes *Service, now func() time.Time) *ScanWorker {
	if now == nil {
		now = time.Now
	}
	return &ScanWorker{nodes: nodes, now: now}
}

func (w *ScanWorker) ProcessScanJob(ctx context.Context, job ScanJob, opts ScanWorkerOptions) (ScanDecision, error) {
	if job.NodeID == "" {
		return ScanDecision{}, fmt.Errorf("node_id is required")
	}
	if job.Attempt <= 0 {
		job.Attempt = 1
	}
	result, err := w.nodes.ScanNode(ctx, job.NodeID)
	if err != nil {
		return ScanDecision{}, err
	}
	return ScanRetryDecision(result, job.Attempt, opts), nil
}

func ScanRetryDecision(result ScanResult, attempt int, opts ScanWorkerOptions) ScanDecision {
	if opts.BaseBackoff <= 0 {
		opts.BaseBackoff = 5 * time.Second
	}
	if opts.MaxBackoff <= 0 {
		opts.MaxBackoff = 5 * time.Minute
	}
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 5
	}
	decision := ScanDecision{Scan: result}
	if result.Status == "REACHABLE" || attempt >= opts.MaxAttempts {
		return decision
	}
	delay := opts.BaseBackoff
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= opts.MaxBackoff {
			delay = opts.MaxBackoff
			break
		}
	}
	decision.Retry = true
	decision.NextAttempt = attempt + 1
	decision.RetryDelay = delay
	return decision
}
