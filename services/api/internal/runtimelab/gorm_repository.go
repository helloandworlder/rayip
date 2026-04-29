package runtimelab

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) UpsertAccount(ctx context.Context, account Account) (Account, error) {
	model := accountModelFrom(account)
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "proxy_account_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"node_id", "runtime_email", "protocol", "listen_ip", "port", "username", "password",
			"expires_at", "egress_limit_bps", "ingress_limit_bps", "max_connections", "status",
			"policy_version", "desired_generation", "applied_generation", "updated_at",
		}),
	}).Create(&model).Error
	if err != nil {
		return Account{}, err
	}
	return model.toAccount(), nil
}

func (r *GormRepository) GetAccount(ctx context.Context, proxyAccountID string) (Account, bool, error) {
	var model accountModel
	err := r.db.WithContext(ctx).Where("proxy_account_id = ?", proxyAccountID).First(&model).Error
	if err == gorm.ErrRecordNotFound {
		return Account{}, false, nil
	}
	if err != nil {
		return Account{}, false, err
	}
	return model.toAccount(), true, nil
}

func (r *GormRepository) ListAccounts(ctx context.Context) ([]Account, error) {
	var models []accountModel
	if err := r.db.WithContext(ctx).Where("status <> ?", string(AccountStatusDeleted)).Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]Account, 0, len(models))
	for _, model := range models {
		items = append(items, model.toAccount())
	}
	return items, nil
}

func (r *GormRepository) SaveApplyResult(ctx context.Context, result ApplyResult) error {
	usageJSON, _ := json.Marshal(result.Usage)
	digestJSON, _ := json.Marshal(result.Digest)
	return r.db.WithContext(ctx).Create(&applyResultModel{
		CommandID:         result.CommandID,
		ProxyAccountID:    result.ProxyAccountID,
		NodeID:            result.NodeID,
		Operation:         string(result.Operation),
		Status:            string(result.Status),
		ErrorCode:         result.ErrorCode,
		ErrorMessage:      result.ErrorMessage,
		AppliedGeneration: result.AppliedGeneration,
		UsageJSON:         string(usageJSON),
		DigestJSON:        string(digestJSON),
		CreatedAt:         result.CreatedAt,
	}).Error
}

func (r *GormRepository) ListApplyResults(ctx context.Context, proxyAccountID string, limit int) ([]ApplyResult, error) {
	query := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if proxyAccountID != "" {
		query = query.Where("proxy_account_id = ?", proxyAccountID)
	}
	var models []applyResultModel
	if err := query.Find(&models).Error; err != nil {
		return nil, err
	}
	items := make([]ApplyResult, 0, len(models))
	for _, model := range models {
		items = append(items, model.toResult())
	}
	return items, nil
}

type accountModel struct {
	ProxyAccountID    string     `gorm:"column:proxy_account_id;type:uuid;primaryKey"`
	NodeID            string     `gorm:"column:node_id;type:uuid"`
	RuntimeEmail      string     `gorm:"column:runtime_email"`
	Protocol          string     `gorm:"column:protocol"`
	ListenIP          string     `gorm:"column:listen_ip"`
	Port              uint32     `gorm:"column:port"`
	Username          string     `gorm:"column:username"`
	Password          string     `gorm:"column:password"`
	ExpiresAt         *time.Time `gorm:"column:expires_at"`
	EgressLimitBPS    uint64     `gorm:"column:egress_limit_bps"`
	IngressLimitBPS   uint64     `gorm:"column:ingress_limit_bps"`
	MaxConnections    uint32     `gorm:"column:max_connections"`
	Status            string     `gorm:"column:status"`
	PolicyVersion     uint64     `gorm:"column:policy_version"`
	DesiredGeneration uint64     `gorm:"column:desired_generation"`
	AppliedGeneration uint64     `gorm:"column:applied_generation"`
	CreatedAt         time.Time  `gorm:"column:created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at"`
}

func (accountModel) TableName() string { return "runtime_lab_accounts" }

func accountModelFrom(account Account) accountModel {
	var expiresAt *time.Time
	if !account.ExpiresAt.IsZero() {
		expiresAt = &account.ExpiresAt
	}
	return accountModel{
		ProxyAccountID:    account.ProxyAccountID,
		NodeID:            account.NodeID,
		RuntimeEmail:      account.RuntimeEmail,
		Protocol:          string(account.Protocol),
		ListenIP:          account.ListenIP,
		Port:              account.Port,
		Username:          account.Username,
		Password:          account.Password,
		ExpiresAt:         expiresAt,
		EgressLimitBPS:    account.EgressLimitBPS,
		IngressLimitBPS:   account.IngressLimitBPS,
		MaxConnections:    account.MaxConnections,
		Status:            string(account.Status),
		PolicyVersion:     account.PolicyVersion,
		DesiredGeneration: account.DesiredGeneration,
		AppliedGeneration: account.AppliedGeneration,
		CreatedAt:         account.CreatedAt,
		UpdatedAt:         account.UpdatedAt,
	}
}

func (m accountModel) toAccount() Account {
	expiresAt := time.Time{}
	if m.ExpiresAt != nil {
		expiresAt = *m.ExpiresAt
	}
	return Account{
		ProxyAccountID:    m.ProxyAccountID,
		NodeID:            m.NodeID,
		RuntimeEmail:      m.RuntimeEmail,
		Protocol:          Protocol(m.Protocol),
		ListenIP:          m.ListenIP,
		Port:              m.Port,
		Username:          m.Username,
		Password:          m.Password,
		ExpiresAt:         expiresAt,
		EgressLimitBPS:    m.EgressLimitBPS,
		IngressLimitBPS:   m.IngressLimitBPS,
		MaxConnections:    m.MaxConnections,
		Status:            AccountStatus(m.Status),
		PolicyVersion:     m.PolicyVersion,
		DesiredGeneration: m.DesiredGeneration,
		AppliedGeneration: m.AppliedGeneration,
		CreatedAt:         m.CreatedAt,
		UpdatedAt:         m.UpdatedAt,
	}
}

type applyResultModel struct {
	CommandID         string    `gorm:"column:command_id;type:uuid;primaryKey"`
	ProxyAccountID    string    `gorm:"column:proxy_account_id"`
	NodeID            string    `gorm:"column:node_id"`
	Operation         string    `gorm:"column:operation"`
	Status            string    `gorm:"column:status"`
	ErrorCode         string    `gorm:"column:error_code"`
	ErrorMessage      string    `gorm:"column:error_message"`
	AppliedGeneration uint64    `gorm:"column:applied_generation"`
	UsageJSON         string    `gorm:"column:usage;type:jsonb"`
	DigestJSON        string    `gorm:"column:digest;type:jsonb"`
	CreatedAt         time.Time `gorm:"column:created_at"`
}

func (applyResultModel) TableName() string { return "runtime_lab_apply_results" }

func (m applyResultModel) toResult() ApplyResult {
	var usage Usage
	var digest Digest
	_ = json.Unmarshal([]byte(m.UsageJSON), &usage)
	_ = json.Unmarshal([]byte(m.DigestJSON), &digest)
	return ApplyResult{
		CommandID:         m.CommandID,
		ProxyAccountID:    m.ProxyAccountID,
		NodeID:            m.NodeID,
		Operation:         Operation(m.Operation),
		Status:            ApplyStatus(m.Status),
		ErrorCode:         m.ErrorCode,
		ErrorMessage:      m.ErrorMessage,
		AppliedGeneration: m.AppliedGeneration,
		Usage:             usage,
		Digest:            digest,
		CreatedAt:         m.CreatedAt,
	}
}
