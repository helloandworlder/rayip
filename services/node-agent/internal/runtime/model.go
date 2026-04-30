package runtime

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
	ApplyID              string
	NodeID               string
	Mode                 ApplyMode
	VersionInfo          string
	Nonce                string
	BaseRevision         uint64
	TargetRevision       uint64
	DeadlineUnixMS       int64
	Resources            []Resource
	RemovedResourceNames []string
}

type Resource struct {
	Name              string
	Kind              ResourceKind
	ResourceVersion   uint64
	RuntimeEmail      string
	Protocol          Protocol
	ListenIP          string
	Port              uint32
	Username          string
	Password          string
	EgressLimitBPS    uint64
	IngressLimitBPS   uint64
	MaxConnections    uint32
	Priority          uint32
	AbuseReportPolicy string
	ExpiresAtUnixMS   int64
}

type ApplyAck struct {
	ApplyID          string
	NodeID           string
	VersionInfo      string
	Nonce            string
	Status           AckStatus
	AppliedRevision  uint64
	LastGoodRevision uint64
	ResourceResults  []ResourceResult
	Digest           Digest
	ErrorDetail      string
}

type ResourceResult struct {
	Name        string
	Status      ResourceResultStatus
	ErrorDetail string
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
	AccountCount  uint64
	EnabledCount  uint64
	DisabledCount uint64
	MaxGeneration uint64
	Hash          string
}
