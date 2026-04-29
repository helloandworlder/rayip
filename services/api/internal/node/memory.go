package node

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	byCode   map[string]NodeRecord
	codeByID map[string]string
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		byCode:   map[string]NodeRecord{},
		codeByID: map[string]string{},
	}
}

func (r *MemoryRepository) UpsertLease(_ context.Context, input LeaseInput, now time.Time) (NodeRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.byCode[input.NodeCode]
	if !ok {
		record = NodeRecord{
			ID:        firstNonEmpty(input.NodeID, uuid.NewString()),
			Code:      input.NodeCode,
			CreatedAt: now,
		}
	}
	record.BundleVersion = input.BundleVersion
	record.AgentVersion = input.AgentVersion
	record.XrayVersion = input.XrayVersion
	record.Capabilities = append([]string(nil), input.Capabilities...)
	record.LastOnlineAt = now
	record.UpdatedAt = now

	r.byCode[record.Code] = record
	r.codeByID[record.ID] = record.Code
	return record, nil
}

func (r *MemoryRepository) List(_ context.Context) ([]NodeRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	records := make([]NodeRecord, 0, len(r.byCode))
	for _, record := range r.byCode {
		records = append(records, record)
	}
	return records, nil
}

type MemoryLeaseStore struct {
	mu     sync.RWMutex
	leases map[string]LeaseSnapshot
}

func NewMemoryLeaseStore() *MemoryLeaseStore {
	return &MemoryLeaseStore{leases: map[string]LeaseSnapshot{}}
}

func (s *MemoryLeaseStore) PutLease(_ context.Context, lease LeaseSnapshot, _ time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.leases[lease.NodeID] = lease
	return nil
}

func (s *MemoryLeaseStore) GetLease(_ context.Context, nodeID string) (LeaseSnapshot, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lease, ok := s.leases[nodeID]
	return lease, ok, nil
}
