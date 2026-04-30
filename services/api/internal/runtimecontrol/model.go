package runtimecontrol

import "time"

type Protocol string

const (
	ProtocolSOCKS5 Protocol = "SOCKS5"
	ProtocolHTTP   Protocol = "HTTP"
)

type ResourceKind string

const (
	ResourceKindProxyAccount ResourceKind = "PROXY_ACCOUNT"
)

type ChangeAction string

const (
	ChangeActionUpsert ChangeAction = "UPSERT"
	ChangeActionRemove ChangeAction = "REMOVE"
)

type ResourceInput struct {
	ProxyAccountID  string
	NodeID          string
	RuntimeEmail    string
	Protocol        Protocol
	ListenIP        string
	Port            uint32
	Username        string
	Password        string
	EgressLimitBPS  uint64
	IngressLimitBPS uint64
	MaxConnections  uint32
	Priority        uint32
	ExpiresAt       time.Time
}

type ResourceState struct {
	ResourceName    string
	ProxyAccountID  string
	NodeID          string
	Kind            ResourceKind
	RuntimeEmail    string
	Protocol        Protocol
	ListenIP        string
	Port            uint32
	Username        string
	Password        string
	EgressLimitBPS  uint64
	IngressLimitBPS uint64
	MaxConnections  uint32
	Priority        uint32
	ExpiresAt       time.Time
	DesiredRevision uint64
	Removed         bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ChangeLogEntry struct {
	ID           string
	NodeID       string
	Seq          uint64
	ResourceName string
	Action       ChangeAction
	Revision     uint64
	CreatedAt    time.Time
}

type OutboxEvent struct {
	ID           string
	Topic        string
	AggregateID  string
	AggregateKey string
	Payload      map[string]any
	PublishedAt  time.Time
	CreatedAt    time.Time
}

type JobStatus string

const (
	JobStatusPending   JobStatus = "PENDING"
	JobStatusSucceeded JobStatus = "SUCCEEDED"
	JobStatusFailed    JobStatus = "FAILED"
	JobStatusRetryable JobStatus = "RETRYABLE"
)

type JobResult struct {
	JobID            string
	NodeID           string
	Status           JobStatus
	BaseRevision     uint64
	TargetRevision   uint64
	AcceptedRevision uint64
	LastGoodRevision uint64
	ApplyID          string
	VersionInfo      string
	Nonce            string
	ErrorDetail      string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type MutationResult struct {
	State  ResourceState
	Change ChangeLogEntry
	Outbox OutboxEvent
}
