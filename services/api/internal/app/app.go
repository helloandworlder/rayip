package app

import (
	"time"

	"github.com/rayip/rayip/services/api/internal/bus"
	"github.com/rayip/rayip/services/api/internal/cache"
	"github.com/rayip/rayip/services/api/internal/config"
	"github.com/rayip/rayip/services/api/internal/db"
	"github.com/rayip/rayip/services/api/internal/grpcapi"
	"github.com/rayip/rayip/services/api/internal/httpapi"
	"github.com/rayip/rayip/services/api/internal/logging"
	"github.com/rayip/rayip/services/api/internal/node"
	"github.com/rayip/rayip/services/api/internal/noderuntime"
	"github.com/rayip/rayip/services/api/internal/runtimecontrol"
	"github.com/rayip/rayip/services/api/internal/runtimelab"
	"go.uber.org/fx"
)

func New() *fx.App {
	return fx.New(
		fx.Provide(
			config.Load,
			logging.NewLogger,
			db.NewSQLDB,
			db.NewEntClient,
			cache.NewRedis,
			bus.NewNATS,
			bus.NewRuntimePublisher,
			func(p *bus.RuntimePublisher) runtimecontrol.OutboxPublisher { return p },
			fx.Annotate(node.NewEntRepository, fx.As(new(node.Repository))),
			fx.Annotate(node.NewRedisLeaseStore, fx.As(new(node.LeaseStore))),
			fx.Annotate(noderuntime.NewEntRepository, fx.As(new(noderuntime.Repository))),
			fx.Annotate(runtimecontrol.NewEntRepository, fx.As(new(runtimecontrol.Repository))),
			fx.Annotate(runtimelab.NewEntRepository, fx.As(new(runtimelab.Repository))),
			grpcapi.NewRuntimeDispatcher,
			func(d *grpcapi.RuntimeDispatcher) runtimelab.Dispatcher { return d },
			func(d *grpcapi.RuntimeDispatcher) runtimecontrol.RuntimeDispatcher { return d },
			func() func() time.Time { return time.Now },
			node.NewService,
			noderuntime.NewService,
			runtimecontrol.NewService,
			runtimecontrol.NewWorker,
			runtimecontrol.NewReconcilePlanner,
			runtimelab.NewService,
			httpapi.NewServer,
			grpcapi.NewControlServer,
			grpcapi.NewGRPCServer,
		),
		fx.Invoke(
			db.RegisterLifecycle,
			cache.RegisterLifecycle,
			bus.RegisterLifecycle,
			runtimecontrol.RegisterRuntimePipelineLifecycle,
			httpapi.RegisterLifecycle,
			grpcapi.RegisterLifecycle,
		),
	)
}
