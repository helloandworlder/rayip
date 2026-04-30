package noderuntime

import (
	"context"
	"sync"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	statuses map[string]Status
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{statuses: map[string]Status{}}
}

func (r *MemoryRepository) UpsertStatus(_ context.Context, status Status) (Status, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	status.Capabilities = append([]string(nil), status.Capabilities...)
	status.UnsellableReasons = append([]UnsellableReason(nil), status.UnsellableReasons...)
	r.statuses[status.NodeID] = status
	return status, nil
}

func (r *MemoryRepository) GetStatus(_ context.Context, nodeID string) (Status, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	status, ok := r.statuses[nodeID]
	if !ok {
		return Status{}, false, nil
	}
	status.Capabilities = append([]string(nil), status.Capabilities...)
	status.UnsellableReasons = append([]UnsellableReason(nil), status.UnsellableReasons...)
	return status, true, nil
}

var _ Repository = (*MemoryRepository)(nil)
