package cache

import (
	"context"

	"github.com/rayip/rayip/services/api/internal/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

func NewRedis(cfg config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}

func RegisterLifecycle(lc fx.Lifecycle, client *redis.Client) {
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			return client.Close()
		},
	})
}
