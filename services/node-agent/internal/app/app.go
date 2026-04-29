package app

import (
	"context"

	"github.com/rayip/rayip/services/node-agent/internal/config"
	"github.com/rayip/rayip/services/node-agent/internal/control"
	"github.com/rayip/rayip/services/node-agent/internal/logging"
	"github.com/rayip/rayip/services/node-agent/internal/runtime"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func New() *fx.App {
	return fx.New(
		fx.Provide(
			config.Load,
			logging.NewLogger,
			fx.Annotate(runtime.NewMemoryCore, fx.As(new(runtime.Core))),
			runtime.NewManager,
			control.NewClient,
		),
		fx.Invoke(registerControlLoop),
	)
}

func registerControlLoop(lc fx.Lifecycle, client *control.Client, log *zap.Logger) {
	var cancel context.CancelFunc
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			runCtx, runCancel := context.WithCancel(context.Background())
			cancel = runCancel
			go func() {
				if err := client.Run(runCtx); err != nil && runCtx.Err() == nil {
					log.Error("control client stopped", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if cancel != nil {
				cancel()
			}
			return nil
		},
	})
}
