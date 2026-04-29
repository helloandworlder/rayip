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

type ResultStatus string

const (
	ResultSuccess   ResultStatus = "SUCCESS"
	ResultFailed    ResultStatus = "FAILED"
	ResultSkipped   ResultStatus = "SKIPPED"
	ResultDuplicate ResultStatus = "DUPLICATE"
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

type Command struct {
	CommandID         string
	NodeID            string
	Operation         Operation
	Account           Account
	DesiredGeneration uint64
	DeadlineUnixMS    int64
}

type Result struct {
	CommandID         string
	Status            ResultStatus
	ErrorCode         string
	ErrorMessage      string
	AppliedGeneration uint64
	Usage             Usage
	Digest            Digest
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
