package runtimelab

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	UpsertAccount(ctx context.Context, account Account) (Account, error)
	GetAccount(ctx context.Context, proxyAccountID string) (Account, bool, error)
	ListAccounts(ctx context.Context) ([]Account, error)
	SaveApplyResult(ctx context.Context, result ApplyResult) error
	ListApplyResults(ctx context.Context, proxyAccountID string, limit int) ([]ApplyResult, error)
}

type Dispatcher interface {
	DispatchRuntimeCommand(ctx context.Context, cmd RuntimeCommand) (ApplyResult, error)
}

type Service struct {
	repo       Repository
	dispatcher Dispatcher
	now        func() time.Time
}

func NewService(repo Repository, dispatcher Dispatcher, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, dispatcher: dispatcher, now: now}
}

func (s *Service) CreateAccount(ctx context.Context, input CreateAccountInput) (Account, ApplyResult, error) {
	if input.NodeID == "" {
		return Account{}, ApplyResult{}, errors.New("node_id is required")
	}
	if input.Protocol != ProtocolSOCKS5 && input.Protocol != ProtocolHTTP {
		return Account{}, ApplyResult{}, errors.New("protocol must be SOCKS5 or HTTP")
	}
	if input.ListenIP == "" {
		input.ListenIP = "127.0.0.1"
	}
	if input.Port == 0 {
		return Account{}, ApplyResult{}, errors.New("port is required")
	}
	if input.Username == "" || input.Password == "" {
		return Account{}, ApplyResult{}, errors.New("username and password are required")
	}
	if input.DesiredGeneration == 0 {
		input.DesiredGeneration = 1
	}

	now := s.now().UTC()
	accountID := uuid.NewString()
	account := Account{
		ProxyAccountID:    accountID,
		NodeID:            input.NodeID,
		RuntimeEmail:      accountID,
		Protocol:          input.Protocol,
		ListenIP:          input.ListenIP,
		Port:              input.Port,
		Username:          input.Username,
		Password:          input.Password,
		ExpiresAt:         input.ExpiresAt,
		EgressLimitBPS:    input.EgressLimitBPS,
		IngressLimitBPS:   input.IngressLimitBPS,
		MaxConnections:    input.MaxConnections,
		Status:            AccountStatusEnabled,
		PolicyVersion:     1,
		DesiredGeneration: input.DesiredGeneration,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	account, err := s.repo.UpsertAccount(ctx, account)
	if err != nil {
		return Account{}, ApplyResult{}, err
	}
	result, err := s.dispatch(ctx, OperationUpsert, account, input.DesiredGeneration)
	if err != nil {
		return account, result, err
	}
	account.AppliedGeneration = result.AppliedGeneration
	account, _ = s.repo.UpsertAccount(ctx, account)
	return account, result, nil
}

func (s *Service) UpsertAccountPolicy(ctx context.Context, proxyAccountID string, input PolicyInput) (Account, ApplyResult, error) {
	account, ok, err := s.repo.GetAccount(ctx, proxyAccountID)
	if err != nil {
		return Account{}, ApplyResult{}, err
	}
	if !ok {
		return Account{}, ApplyResult{}, fmt.Errorf("account %s not found", proxyAccountID)
	}
	if input.DesiredGeneration == 0 {
		input.DesiredGeneration = account.DesiredGeneration + 1
	}
	if input.DesiredGeneration <= account.AppliedGeneration {
		result := ApplyResult{
			CommandID:         uuid.NewString(),
			ProxyAccountID:    account.ProxyAccountID,
			NodeID:            account.NodeID,
			Operation:         OperationUpdatePolicy,
			Status:            ApplyStatusDuplicate,
			AppliedGeneration: account.AppliedGeneration,
			CreatedAt:         s.now().UTC(),
		}
		_ = s.repo.SaveApplyResult(ctx, result)
		return account, result, nil
	}

	account.EgressLimitBPS = input.EgressLimitBPS
	account.IngressLimitBPS = input.IngressLimitBPS
	account.MaxConnections = input.MaxConnections
	account.PolicyVersion++
	account.DesiredGeneration = input.DesiredGeneration
	account.UpdatedAt = s.now().UTC()
	account, err = s.repo.UpsertAccount(ctx, account)
	if err != nil {
		return Account{}, ApplyResult{}, err
	}
	result, err := s.dispatch(ctx, OperationUpdatePolicy, account, input.DesiredGeneration)
	if err != nil {
		return account, result, err
	}
	account.AppliedGeneration = result.AppliedGeneration
	account, _ = s.repo.UpsertAccount(ctx, account)
	return account, result, nil
}

func (s *Service) DisableAccount(ctx context.Context, proxyAccountID string) (Account, ApplyResult, error) {
	account, ok, err := s.repo.GetAccount(ctx, proxyAccountID)
	if err != nil {
		return Account{}, ApplyResult{}, err
	}
	if !ok {
		return Account{}, ApplyResult{}, fmt.Errorf("account %s not found", proxyAccountID)
	}
	account.Status = AccountStatusDisabled
	account.DesiredGeneration++
	account.UpdatedAt = s.now().UTC()
	account, err = s.repo.UpsertAccount(ctx, account)
	if err != nil {
		return Account{}, ApplyResult{}, err
	}
	result, err := s.dispatch(ctx, OperationDisable, account, account.DesiredGeneration)
	if err == nil {
		account.AppliedGeneration = result.AppliedGeneration
		account, _ = s.repo.UpsertAccount(ctx, account)
	}
	return account, result, err
}

func (s *Service) DeleteAccount(ctx context.Context, proxyAccountID string) (ApplyResult, error) {
	account, ok, err := s.repo.GetAccount(ctx, proxyAccountID)
	if err != nil {
		return ApplyResult{}, err
	}
	if !ok {
		return ApplyResult{}, fmt.Errorf("account %s not found", proxyAccountID)
	}
	account.Status = AccountStatusDeleted
	account.DesiredGeneration++
	account.UpdatedAt = s.now().UTC()
	account, err = s.repo.UpsertAccount(ctx, account)
	if err != nil {
		return ApplyResult{}, err
	}
	return s.dispatch(ctx, OperationDelete, account, account.DesiredGeneration)
}

func (s *Service) GetUsage(ctx context.Context, proxyAccountID string) (ApplyResult, error) {
	account, ok, err := s.repo.GetAccount(ctx, proxyAccountID)
	if err != nil {
		return ApplyResult{}, err
	}
	if !ok {
		return ApplyResult{}, fmt.Errorf("account %s not found", proxyAccountID)
	}
	return s.dispatch(ctx, OperationGetUsage, account, account.DesiredGeneration)
}

func (s *Service) ProbeAccount(ctx context.Context, proxyAccountID string) (ApplyResult, error) {
	account, ok, err := s.repo.GetAccount(ctx, proxyAccountID)
	if err != nil {
		return ApplyResult{}, err
	}
	if !ok {
		return ApplyResult{}, fmt.Errorf("account %s not found", proxyAccountID)
	}
	return s.dispatch(ctx, OperationProbe, account, account.DesiredGeneration)
}

func (s *Service) GetDigest(ctx context.Context, nodeID string) (ApplyResult, error) {
	if nodeID == "" {
		return ApplyResult{}, errors.New("node_id is required")
	}
	return s.dispatch(ctx, OperationGetDigest, Account{NodeID: nodeID}, 0)
}

func (s *Service) ListAccounts(ctx context.Context) ([]Account, error) {
	return s.repo.ListAccounts(ctx)
}

func (s *Service) ListApplyResults(ctx context.Context, proxyAccountID string, limit int) ([]ApplyResult, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.repo.ListApplyResults(ctx, proxyAccountID, limit)
}

func (s *Service) dispatch(ctx context.Context, operation Operation, account Account, generation uint64) (ApplyResult, error) {
	command := RuntimeCommand{
		CommandID:         uuid.NewString(),
		NodeID:            account.NodeID,
		Operation:         operation,
		Account:           account,
		DesiredGeneration: generation,
		DeadlineUnixMS:    s.now().Add(8 * time.Second).UnixMilli(),
	}
	result, err := s.dispatcher.DispatchRuntimeCommand(ctx, command)
	if result.CommandID == "" {
		result.CommandID = command.CommandID
	}
	result.ProxyAccountID = account.ProxyAccountID
	result.NodeID = account.NodeID
	result.Operation = operation
	result.CreatedAt = s.now().UTC()
	if err != nil {
		result.Status = ApplyStatusFailed
		result.ErrorCode = "DISPATCH_FAILED"
		result.ErrorMessage = err.Error()
	}
	if saveErr := s.repo.SaveApplyResult(ctx, result); saveErr != nil && err == nil {
		err = saveErr
	}
	return result, err
}
