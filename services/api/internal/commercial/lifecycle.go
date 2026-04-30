package commercial

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

func RegisterLifecycle(lc fx.Lifecycle, service *Service, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := service.BootstrapDefaults(ctx); err != nil {
				return err
			}
			log.Info("commercial defaults bootstrapped")
			return nil
		},
	})
}
