package runtimecontrol

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryRepository struct {
	mu      sync.RWMutex
	states  map[string]ResourceState
	seqByID map[string]uint64
	changes []ChangeLogEntry
	outbox  []OutboxEvent
	jobs    []JobResult
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		states:  map[string]ResourceState{},
		seqByID: map[string]uint64{},
	}
}

func (r *MemoryRepository) UpsertResource(_ context.Context, input ResourceInput, now time.Time) (MutationResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	resourceName := resourceName(input.RuntimeEmail)
	state := r.states[input.ProxyAccountID]
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	state.ResourceName = resourceName
	state.ProxyAccountID = input.ProxyAccountID
	state.NodeID = input.NodeID
	state.Kind = ResourceKindProxyAccount
	state.RuntimeEmail = input.RuntimeEmail
	state.Protocol = input.Protocol
	state.ListenIP = input.ListenIP
	state.Port = input.Port
	state.Username = input.Username
	state.Password = input.Password
	state.EgressLimitBPS = input.EgressLimitBPS
	state.IngressLimitBPS = input.IngressLimitBPS
	state.MaxConnections = input.MaxConnections
	state.Priority = input.Priority
	state.ExpiresAt = input.ExpiresAt
	state.DesiredRevision++
	state.Removed = false
	state.UpdatedAt = now
	r.states[input.ProxyAccountID] = state

	change, outbox := r.appendChangeLocked(state.NodeID, state.ResourceName, ChangeActionUpsert, state.DesiredRevision, now)
	return MutationResult{State: state, Change: change, Outbox: outbox}, nil
}

func (r *MemoryRepository) RemoveResource(_ context.Context, proxyAccountID string, now time.Time) (MutationResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	state, ok := r.states[proxyAccountID]
	if !ok {
		return MutationResult{}, fmt.Errorf("resource %s not found", proxyAccountID)
	}
	state.DesiredRevision++
	state.Removed = true
	state.UpdatedAt = now
	r.states[proxyAccountID] = state

	change, outbox := r.appendChangeLocked(state.NodeID, state.ResourceName, ChangeActionRemove, state.DesiredRevision, now)
	return MutationResult{State: state, Change: change, Outbox: outbox}, nil
}

func (r *MemoryRepository) ListChanges(_ context.Context, nodeID string, afterSeq uint64, limit int) ([]ChangeLogEntry, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	changes := []ChangeLogEntry{}
	for _, change := range r.changes {
		if change.NodeID == nodeID && change.Seq > afterSeq {
			changes = append(changes, change)
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Seq < changes[j].Seq })
	if len(changes) > limit {
		changes = changes[:limit]
	}
	return changes, nil
}

func (r *MemoryRepository) GetResourceByName(_ context.Context, resourceName string) (ResourceState, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, state := range r.states {
		if state.ResourceName == resourceName {
			return state, true, nil
		}
	}
	return ResourceState{}, false, nil
}

func (r *MemoryRepository) ListResourcesByNode(_ context.Context, nodeID string, includeRemoved bool, offset int, limit int) ([]ResourceState, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	states := []ResourceState{}
	for _, state := range r.states {
		if state.NodeID != nodeID {
			continue
		}
		if state.Removed && !includeRemoved {
			continue
		}
		states = append(states, state)
	}
	sort.Slice(states, func(i, j int) bool { return states[i].ResourceName < states[j].ResourceName })
	if offset >= len(states) {
		return []ResourceState{}, nil
	}
	states = states[offset:]
	if len(states) > limit {
		states = states[:limit]
	}
	return append([]ResourceState(nil), states...), nil
}

func (r *MemoryRepository) ListOutbox(_ context.Context, limit int) ([]OutboxEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	pending := []OutboxEvent{}
	for _, event := range r.outbox {
		if event.PublishedAt.IsZero() {
			pending = append(pending, event)
		}
	}
	if limit > len(pending) {
		limit = len(pending)
	}
	return append([]OutboxEvent(nil), pending[:limit]...), nil
}

func (r *MemoryRepository) MarkOutboxPublished(_ context.Context, eventID string, publishedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for idx := range r.outbox {
		if r.outbox[idx].ID == eventID {
			r.outbox[idx].PublishedAt = publishedAt
			return nil
		}
	}
	return fmt.Errorf("outbox event %s not found", eventID)
}

func (r *MemoryRepository) SaveJobResult(_ context.Context, result JobResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs = append(r.jobs, result)
	return nil
}

func (r *MemoryRepository) appendChangeLocked(nodeID string, resourceName string, action ChangeAction, revision uint64, now time.Time) (ChangeLogEntry, OutboxEvent) {
	r.seqByID[nodeID]++
	change := ChangeLogEntry{
		ID:           uuid.NewString(),
		NodeID:       nodeID,
		Seq:          r.seqByID[nodeID],
		ResourceName: resourceName,
		Action:       action,
		Revision:     revision,
		CreatedAt:    now,
	}
	outbox := OutboxEvent{
		ID:           uuid.NewString(),
		Topic:        "rayip.runtime.apply.v1",
		AggregateID:  change.ID,
		AggregateKey: nodeID,
		Payload: map[string]any{
			"change_id":     change.ID,
			"node_id":       nodeID,
			"seq":           change.Seq,
			"resource_name": resourceName,
			"action":        string(action),
		},
		CreatedAt: now,
	}
	r.changes = append(r.changes, change)
	r.outbox = append(r.outbox, outbox)
	return change, outbox
}

func resourceName(runtimeEmail string) string {
	return "proxy/" + runtimeEmail
}

var _ Repository = (*MemoryRepository)(nil)
