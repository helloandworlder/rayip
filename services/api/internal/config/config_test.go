package config_test

import (
	"testing"

	"github.com/rayip/rayip/services/api/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("RAYIP_HTTP_ADDR", "")
	t.Setenv("RAYIP_GRPC_ADDR", "")
	t.Setenv("RAYIP_POSTGRES_DSN", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.Addr != ":8080" {
		t.Fatalf("HTTP addr = %q, want :8080", cfg.HTTP.Addr)
	}
	if cfg.GRPC.Addr != ":9090" {
		t.Fatalf("gRPC addr = %q, want :9090", cfg.GRPC.Addr)
	}
	if cfg.Service.Name != "rayip-api" {
		t.Fatalf("service name = %q, want rayip-api", cfg.Service.Name)
	}
	if cfg.Node.EnrollmentToken != "dev-enrollment-token" {
		t.Fatalf("node enrollment token = %q, want dev-enrollment-token", cfg.Node.EnrollmentToken)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("RAYIP_HTTP_ADDR", ":18080")
	t.Setenv("RAYIP_GRPC_ADDR", ":19090")
	t.Setenv("RAYIP_SERVICE_INSTANCE_ID", "api-test")
	t.Setenv("RAYIP_NODE_ENROLLMENT_TOKEN", "ztp-token")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.HTTP.Addr != ":18080" {
		t.Fatalf("HTTP addr = %q, want :18080", cfg.HTTP.Addr)
	}
	if cfg.GRPC.Addr != ":19090" {
		t.Fatalf("gRPC addr = %q, want :19090", cfg.GRPC.Addr)
	}
	if cfg.Service.InstanceID != "api-test" {
		t.Fatalf("instance id = %q, want api-test", cfg.Service.InstanceID)
	}
	if cfg.Node.EnrollmentToken != "ztp-token" {
		t.Fatalf("node enrollment token = %q, want ztp-token", cfg.Node.EnrollmentToken)
	}
}

func TestLoadRedisURLEnv(t *testing.T) {
	t.Setenv("RAYIP_REDIS_ADDR", "")
	t.Setenv("RAYIP_REDIS_URL", "redis://:secret@redis.railway.internal:6379/2")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Redis.Addr != "redis.railway.internal:6379" {
		t.Fatalf("redis addr = %q, want redis.railway.internal:6379", cfg.Redis.Addr)
	}
	if cfg.Redis.Password != "secret" {
		t.Fatalf("redis password = %q, want secret", cfg.Redis.Password)
	}
	if cfg.Redis.DB != 2 {
		t.Fatalf("redis db = %d, want 2", cfg.Redis.DB)
	}
}
