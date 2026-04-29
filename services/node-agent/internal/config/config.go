package config

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Node    NodeConfig
	API     APIConfig
	Runtime RuntimeConfig
	Lease   LeaseConfig
}

type NodeConfig struct {
	Code            string
	EnrollmentToken string
}

type APIConfig struct {
	GRPCAddr string
}

type RuntimeConfig struct {
	AgentVersion string
	BundleDir    string
	ManifestPath string
}

type LeaseConfig struct {
	Interval time.Duration
	TTL      time.Duration
}

func Load() (Config, error) {
	v := viper.New()
	v.SetEnvPrefix("RAYIP_AGENT")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	_ = v.BindEnv("node.enrollment_token", "RAYIP_AGENT_ENROLLMENT_TOKEN")

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "local-node"
	}

	v.SetDefault("node.code", hostname)
	v.SetDefault("node.enrollment_token", "dev-enrollment-token")
	v.SetDefault("api.grpc_addr", "127.0.0.1:9090")
	v.SetDefault("runtime.agent_version", "dev-agent")
	v.SetDefault("runtime.bundle_dir", "/opt/rayip/runtime")
	v.SetDefault("lease.interval", "10s")
	v.SetDefault("lease.ttl", "45s")

	bundleDir := v.GetString("runtime.bundle_dir")
	return Config{
		Node: NodeConfig{
			Code:            v.GetString("node.code"),
			EnrollmentToken: v.GetString("node.enrollment_token"),
		},
		API: APIConfig{GRPCAddr: v.GetString("api.grpc_addr")},
		Runtime: RuntimeConfig{
			AgentVersion: v.GetString("runtime.agent_version"),
			BundleDir:    bundleDir,
			ManifestPath: filepath.Join(bundleDir, "runtime-manifest.json"),
		},
		Lease: LeaseConfig{
			Interval: v.GetDuration("lease.interval"),
			TTL:      v.GetDuration("lease.ttl"),
		},
	}, nil
}
