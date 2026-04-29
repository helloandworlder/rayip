package node

import "time"

type Status string

const (
	StatusOnline  Status = "ONLINE"
	StatusOffline Status = "OFFLINE"
)

type LeaseInput struct {
	NodeID          string
	NodeCode        string
	SessionID       string
	APIInstanceID   string
	BundleVersion   string
	AgentVersion    string
	XrayVersion     string
	Capabilities    []string
	Sequence        uint64
	LeaseTTLSeconds int
}

type NodeRecord struct {
	ID            string
	Code          string
	BundleVersion string
	AgentVersion  string
	XrayVersion   string
	Capabilities  []string
	LastOnlineAt  time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type LeaseSnapshot struct {
	NodeID          string    `json:"node_id"`
	NodeCode        string    `json:"node_code"`
	SessionID       string    `json:"session_id"`
	APIInstanceID   string    `json:"api_instance_id"`
	BundleVersion   string    `json:"bundle_version"`
	AgentVersion    string    `json:"agent_version"`
	XrayVersion     string    `json:"xray_version"`
	Capabilities    []string  `json:"capabilities"`
	Sequence        uint64    `json:"sequence"`
	RenewedAt       time.Time `json:"renewed_at"`
	ExpiresAt       time.Time `json:"expires_at"`
	LeaseTTLSeconds int       `json:"lease_ttl_seconds"`
}

type Summary struct {
	ID             string    `json:"id"`
	Code           string    `json:"code"`
	Status         Status    `json:"status"`
	LastOnlineAt   time.Time `json:"last_online_at"`
	BundleVersion  string    `json:"bundle_version"`
	AgentVersion   string    `json:"agent_version"`
	XrayVersion    string    `json:"xray_version"`
	APIInstanceID  string    `json:"api_instance_id"`
	SessionID      string    `json:"session_id"`
	Capabilities   []string  `json:"capabilities"`
	LeaseExpiresAt time.Time `json:"lease_expires_at,omitempty"`
}
