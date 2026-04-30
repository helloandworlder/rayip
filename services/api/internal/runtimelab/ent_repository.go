package runtimelab

import (
	"context"
	"encoding/json"
	"time"

	"entgo.io/ent/dialect/sql"
	apiEnt "github.com/rayip/rayip/services/api/ent"
	entApply "github.com/rayip/rayip/services/api/ent/runtimeapplyresult"
	entAccount "github.com/rayip/rayip/services/api/ent/runtimelabaccount"
)

type EntRepository struct {
	client *apiEnt.Client
}

func NewEntRepository(client *apiEnt.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) UpsertAccount(ctx context.Context, account Account) (Account, error) {
	create := r.client.RuntimeLabAccount.Create().
		SetID(account.ProxyAccountID).
		SetNodeID(account.NodeID).
		SetRuntimeEmail(account.RuntimeEmail).
		SetProtocol(string(account.Protocol)).
		SetListenIP(account.ListenIP).
		SetPort(account.Port).
		SetUsername(account.Username).
		SetPassword(account.Password).
		SetEgressLimitBps(account.EgressLimitBPS).
		SetIngressLimitBps(account.IngressLimitBPS).
		SetMaxConnections(account.MaxConnections).
		SetStatus(string(account.Status)).
		SetPolicyVersion(account.PolicyVersion).
		SetDesiredGeneration(account.DesiredGeneration).
		SetAppliedGeneration(account.AppliedGeneration).
		SetCreatedAt(account.CreatedAt).
		SetUpdatedAt(account.UpdatedAt)
	if !account.ExpiresAt.IsZero() {
		create.SetExpiresAt(account.ExpiresAt)
	}
	if err := create.OnConflict(sql.ConflictColumns("proxy_account_id")).UpdateNewValues().Exec(ctx); err != nil {
		return Account{}, err
	}
	item, err := r.client.RuntimeLabAccount.Get(ctx, account.ProxyAccountID)
	if err != nil {
		return Account{}, err
	}
	return accountFromEnt(item), nil
}

func (r *EntRepository) GetAccount(ctx context.Context, proxyAccountID string) (Account, bool, error) {
	item, err := r.client.RuntimeLabAccount.Get(ctx, proxyAccountID)
	if apiEnt.IsNotFound(err) {
		return Account{}, false, nil
	}
	if err != nil {
		return Account{}, false, err
	}
	return accountFromEnt(item), true, nil
}

func (r *EntRepository) ListAccounts(ctx context.Context) ([]Account, error) {
	items, err := r.client.RuntimeLabAccount.Query().
		Where(entAccount.StatusNEQ(string(AccountStatusDeleted))).
		Order(apiEnt.Desc(entAccount.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	accounts := make([]Account, 0, len(items))
	for _, item := range items {
		accounts = append(accounts, accountFromEnt(item))
	}
	return accounts, nil
}

func (r *EntRepository) SaveApplyResult(ctx context.Context, result ApplyResult) error {
	usageMap, err := structToMap(result.Usage)
	if err != nil {
		return err
	}
	digestMap, err := structToMap(result.Digest)
	if err != nil {
		return err
	}
	return r.client.RuntimeApplyResult.Create().
		SetID(result.ApplyID).
		SetProxyAccountID(result.ProxyAccountID).
		SetNodeID(result.NodeID).
		SetOperation(string(result.Operation)).
		SetStatus(string(result.Status)).
		SetVersionInfo(result.VersionInfo).
		SetNonce(result.Nonce).
		SetAppliedRevision(result.AppliedRevision).
		SetLastGoodRevision(result.LastGoodRevision).
		SetErrorDetail(result.ErrorDetail).
		SetUsage(usageMap).
		SetDigest(digestMap).
		SetCreatedAt(result.CreatedAt).
		OnConflict(sql.ConflictColumns("apply_id")).
		UpdateNewValues().
		Exec(ctx)
}

func (r *EntRepository) ListApplyResults(ctx context.Context, proxyAccountID string, limit int) ([]ApplyResult, error) {
	query := r.client.RuntimeApplyResult.Query().Order(apiEnt.Desc(entApply.FieldCreatedAt)).Limit(limit)
	if proxyAccountID != "" {
		query = query.Where(entApply.ProxyAccountID(proxyAccountID))
	}
	items, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]ApplyResult, 0, len(items))
	for _, item := range items {
		result, err := applyResultFromEnt(item)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (r *EntRepository) LatestUsage(ctx context.Context, proxyAccountID string) (Usage, bool, error) {
	items, err := r.ListApplyResults(ctx, proxyAccountID, 50)
	if err != nil {
		return Usage{}, false, err
	}
	for _, item := range items {
		if item.Usage.ProxyAccountID == "" && item.Usage.RuntimeEmail == "" {
			continue
		}
		return item.Usage, true, nil
	}
	return Usage{}, false, nil
}

func (r *EntRepository) LatestDigest(ctx context.Context, nodeID string) (Digest, bool, error) {
	items, err := r.client.RuntimeApplyResult.Query().
		Where(entApply.NodeID(nodeID)).
		Order(apiEnt.Desc(entApply.FieldCreatedAt)).
		Limit(50).
		All(ctx)
	if err != nil {
		return Digest{}, false, err
	}
	for _, item := range items {
		result, err := applyResultFromEnt(item)
		if err != nil {
			return Digest{}, false, err
		}
		if result.Digest.Hash == "" && result.Digest.AccountCount == 0 && result.Digest.MaxGeneration == 0 {
			continue
		}
		return result.Digest, true, nil
	}
	return Digest{}, false, nil
}

func (r *EntRepository) LatestNodeRevision(ctx context.Context, nodeID string) (uint64, bool, error) {
	item, err := r.client.RuntimeApplyResult.Query().
		Where(entApply.NodeID(nodeID), entApply.LastGoodRevisionGT(0)).
		Order(apiEnt.Desc(entApply.FieldCreatedAt)).
		First(ctx)
	if apiEnt.IsNotFound(err) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return item.LastGoodRevision, true, nil
}

func accountFromEnt(item *apiEnt.RuntimeLabAccount) Account {
	expiresAt := time.Time{}
	if item.ExpiresAt != nil {
		expiresAt = *item.ExpiresAt
	}
	return Account{
		ProxyAccountID:    item.ID,
		NodeID:            item.NodeID,
		RuntimeEmail:      item.RuntimeEmail,
		Protocol:          Protocol(item.Protocol),
		ListenIP:          item.ListenIP,
		Port:              item.Port,
		Username:          item.Username,
		Password:          item.Password,
		ExpiresAt:         expiresAt,
		EgressLimitBPS:    item.EgressLimitBps,
		IngressLimitBPS:   item.IngressLimitBps,
		MaxConnections:    item.MaxConnections,
		Status:            AccountStatus(item.Status),
		PolicyVersion:     item.PolicyVersion,
		DesiredGeneration: item.DesiredGeneration,
		AppliedGeneration: item.AppliedGeneration,
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

func applyResultFromEnt(item *apiEnt.RuntimeApplyResult) (ApplyResult, error) {
	var usage Usage
	if err := mapToStruct(item.Usage, &usage); err != nil {
		return ApplyResult{}, err
	}
	var digest Digest
	if err := mapToStruct(item.Digest, &digest); err != nil {
		return ApplyResult{}, err
	}
	return ApplyResult{
		ApplyID:          item.ID,
		ProxyAccountID:   item.ProxyAccountID,
		NodeID:           item.NodeID,
		Operation:        Operation(item.Operation),
		Status:           ApplyStatus(item.Status),
		VersionInfo:      item.VersionInfo,
		Nonce:            item.Nonce,
		AppliedRevision:  item.AppliedRevision,
		LastGoodRevision: item.LastGoodRevision,
		ErrorDetail:      item.ErrorDetail,
		Usage:            usage,
		Digest:           digest,
		CreatedAt:        item.CreatedAt,
	}, nil
}

func structToMap(value any) (map[string]any, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	result := map[string]any{}
	if err := json.Unmarshal(payload, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func mapToStruct(value map[string]any, target any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, target)
}
