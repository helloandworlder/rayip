package runtimecontrol

import (
	"context"
	"encoding/json"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/rayip/rayip/services/api/internal/noderuntime"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	RuntimeStreamName   = "RAYIP_RUNTIME"
	RuntimeApplySubject = "rayip.runtime.apply.v1"
	RuntimeConsumerName = "runtime-apply-worker-v2"
	RuntimeQueueName    = "runtime-apply-workers"
	outboxPublishPeriod = 2 * time.Second
)

type runtimeEventPayload struct {
	NodeID string `json:"node_id"`
	Seq    uint64 `json:"seq"`
}

func RegisterRuntimePipelineLifecycle(lc fx.Lifecycle, service *Service, publisher OutboxPublisher, conn *nats.Conn, worker *Worker, nodeRuntime *noderuntime.Service, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			js, err := conn.JetStream()
			if err != nil {
				return err
			}
			if _, err := js.AddStream(&nats.StreamConfig{
				Name:     RuntimeStreamName,
				Subjects: []string{RuntimeApplySubject},
				Storage:  nats.FileStorage,
			}); err != nil {
				if _, updateErr := js.UpdateStream(&nats.StreamConfig{
					Name:     RuntimeStreamName,
					Subjects: []string{RuntimeApplySubject},
					Storage:  nats.FileStorage,
				}); updateErr != nil {
					return err
				}
			}

			stop := make(chan struct{})
			go runOutboxPublisher(stop, service, publisher, log)
			sub, err := js.QueueSubscribe(RuntimeApplySubject, RuntimeQueueName, func(msg *nats.Msg) {
				handleRuntimeMessage(msg, worker, nodeRuntime, log)
			}, nats.Durable(RuntimeConsumerName), nats.ManualAck(), nats.AckExplicit())
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

func runOutboxPublisher(stop <-chan struct{}, service *Service, publisher OutboxPublisher, log *zap.Logger) {
	ticker := time.NewTicker(outboxPublishPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), outboxPublishPeriod)
			if _, err := service.PublishPendingOutbox(ctx, publisher, 100); err != nil {
				log.Warn("runtime outbox publish failed", zap.Error(err))
			}
			cancel()
		}
	}
}

func handleRuntimeMessage(msg *nats.Msg, worker *Worker, nodeRuntime *noderuntime.Service, log *zap.Logger) {
	var payload runtimeEventPayload
	if err := json.Unmarshal(msg.Data, &payload); err != nil {
		log.Warn("runtime event payload invalid", zap.Error(err))
		_ = msg.Term()
		return
	}
	if payload.NodeID == "" {
		log.Warn("runtime event missing node_id")
		_ = msg.Term()
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	afterSeq := uint64(0)
	if status, ok, err := nodeRuntime.GetStatus(ctx, payload.NodeID); err == nil && ok {
		afterSeq = status.LastGoodRevision
	}
	if _, err := worker.ProcessNodeChanges(ctx, payload.NodeID, afterSeq, 100); err != nil {
		log.Warn("runtime worker process failed", zap.String("node_id", payload.NodeID), zap.Error(err))
		_ = msg.Nak()
		return
	}
	_ = msg.Ack()
}
