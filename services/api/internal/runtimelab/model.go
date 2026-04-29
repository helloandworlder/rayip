package runtimelab

import "time"

type Protocol string

const (
	ProtocolSOCKS5 Protocol = "SOCKS5"
	ProtocolHTTP   Protocol = "HTTP"
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
	OperationDisable      Operation = "DISABLE"
	OperationUpdatePolicy Operation = "UPDATE_POLICY"
	OperationGetUsage     Operation = "GET_USAGE"
	OperationGetDigest    Operation = "GET_DIGEST"
	OperationProbe        Operation = "PROBE"
)

type ApplyStatus string

const (
	ApplyStatusSuccess   ApplyStatus = "SUCCESS"
	ApplyStatusFailed    ApplyStatus = "FAILED"
	ApplyStatusSkipped   ApplyStatus = "SKIPPED"
	ApplyStatusDuplicate ApplyStatus = "DUPLICATE"
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

type RuntimeCommand struct {
	CommandID         string    `json:"command_id"`
	NodeID            string    `json:"node_id"`
	Operation         Operation `json:"operation"`
	Account           Account   `json:"account"`
	DesiredGeneration uint64    `json:"desired_generation"`
	DeadlineUnixMS    int64     `json:"deadline_unix_ms"`
}

type ApplyResult struct {
	CommandID         string      `json:"command_id"`
	ProxyAccountID    string      `json:"proxy_account_id,omitempty"`
	NodeID            string      `json:"node_id,omitempty"`
	Operation         Operation   `json:"operation,omitempty"`
	Status            ApplyStatus `json:"status"`
	ErrorCode         string      `json:"error_code,omitempty"`
	ErrorMessage      string      `json:"error_message,omitempty"`
	AppliedGeneration uint64      `json:"applied_generation"`
	Usage             Usage       `json:"usage,omitempty"`
	Digest            Digest      `json:"digest,omitempty"`
	CreatedAt         time.Time   `json:"created_at,omitempty"`
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
