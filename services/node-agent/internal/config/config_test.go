package config_test

import (
	"testing"

	"github.com/rayip/rayip/services/node-agent/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("RAYIP_AGENT_NODE_CODE", "")
	t.Setenv("RAYIP_AGENT_API_GRPC_ADDR", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Node.Code == "" {
		t.Fatal("node code should have a stable default")
	}
	if cfg.API.GRPCAddr != "127.0.0.1:9090" {
		t.Fatalf("API gRPC addr = %q, want 127.0.0.1:9090", cfg.API.GRPCAddr)
	}
	if cfg.API.UseTLS {
		t.Fatal("API TLS should default to false")
	}
	if cfg.Runtime.BundleDir == "" {
		t.Fatal("runtime bundle dir should have a default")
	}
	if cfg.Runtime.ManifestPath == "" {
		t.Fatal("runtime manifest path should have a default")
	}
	if cfg.Runtime.CoreMode != "xray" {
		t.Fatalf("runtime core mode = %q, want xray", cfg.Runtime.CoreMode)
	}
	if cfg.Runtime.XrayBinaryPath == "" || cfg.Runtime.XrayConfigPath == "" {
		t.Fatalf("xray runtime paths should have defaults: %#v", cfg.Runtime)
	}
	if cfg.Runtime.XrayGRPCAddr != "auto" {
		t.Fatalf("xray grpc addr = %q, want auto", cfg.Runtime.XrayGRPCAddr)
	}
	if cfg.Probe.PublicIPURL == "" || cfg.Probe.Port != 18080 || len(cfg.Probe.Protocols) == 0 {
		t.Fatalf("probe defaults = %#v", cfg.Probe)
	}
}

func TestLoadGRPCSTargetEnablesTLS(t *testing.T) {
	t.Setenv("RAYIP_AGENT_API_GRPC_ADDR", "grpcs://api-production-c00f.up.railway.app:443")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.API.GRPCAddr != "api-production-c00f.up.railway.app:443" {
		t.Fatalf("API gRPC addr = %q, want host:443", cfg.API.GRPCAddr)
	}
	if !cfg.API.UseTLS {
		t.Fatal("API TLS should be enabled for grpcs target")
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("RAYIP_AGENT_NODE_CODE", "nyc-home-001")
	t.Setenv("RAYIP_AGENT_API_GRPC_ADDR", "api.internal:9090")
	t.Setenv("RAYIP_AGENT_ENROLLMENT_TOKEN", "token-1")
	t.Setenv("RAYIP_AGENT_RUNTIME_BUNDLE_DIR", "/opt/rayip/runtime")
	t.Setenv("RAYIP_AGENT_RUNTIME_CORE_MODE", "xray")
	t.Setenv("RAYIP_AGENT_RUNTIME_XRAY_GRPC_ADDR", "127.0.0.1:10085")
	t.Setenv("RAYIP_AGENT_RUNTIME_XRAY_BINARY_PATH", "/opt/rayip/runtime/xray")
	t.Setenv("RAYIP_AGENT_RUNTIME_XRAY_CONFIG_PATH", "/opt/rayip/runtime/config.json")
	t.Setenv("RAYIP_AGENT_PROBE_PUBLIC_IP_URL", "http://probe.local/ip")
	t.Setenv("RAYIP_AGENT_PROBE_SCAN_HOST", "node-agent.example.net")
	t.Setenv("RAYIP_AGENT_PROBE_PORT", "28080")
	t.Setenv("RAYIP_AGENT_PROBE_PROTOCOLS", "SOCKS5")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Node.Code != "nyc-home-001" {
		t.Fatalf("node code = %q, want nyc-home-001", cfg.Node.Code)
	}
	if cfg.API.GRPCAddr != "api.internal:9090" {
		t.Fatalf("API gRPC addr = %q, want api.internal:9090", cfg.API.GRPCAddr)
	}
	if cfg.Node.EnrollmentToken != "token-1" {
		t.Fatalf("enrollment token = %q, want token-1", cfg.Node.EnrollmentToken)
	}
	if cfg.Runtime.BundleDir != "/opt/rayip/runtime" {
		t.Fatalf("runtime bundle dir = %q, want /opt/rayip/runtime", cfg.Runtime.BundleDir)
	}
	if cfg.Runtime.ManifestPath != "/opt/rayip/runtime/runtime-manifest.json" {
		t.Fatalf("runtime manifest path = %q, want manifest inside bundle dir", cfg.Runtime.ManifestPath)
	}
	if cfg.Runtime.CoreMode != "xray" || cfg.Runtime.XrayGRPCAddr != "127.0.0.1:10085" {
		t.Fatalf("runtime core config = %#v", cfg.Runtime)
	}
	if cfg.Runtime.XrayBinaryPath != "/opt/rayip/runtime/xray" || cfg.Runtime.XrayConfigPath != "/opt/rayip/runtime/config.json" {
		t.Fatalf("runtime xray paths = %#v", cfg.Runtime)
	}
	if cfg.Probe.PublicIPURL != "http://probe.local/ip" || cfg.Probe.ScanHost != "node-agent.example.net" || cfg.Probe.Port != 28080 || len(cfg.Probe.Protocols) != 1 || cfg.Probe.Protocols[0] != "SOCKS5" {
		t.Fatalf("probe config = %#v", cfg.Probe)
	}
}
