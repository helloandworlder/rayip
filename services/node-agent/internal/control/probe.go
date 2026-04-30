package control

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	controlv1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/control/v1"
	"github.com/rayip/rayip/services/node-agent/internal/config"
)

func probePublicReachability(cfg config.ProbeConfig, now func() time.Time) (*controlv1.NodeProbeObservation, error) {
	if now == nil {
		now = time.Now
	}
	candidates, candidateErr := scanCandidatePublicIPv4s()
	if cfg.PublicIPURL == "" {
		return &controlv1.NodeProbeObservation{
			ProbePort:          cfg.Port,
			ProbeProtocols:     append([]string(nil), cfg.Protocols...),
			CheckedAtUnixMilli: now().UTC().UnixMilli(),
			ScanHost:           cfg.ScanHost,
			CandidatePublicIps: candidates,
		}, candidateErr
	}
	if len(candidates) > 0 {
		candidates = reachableProbeCandidates(candidates, func(ip string) boundPublicIPResult {
			return probePublicIPFromSource(cfg, ip)
		})
	}
	return &controlv1.NodeProbeObservation{
		ProbePort:          cfg.Port,
		ProbeProtocols:     append([]string(nil), cfg.Protocols...),
		CheckedAtUnixMilli: now().UTC().UnixMilli(),
		ScanHost:           cfg.ScanHost,
		CandidatePublicIps: candidates,
	}, candidateErr
}

var errSourceIPUnavailable = errors.New("source ip unavailable")

type boundPublicIPResult struct {
	publicIP string
	err      error
}

func reachableProbeCandidates(candidates []string, probe func(string) boundPublicIPResult) []string {
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		result := probe(candidate)
		if result.err == nil && result.publicIP == candidate {
			out = append(out, candidate)
		}
	}
	return out
}

func probePublicIPFromSource(cfg config.ProbeConfig, sourceIP string) boundPublicIPResult {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	localIP := net.ParseIP(sourceIP)
	if localIP == nil {
		return boundPublicIPResult{err: errSourceIPUnavailable}
	}
	dialer := &net.Dialer{
		Timeout:   timeout,
		LocalAddr: &net.TCPAddr{IP: localIP},
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.PublicIPURL, nil)
	if err != nil {
		return boundPublicIPResult{err: err}
	}
	resp, err := client.Do(req)
	if err != nil {
		return boundPublicIPResult{err: fmt.Errorf("%w: %v", errSourceIPUnavailable, err)}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return boundPublicIPResult{err: fmt.Errorf("public ip probe returned HTTP %d", resp.StatusCode)}
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil {
		return boundPublicIPResult{err: err}
	}
	publicIP := strings.TrimSpace(string(payload))
	if net.ParseIP(publicIP) == nil {
		return boundPublicIPResult{err: fmt.Errorf("public ip probe returned invalid IP %q", publicIP)}
	}
	return boundPublicIPResult{publicIP: publicIP}
}

func scanCandidatePublicIPv4s() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	candidates := []string{}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip := extractIPv4(addr)
			if !isPublicIPv4(ip) {
				continue
			}
			value := ip.String()
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			candidates = append(candidates, value)
		}
	}
	return candidates, nil
}

func extractIPv4(addr net.Addr) net.IP {
	switch value := addr.(type) {
	case *net.IPNet:
		return value.IP.To4()
	case *net.IPAddr:
		return value.IP.To4()
	default:
		return nil
	}
}

func isPublicIPv4(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsMulticast() {
		return false
	}
	privateCIDRs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"100.64.0.0/10",
		"224.0.0.0/4",
	}
	for _, cidr := range privateCIDRs {
		_, block, _ := net.ParseCIDR(cidr)
		if block.Contains(ip) {
			return false
		}
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
