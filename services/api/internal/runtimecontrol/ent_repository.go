package runtimecontrol

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	apiEnt "github.com/rayip/rayip/services/api/ent"
	entOutbox "github.com/rayip/rayip/services/api/ent/outboxevent"
	entState "github.com/rayip/rayip/services/api/ent/runtimeaccountstate"
	entChange "github.com/rayip/rayip/services/api/ent/runtimechangelog"
)

type EntRepository struct {
	client *apiEnt.Client
}

func NewEntRepository(client *apiEnt.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) UpsertResource(ctx context.Context, input ResourceInput, now time.Time) (MutationResult, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return MutationResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	existing, err := tx.RuntimeAccountState.Get(ctx, input.ProxyAccountID)
	if err != nil && !apiEnt.IsNotFound(err) {
		return MutationResult{}, err
	}
	revision := uint64(1)
	createdAt := now
	if existing != nil {
		revision = existing.DesiredRevision + 1
		createdAt = existing.CreatedAt
	}
	state := ResourceState{
		ResourceName:    resourceName(input.RuntimeEmail),
		ProxyAccountID:  input.ProxyAccountID,
		NodeID:          input.NodeID,
		Kind:            ResourceKindProxyAccount,
		RuntimeEmail:    input.RuntimeEmail,
		Protocol:        input.Protocol,
		ListenIP:        input.ListenIP,
		Port:            input.Port,
		Username:        input.Username,
		Password:        input.Password,
		EgressLimitBPS:  input.EgressLimitBPS,
		IngressLimitBPS: input.IngressLimitBPS,
		MaxConnections:  input.MaxConnections,
		Priority:        input.Priority,
		ExpiresAt:       input.ExpiresAt,
		DesiredRevision: revision,
		Removed:         false,
		CreatedAt:       createdAt,
		UpdatedAt:       now,
	}
	if err := upsertState(ctx, tx, state); err != nil {
		return MutationResult{}, err
	}
	change, outbox, err := appendChange(ctx, tx, state.NodeID, state.ResourceName, ChangeActionUpsert, state.DesiredRevision, now)
	if err != nil {
		return MutationResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return MutationResult{}, err
	}
	committed = true
	return MutationResult{State: state, Change: change, Outbox: outbox}, nil
}

func (r *EntRepository) RemoveResource(ctx context.Context, proxyAccountID string, now time.Time) (MutationResult, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return MutationResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	existing, err := tx.RuntimeAccountState.Get(ctx, proxyAccountID)
	if apiEnt.IsNotFound(err) {
		return MutationResult{}, fmt.Errorf("resource %s not found", proxyAccountID)
	}
	if err != nil {
		return MutationResult{}, err
	}
	state := stateFromEnt(existing)
	state.DesiredRevision++
	state.Removed = true
	state.UpdatedAt = now
	if err := tx.RuntimeAccountState.UpdateOneID(proxyAccountID).
		SetDesiredRevision(state.DesiredRevision).
		SetRemoved(true).
		SetUpdatedAt(now).
		Exec(ctx); err != nil {
		return MutationResult{}, err
	}
	change, outbox, err := appendChange(ctx, tx, state.NodeID, state.ResourceName, ChangeActionRemove, state.DesiredRevision, now)
	if err != nil {
		return MutationResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return MutationResult{}, err
	}
	committed = true
	return MutationResult{State: state, Change: change, Outbox: outbox}, nil
}

func (r *EntRepository) ListChanges(ctx context.Context, nodeID string, afterSeq uint64, limit int) ([]ChangeLogEntry, error) {
	items, err := r.client.RuntimeChangeLog.Query().
		Where(entChange.NodeID(nodeID), entChange.SeqGT(afterSeq)).
		Order(apiEnt.Asc(entChange.FieldSeq)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	changes := make([]ChangeLogEntry, 0, len(items))
	for _, item := range items {
		changes = append(changes, changeFromEnt(item))
	}
	return changes, nil
}

func (r *EntRepository) GetResourceByName(ctx context.Context, resourceName string) (ResourceState, bool, error) {
	item, err := r.client.RuntimeAccountState.Query().
		Where(entState.ResourceName(resourceName)).
		Only(ctx)
	if apiEnt.IsNotFound(err) {
		return ResourceState{}, false, nil
	}
	if err != nil {
		return ResourceState{}, false, err
	}
	return stateFromEnt(item), true, nil
}

func (r *EntRepository) ListResourcesByNode(ctx context.Context, nodeID string, includeRemoved bool, offset int, limit int) ([]ResourceState, error) {
	query := r.client.RuntimeAccountState.Query().
		Where(entState.NodeID(nodeID)).
		Order(apiEnt.Asc(entState.FieldResourceName)).
		Offset(offset).
		Limit(limit)
	if !includeRemoved {
		query = query.Where(entState.Removed(false))
	}
	items, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	states := make([]ResourceState, 0, len(items))
	for _, item := range items {
		states = append(states, stateFromEnt(item))
	}
	return states, nil
}

func (r *EntRepository) ListOutbox(ctx context.Context, limit int) ([]OutboxEvent, error) {
	items, err := r.client.OutboxEvent.Query().
		Where(entOutbox.PublishedAtIsNil()).
		Order(apiEnt.Asc(entOutbox.FieldCreatedAt)).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	outbox := make([]OutboxEvent, 0, len(items))
	for _, item := range items {
		outbox = append(outbox, outboxFromEnt(item))
	}
	return outbox, nil
}

func (r *EntRepository) MarkOutboxPublished(ctx context.Context, eventID string, publishedAt time.Time) error {
	return r.client.OutboxEvent.UpdateOneID(eventID).
		SetPublishedAt(publishedAt).
		Exec(ctx)
}

func (r *EntRepository) SaveJobResult(ctx context.Context, result JobResult) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if err := tx.NodeJob.Create().
		SetID(result.JobID).
		SetNodeID(result.NodeID).
		SetStatus(string(result.Status)).
		SetBaseRevision(result.BaseRevision).
		SetTargetRevision(result.TargetRevision).
		SetAcceptedRevision(result.AcceptedRevision).
		SetLastGoodRevision(result.LastGoodRevision).
		SetApplyID(result.ApplyID).
		SetVersionInfo(result.VersionInfo).
		SetNonce(result.Nonce).
		SetErrorDetail(result.ErrorDetail).
		SetCreatedAt(result.CreatedAt).
		SetUpdatedAt(result.UpdatedAt).
		Exec(ctx); err != nil {
		return err
	}
	if err := tx.NodeJobAttempt.Create().
		SetID(uuid.NewString()).
		SetJobID(result.JobID).
		SetNodeID(result.NodeID).
		SetStatus(string(result.Status)).
		SetApplyID(result.ApplyID).
		SetErrorDetail(result.ErrorDetail).
		SetCreatedAt(result.UpdatedAt).
		Exec(ctx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func upsertState(ctx context.Context, tx *apiEnt.Tx, state ResourceState) error {
	create := tx.RuntimeAccountState.Create().
		SetID(state.ProxyAccountID).
		SetNodeID(state.NodeID).
		SetResourceName(state.ResourceName).
		SetKind(string(state.Kind)).
		SetRuntimeEmail(state.RuntimeEmail).
		SetProtocol(string(state.Protocol)).
		SetListenIP(state.ListenIP).
		SetPort(state.Port).
		SetUsername(state.Username).
		SetPassword(state.Password).
		SetEgressLimitBps(state.EgressLimitBPS).
		SetIngressLimitBps(state.IngressLimitBPS).
		SetMaxConnections(state.MaxConnections).
		SetPriority(state.Priority).
		SetDesiredRevision(state.DesiredRevision).
		SetRemoved(state.Removed).
		SetCreatedAt(state.CreatedAt).
		SetUpdatedAt(state.UpdatedAt)
	if !state.ExpiresAt.IsZero() {
		create.SetExpiresAt(state.ExpiresAt)
	}
	return create.OnConflict(sql.ConflictColumns("proxy_account_id")).UpdateNewValues().Exec(ctx)
}

func appendChange(ctx context.Context, tx *apiEnt.Tx, nodeID string, resourceName string, action ChangeAction, revision uint64, now time.Time) (ChangeLogEntry, OutboxEvent, error) {
	maxSeq, err := tx.RuntimeChangeLog.Query().
		Where(entChange.NodeID(nodeID)).
		Aggregate(apiEnt.Max(entChange.FieldSeq)).
		Int(ctx)
	if err != nil {
		return ChangeLogEntry{}, OutboxEvent{}, err
	}
	seq := uint64(maxSeq) + 1
	changeID := uuid.NewString()
	changeItem, err := tx.RuntimeChangeLog.Create().
		SetID(changeID).
		SetNodeID(nodeID).
		SetSeq(seq).
		SetResourceName(resourceName).
		SetAction(string(action)).
		SetRevision(revision).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return ChangeLogEntry{}, OutboxEvent{}, err
	}
	outboxItem, err := tx.OutboxEvent.Create().
		SetID(uuid.NewString()).
		SetTopic("rayip.runtime.apply.v1").
		SetAggregateID(changeID).
		SetAggregateKey(nodeID).
		SetPayload(map[string]any{
			"change_id":     changeID,
			"node_id":       nodeID,
			"seq":           seq,
			"resource_name": resourceName,
			"action":        string(action),
		}).
		SetCreatedAt(now).
		Save(ctx)
	if err != nil {
		return ChangeLogEntry{}, OutboxEvent{}, err
	}
	return changeFromEnt(changeItem), outboxFromEnt(outboxItem), nil
}

func stateFromEnt(item *apiEnt.RuntimeAccountState) ResourceState {
	expiresAt := time.Time{}
	if item.ExpiresAt != nil {
		expiresAt = *item.ExpiresAt
	}
	return ResourceState{
		ResourceName:    item.ResourceName,
		ProxyAccountID:  item.ID,
		NodeID:          item.NodeID,
		Kind:            ResourceKind(item.Kind),
		RuntimeEmail:    item.RuntimeEmail,
		Protocol:        Protocol(item.Protocol),
		ListenIP:        item.ListenIP,
		Port:            item.Port,
		Username:        item.Username,
		Password:        item.Password,
		EgressLimitBPS:  item.EgressLimitBps,
		IngressLimitBPS: item.IngressLimitBps,
		MaxConnections:  item.MaxConnections,
		Priority:        item.Priority,
		ExpiresAt:       expiresAt,
		DesiredRevision: item.DesiredRevision,
		Removed:         item.Removed,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

func changeFromEnt(item *apiEnt.RuntimeChangeLog) ChangeLogEntry {
	return ChangeLogEntry{
		ID:           item.ID,
		NodeID:       item.NodeID,
		Seq:          item.Seq,
		ResourceName: item.ResourceName,
		Action:       ChangeAction(item.Action),
		Revision:     item.Revision,
		CreatedAt:    item.CreatedAt,
	}
}

func outboxFromEnt(item *apiEnt.OutboxEvent) OutboxEvent {
	publishedAt := time.Time{}
	if item.PublishedAt != nil {
		publishedAt = *item.PublishedAt
	}
	return OutboxEvent{
		ID:           item.ID,
		Topic:        item.Topic,
		AggregateID:  item.AggregateID,
		AggregateKey: item.AggregateKey,
		Payload:      item.Payload,
		PublishedAt:  publishedAt,
		CreatedAt:    item.CreatedAt,
	}
}

var _ Repository = (*EntRepository)(nil)
