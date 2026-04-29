package bus

import (
	"context"

	"github.com/nats-io/nats.go"
	"github.com/rayip/rayip/services/api/internal/config"
	"go.uber.org/fx"
)

func NewNATS(cfg config.Config) (*nats.Conn, error) {
	return nats.Connect(
		cfg.NATS.URL,
		nats.Name(cfg.Service.InstanceID),
		nats.MaxReconnects(-1),
	)
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
