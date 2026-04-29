package db

import (
	"context"
	"database/sql"
	"embed"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/rayip/rayip/services/api/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func NewGorm(cfg config.Config) (*gorm.DB, error) {
	gormLog := logger.Default.LogMode(logger.Silent)
	if cfg.Service.Env == "prod" {
		gormLog = logger.Default.LogMode(logger.Warn)
	}
	return gorm.Open(postgres.Open(cfg.Postgres.DSN), &gorm.Config{Logger: gormLog})
}

func SQLDB(gdb *gorm.DB) (*sql.DB, error) {
	return gdb.DB()
}

func RegisterLifecycle(lc fx.Lifecycle, cfg config.Config, sqlDB *sql.DB, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			sqlDB.SetMaxOpenConns(30)
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetConnMaxLifetime(30 * time.Minute)
			if cfg.Postgres.RunMigrations {
				goose.SetBaseFS(migrationsFS)
				if err := goose.SetDialect("postgres"); err != nil {
					return err
				}
				if err := goose.UpContext(ctx, sqlDB, "migrations"); err != nil {
					return err
				}
				log.Info("database migrations applied")
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return sqlDB.Close()
		},
	})
}
