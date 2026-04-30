package runtimelab

import "time"

type Protocol string

const (
	ProtocolSOCKS5 Protocol = "SOCKS5"
	ProtocolHTTP   Protocol = "HTTP"
	ProtocolMixed  Protocol = "MIXED"
)

type AccountStatus string

const (
	AccountStatusEnabled  AccountStatus = "ENABLED"
	AccountStatusDisabled AccountStatus = "DISABLED"
	AccountStatusDeleted  AccountStatus = "DELETED"
)

type Operation string

const (
	OperationUpsert       Operation = "UPSERT"
	OperationDelete       Operation = "DELETE"
	OperationUpdatePolicy Operation = "UPDATE_POLICY"
	OperationGetUsage     Operation = "GET_USAGE"
	OperationGetDigest    Operation = "GET_DIGEST"
	OperationProbe        Operation = "PROBE"
)

type ApplyStatus string

const (
	ApplyStatusACK       ApplyStatus = "ACK"
	ApplyStatusNACK      ApplyStatus = "NACK"
	ApplyStatusPartial   ApplyStatus = "PARTIAL"
	ApplyStatusFailed    ApplyStatus = "FAILED"
	ApplyStatusDuplicate ApplyStatus = "DUPLICATE"
)

type ApplyMode string

const (
	ApplyModeDelta    ApplyMode = "DELTA"
	ApplyModeSnapshot ApplyMode = "SNAPSHOT"
)

type ResourceKind string

const (
	ResourceKindProxyAccount ResourceKind = "PROXY_ACCOUNT"
)

type Account struct {
	ProxyAccountID    string        `json:"proxy_account_id"`
	NodeID            string        `json:"node_id"`
	RuntimeEmail      string        `json:"runtime_email"`
	Protocol          Protocol      `json:"protocol"`
	ListenIP          string        `json:"listen_ip"`
	Port              uint32        `json:"port"`
	Username          string        `json:"username"`
	Password          string        `json:"password"`
	ExpiresAt         time.Time     `json:"expires_at,omitempty"`
	EgressLimitBPS    uint64        `json:"egress_limit_bps"`
	IngressLimitBPS   uint64        `json:"ingress_limit_bps"`
	MaxConnections    uint32        `json:"max_connections"`
	Status            AccountStatus `json:"status"`
	PolicyVersion     uint64        `json:"policy_version"`
	DesiredGeneration uint64        `json:"desired_generation"`
	AppliedGeneration uint64        `json:"applied_generation"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type CreateAccountInput struct {
	NodeID            string    `json:"node_id"`
	Protocol          Protocol  `json:"protocol"`
	ListenIP          string    `json:"listen_ip"`
	Port              uint32    `json:"port"`
	Username          string    `json:"username"`
	Password          string    `json:"password"`
	ExpiresAt         time.Time `json:"expires_at,omitempty"`
	EgressLimitBPS    uint64    `json:"egress_limit_bps"`
	IngressLimitBPS   uint64    `json:"ingress_limit_bps"`
	MaxConnections    uint32    `json:"max_connections"`
	DesiredGeneration uint64    `json:"desired_generation"`
}

type PolicyInput struct {
	EgressLimitBPS    uint64 `json:"egress_limit_bps"`
	IngressLimitBPS   uint64 `json:"ingress_limit_bps"`
	MaxConnections    uint32 `json:"max_connections"`
	DesiredGeneration uint64 `json:"desired_generation"`
}

type RuntimeApply struct {
	ApplyID              string            `json:"apply_id"`
	NodeID               string            `json:"node_id"`
	Mode                 ApplyMode         `json:"mode"`
	VersionInfo          string            `json:"version_info"`
	Nonce                string            `json:"nonce"`
	BaseRevision         uint64            `json:"base_revision"`
	TargetRevision       uint64            `json:"target_revision"`
	DeadlineUnixMS       int64             `json:"deadline_unix_ms"`
	Resources            []RuntimeResource `json:"resources"`
	RemovedResourceNames []string          `json:"removed_resource_names"`
	QueryOperation       Operation         `json:"query_operation,omitempty"`
	QueryResourceName    string            `json:"query_resource_name,omitempty"`
}

type RuntimeResource struct {
	Name              string       `json:"name"`
	Kind              ResourceKind `json:"kind"`
	ResourceVersion   uint64       `json:"resource_version"`
	RuntimeEmail      string       `json:"runtime_email"`
	Protocol          Protocol     `json:"protocol"`
	ListenIP          string       `json:"listen_ip"`
	Port              uint32       `json:"port"`
	Username          string       `json:"username"`
	Password          string       `json:"password"`
	EgressLimitBPS    uint64       `json:"egress_limit_bps"`
	IngressLimitBPS   uint64       `json:"ingress_limit_bps"`
	MaxConnections    uint32       `json:"max_connections"`
	Priority          uint32       `json:"priority"`
	AbuseReportPolicy string       `json:"abuse_report_policy"`
	ExpiresAtUnixMS   int64        `json:"expires_at_unix_ms"`
}

type ApplyResult struct {
	ApplyID          string      `json:"apply_id"`
	ProxyAccountID   string      `json:"proxy_account_id,omitempty"`
	NodeID           string      `json:"node_id,omitempty"`
	Operation        Operation   `json:"operation,omitempty"`
	Status           ApplyStatus `json:"status"`
	VersionInfo      string      `json:"version_info,omitempty"`
	Nonce            string      `json:"nonce,omitempty"`
	AppliedRevision  uint64      `json:"applied_revision"`
	LastGoodRevision uint64      `json:"last_good_revision"`
	ErrorDetail      string      `json:"error_detail,omitempty"`
	Usage            Usage       `json:"usage,omitempty"`
	Digest           Digest      `json:"digest,omitempty"`
	CreatedAt        time.Time   `json:"created_at,omitempty"`
}

type Usage struct {
	ProxyAccountID    string `json:"proxy_account_id,omitempty"`
	RuntimeEmail      string `json:"runtime_email,omitempty"`
	RxBytes           uint64 `json:"rx_bytes"`
	TxBytes           uint64 `json:"tx_bytes"`
	ActiveConnections uint64 `json:"active_connections"`
	RxBytesPerSecond  uint64 `json:"rx_bytes_per_second"`
	TxBytesPerSecond  uint64 `json:"tx_bytes_per_second"`
}

type Digest struct {
	AccountCount  uint64 `json:"account_count"`
	EnabledCount  uint64 `json:"enabled_count"`
	DisabledCount uint64 `json:"disabled_count"`
	MaxGeneration uint64 `json:"max_generation"`
	Hash          string `json:"hash"`
}
