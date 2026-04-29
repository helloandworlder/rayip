package logging

import (
	"github.com/rayip/rayip/services/api/internal/config"
	"go.uber.org/zap"
)

func NewLogger(cfg config.Config) (*zap.Logger, error) {
	if cfg.Service.Env == "prod" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
