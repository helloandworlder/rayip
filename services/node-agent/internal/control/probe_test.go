package control

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rayip/rayip/services/node-agent/internal/config"
)

func TestProbePublicReachabilityDoesNotTrustDefaultEgressIP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("198.51.100.42\n"))
	}))
	defer server.Close()

	checkedAt := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	probe, err := probePublicReachability(config.ProbeConfig{
		PublicIPURL: server.URL,
		ScanHost:    "node-agent.example.net",
		Port:        18080,
		Protocols:   []string{"SOCKS5", "HTTP"},
		Timeout:     time.Second,
	}, func() time.Time { return checkedAt })
	if err != nil {
		t.Fatalf("probePublicReachability() error = %v", err)
	}
	if probe.PublicIp != "" || probe.ScanHost != "node-agent.example.net" || probe.ProbePort != 18080 || probe.CheckedAtUnixMilli != checkedAt.UnixMilli() {
		t.Fatalf("probe = %#v", probe)
	}
	if len(probe.ProbeProtocols) != 2 || probe.ProbeProtocols[0] != "SOCKS5" || probe.ProbeProtocols[1] != "HTTP" {
		t.Fatalf("probe protocols = %#v", probe.ProbeProtocols)
	}
}

func TestReachableProbeCandidatesKeepsOnlyIPsThatEgressAsThemselves(t *testing.T) {
	candidates := []string{"204.42.251.2", "204.42.251.3", "204.42.251.236"}
	results := map[string]boundPublicIPResult{
		"204.42.251.2":   {publicIP: "204.42.251.2"},
		"204.42.251.3":   {publicIP: "203.0.113.99"},
		"204.42.251.236": {err: errSourceIPUnavailable},
	}

	got := reachableProbeCandidates(candidates, func(ip string) boundPublicIPResult {
		return results[ip]
	})

	if len(got) != 1 || got[0] != "204.42.251.2" {
		t.Fatalf("reachableProbeCandidates() = %#v, want only 204.42.251.2", got)
	}
}

func TestIsPublicIPv4FiltersPrivateAndCGNAT(t *testing.T) {
	cases := map[string]bool{
		"10.0.0.1":       false,
		"172.16.4.1":     false,
		"192.168.1.10":   false,
		"100.64.1.10":    false,
		"127.0.0.1":      false,
		"169.254.10.1":   false,
		"0.0.0.0":        false,
		"8.8.8.8":        true,
		"204.42.251.236": true,
	}
	for raw, want := range cases {
		if got := isPublicIPv4(net.ParseIP(raw)); got != want {
			t.Fatalf("isPublicIPv4(%s) = %v, want %v", raw, got, want)
		}
	}
}
