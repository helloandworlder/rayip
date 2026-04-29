package runtime

import (
	"context"
	"errors"
	"sync"
)

type Manager struct {
	core        Core
	mu          sync.Mutex
	generations map[string]uint64
}

func NewManager(core Core) *Manager {
	return &Manager{core: core, generations: map[string]uint64{}}
}

func (m *Manager) Apply(ctx context.Context, cmd Command) (Result, error) {
	if cmd.CommandID == "" {
		return Result{}, errors.New("command_id is required")
	}

	accountID := cmd.Account.ProxyAccountID
	if accountID != "" && cmd.Operation != OperationGetUsage && cmd.Operation != OperationGetDigest && cmd.Operation != OperationProbe {
		m.mu.Lock()
		applied := m.generations[accountID]
		if cmd.DesiredGeneration > 0 && applied >= cmd.DesiredGeneration {
			m.mu.Unlock()
			digest, _ := m.core.Digest(ctx)
			return Result{
				CommandID:         cmd.CommandID,
				Status:            ResultDuplicate,
				AppliedGeneration: applied,
				Digest:            digest,
			}, nil
		}
		m.mu.Unlock()
	}

	result := Result{CommandID: cmd.CommandID, Status: ResultSuccess, AppliedGeneration: cmd.DesiredGeneration}
	switch cmd.Operation {
	case OperationUpsert, OperationUpdatePolicy:
		account := cmd.Account
		if account.RuntimeEmail == "" {
			account.RuntimeEmail = account.ProxyAccountID
		}
		if account.Status == "" {
			account.Status = AccountStatusEnabled
		}
		account.DesiredGeneration = cmd.DesiredGeneration
		if err := m.core.UpsertAccount(ctx, account); err != nil {
			return failed(cmd.CommandID, cmd.DesiredGeneration, err), err
		}
		m.remember(account.ProxyAccountID, cmd.DesiredGeneration)
	case OperationDisable:
		if err := m.core.DisableAccount(ctx, accountID, cmd.DesiredGeneration); err != nil {
			return failed(cmd.CommandID, cmd.DesiredGeneration, err), err
		}
		m.remember(accountID, cmd.DesiredGeneration)
	case OperationDelete:
		if err := m.core.DeleteAccount(ctx, accountID); err != nil {
			return failed(cmd.CommandID, cmd.DesiredGeneration, err), err
		}
		m.remember(accountID, cmd.DesiredGeneration)
	case OperationGetUsage:
		usage, err := m.core.Usage(ctx, accountID)
		if err != nil {
			return failed(cmd.CommandID, cmd.DesiredGeneration, err), err
		}
		result.Usage = usage
	case OperationProbe:
		usage, err := m.core.Probe(ctx, accountID)
		if err != nil {
			return failed(cmd.CommandID, cmd.DesiredGeneration, err), err
		}
		result.Usage = usage
	case OperationGetDigest:
	default:
		err := errors.New("unsupported runtime operation")
		return failed(cmd.CommandID, cmd.DesiredGeneration, err), err
	}

	digest, err := m.core.Digest(ctx)
	if err != nil {
		return failed(cmd.CommandID, cmd.DesiredGeneration, err), err
	}
	result.Digest = digest
	return result, nil
}

func (m *Manager) remember(proxyAccountID string, generation uint64) {
	if proxyAccountID == "" {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if generation > m.generations[proxyAccountID] {
		m.generations[proxyAccountID] = generation
	}
}

func failed(commandID string, generation uint64, err error) Result {
	return Result{
		CommandID:         commandID,
		Status:            ResultFailed,
		ErrorCode:         "RUNTIME_APPLY_FAILED",
		ErrorMessage:      err.Error(),
		AppliedGeneration: generation,
	}
}
