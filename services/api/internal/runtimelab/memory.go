package runtimelab

import (
	"context"
	"sort"
	"sync"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	accounts map[string]Account
	results  []ApplyResult
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{accounts: map[string]Account{}}
}

func (r *MemoryRepository) UpsertAccount(_ context.Context, account Account) (Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.accounts[account.ProxyAccountID] = account
	return account, nil
}

func (r *MemoryRepository) GetAccount(_ context.Context, proxyAccountID string) (Account, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	account, ok := r.accounts[proxyAccountID]
	return account, ok, nil
}

func (r *MemoryRepository) ListAccounts(_ context.Context) ([]Account, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]Account, 0, len(r.accounts))
	for _, account := range r.accounts {
		if account.Status != AccountStatusDeleted {
			items = append(items, account)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryRepository) SaveApplyResult(_ context.Context, result ApplyResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results = append(r.results, result)
	return nil
}

func (r *MemoryRepository) ListApplyResults(_ context.Context, proxyAccountID string, limit int) ([]ApplyResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []ApplyResult{}
	for i := len(r.results) - 1; i >= 0 && len(items) < limit; i-- {
		result := r.results[i]
		if proxyAccountID == "" || result.ProxyAccountID == proxyAccountID {
			items = append(items, result)
		}
	}
	return items, nil
}

func (r *MemoryRepository) LatestUsage(_ context.Context, proxyAccountID string) (Usage, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := len(r.results) - 1; i >= 0; i-- {
		result := r.results[i]
		if result.ProxyAccountID != proxyAccountID {
			continue
		}
		if result.Usage.ProxyAccountID == "" && result.Usage.RuntimeEmail == "" {
			continue
		}
		return result.Usage, true, nil
	}
	return Usage{}, false, nil
}

func (r *MemoryRepository) LatestDigest(_ context.Context, nodeID string) (Digest, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := len(r.results) - 1; i >= 0; i-- {
		result := r.results[i]
		if result.NodeID != nodeID {
			continue
		}
		if result.Digest.Hash == "" && result.Digest.AccountCount == 0 && result.Digest.MaxGeneration == 0 {
			continue
		}
		return result.Digest, true, nil
	}
	return Digest{}, false, nil
}

func (r *MemoryRepository) LatestNodeRevision(_ context.Context, nodeID string) (uint64, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := len(r.results) - 1; i >= 0; i-- {
		result := r.results[i]
		if result.NodeID != nodeID {
			continue
		}
		if result.Operation != OperationUpsert && result.Operation != OperationUpdatePolicy && result.Operation != OperationDelete {
			continue
		}
		return result.LastGoodRevision, true, nil
	}
	return 0, false, nil
}
