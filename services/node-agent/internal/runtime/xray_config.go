package runtime

import (
	"encoding/json"
	"net"
	"strconv"
)

type XrayRuntimeConfig struct {
	Log       map[string]string    `json:"log"`
	API       xrayAPIConfig        `json:"api"`
	Stats     map[string]any       `json:"stats"`
	Policy    map[string]any       `json:"policy"`
	Outbounds []xrayOutboundConfig `json:"outbounds"`
}

type xrayAPIConfig struct {
	Tag      string   `json:"tag"`
	Listen   string   `json:"listen"`
	Services []string `json:"services"`
}

type xrayOutboundConfig struct {
	Protocol string `json:"protocol"`
	Tag      string `json:"tag"`
}

func BuildXrayRuntimeConfig(apiAddr string) ([]byte, error) {
	host, port, err := net.SplitHostPort(apiAddr)
	if err != nil {
		return nil, err
	}
	if host == "" {
		host = "127.0.0.1"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return nil, err
	}
	cfg := XrayRuntimeConfig{
		Log: map[string]string{"loglevel": "warning"},
		API: xrayAPIConfig{
			Tag:    "rayip-api",
			Listen: net.JoinHostPort(host, port),
			Services: []string{
				"HandlerService",
				"StatsService",
				"RayIPRuntimeService",
				"ReflectionService",
			},
		},
		Stats:  map[string]any{},
		Policy: map[string]any{"system": map[string]bool{"statsInboundUplink": true, "statsInboundDownlink": true}},
		Outbounds: []xrayOutboundConfig{
			{Protocol: "freedom", Tag: "direct"},
			{Protocol: "blackhole", Tag: "blocked"},
		},
	}
	return json.MarshalIndent(cfg, "", "  ")
}
