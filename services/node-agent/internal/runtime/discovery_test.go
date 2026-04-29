package runtime_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/rayip/rayip/services/node-agent/internal/runtime"
)

func TestDiscoverReadsRuntimeManifest(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "runtime-manifest.json")
	if err := os.WriteFile(manifest, []byte(`{
		"bundle_version": "rayip-runtime-v26.3.27.1",
		"xray_version": "v26.3.27-rayip.1",
		"extension_abi": "rayip.runtime.v1",
		"binary_sha256": "sha256:binary",
		"manifest_sha256": "sha256:manifest",
		"capabilities": ["socks5", "http", "account-rate-limit"]
	}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	info, err := runtime.Discover(runtime.DiscoveryConfig{
		AgentVersion: "agent-1",
		ManifestPath: manifest,
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if info.AgentVersion != "agent-1" || info.BundleVersion != "rayip-runtime-v26.3.27.1" || info.XrayVersion != "v26.3.27-rayip.1" {
		t.Fatalf("unexpected runtime info: %#v", info)
	}
	if info.ExtensionABI != "rayip.runtime.v1" || info.BinarySHA256 != "sha256:binary" || info.ManifestSHA256 != "sha256:manifest" {
		t.Fatalf("manifest metadata not discovered: %#v", info)
	}
	if !reflect.DeepEqual(info.Capabilities, []string{"socks5", "http", "account-rate-limit"}) {
		t.Fatalf("capabilities = %#v", info.Capabilities)
	}
}

func TestDiscoverFallsBackToBootstrapUnknownWhenManifestMissing(t *testing.T) {
	info, err := runtime.Discover(runtime.DiscoveryConfig{
		AgentVersion: "agent-1",
		ManifestPath: filepath.Join(t.TempDir(), "missing.json"),
	})
	if err != nil {
		t.Fatalf("Discover() missing manifest error = %v", err)
	}
	if info.BundleVersion != "unknown" || info.XrayVersion != "unknown" {
		t.Fatalf("fallback runtime info: %#v", info)
	}
	if len(info.Capabilities) != 0 {
		t.Fatalf("fallback capabilities = %#v, want empty", info.Capabilities)
	}
}
