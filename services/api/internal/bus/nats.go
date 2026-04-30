package bus

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
	"github.com/rayip/rayip/services/api/internal/config"
	"github.com/rayip/rayip/services/api/internal/runtimecontrol"
	"go.uber.org/fx"
)

func NewNATS(cfg config.Config) (*nats.Conn, error) {
	return nats.Connect(
		cfg.NATS.URL,
		nats.Name(cfg.Service.InstanceID),
		nats.MaxReconnects(-1),
	)
}

type RuntimePublisher struct {
	js nats.JetStreamContext
}

func NewRuntimePublisher(conn *nats.Conn) (*RuntimePublisher, error) {
	js, err := conn.JetStream()
	if err != nil {
		return nil, err
	}
	if _, err := js.AddStream(&nats.StreamConfig{
		Name:     runtimecontrol.RuntimeStreamName,
		Subjects: []string{runtimecontrol.RuntimeApplySubject},
		Storage:  nats.FileStorage,
	}); err != nil {
		if _, updateErr := js.UpdateStream(&nats.StreamConfig{
			Name:     runtimecontrol.RuntimeStreamName,
			Subjects: []string{runtimecontrol.RuntimeApplySubject},
			Storage:  nats.FileStorage,
		}); updateErr != nil {
			return nil, err
		}
	}
	return &RuntimePublisher{js: js}, nil
}

func (p *RuntimePublisher) PublishRuntimeApply(ctx context.Context, event runtimecontrol.OutboxEvent) error {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return err
	}
	_, err = p.js.PublishMsg(&nats.Msg{
		Subject: event.Topic,
		Data:    payload,
		Header: nats.Header{
			nats.MsgIdHdr: []string{event.ID},
		},
	}, nats.Context(ctx))
	return err
}

func RegisterLifecycle(lc fx.Lifecycle, conn *nats.Conn) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			conn.Drain()
			conn.Close()
			return nil
		},
	})
}
