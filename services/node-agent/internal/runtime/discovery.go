package runtime

import (
	"encoding/json"
	"errors"
	"os"
)

type DiscoveryConfig struct {
	AgentVersion string
	ManifestPath string
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
	return info, nil
}
