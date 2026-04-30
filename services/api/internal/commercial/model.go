package commercial

import "time"

type UserStatus string

const (
	UserStatusActive   UserStatus = "ACTIVE"
	UserStatusDisabled UserStatus = "DISABLED"
)

type SessionScope string

const (
	SessionScopeUser  SessionScope = "USER"
	SessionScopeAdmin SessionScope = "ADMIN"
)

type LedgerType string

const (
	LedgerTypeCreditRecharge LedgerType = "CREDIT_RECHARGE"
	LedgerTypeHold           LedgerType = "HOLD"
	LedgerTypeHoldRelease    LedgerType = "HOLD_RELEASE"
	LedgerTypeDebitPurchase  LedgerType = "DEBIT_PURCHASE"
	LedgerTypeCreditRefund   LedgerType = "CREDIT_REFUND"
)

type PaymentOrderStatus string

const (
	PaymentOrderStatusPending PaymentOrderStatus = "PENDING"
	PaymentOrderStatusPaid    PaymentOrderStatus = "PAID"
)

type InventoryStatus string

const (
	InventoryStatusAvailable InventoryStatus = "AVAILABLE"
	InventoryStatusReserved  InventoryStatus = "RESERVED"
	InventoryStatusSold      InventoryStatus = "SOLD"
	InventoryStatusDisabled  InventoryStatus = "DISABLED"
	InventoryStatusHold      InventoryStatus = "HOLD"
)

type ReservationStatus string

const (
	ReservationStatusActive    ReservationStatus = "ACTIVE"
	ReservationStatusConfirmed ReservationStatus = "CONFIRMED"
	ReservationStatusReleased  ReservationStatus = "RELEASED"
)

type OrderStatus string

const (
	OrderStatusCreated           OrderStatus = "CREATED"
	OrderStatusWalletHeld        OrderStatus = "WALLET_HELD"
	OrderStatusInventoryReserved OrderStatus = "INVENTORY_RESERVED"
	OrderStatusPendingRuntime    OrderStatus = "PENDING_RUNTIME"
	OrderStatusDelivered         OrderStatus = "DELIVERED"
	OrderStatusFulfillmentFailed OrderStatus = "FULFILLMENT_FAILED"
	OrderStatusDisabled          OrderStatus = "DISABLED"
	OrderStatusRuntimePending    OrderStatus = "RUNTIME_PENDING"
)

type ProxyLifecycleStatus string

const (
	ProxyLifecycleActive         ProxyLifecycleStatus = "ACTIVE"
	ProxyLifecycleExpiringSoon   ProxyLifecycleStatus = "EXPIRING_SOON"
	ProxyLifecycleExpired        ProxyLifecycleStatus = "EXPIRED"
	ProxyLifecycleDisabled       ProxyLifecycleStatus = "DISABLED"
	ProxyLifecycleRuntimePending ProxyLifecycleStatus = "RUNTIME_PENDING"
	ProxyLifecycleRuntimeFailed  ProxyLifecycleStatus = "RUNTIME_FAILED"
)

type FulfillmentJobStatus string

const (
	FulfillmentJobPending   FulfillmentJobStatus = "PENDING"
	FulfillmentJobSucceeded FulfillmentJobStatus = "SUCCEEDED"
	FulfillmentJobFailed    FulfillmentJobStatus = "FAILED"
)

type Protocol string

const (
	ProtocolSOCKS5 Protocol = "SOCKS5"
	ProtocolHTTP   Protocol = "HTTP"
)

type RuntimeApplyStatus string

const (
	RuntimeApplyStatusACK     RuntimeApplyStatus = "ACK"
	RuntimeApplyStatusNACK    RuntimeApplyStatus = "NACK"
	RuntimeApplyStatusPartial RuntimeApplyStatus = "PARTIAL"
	RuntimeApplyStatusFailed  RuntimeApplyStatus = "FAILED"
	RuntimeApplyStatusTimeout RuntimeApplyStatus = "TIMEOUT"
)

type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Status       UserStatus `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type AdminUser struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Session struct {
	ID        string       `json:"id"`
	SubjectID string       `json:"subject_id"`
	Scope     SessionScope `json:"scope"`
	ExpiresAt time.Time    `json:"expires_at"`
	CreatedAt time.Time    `json:"created_at"`
}

type Wallet struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	BalanceCents   int64     `json:"balance_cents"`
	HeldCents      int64     `json:"held_cents"`
	AvailableCents int64     `json:"available_cents"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type WalletLedger struct {
	ID             string     `json:"id"`
	WalletID       string     `json:"wallet_id"`
	UserID         string     `json:"user_id"`
	Type           LedgerType `json:"type"`
	AmountCents    int64      `json:"amount_cents"`
	BalanceAfter   int64      `json:"balance_after_cents"`
	HeldAfter      int64      `json:"held_after_cents"`
	ReferenceType  string     `json:"reference_type"`
	ReferenceID    string     `json:"reference_id"`
	IdempotencyKey string     `json:"idempotency_key"`
	CreatedAt      time.Time  `json:"created_at"`
}

type WalletHold struct {
	ID          string    `json:"id"`
	WalletID    string    `json:"wallet_id"`
	UserID      string    `json:"user_id"`
	OrderID     string    `json:"order_id"`
	AmountCents int64     `json:"amount_cents"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PaymentOrder struct {
	ID              string             `json:"id"`
	UserID          string             `json:"user_id"`
	AmountCents     int64              `json:"amount_cents"`
	Status          PaymentOrderStatus `json:"status"`
	Provider        string             `json:"provider"`
	ProviderTradeNo string             `json:"provider_trade_no"`
	PaidAt          time.Time          `json:"paid_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type AuditLog struct {
	ID        string         `json:"id"`
	ActorID   string         `json:"actor_id"`
	ActorType string         `json:"actor_type"`
	Action    string         `json:"action"`
	TargetID  string         `json:"target_id"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

type Region struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type City struct {
	ID        string    `json:"id"`
	RegionID  string    `json:"region_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Line struct {
	ID        string    `json:"id"`
	RegionID  string    `json:"region_id"`
	CityID    string    `json:"city_id"`
	NodeID    string    `json:"node_id"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Product struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IPType    string    `json:"ip_type"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProductPrice struct {
	ID           string    `json:"id"`
	ProductID    string    `json:"product_id"`
	Protocol     Protocol  `json:"protocol"`
	DurationDays int       `json:"duration_days"`
	UnitCents    int64     `json:"unit_cents"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RatePolicy struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	EgressLimitBPS  uint64    `json:"egress_limit_bps"`
	IngressLimitBPS uint64    `json:"ingress_limit_bps"`
	MaxConnections  uint32    `json:"max_connections"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type NodeRuntimeStatus struct {
	NodeID            string    `json:"node_id"`
	LeaseOnline       bool      `json:"lease_online"`
	RuntimeVerdict    string    `json:"runtime_verdict"`
	Sellable          bool      `json:"sellable"`
	Capabilities      []string  `json:"capabilities"`
	UnsellableReasons []string  `json:"unsellable_reasons"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type NodeInventoryIP struct {
	ID              string          `json:"id"`
	LineID          string          `json:"line_id"`
	NodeID          string          `json:"node_id"`
	IP              string          `json:"ip"`
	Port            uint32          `json:"port"`
	Protocols       []Protocol      `json:"protocols"`
	Status          InventoryStatus `json:"status"`
	ManualHold      bool            `json:"manual_hold"`
	ComplianceHold  bool            `json:"compliance_hold"`
	SoldOrderID     string          `json:"sold_order_id,omitempty"`
	ReservedOrderID string          `json:"reserved_order_id,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type InventoryReservation struct {
	ID          string            `json:"id"`
	InventoryID string            `json:"inventory_id"`
	UserID      string            `json:"user_id"`
	OrderID     string            `json:"order_id"`
	Status      ReservationStatus `json:"status"`
	ExpiresAt   time.Time         `json:"expires_at"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type ProxyOrder struct {
	ID             string      `json:"id"`
	UserID         string      `json:"user_id"`
	ProductID      string      `json:"product_id"`
	InventoryID    string      `json:"inventory_id"`
	ReservationID  string      `json:"reservation_id"`
	WalletHoldID   string      `json:"wallet_hold_id"`
	ProxyAccountID string      `json:"proxy_account_id"`
	IdempotencyKey string      `json:"idempotency_key"`
	Protocol       Protocol    `json:"protocol"`
	DurationDays   int         `json:"duration_days"`
	Quantity       int         `json:"quantity"`
	AmountCents    int64       `json:"amount_cents"`
	Status         OrderStatus `json:"status"`
	FailureReason  string      `json:"failure_reason,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	DeliveredAt    time.Time   `json:"delivered_at,omitempty"`
	ExpiresAt      time.Time   `json:"expires_at,omitempty"`
}

type ProxyAccount struct {
	ID              string               `json:"id"`
	OrderID         string               `json:"order_id"`
	UserID          string               `json:"user_id"`
	NodeID          string               `json:"node_id"`
	InventoryID     string               `json:"inventory_id"`
	Protocol        Protocol             `json:"protocol"`
	ListenIP        string               `json:"listen_ip"`
	Port            uint32               `json:"port"`
	Username        string               `json:"username"`
	Password        string               `json:"password,omitempty"`
	ConnectionURI   string               `json:"connection_uri,omitempty"`
	RuntimeEmail    string               `json:"runtime_email"`
	EgressLimitBPS  uint64               `json:"egress_limit_bps"`
	IngressLimitBPS uint64               `json:"ingress_limit_bps"`
	MaxConnections  uint32               `json:"max_connections"`
	Status          string               `json:"status"`
	LifecycleStatus ProxyLifecycleStatus `json:"lifecycle_status"`
	ExpiresAt       time.Time            `json:"expires_at"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}

type FulfillmentJob struct {
	ID             string               `json:"id"`
	OrderID        string               `json:"order_id"`
	ProxyAccountID string               `json:"proxy_account_id"`
	Status         FulfillmentJobStatus `json:"status"`
	ErrorDetail    string               `json:"error_detail,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
	UpdatedAt      time.Time            `json:"updated_at"`
}

type FulfillmentAttempt struct {
	ID        string    `json:"id"`
	JobID     string    `json:"job_id"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type CatalogRegion struct {
	Region      Region        `json:"region"`
	Cities      []CatalogCity `json:"cities"`
	Available   int           `json:"available"`
	Disabled    bool          `json:"disabled"`
	DisabledWhy []string      `json:"disabled_reasons"`
}

type CatalogCity struct {
	City      City          `json:"city"`
	Lines     []CatalogLine `json:"lines"`
	Available int           `json:"available"`
}

type CatalogLine struct {
	Line         Line     `json:"line"`
	Available    int      `json:"available"`
	InventoryIDs []string `json:"inventory_ids"`
	Sellable     bool     `json:"sellable"`
	Reasons      []string `json:"reasons"`
}

type Catalog struct {
	Product        Product         `json:"product"`
	Prices         []ProductPrice  `json:"prices"`
	Regions        []CatalogRegion `json:"regions"`
	TotalAvailable int             `json:"total_available"`
}

type Quote struct {
	ProductID    string   `json:"product_id"`
	Protocol     Protocol `json:"protocol"`
	DurationDays int      `json:"duration_days"`
	Quantity     int      `json:"quantity"`
	UnitCents    int64    `json:"unit_cents"`
	TotalCents   int64    `json:"total_cents"`
}

type RuntimeProxyAccountInput struct {
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
	ExpiresAt       time.Time
}

type RuntimeMutationResult struct {
	ProxyAccountID string
	NodeID         string
}

type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreatePaymentOrderInput struct {
	AmountCents int64 `json:"amount_cents"`
}

type PaymentCallbackInput struct {
	PaymentOrderID  string `json:"payment_order_id"`
	ProviderTradeNo string `json:"provider_trade_no"`
	PaidAmountCents int64  `json:"paid_amount_cents"`
}

type QuoteInput struct {
	ProductID    string   `json:"product_id"`
	Protocol     Protocol `json:"protocol"`
	DurationDays int      `json:"duration_days"`
	Quantity     int      `json:"quantity"`
}

type CreateReservationInput struct {
	InventoryID string   `json:"inventory_id"`
	Protocol    Protocol `json:"protocol"`
	OrderID     string   `json:"order_id"`
	TTLSeconds  int      `json:"ttl_seconds"`
}

type CreateOrderInput struct {
	ProductID       string   `json:"product_id"`
	InventoryID     string   `json:"inventory_id"`
	Protocol        Protocol `json:"protocol"`
	DurationDays    int      `json:"duration_days"`
	Quantity        int      `json:"quantity"`
	IdempotencyKey  string   `json:"idempotency_key"`
	EgressLimitBPS  uint64   `json:"egress_limit_bps"`
	IngressLimitBPS uint64   `json:"ingress_limit_bps"`
	MaxConnections  uint32   `json:"max_connections"`
}

type RuntimeApplySettlementInput struct {
	ProxyAccountID string
	Status         RuntimeApplyStatus
	ErrorDetail    string
	AppliedAt      time.Time
}

type UpsertProductInput struct {
	ID      string                    `json:"id"`
	Name    string                    `json:"name"`
	IPType  string                    `json:"ip_type"`
	Enabled bool                      `json:"enabled"`
	Prices  []UpsertProductPriceInput `json:"prices"`
}

type UpsertProductPriceInput struct {
	ID           string   `json:"id"`
	ProductID    string   `json:"product_id"`
	Protocol     Protocol `json:"protocol"`
	DurationDays int      `json:"duration_days"`
	UnitCents    int64    `json:"unit_cents"`
}

type RenewProxyInput struct {
	DurationDays   int    `json:"duration_days"`
	IdempotencyKey string `json:"idempotency_key"`
}

type UpsertLineInput struct {
	ID       string `json:"id"`
	RegionID string `json:"region_id"`
	CityID   string `json:"city_id"`
	NodeID   string `json:"node_id"`
	Name     string `json:"name"`
	Enabled  bool   `json:"enabled"`
}

type UpsertInventoryInput struct {
	ID             string          `json:"id"`
	LineID         string          `json:"line_id"`
	NodeID         string          `json:"node_id"`
	IP             string          `json:"ip"`
	Port           uint32          `json:"port"`
	Protocols      []Protocol      `json:"protocols"`
	Status         InventoryStatus `json:"status"`
	ManualHold     bool            `json:"manual_hold"`
	ComplianceHold bool            `json:"compliance_hold"`
}

type LedgerFilter struct {
	UserID string
}

type OrderFilter struct {
	UserID string
}
