package node

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	ScanStreamName     = "RAYIP_NODE_SCAN"
	ScanSubject        = "rayip.node.scan.v1"
	ScanConsumerName   = "node-scan-worker-v2"
	ScanQueueName      = "node-scan-workers"
	scanSchedulePeriod = 30 * time.Second
)

type NATSPublisher struct {
	js nats.JetStreamContext
}

func NewNATSScanPublisher(conn *nats.Conn) (*NATSPublisher, error) {
	js, err := conn.JetStream()
	if err != nil {
		return nil, err
	}
	if err := ensureScanStream(js); err != nil {
		return nil, err
	}
	return &NATSPublisher{js: js}, nil
}

func (p *NATSPublisher) PublishNodeScan(ctx context.Context, job ScanJob) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	_, err = p.js.PublishMsg(&nats.Msg{
		Subject: ScanSubject,
		Data:    payload,
		Header: nats.Header{
			nats.MsgIdHdr: []string{job.JobID},
		},
	}, nats.Context(ctx))
	return err
}

func RegisterScanPipelineLifecycle(lc fx.Lifecycle, conn *nats.Conn, scheduler *ScanScheduler, publisher ScanPublisher, worker *ScanWorker, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			js, err := conn.JetStream()
			if err != nil {
				return err
			}
			if err := ensureScanStream(js); err != nil {
				return err
			}
			stop := make(chan struct{})
			go runScanScheduler(stop, scheduler, publisher, log)
			sub, err := js.QueueSubscribe(ScanSubject, ScanQueueName, func(msg *nats.Msg) {
				handleScanMessage(msg, worker, log)
			}, nats.Durable(ScanConsumerName), nats.ManualAck(), nats.AckExplicit())
			if err != nil {
				close(stop)
				return err
			}
			lc.Append(fx.Hook{
				OnStop: func(context.Context) error {
					close(stop)
					return sub.Drain()
				},
			})
			return nil
		},
	})
}

func ensureScanStream(js nats.JetStreamContext) error {
	cfg := &nats.StreamConfig{
		Name:     ScanStreamName,
		Subjects: []string{ScanSubject},
		Storage:  nats.FileStorage,
	}
	if _, err := js.AddStream(cfg); err != nil {
		if _, updateErr := js.UpdateStream(cfg); updateErr != nil {
			return err
		}
	}
	return nil
}

func runScanScheduler(stop <-chan struct{}, scheduler *ScanScheduler, publisher ScanPublisher, log *zap.Logger) {
	ticker := time.NewTicker(scanSchedulePeriod)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), scanSchedulePeriod)
			if _, err := scheduler.EnqueueDueScans(ctx, publisher, ScanScheduleOptions{Interval: 5 * time.Minute, Limit: 100}); err != nil {
				log.Warn("node scan enqueue failed", zap.Error(err))
			}
			cancel()
		}
	}
}

func handleScanMessage(msg *nats.Msg, worker *ScanWorker, log *zap.Logger) {
	var job ScanJob
	if err := json.Unmarshal(msg.Data, &job); err != nil {
		log.Warn("node scan payload invalid", zap.Error(err))
		_ = msg.Term()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	decision, err := worker.ProcessScanJob(ctx, job, ScanWorkerOptions{
		BaseBackoff: 5 * time.Second,
		MaxBackoff:  5 * time.Minute,
		MaxAttempts: 5,
	})
	if err != nil {
		log.Warn("node scan worker failed", zap.String("node_id", job.NodeID), zap.Error(err))
		_ = msg.NakWithDelay(30 * time.Second)
		return
	}
	if decision.Retry {
		log.Warn("node scan retry scheduled",
			zap.String("node_id", job.NodeID),
			zap.String("status", decision.Scan.Status),
			zap.Duration("delay", decision.RetryDelay),
			zap.Int("next_attempt", decision.NextAttempt),
		)
		_ = msg.NakWithDelay(decision.RetryDelay)
		return
	}
	_ = msg.Ack()
}
