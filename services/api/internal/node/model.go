package node

import "time"

type Status string

const (
	StatusOnline  Status = "ONLINE"
	StatusOffline Status = "OFFLINE"
)

type LeaseInput struct {
	NodeID             string
	NodeCode           string
	SessionID          string
	APIInstanceID      string
	BundleVersion      string
	AgentVersion       string
	XrayVersion        string
	Capabilities       []string
	PublicIP           string
	CandidatePublicIPs []string
	ScanHost           string
	ProbePort          uint32
	ProbeProtocols     []string
	ProbeCheckedAt     time.Time
	Sequence           uint64
	LeaseTTLSeconds    int
}

type NodeRecord struct {
	ID                 string
	Code               string
	BundleVersion      string
	AgentVersion       string
	XrayVersion        string
	Capabilities       []string
	PublicIP           string
	CandidatePublicIPs []string
	ScanHost           string
	ProbePort          uint32
	ProbeProtocols     []string
	ProbeCheckedAt     time.Time
	LastScanStatus     string
	LastScanError      string
	LastScanLatency    time.Duration
	LastScanAt         time.Time
	LastOnlineAt       time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type LeaseSnapshot struct {
	NodeID             string    `json:"node_id"`
	NodeCode           string    `json:"node_code"`
	SessionID          string    `json:"session_id"`
	APIInstanceID      string    `json:"api_instance_id"`
	BundleVersion      string    `json:"bundle_version"`
	AgentVersion       string    `json:"agent_version"`
	XrayVersion        string    `json:"xray_version"`
	Capabilities       []string  `json:"capabilities"`
	PublicIP           string    `json:"public_ip"`
	CandidatePublicIPs []string  `json:"candidate_public_ips"`
	ScanHost           string    `json:"scan_host"`
	ProbePort          uint32    `json:"probe_port"`
	ProbeProtocols     []string  `json:"probe_protocols"`
	ProbeCheckedAt     time.Time `json:"probe_checked_at"`
	Sequence           uint64    `json:"sequence"`
	RenewedAt          time.Time `json:"renewed_at"`
	ExpiresAt          time.Time `json:"expires_at"`
	LeaseTTLSeconds    int       `json:"lease_ttl_seconds"`
}

type Summary struct {
	ID                 string    `json:"id"`
	Code               string    `json:"code"`
	Status             Status    `json:"status"`
	LastOnlineAt       time.Time `json:"last_online_at"`
	BundleVersion      string    `json:"bundle_version"`
	AgentVersion       string    `json:"agent_version"`
	XrayVersion        string    `json:"xray_version"`
	APIInstanceID      string    `json:"api_instance_id"`
	SessionID          string    `json:"session_id"`
	Capabilities       []string  `json:"capabilities"`
	PublicIP           string    `json:"public_ip"`
	CandidatePublicIPs []string  `json:"candidate_public_ips"`
	ScanHost           string    `json:"scan_host"`
	ProbePort          uint32    `json:"probe_port"`
	ProbeProtocols     []string  `json:"probe_protocols"`
	ProbeCheckedAt     time.Time `json:"probe_checked_at"`
	LastScanStatus     string    `json:"last_scan_status"`
	LastScanError      string    `json:"last_scan_error,omitempty"`
	LastScanLatencyMs  int64     `json:"last_scan_latency_ms"`
	LastScanAt         time.Time `json:"last_scan_at,omitempty"`
	LeaseExpiresAt     time.Time `json:"lease_expires_at,omitempty"`
}

type ScanResult struct {
	NodeID    string        `json:"node_id"`
	Target    string        `json:"target"`
	Status    string        `json:"status"`
	Error     string        `json:"error,omitempty"`
	Latency   time.Duration `json:"-"`
	LatencyMs int64         `json:"latency_ms"`
	ScannedAt time.Time     `json:"scanned_at"`
}
