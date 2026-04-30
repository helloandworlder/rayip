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
	Probe   ProbeConfig
}

type NodeConfig struct {
	Code            string
	EnrollmentToken string
}

type APIConfig struct {
	GRPCAddr string
}

type RuntimeConfig struct {
	AgentVersion   string
	BundleDir      string
	ManifestPath   string
	CoreMode       string
	XrayGRPCAddr   string
	XrayBinaryPath string
	XrayConfigPath string
	XrayAutoStart  bool
}

type LeaseConfig struct {
	Interval time.Duration
	TTL      time.Duration
}

type ProbeConfig struct {
	PublicIPURL string
	ScanHost    string
	Port        uint32
	Protocols   []string
	Timeout     time.Duration
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
	v.SetDefault("runtime.core_mode", "xray")
	v.SetDefault("runtime.xray_grpc_addr", "auto")
	v.SetDefault("runtime.xray_binary_path", "./third_party/xray-core/xray")
	v.SetDefault("runtime.xray_auto_start", true)
	v.SetDefault("lease.interval", "10s")
	v.SetDefault("lease.ttl", "45s")
	v.SetDefault("probe.public_ip_url", "https://api.ipify.org")
	v.SetDefault("probe.scan_host", "")
	v.SetDefault("probe.port", 18080)
	v.SetDefault("probe.protocols", "SOCKS5,HTTP")
	v.SetDefault("probe.timeout", "5s")

	bundleDir := v.GetString("runtime.bundle_dir")
	configPath := v.GetString("runtime.xray_config_path")
	if configPath == "" {
		configPath = filepath.Join(bundleDir, "xray-runtime.json")
	}
	return Config{
		Node: NodeConfig{
			Code:            v.GetString("node.code"),
			EnrollmentToken: v.GetString("node.enrollment_token"),
		},
		API: APIConfig{GRPCAddr: v.GetString("api.grpc_addr")},
		Runtime: RuntimeConfig{
			AgentVersion:   v.GetString("runtime.agent_version"),
			BundleDir:      bundleDir,
			ManifestPath:   filepath.Join(bundleDir, "runtime-manifest.json"),
			CoreMode:       v.GetString("runtime.core_mode"),
			XrayGRPCAddr:   v.GetString("runtime.xray_grpc_addr"),
			XrayBinaryPath: v.GetString("runtime.xray_binary_path"),
			XrayConfigPath: configPath,
			XrayAutoStart:  v.GetBool("runtime.xray_auto_start"),
		},
		Lease: LeaseConfig{
			Interval: v.GetDuration("lease.interval"),
			TTL:      v.GetDuration("lease.ttl"),
		},
		Probe: ProbeConfig{
			PublicIPURL: v.GetString("probe.public_ip_url"),
			ScanHost:    v.GetString("probe.scan_host"),
			Port:        uint32(v.GetUint("probe.port")),
			Protocols:   splitCSV(v.GetString("probe.protocols")),
			Timeout:     v.GetDuration("probe.timeout"),
		},
	}, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
