package httpapi

import (
	"context"
	"database/sql"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/nats-io/nats.go"
	"github.com/rayip/rayip/services/api/internal/config"
	"github.com/rayip/rayip/services/api/internal/node"
	"github.com/rayip/rayip/services/api/internal/runtimelab"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServerParams struct {
	fx.In

	Config config.Config
	SQLDB  *sql.DB
	Redis  *redis.Client
	NATS   *nats.Conn
	Nodes  *node.Service
	Lab    *runtimelab.Service
}

func NewServer(p ServerParams) *fiber.App {
	app := fiber.New(fiber.Config{AppName: "RayIP API"})
	RegisterHealthRoutes(app, HealthOptions{
		ServiceName: p.Config.Service.Name,
		Version:     p.Config.Service.Version,
		InstanceID:  p.Config.Service.InstanceID,
		ReadyCheck:  readyCheck(p.SQLDB, p.Redis, p.NATS),
	})
	RegisterNodeRoutes(app, p.Nodes)
	RegisterRuntimeLabRoutes(app, p.Lab)
	return app
}

func RegisterLifecycle(lc fx.Lifecycle, app *fiber.App, cfg config.Config, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := app.Listen(cfg.HTTP.Addr); err != nil {
					log.Error("http server stopped", zap.Error(err))
				}
			}()
			log.Info("http server listening", zap.String("addr", cfg.HTTP.Addr))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			done := make(chan error, 1)
			go func() { done <- app.Shutdown() }()
			select {
			case err := <-done:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
}

func readyCheck(sqlDB *sql.DB, redisClient *redis.Client, natsConn *nats.Conn) func() ReadyReport {
	return func() ReadyReport {
		ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
		defer cancel()

		checks := map[string]string{}
		status := "ok"
		if err := sqlDB.PingContext(ctx); err != nil {
			checks["postgres"] = "error: " + err.Error()
			status = "degraded"
		} else {
			checks["postgres"] = "ok"
		}

		if err := redisClient.Ping(ctx).Err(); err != nil {
			checks["redis"] = "error: " + err.Error()
			status = "degraded"
		} else {
			checks["redis"] = "ok"
		}

		if natsConn == nil || !natsConn.IsConnected() {
			checks["nats"] = "error: disconnected"
			status = "degraded"
		} else {
			checks["nats"] = "ok"
		}

		return ReadyReport{Status: status, Checks: checks}
	}
}
