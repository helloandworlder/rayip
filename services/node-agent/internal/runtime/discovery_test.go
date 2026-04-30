package runtime_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	runtimev1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/runtime/v1"
	"github.com/rayip/rayip/services/node-agent/internal/runtime"
	"google.golang.org/grpc"
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

func TestDiscoverOverridesManifestWithXrayExtension(t *testing.T) {
	addr, cleanup := runtimeDiscoveryServer(t)
	defer cleanup()
	dir := t.TempDir()
	manifest := filepath.Join(dir, "runtime-manifest.json")
	if err := os.WriteFile(manifest, []byte(`{
		"bundle_version": "rayip-runtime-v26.3.27.1",
		"xray_version": "v26.3.27-rayip.1",
		"extension_abi": "stale",
		"binary_sha256": "sha256:binary",
		"manifest_sha256": "sha256:manifest",
		"capabilities": ["stale-capability"]
	}`), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	info, err := runtime.Discover(runtime.DiscoveryConfig{
		AgentVersion: "agent-1",
		ManifestPath: manifest,
		CoreMode:     "xray",
		XrayGRPCAddr: addr,
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	if info.ExtensionABI != "rayip.runtime.v1" {
		t.Fatalf("extension abi = %q, want rayip.runtime.v1", info.ExtensionABI)
	}
	if !reflect.DeepEqual(info.Capabilities, []string{"socks5", "rayip-runtime"}) {
		t.Fatalf("capabilities = %#v", info.Capabilities)
	}
	if info.RuntimeDigest != "digest-1" || info.LastGoodGeneration != 9 {
		t.Fatalf("runtime digest/generation not discovered: %#v", info)
	}
}

func TestDiscoverComputesXrayBinaryHashWhenManifestMissing(t *testing.T) {
	addr, cleanup := runtimeDiscoveryServer(t)
	defer cleanup()
	dir := t.TempDir()
	binaryPath := filepath.Join(dir, "xray")
	binaryPayload := []byte("fake-xray-binary")
	if err := os.WriteFile(binaryPath, binaryPayload, 0o700); err != nil {
		t.Fatalf("write binary: %v", err)
	}

	info, err := runtime.Discover(runtime.DiscoveryConfig{
		AgentVersion: "agent-1",
		ManifestPath: filepath.Join(dir, "missing.json"),
		CoreMode:     "xray",
		XrayGRPCAddr: addr,
		BinaryPath:   binaryPath,
	})
	if err != nil {
		t.Fatalf("Discover() error = %v", err)
	}
	expectedSum := sha256.Sum256(binaryPayload)
	expectedHash := "sha256:" + hex.EncodeToString(expectedSum[:])
	if info.BinarySHA256 != expectedHash {
		t.Fatalf("binary hash = %q, want %q", info.BinarySHA256, expectedHash)
	}
	if info.ExtensionABI != "rayip.runtime.v1" || info.RuntimeDigest != "digest-1" {
		t.Fatalf("runtime extension metadata not discovered: %#v", info)
	}
}

type discoveryRuntimeServer struct {
	runtimev1.UnimplementedRuntimeServiceServer
}

func (s discoveryRuntimeServer) GetCapabilities(context.Context, *runtimev1.GetCapabilitiesRequest) (*runtimev1.GetCapabilitiesResponse, error) {
	return &runtimev1.GetCapabilitiesResponse{
		ExtensionAbi: "rayip.runtime.v1",
		Capabilities: []string{"socks5", "rayip-runtime"},
		Digest: &runtimev1.Digest{
			MaxGeneration: 9,
			Hash:          "digest-1",
		},
	}, nil
}

func runtimeDiscoveryServer(t *testing.T) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	server := grpc.NewServer()
	runtimev1.RegisterRuntimeServiceServer(server, discoveryRuntimeServer{})
	go func() {
		_ = server.Serve(listener)
	}()

	return listener.Addr().String(), func() {
		server.Stop()
		_ = listener.Close()
	}
}
