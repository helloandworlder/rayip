package config

import (
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

type Config struct {
	Service  ServiceConfig
	HTTP     HTTPConfig
	GRPC     GRPCConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	NATS     NATSConfig
	Node     NodeConfig
}

type ServiceConfig struct {
	Name       string
	Version    string
	InstanceID string
	Env        string
}

type HTTPConfig struct {
	Addr string
}

type GRPCConfig struct {
	Addr string
}

type PostgresConfig struct {
	DSN           string
	RunMigrations bool
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type NATSConfig struct {
	URL string
}

type NodeConfig struct {
	LeaseTTLSeconds int
	EnrollmentToken string
}

func Load() (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("RAYIP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "local"
	}

	v.SetDefault("service.name", "rayip-api")
	v.SetDefault("service.version", "0.1.0-dev")
	v.SetDefault("service.instance_id", "api-"+hostname+"-"+uuid.NewString()[:8])
	v.SetDefault("service.env", "dev")
	v.SetDefault("http.addr", ":8080")
	v.SetDefault("grpc.addr", ":9090")
	if port := os.Getenv("PORT"); port != "" {
		v.SetDefault("http.addr", ":"+port)
		v.SetDefault("grpc.addr", ":"+port)
	}
	v.SetDefault("postgres.dsn", "postgres://rayip:rayip@localhost:5432/rayip?sslmode=disable")
	v.SetDefault("postgres.run_migrations", true)
	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.url", "")
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("node.lease_ttl_seconds", 45)
	v.SetDefault("node.enrollment_token", "dev-enrollment-token")

	redisAddr := v.GetString("redis.addr")
	redisPassword := v.GetString("redis.password")
	redisDB := v.GetInt("redis.db")
	if rawURL := v.GetString("redis.url"); rawURL != "" {
		if parsed, err := url.Parse(rawURL); err == nil {
			if parsed.Host != "" {
				redisAddr = parsed.Host
			}
			if password, ok := parsed.User.Password(); ok {
				redisPassword = password
			}
			if db, err := strconv.Atoi(strings.TrimPrefix(parsed.Path, "/")); err == nil {
				redisDB = db
			}
		}
	}

	return Config{
		Service: ServiceConfig{
			Name:       v.GetString("service.name"),
			Version:    v.GetString("service.version"),
			InstanceID: v.GetString("service.instance_id"),
			Env:        v.GetString("service.env"),
		},
		HTTP: HTTPConfig{Addr: v.GetString("http.addr")},
		GRPC: GRPCConfig{Addr: v.GetString("grpc.addr")},
		Postgres: PostgresConfig{
			DSN:           v.GetString("postgres.dsn"),
			RunMigrations: v.GetBool("postgres.run_migrations"),
		},
		Redis: RedisConfig{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       redisDB,
		},
		NATS: NATSConfig{URL: v.GetString("nats.url")},
		Node: NodeConfig{
			LeaseTTLSeconds: v.GetInt("node.lease_ttl_seconds"),
			EnrollmentToken: v.GetString("node.enrollment_token"),
		},
	}, nil
}
