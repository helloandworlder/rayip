package runtime_test

import (
	"encoding/json"
	"testing"

	"github.com/rayip/rayip/services/node-agent/internal/runtime"
)

func TestBuildXrayRuntimeConfigIncludesRayIPService(t *testing.T) {
	payload, err := runtime.BuildXrayRuntimeConfig("127.0.0.1:10085")
	if err != nil {
		t.Fatalf("BuildXrayRuntimeConfig() error = %v", err)
	}
	var cfg struct {
		API struct {
			Listen   string   `json:"listen"`
			Services []string `json:"services"`
		} `json:"api"`
	}
	if err := json.Unmarshal(payload, &cfg); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if cfg.API.Listen != "127.0.0.1:10085" {
		t.Fatalf("api listen = %q, want 127.0.0.1:10085", cfg.API.Listen)
	}
	if !contains(cfg.API.Services, "RayIPRuntimeService") {
		t.Fatalf("api services = %#v, want RayIPRuntimeService", cfg.API.Services)
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
