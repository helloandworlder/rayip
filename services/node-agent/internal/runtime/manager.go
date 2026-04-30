package runtime

import (
	"context"
	"errors"
	"sync"
)

type Manager struct {
	core             Core
	mu               sync.Mutex
	lastGoodRevision uint64
	lastSeenNonce    string
	lastVersionInfo  string
	seenApplies      map[string]ApplyAck
}

func NewManager(core Core) *Manager {
	return &Manager{core: core, seenApplies: map[string]ApplyAck{}}
}

func (m *Manager) Apply(ctx context.Context, apply Apply) (ApplyAck, error) {
	if apply.ApplyID == "" {
		return ApplyAck{}, errors.New("apply_id is required")
	}
	m.mu.Lock()
	if previous, ok := m.seenApplies[apply.ApplyID]; ok && apply.Nonce == m.lastSeenNonce && apply.VersionInfo == m.lastVersionInfo {
		m.mu.Unlock()
		return previous, nil
	}
	lastGood := m.lastGoodRevision
	if apply.QueryOperation != "" {
		m.mu.Unlock()
		return m.applyQuery(ctx, apply, lastGood)
	}
	if apply.FairnessState != (FairnessState{}) {
		m.mu.Unlock()
		return m.applyFairness(ctx, apply, lastGood)
	}
	if apply.Mode == ApplyModeDelta && apply.BaseRevision != lastGood {
		ack := m.baseAckLocked(apply)
		ack.Status = AckStatusNACK
		ack.AppliedRevision = lastGood
		ack.ErrorDetail = "base revision does not match last good revision"
		m.mu.Unlock()
		return ack, errors.New(ack.ErrorDetail)
	}
	m.mu.Unlock()

	ack := ApplyAck{
		ApplyID:          apply.ApplyID,
		NodeID:           apply.NodeID,
		VersionInfo:      apply.VersionInfo,
		Nonce:            apply.Nonce,
		Status:           AckStatusACK,
		AppliedRevision:  apply.TargetRevision,
		LastGoodRevision: apply.TargetRevision,
	}

	for _, resource := range apply.Resources {
		if resource.Kind != ResourceKindProxyAccount {
			ack.Status = AckStatusPartial
			ack.ResourceResults = append(ack.ResourceResults, ResourceResult{Name: resource.Name, Status: ResourceResultFailed, ErrorDetail: "unsupported resource kind"})
			continue
		}
		account := accountFromResource(resource)
		if err := m.core.UpsertAccount(ctx, account); err != nil {
			ack.Status = AckStatusPartial
			ack.ResourceResults = append(ack.ResourceResults, ResourceResult{Name: resource.Name, Status: ResourceResultFailed, ErrorDetail: err.Error()})
			continue
		}
		ack.ResourceResults = append(ack.ResourceResults, ResourceResult{Name: resource.Name, Status: ResourceResultApplied})
	}
	for _, name := range apply.RemovedResourceNames {
		if err := m.core.DeleteAccount(ctx, proxyAccountIDFromResourceName(name)); err != nil {
			ack.Status = AckStatusPartial
			ack.ResourceResults = append(ack.ResourceResults, ResourceResult{Name: name, Status: ResourceResultFailed, ErrorDetail: err.Error()})
			continue
		}
		ack.ResourceResults = append(ack.ResourceResults, ResourceResult{Name: name, Status: ResourceResultRemoved})
	}

	digest, err := m.core.Digest(ctx)
	if err != nil {
		ack.Status = AckStatusNACK
		ack.ErrorDetail = err.Error()
		return ack, err
	}
	ack.Digest = digest
	if ack.Status == AckStatusPartial {
		ack.AppliedRevision = lastGood
		ack.LastGoodRevision = lastGood
		return ack, errors.New("runtime apply partially failed")
	}

	m.mu.Lock()
	m.lastGoodRevision = apply.TargetRevision
	m.lastSeenNonce = apply.Nonce
	m.lastVersionInfo = apply.VersionInfo
	ack.LastGoodRevision = m.lastGoodRevision
	m.seenApplies[apply.ApplyID] = ack
	m.mu.Unlock()
	return ack, nil
}

func (m *Manager) applyFairness(ctx context.Context, apply Apply, lastGood uint64) (ApplyAck, error) {
	ack := ApplyAck{
		ApplyID:          apply.ApplyID,
		NodeID:           apply.NodeID,
		VersionInfo:      apply.VersionInfo,
		Nonce:            apply.Nonce,
		Status:           AckStatusACK,
		AppliedRevision:  lastGood,
		LastGoodRevision: lastGood,
	}
	if err := m.core.SetFairnessState(ctx, apply.FairnessState); err != nil {
		ack.Status = AckStatusNACK
		ack.ErrorDetail = err.Error()
		return ack, err
	}
	if digest, err := m.core.Digest(ctx); err == nil {
		ack.Digest = digest
	}
	return ack, nil
}

func (m *Manager) applyQuery(ctx context.Context, apply Apply, lastGood uint64) (ApplyAck, error) {
	ack := ApplyAck{
		ApplyID:          apply.ApplyID,
		NodeID:           apply.NodeID,
		VersionInfo:      apply.VersionInfo,
		Nonce:            apply.Nonce,
		Status:           AckStatusACK,
		AppliedRevision:  lastGood,
		LastGoodRevision: lastGood,
	}
	switch apply.QueryOperation {
	case "GET_USAGE":
		proxyAccountID := proxyAccountIDFromResourceName(apply.QueryResourceName)
		usage, err := m.core.Usage(ctx, proxyAccountID)
		if err != nil {
			ack.Status = AckStatusNACK
			ack.ErrorDetail = err.Error()
			return ack, err
		}
		ack.Usage = usage
	default:
		ack.Status = AckStatusNACK
		ack.ErrorDetail = "unsupported query operation"
		return ack, errors.New(ack.ErrorDetail)
	}
	if digest, err := m.core.Digest(ctx); err == nil {
		ack.Digest = digest
	}
	return ack, nil
}

func (m *Manager) baseAckLocked(apply Apply) ApplyAck {
	return ApplyAck{
		ApplyID:          apply.ApplyID,
		NodeID:           apply.NodeID,
		VersionInfo:      apply.VersionInfo,
		Nonce:            apply.Nonce,
		LastGoodRevision: m.lastGoodRevision,
	}
}

func accountFromResource(resource Resource) Account {
	email := resource.RuntimeEmail
	if email == "" {
		email = resource.Name
	}
	priority := resource.Priority
	if priority == 0 {
		priority = 1
	}
	return Account{
		ProxyAccountID:    proxyAccountIDFromResourceName(resource.Name),
		RuntimeEmail:      email,
		Protocol:          resource.Protocol,
		ListenIP:          resource.ListenIP,
		Port:              resource.Port,
		Username:          resource.Username,
		Password:          resource.Password,
		ExpiresAtUnixMS:   resource.ExpiresAtUnixMS,
		EgressLimitBPS:    resource.EgressLimitBPS,
		IngressLimitBPS:   resource.IngressLimitBPS,
		MaxConnections:    resource.MaxConnections,
		Status:            AccountStatusEnabled,
		Priority:          priority,
		AbuseAction:       AbuseActionReportOnly,
		DesiredGeneration: resource.ResourceVersion,
	}
}

func proxyAccountIDFromResourceName(name string) string {
	const prefix = "proxy/"
	if len(name) > len(prefix) && name[:len(prefix)] == prefix {
		return name[len(prefix):]
	}
	return name
}
