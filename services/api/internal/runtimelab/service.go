package runtimelab

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	UpsertAccount(ctx context.Context, account Account) (Account, error)
	GetAccount(ctx context.Context, proxyAccountID string) (Account, bool, error)
	ListAccounts(ctx context.Context) ([]Account, error)
	SaveApplyResult(ctx context.Context, result ApplyResult) error
	ListApplyResults(ctx context.Context, proxyAccountID string, limit int) ([]ApplyResult, error)
	LatestUsage(ctx context.Context, proxyAccountID string) (Usage, bool, error)
	LatestDigest(ctx context.Context, nodeID string) (Digest, bool, error)
	LatestNodeRevision(ctx context.Context, nodeID string) (uint64, bool, error)
}

type Dispatcher interface {
	DispatchRuntimeApply(ctx context.Context, apply RuntimeApply) (ApplyResult, error)
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

func (s *Service) SaveApplyResult(ctx context.Context, result ApplyResult) error {
	if result.ApplyID == "" {
		return errors.New("apply_id is required")
	}
	if result.CreatedAt.IsZero() {
		result.CreatedAt = s.now().UTC()
	}
	return s.repo.SaveApplyResult(ctx, result)
}

func (s *Service) CreateAccount(ctx context.Context, input CreateAccountInput) (Account, ApplyResult, error) {
	if input.NodeID == "" {
		return Account{}, ApplyResult{}, errors.New("node_id is required")
	}
	if input.Protocol != ProtocolSOCKS5 && input.Protocol != ProtocolHTTP && input.Protocol != ProtocolMixed {
		return Account{}, ApplyResult{}, errors.New("protocol must be SOCKS5, HTTP, or MIXED")
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
	result, err := s.dispatchResource(ctx, OperationUpsert, account, input.DesiredGeneration)
	if err != nil {
		return account, result, err
	}
	account.AppliedGeneration = result.AppliedRevision
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
			ApplyID:          uuid.NewString(),
			ProxyAccountID:   account.ProxyAccountID,
			NodeID:           account.NodeID,
			Operation:        OperationUpdatePolicy,
			Status:           ApplyStatusDuplicate,
			AppliedRevision:  account.AppliedGeneration,
			LastGoodRevision: account.AppliedGeneration,
			CreatedAt:        s.now().UTC(),
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
	result, err := s.dispatchResource(ctx, OperationUpdatePolicy, account, input.DesiredGeneration)
	if err != nil {
		return account, result, err
	}
	account.AppliedGeneration = result.AppliedRevision
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
	result, err := s.dispatchRemove(ctx, OperationDelete, account, account.DesiredGeneration)
	if err == nil {
		account.AppliedGeneration = result.AppliedRevision
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
	return s.dispatchRemove(ctx, OperationDelete, account, account.DesiredGeneration)
}

func (s *Service) GetUsage(ctx context.Context, proxyAccountID string) (ApplyResult, error) {
	account, ok, err := s.repo.GetAccount(ctx, proxyAccountID)
	if err != nil {
		return ApplyResult{}, err
	}
	if !ok {
		return ApplyResult{}, fmt.Errorf("account %s not found", proxyAccountID)
	}
	usage, ok, err := s.repo.LatestUsage(ctx, proxyAccountID)
	if err != nil {
		return ApplyResult{}, err
	}
	if !ok {
		usage = Usage{ProxyAccountID: account.ProxyAccountID, RuntimeEmail: account.RuntimeEmail}
	}
	result := ApplyResult{
		ApplyID:          uuid.NewString(),
		ProxyAccountID:   account.ProxyAccountID,
		NodeID:           account.NodeID,
		Operation:        OperationGetUsage,
		Status:           ApplyStatusACK,
		AppliedRevision:  account.AppliedGeneration,
		LastGoodRevision: account.AppliedGeneration,
		Usage:            usage,
		CreatedAt:        s.now().UTC(),
	}
	_ = s.repo.SaveApplyResult(ctx, result)
	return result, nil
}

func (s *Service) ProbeAccount(ctx context.Context, proxyAccountID string) (ApplyResult, error) {
	account, ok, err := s.repo.GetAccount(ctx, proxyAccountID)
	if err != nil {
		return ApplyResult{}, err
	}
	if !ok {
		return ApplyResult{}, fmt.Errorf("account %s not found", proxyAccountID)
	}
	return s.localUnsupportedResult(ctx, OperationProbe, account, "probe is not a runtime apply operation")
}

func (s *Service) GetDigest(ctx context.Context, nodeID string) (ApplyResult, error) {
	if nodeID == "" {
		return ApplyResult{}, errors.New("node_id is required")
	}
	digest, ok, err := s.repo.LatestDigest(ctx, nodeID)
	if err != nil {
		return ApplyResult{}, err
	}
	if !ok {
		return ApplyResult{}, fmt.Errorf("node %s has no runtime digest yet", nodeID)
	}
	result := ApplyResult{
		ApplyID:   uuid.NewString(),
		NodeID:    nodeID,
		Operation: OperationGetDigest,
		Status:    ApplyStatusACK,
		Digest:    digest,
		CreatedAt: s.now().UTC(),
	}
	_ = s.repo.SaveApplyResult(ctx, result)
	return result, nil
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

func (s *Service) dispatchResource(ctx context.Context, operation Operation, account Account, revision uint64) (ApplyResult, error) {
	nodeBaseRevision, targetRevision, err := s.nextNodeRevision(ctx, account.NodeID)
	if err != nil {
		return ApplyResult{}, err
	}
	resourceRevision := revision
	if resourceRevision == 0 {
		resourceRevision = targetRevision
	}
	apply := s.newDeltaApply(account.NodeID, nodeBaseRevision, targetRevision)
	apply.Resources = []RuntimeResource{resourceFromAccount(account, resourceRevision)}
	result, err := s.dispatch(ctx, operation, account, apply)
	if s.shouldRetryRevisionMismatch(result, err) {
		retryBase := result.LastGoodRevision
		retryTarget := retryBase + 1
		retry := s.newDeltaApply(account.NodeID, retryBase, retryTarget)
		retry.Resources = []RuntimeResource{resourceFromAccount(account, retryTarget)}
		return s.dispatch(ctx, operation, account, retry)
	}
	return result, err
}

func (s *Service) dispatchRemove(ctx context.Context, operation Operation, account Account, revision uint64) (ApplyResult, error) {
	nodeBaseRevision, targetRevision, err := s.nextNodeRevision(ctx, account.NodeID)
	if err != nil {
		return ApplyResult{}, err
	}
	apply := s.newDeltaApply(account.NodeID, nodeBaseRevision, targetRevision)
	apply.RemovedResourceNames = []string{resourceName(account)}
	result, err := s.dispatch(ctx, operation, account, apply)
	if s.shouldRetryRevisionMismatch(result, err) {
		retryBase := result.LastGoodRevision
		retry := s.newDeltaApply(account.NodeID, retryBase, retryBase+1)
		retry.RemovedResourceNames = []string{resourceName(account)}
		return s.dispatch(ctx, operation, account, retry)
	}
	return result, err
}

func (s *Service) nextNodeRevision(ctx context.Context, nodeID string) (uint64, uint64, error) {
	baseRevision, ok, err := s.repo.LatestNodeRevision(ctx, nodeID)
	if err != nil {
		return 0, 0, err
	}
	if !ok {
		baseRevision = 0
	}
	return baseRevision, baseRevision + 1, nil
}

func (s *Service) shouldRetryRevisionMismatch(result ApplyResult, err error) bool {
	if err == nil {
		return false
	}
	return (result.Status == ApplyStatusNACK || result.Status == ApplyStatusFailed) && strings.Contains(result.ErrorDetail, "base revision")
}

func (s *Service) newDeltaApply(nodeID string, baseRevision uint64, targetRevision uint64) RuntimeApply {
	applyID := uuid.NewString()
	return RuntimeApply{
		ApplyID:        applyID,
		NodeID:         nodeID,
		Mode:           ApplyModeDelta,
		VersionInfo:    fmt.Sprintf("revision/%d", targetRevision),
		Nonce:          uuid.NewString(),
		BaseRevision:   baseRevision,
		TargetRevision: targetRevision,
		DeadlineUnixMS: s.now().Add(8 * time.Second).UnixMilli(),
	}
}

func (s *Service) dispatch(ctx context.Context, operation Operation, account Account, apply RuntimeApply) (ApplyResult, error) {
	result, err := s.dispatcher.DispatchRuntimeApply(ctx, apply)
	if result.ApplyID == "" {
		result.ApplyID = apply.ApplyID
	}
	result.ProxyAccountID = account.ProxyAccountID
	result.NodeID = account.NodeID
	result.Operation = operation
	result.CreatedAt = s.now().UTC()
	if err != nil {
		result.Status = ApplyStatusFailed
		result.ErrorDetail = err.Error()
	}
	if saveErr := s.repo.SaveApplyResult(ctx, result); saveErr != nil && err == nil {
		err = saveErr
	}
	return result, err
}

func (s *Service) localUnsupportedResult(ctx context.Context, operation Operation, account Account, detail string) (ApplyResult, error) {
	result := ApplyResult{
		ApplyID:          uuid.NewString(),
		ProxyAccountID:   account.ProxyAccountID,
		NodeID:           account.NodeID,
		Operation:        operation,
		Status:           ApplyStatusFailed,
		ErrorDetail:      detail,
		AppliedRevision:  account.AppliedGeneration,
		LastGoodRevision: account.AppliedGeneration,
		CreatedAt:        s.now().UTC(),
	}
	_ = s.repo.SaveApplyResult(ctx, result)
	return result, errors.New(detail)
}

func resourceFromAccount(account Account, revision uint64) RuntimeResource {
	expiresAt := int64(0)
	if !account.ExpiresAt.IsZero() {
		expiresAt = account.ExpiresAt.UnixMilli()
	}
	return RuntimeResource{
		Name:              resourceName(account),
		Kind:              ResourceKindProxyAccount,
		ResourceVersion:   revision,
		RuntimeEmail:      account.RuntimeEmail,
		Protocol:          account.Protocol,
		ListenIP:          account.ListenIP,
		Port:              account.Port,
		Username:          account.Username,
		Password:          account.Password,
		EgressLimitBPS:    account.EgressLimitBPS,
		IngressLimitBPS:   account.IngressLimitBPS,
		MaxConnections:    account.MaxConnections,
		Priority:          1,
		AbuseReportPolicy: "REPORT_ONLY",
		ExpiresAtUnixMS:   expiresAt,
	}
}

func resourceName(account Account) string {
	if account.RuntimeEmail != "" {
		return "proxy/" + account.RuntimeEmail
	}
	return "proxy/" + account.ProxyAccountID
}
