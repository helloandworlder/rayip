package db

import (
	"context"
	"database/sql"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"
	_ "github.com/jackc/pgx/v5/stdlib"
	apiEnt "github.com/rayip/rayip/services/api/ent"
	"github.com/rayip/rayip/services/api/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewSQLDB(cfg config.Config) (*sql.DB, error) {
	return sql.Open("pgx", cfg.Postgres.DSN)
}

func NewEntClient(sqlDB *sql.DB, cfg config.Config) *apiEnt.Client {
	driver := entsql.OpenDB(dialect.Postgres, sqlDB)
	options := []apiEnt.Option{apiEnt.Driver(driver)}
	if cfg.Service.Env != "prod" {
		options = append(options, apiEnt.Debug())
	}
	return apiEnt.NewClient(options...)
}

func RegisterLifecycle(lc fx.Lifecycle, cfg config.Config, sqlDB *sql.DB, client *apiEnt.Client, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			sqlDB.SetMaxOpenConns(30)
			sqlDB.SetMaxIdleConns(10)
			sqlDB.SetConnMaxLifetime(30 * time.Minute)
			if cfg.Postgres.RunMigrations {
				if err := client.Schema.Create(ctx, schema.WithForeignKeys(false)); err != nil {
					return err
				}
				log.Info("ent schema migration applied")
			}
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return client.Close()
		},
	})
}
