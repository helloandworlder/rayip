package app

import (
	"context"
	"strings"
	"time"

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
			newRuntimeEndpoint,
			fx.Annotate(newRuntimeCore, fx.As(new(runtime.Core))),
			runtime.NewManager,
			control.NewClient,
		),
		fx.Invoke(registerControlLoop),
	)
}

func newRuntimeEndpoint(cfg config.Config) *runtime.Endpoint {
	return runtime.NewEndpoint(cfg.Runtime.XrayGRPCAddr)
}

func newRuntimeCore(lc fx.Lifecycle, cfg config.Config, endpoint *runtime.Endpoint) (runtime.Core, error) {
	if strings.EqualFold(cfg.Runtime.CoreMode, "xray") {
		var (
			process *runtime.XrayProcess
			core    *runtime.XrayCore
			cancel  context.CancelFunc
		)
		if cfg.Runtime.XrayAutoStart {
			started, err := startManagedRuntime(cfg, endpoint)
			if err != nil {
				return nil, err
			}
			process = started.process
			core = started.core
			cancel = started.cancel
			lc.Append(fx.Hook{OnStop: func(ctx context.Context) error {
				cancel()
				return process.Stop(ctx)
			}})
			return core, nil
		}
		connectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		endpoint.SetGRPCAddr(cfg.Runtime.XrayGRPCAddr)
		return runtime.NewXrayCore(connectCtx, cfg.Runtime.XrayGRPCAddr)
	}
	return runtime.NewMemoryCore(), nil
}

type managedRuntime struct {
	process *runtime.XrayProcess
	core    *runtime.XrayCore
	cancel  context.CancelFunc
}

func startManagedRuntime(cfg config.Config, endpoint *runtime.Endpoint) (managedRuntime, error) {
	var lastErr error
	for attempt := 0; attempt < 8; attempt++ {
		addr, err := runtime.ResolveXrayAPIAddr("auto")
		if err != nil {
			lastErr = err
			continue
		}
		startCtx, cancel := context.WithCancel(context.Background())
		process, err := runtime.StartXrayProcess(startCtx, runtime.XrayProcessConfig{
			BinaryPath: cfg.Runtime.XrayBinaryPath,
			ConfigPath: cfg.Runtime.XrayConfigPath,
			GRPCAddr:   addr,
		})
		if err != nil {
			cancel()
			lastErr = err
			continue
		}
		connectCtx, connectCancel := context.WithTimeout(context.Background(), 5*time.Second)
		core, err := runtime.NewXrayCore(connectCtx, addr)
		connectCancel()
		if err == nil {
			endpoint.SetGRPCAddr(addr)
			return managedRuntime{process: process, core: core, cancel: cancel}, nil
		}
		lastErr = err
		_ = process.Stop(context.Background())
		cancel()
	}
	return managedRuntime{}, lastErr
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
