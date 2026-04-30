package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"time"

	runtimev1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/runtime/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var runtimeDialContext func(context.Context, string) (net.Conn, error)

type DiscoveryConfig struct {
	AgentVersion string
	ManifestPath string
	CoreMode     string
	XrayGRPCAddr string
	BinaryPath   string
}

type DiscoveryInfo struct {
	AgentVersion       string   `json:"agent_version"`
	XrayVersion        string   `json:"xray_version"`
	BundleVersion      string   `json:"bundle_version"`
	ExtensionABI       string   `json:"extension_abi"`
	Capabilities       []string `json:"capabilities"`
	BinarySHA256       string   `json:"binary_sha256"`
	ManifestSHA256     string   `json:"manifest_sha256"`
	RuntimeDigest      string   `json:"runtime_digest"`
	LastGoodGeneration uint64   `json:"last_good_generation"`
}

func Discover(cfg DiscoveryConfig) (DiscoveryInfo, error) {
	info := DiscoveryInfo{
		AgentVersion:  cfg.AgentVersion,
		XrayVersion:   "unknown",
		BundleVersion: "unknown",
		Capabilities:  []string{},
	}
	payload, err := os.ReadFile(cfg.ManifestPath)
	if errors.Is(err, os.ErrNotExist) {
		if strings.EqualFold(cfg.CoreMode, "xray") {
			return finalizeXrayDiscovery(cfg, info)
		}
		return info, nil
	}
	if err != nil {
		return DiscoveryInfo{}, err
	}

	var manifest struct {
		XrayVersion        string   `json:"xray_version"`
		BundleVersion      string   `json:"bundle_version"`
		ExtensionABI       string   `json:"extension_abi"`
		Capabilities       []string `json:"capabilities"`
		BinarySHA256       string   `json:"binary_sha256"`
		ManifestSHA256     string   `json:"manifest_sha256"`
		RuntimeDigest      string   `json:"runtime_digest"`
		LastGoodGeneration uint64   `json:"last_good_generation"`
	}
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return DiscoveryInfo{}, err
	}
	if manifest.XrayVersion != "" {
		info.XrayVersion = manifest.XrayVersion
	}
	if manifest.BundleVersion != "" {
		info.BundleVersion = manifest.BundleVersion
	}
	info.ExtensionABI = manifest.ExtensionABI
	info.BinarySHA256 = manifest.BinarySHA256
	info.ManifestSHA256 = manifest.ManifestSHA256
	info.RuntimeDigest = manifest.RuntimeDigest
	info.LastGoodGeneration = manifest.LastGoodGeneration
	if manifest.Capabilities != nil {
		info.Capabilities = manifest.Capabilities
	}
	if strings.EqualFold(cfg.CoreMode, "xray") {
		return finalizeXrayDiscovery(cfg, info)
	}
	return info, nil
}

func finalizeXrayDiscovery(cfg DiscoveryConfig, info DiscoveryInfo) (DiscoveryInfo, error) {
	discovered, err := discoverXrayExtension(cfg.XrayGRPCAddr)
	if err != nil {
		return DiscoveryInfo{}, err
	}
	if discovered.ExtensionABI != "" {
		info.ExtensionABI = discovered.ExtensionABI
	}
	if discovered.Capabilities != nil {
		info.Capabilities = discovered.Capabilities
	}
	if discovered.RuntimeDigest != "" {
		info.RuntimeDigest = discovered.RuntimeDigest
	}
	if discovered.LastGoodGeneration > 0 {
		info.LastGoodGeneration = discovered.LastGoodGeneration
	}
	if info.BinarySHA256 == "" && cfg.BinaryPath != "" {
		hash, err := fileSHA256(cfg.BinaryPath)
		if err != nil {
			return DiscoveryInfo{}, err
		}
		info.BinarySHA256 = hash
	}
	return info, nil
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil)), nil
}

func discoverXrayExtension(addr string) (DiscoveryInfo, error) {
	if addr == "" {
		return DiscoveryInfo{}, errors.New("xray runtime gRPC addr is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if runtimeDialContext != nil {
		opts = append(opts, grpc.WithContextDialer(runtimeDialContext))
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		return DiscoveryInfo{}, err
	}
	defer conn.Close()
	response, err := runtimev1.NewRuntimeServiceClient(conn).GetCapabilities(ctx, &runtimev1.GetCapabilitiesRequest{})
	if err != nil {
		return DiscoveryInfo{}, err
	}
	digest := response.GetDigest()
	info := DiscoveryInfo{
		ExtensionABI:  response.GetExtensionAbi(),
		Capabilities:  response.GetCapabilities(),
		RuntimeDigest: digest.GetHash(),
	}
	if digest != nil {
		info.LastGoodGeneration = digest.GetMaxGeneration()
	}
	return info, nil
}
