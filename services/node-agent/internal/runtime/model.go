package runtime

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

type Direction string

const (
	DirectionEgress  Direction = "EGRESS"
	DirectionIngress Direction = "INGRESS"
)

type AbuseAction string

const (
	AbuseActionReportOnly       AbuseAction = "REPORT_ONLY"
	AbuseActionDisableAndReport AbuseAction = "DISABLE_AND_REPORT"
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

type AckStatus string

const (
	AckStatusACK     AckStatus = "ACK"
	AckStatusNACK    AckStatus = "NACK"
	AckStatusPartial AckStatus = "PARTIAL"
)

type ResourceResultStatus string

const (
	ResourceResultApplied ResourceResultStatus = "APPLIED"
	ResourceResultRemoved ResourceResultStatus = "REMOVED"
	ResourceResultFailed  ResourceResultStatus = "FAILED"
)

type Account struct {
	ProxyAccountID    string
	RuntimeEmail      string
	Protocol          Protocol
	ListenIP          string
	Port              uint32
	Username          string
	Password          string
	ExpiresAtUnixMS   int64
	EgressLimitBPS    uint64
	IngressLimitBPS   uint64
	MaxConnections    uint32
	Status            AccountStatus
	Priority          uint32
	AbuseBytesPerMin  uint64
	AbuseAction       AbuseAction
	PolicyVersion     uint64
	DesiredGeneration uint64
}

type AbuseEvent struct {
	ProxyAccountID string
	RuntimeEmail   string
	Action         AbuseAction
	WindowBytes    uint64
	Threshold      uint64
}

type Apply struct {
	ApplyID              string     `json:"apply_id"`
	NodeID               string     `json:"node_id"`
	Mode                 ApplyMode  `json:"mode"`
	VersionInfo          string     `json:"version_info"`
	Nonce                string     `json:"nonce"`
	BaseRevision         uint64     `json:"base_revision"`
	TargetRevision       uint64     `json:"target_revision"`
	DeadlineUnixMS       int64      `json:"deadline_unix_ms"`
	Resources            []Resource `json:"resources"`
	RemovedResourceNames []string   `json:"removed_resource_names"`
}

type Resource struct {
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

type ApplyAck struct {
	ApplyID          string           `json:"apply_id"`
	NodeID           string           `json:"node_id"`
	VersionInfo      string           `json:"version_info,omitempty"`
	Nonce            string           `json:"nonce,omitempty"`
	Status           AckStatus        `json:"status"`
	AppliedRevision  uint64           `json:"applied_revision"`
	LastGoodRevision uint64           `json:"last_good_revision"`
	ResourceResults  []ResourceResult `json:"resource_results,omitempty"`
	Digest           Digest           `json:"digest,omitempty"`
	ErrorDetail      string           `json:"error_detail,omitempty"`
}

type ResourceResult struct {
	Name        string               `json:"name"`
	Status      ResourceResultStatus `json:"status"`
	ErrorDetail string               `json:"error_detail,omitempty"`
}

type Usage struct {
	ProxyAccountID    string
	RuntimeEmail      string
	RxBytes           uint64
	TxBytes           uint64
	ActiveConnections uint64
	RxBytesPerSecond  uint64
	TxBytesPerSecond  uint64
}

type Digest struct {
	AccountCount  uint64 `json:"account_count"`
	EnabledCount  uint64 `json:"enabled_count"`
	DisabledCount uint64 `json:"disabled_count"`
	MaxGeneration uint64 `json:"max_generation"`
	Hash          string `json:"hash"`
}
