package node

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"time"
)

type Repository interface {
	UpsertLease(ctx context.Context, input LeaseInput, now time.Time) (NodeRecord, error)
	Get(ctx context.Context, nodeID string) (NodeRecord, bool, error)
	List(ctx context.Context) ([]NodeRecord, error)
	SaveScanResult(ctx context.Context, nodeID string, result ScanResult) error
}

type LeaseStore interface {
	PutLease(ctx context.Context, lease LeaseSnapshot, ttl time.Duration) error
	GetLease(ctx context.Context, nodeID string) (LeaseSnapshot, bool, error)
}

type Service struct {
	repo   Repository
	leases LeaseStore
	now    func() time.Time
	dial   func(ctx context.Context, network, address string) (net.Conn, error)
}

func NewService(repo Repository, leases LeaseStore, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, leases: leases, now: now, dial: (&net.Dialer{Timeout: 3 * time.Second}).DialContext}
}

func (s *Service) RegisterLease(ctx context.Context, input LeaseInput) (Summary, error) {
	if input.NodeCode == "" {
		return Summary{}, errors.New("node code is required")
	}
	if input.SessionID == "" {
		return Summary{}, errors.New("session id is required")
	}
	if input.LeaseTTLSeconds <= 0 {
		input.LeaseTTLSeconds = 45
	}

	now := s.now().UTC()
	record, err := s.repo.UpsertLease(ctx, input, now)
	if err != nil {
		return Summary{}, err
	}

	lease := LeaseSnapshot{
		NodeID:             record.ID,
		NodeCode:           record.Code,
		SessionID:          input.SessionID,
		APIInstanceID:      input.APIInstanceID,
		BundleVersion:      input.BundleVersion,
		AgentVersion:       input.AgentVersion,
		XrayVersion:        input.XrayVersion,
		Capabilities:       append([]string(nil), input.Capabilities...),
		PublicIP:           input.PublicIP,
		CandidatePublicIPs: append([]string(nil), input.CandidatePublicIPs...),
		ScanHost:           input.ScanHost,
		ProbePort:          input.ProbePort,
		ProbeProtocols:     append([]string(nil), input.ProbeProtocols...),
		ProbeCheckedAt:     input.ProbeCheckedAt,
		Sequence:           input.Sequence,
		RenewedAt:          now,
		ExpiresAt:          now.Add(time.Duration(input.LeaseTTLSeconds) * time.Second),
		LeaseTTLSeconds:    input.LeaseTTLSeconds,
	}
	if err := s.leases.PutLease(ctx, lease, time.Duration(input.LeaseTTLSeconds)*time.Second); err != nil {
		return Summary{}, err
	}

	return summaryFrom(record, lease, now), nil
}

func (s *Service) ScanNode(ctx context.Context, nodeID string) (ScanResult, error) {
	record, ok, err := s.repo.Get(ctx, nodeID)
	if err != nil {
		return ScanResult{}, err
	}
	if !ok {
		return ScanResult{}, errors.New("node not found")
	}
	targetHosts := scanTargets(record)
	if len(targetHosts) == 0 || record.ProbePort == 0 {
		result := ScanResult{
			NodeID:     record.ID,
			Target:     fmt.Sprintf("%s:%d", firstNonEmpty(targetHosts...), record.ProbePort),
			Status:     "FAILED",
			ReasonCode: ScanReasonNoCandidatePublicIP,
			Error:      "node has no candidate public ip or probe port",
			ScannedAt:  s.now().UTC(),
		}
		_ = s.repo.SaveScanResult(ctx, record.ID, result)
		return result, nil
	}
	if result, blocked := blockedCandidateResult(record, s.now().UTC()); blocked {
		_ = s.repo.SaveScanResult(ctx, record.ID, result)
		return result, nil
	}
	start := s.now().UTC()
	result := ScanResult{NodeID: record.ID, Status: "UNREACHABLE", ScannedAt: start}
	errorsByTarget := []string{}
	for _, targetHost := range targetHosts {
		target := net.JoinHostPort(targetHost, fmt.Sprintf("%d", record.ProbePort))
		result.Target = target
		conn, err := s.dial(ctx, "tcp", target)
		if err == nil {
			result.Status = "REACHABLE"
			result.Error = ""
			result.Latency = s.now().UTC().Sub(start)
			result.LatencyMs = result.Latency.Milliseconds()
			_ = conn.Close()
			break
		}
		errorsByTarget = append(errorsByTarget, target+": "+err.Error())
	}
	if result.Status != "REACHABLE" {
		result.ReasonCode = ScanReasonIngressUnreachable
		result.Error = strings.Join(errorsByTarget, "; ")
		result.Latency = s.now().UTC().Sub(start)
		result.LatencyMs = result.Latency.Milliseconds()
	}
	if saveErr := s.repo.SaveScanResult(ctx, record.ID, result); saveErr != nil {
		return result, saveErr
	}
	return result, nil
}

func (s *Service) ListNodes(ctx context.Context) ([]Summary, error) {
	records, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	summaries := make([]Summary, 0, len(records))
	for _, record := range records {
		lease, ok, err := s.leases.GetLease(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		if !ok {
			lease = LeaseSnapshot{
				NodeID:             record.ID,
				NodeCode:           record.Code,
				BundleVersion:      record.BundleVersion,
				AgentVersion:       record.AgentVersion,
				XrayVersion:        record.XrayVersion,
				Capabilities:       record.Capabilities,
				PublicIP:           record.PublicIP,
				CandidatePublicIPs: record.CandidatePublicIPs,
				ScanHost:           record.ScanHost,
				ProbePort:          record.ProbePort,
				ProbeProtocols:     record.ProbeProtocols,
				ProbeCheckedAt:     record.ProbeCheckedAt,
			}
		}
		summaries = append(summaries, summaryFrom(record, lease, now))
	}
	return summaries, nil
}

func summaryFrom(record NodeRecord, lease LeaseSnapshot, now time.Time) Summary {
	status := StatusOffline
	if !lease.ExpiresAt.IsZero() && lease.ExpiresAt.After(now) {
		status = StatusOnline
	}
	return Summary{
		ID:                 record.ID,
		Code:               record.Code,
		Status:             status,
		LastOnlineAt:       record.LastOnlineAt,
		BundleVersion:      firstNonEmpty(lease.BundleVersion, record.BundleVersion),
		AgentVersion:       firstNonEmpty(lease.AgentVersion, record.AgentVersion),
		XrayVersion:        firstNonEmpty(lease.XrayVersion, record.XrayVersion),
		APIInstanceID:      lease.APIInstanceID,
		SessionID:          lease.SessionID,
		Capabilities:       append([]string(nil), firstNonNil(lease.Capabilities, record.Capabilities)...),
		PublicIP:           firstNonEmpty(lease.PublicIP, record.PublicIP),
		CandidatePublicIPs: append([]string(nil), firstNonNil(lease.CandidatePublicIPs, record.CandidatePublicIPs)...),
		ScanHost:           firstNonEmpty(lease.ScanHost, record.ScanHost),
		ProbePort:          firstNonZero(lease.ProbePort, record.ProbePort),
		ProbeProtocols:     append([]string(nil), firstNonNil(lease.ProbeProtocols, record.ProbeProtocols)...),
		ProbeCheckedAt:     firstNonZeroTime(lease.ProbeCheckedAt, record.ProbeCheckedAt),
		LastScanStatus:     record.LastScanStatus,
		LastScanError:      record.LastScanError,
		LastScanReasonCode: string(record.LastScanReasonCode),
		LastScanLatencyMs:  record.LastScanLatency.Milliseconds(),
		LastScanAt:         record.LastScanAt,
		LeaseExpiresAt:     lease.ExpiresAt,
	}
}

func (s *Service) SetDialerForTest(dial func(ctx context.Context, network, address string) (net.Conn, error)) {
	s.dial = dial
}

func scanTargets(record NodeRecord) []string {
	if len(record.CandidatePublicIPs) > 0 {
		return append([]string(nil), record.CandidatePublicIPs...)
	}
	if record.ScanHost != "" {
		return []string{record.ScanHost}
	}
	if record.PublicIP != "" {
		return []string{record.PublicIP}
	}
	return nil
}

func blockedCandidateResult(record NodeRecord, scannedAt time.Time) (ScanResult, bool) {
	targetHosts := append([]string(nil), record.CandidatePublicIPs...)
	if len(targetHosts) == 0 && record.PublicIP != "" && record.ScanHost == "" {
		targetHosts = []string{record.PublicIP}
	}
	for _, host := range targetHosts {
		reason := classifyBlockedIP(host)
		if reason == "" {
			continue
		}
		return ScanResult{
			NodeID:     record.ID,
			Target:     net.JoinHostPort(host, fmt.Sprintf("%d", record.ProbePort)),
			Status:     "FAILED",
			ReasonCode: reason,
			Error:      fmt.Sprintf("candidate public ip %s is %s", host, reason),
			ScannedAt:  scannedAt,
		}, true
	}
	return ScanResult{}, false
}

func classifyBlockedIP(host string) ScanReasonCode {
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return ""
	}
	if addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() || addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return ScanReasonPrivateIP
	}
	cgnat := netip.MustParsePrefix("100.64.0.0/10")
	if addr.Is4() && cgnat.Contains(addr) {
		return ScanReasonCGNAT
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonNil[T any](values ...[]T) []T {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func firstNonZero(values ...uint32) uint32 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func firstNonZeroTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}
