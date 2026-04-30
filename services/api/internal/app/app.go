package app

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rayip/rayip/services/api/internal/bus"
	"github.com/rayip/rayip/services/api/internal/cache"
	"github.com/rayip/rayip/services/api/internal/commercial"
	"github.com/rayip/rayip/services/api/internal/config"
	"github.com/rayip/rayip/services/api/internal/db"
	"github.com/rayip/rayip/services/api/internal/grpcapi"
	"github.com/rayip/rayip/services/api/internal/httpapi"
	"github.com/rayip/rayip/services/api/internal/logging"
	"github.com/rayip/rayip/services/api/internal/netmux"
	"github.com/rayip/rayip/services/api/internal/node"
	"github.com/rayip/rayip/services/api/internal/noderuntime"
	"github.com/rayip/rayip/services/api/internal/runtimecontrol"
	"github.com/rayip/rayip/services/api/internal/runtimelab"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
			node.NewNATSScanPublisher,
			func(p *bus.RuntimePublisher) runtimecontrol.OutboxPublisher { return p },
			func(p *node.NATSPublisher) node.ScanPublisher { return p },
			fx.Annotate(node.NewEntRepository, fx.As(new(node.Repository))),
			fx.Annotate(node.NewRedisLeaseStore, fx.As(new(node.LeaseStore))),
			fx.Annotate(noderuntime.NewEntRepository, fx.As(new(noderuntime.Repository))),
			fx.Annotate(runtimecontrol.NewEntRepository, fx.As(new(runtimecontrol.Repository))),
			fx.Annotate(runtimelab.NewEntRepository, fx.As(new(runtimelab.Repository))),
			fx.Annotate(commercial.NewEntRepository, fx.As(new(commercial.Repository))),
			commercial.NewRuntimeControlAdapter,
			func(a *commercial.RuntimeControlAdapter) commercial.RuntimeWriter { return a },
			grpcapi.NewRuntimeDispatcher,
			func(d *grpcapi.RuntimeDispatcher) runtimelab.Dispatcher { return d },
			func(d *grpcapi.RuntimeDispatcher) runtimecontrol.RuntimeDispatcher { return d },
			func() func() time.Time { return time.Now },
			node.NewService,
			node.NewScanScheduler,
			node.NewScanWorker,
			noderuntime.NewService,
			runtimecontrol.NewService,
			runtimecontrol.NewWorker,
			runtimecontrol.NewReconcilePlanner,
			runtimelab.NewService,
			commercial.NewService,
			httpapi.NewServer,
			grpcapi.NewControlServer,
			grpcapi.NewGRPCServer,
		),
		fx.Invoke(
			db.RegisterLifecycle,
			commercial.RegisterLifecycle,
			cache.RegisterLifecycle,
			bus.RegisterLifecycle,
			runtimecontrol.RegisterRuntimePipelineLifecycle,
			node.RegisterScanPipelineLifecycle,
			RegisterNetworkLifecycle,
		),
	)
}

func RegisterNetworkLifecycle(lc fx.Lifecycle, cfg config.Config, httpServer *fiber.App, grpcServer *grpc.Server, log *zap.Logger) {
	if !sameAddr(cfg.HTTP.Addr, cfg.GRPC.Addr) {
		httpapi.RegisterLifecycle(lc, httpServer, cfg, log)
		grpcapi.RegisterLifecycle(lc, cfg, grpcServer, log)
		return
	}

	var mux *netmux.Mux
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", cfg.HTTP.Addr)
			if err != nil {
				return err
			}
			mux = netmux.New(listener, func(prefix []byte) string {
				if netmux.IsGRPCPrefix(prefix) {
					return "grpc"
				}
				return "http"
			})
			httpListener := mux.Listener("http")
			grpcListener := mux.Listener("grpc")

			go func() {
				if err := grpcServer.Serve(grpcListener); err != nil && !errors.Is(err, net.ErrClosed) {
					log.Error("grpc server stopped", zap.Error(err))
				}
			}()
			go func() {
				err := httpServer.Listener(httpListener, fiber.ListenConfig{DisableStartupMessage: true})
				if err != nil && !errors.Is(err, net.ErrClosed) {
					log.Error("http server stopped", zap.Error(err))
				}
			}()
			go func() {
				if err := mux.Serve(); err != nil && !errors.Is(err, net.ErrClosed) {
					log.Error("api mux stopped", zap.Error(err))
				}
			}()
			log.Info("api mux listening", zap.String("addr", cfg.HTTP.Addr))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopped := make(chan struct{})
			go func() {
				grpcServer.GracefulStop()
				_ = httpServer.Shutdown()
				if mux != nil {
					_ = mux.Close()
				}
				close(stopped)
			}()
			select {
			case <-stopped:
				return nil
			case <-ctx.Done():
				grpcServer.Stop()
				return ctx.Err()
			}
		},
	})
}

func sameAddr(httpAddr, grpcAddr string) bool {
	return normalizeListenAddr(httpAddr) == normalizeListenAddr(grpcAddr)
}

func normalizeListenAddr(addr string) string {
	if addr == "" {
		return ""
	}
	if strings.HasPrefix(addr, ":") {
		return "0.0.0.0" + addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "::" {
		host = "0.0.0.0"
	}
	if port == "" {
		port = os.Getenv("PORT")
	}
	return net.JoinHostPort(host, port)
}
