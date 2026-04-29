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
	"github.com/rayip/rayip/services/api/internal/runtimelab"
	"go.uber.org/fx"
)

func New() *fx.App {
	return fx.New(
		fx.Provide(
			config.Load,
			logging.NewLogger,
			db.NewGorm,
			db.SQLDB,
			cache.NewRedis,
			bus.NewNATS,
			fx.Annotate(node.NewGormRepository, fx.As(new(node.Repository))),
			fx.Annotate(node.NewRedisLeaseStore, fx.As(new(node.LeaseStore))),
			fx.Annotate(runtimelab.NewGormRepository, fx.As(new(runtimelab.Repository))),
			grpcapi.NewRuntimeDispatcher,
			func(d *grpcapi.RuntimeDispatcher) runtimelab.Dispatcher { return d },
			func() func() time.Time { return time.Now },
			node.NewService,
			runtimelab.NewService,
			httpapi.NewServer,
			grpcapi.NewControlServer,
			grpcapi.NewGRPCServer,
		),
		fx.Invoke(
			db.RegisterLifecycle,
			cache.RegisterLifecycle,
			bus.RegisterLifecycle,
			httpapi.RegisterLifecycle,
			grpcapi.RegisterLifecycle,
		),
	)
}
