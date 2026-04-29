package node

import (
	"context"
	"errors"
	"time"
)

type Repository interface {
	UpsertLease(ctx context.Context, input LeaseInput, now time.Time) (NodeRecord, error)
	List(ctx context.Context) ([]NodeRecord, error)
}

type LeaseStore interface {
	PutLease(ctx context.Context, lease LeaseSnapshot, ttl time.Duration) error
	GetLease(ctx context.Context, nodeID string) (LeaseSnapshot, bool, error)
}

type Service struct {
	repo   Repository
	leases LeaseStore
	now    func() time.Time
}

func NewService(repo Repository, leases LeaseStore, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, leases: leases, now: now}
}

func (s *Service) RegisterLease(ctx context.Context, input LeaseInput) (Summary, error) {
	if input.NodeCode == "" {
		return Summary{}, errors.New("node code is required")
	}
	if input.SessionID == "" {
		return Summary{}, errors.New("session id is required")
	}
	if input.LeaseTTLSeconds <= 0 {
		input.LeaseTTLSeconds = 45
	}

	now := s.now().UTC()
	record, err := s.repo.UpsertLease(ctx, input, now)
	if err != nil {
		return Summary{}, err
	}

	lease := LeaseSnapshot{
		NodeID:          record.ID,
		NodeCode:        record.Code,
		SessionID:       input.SessionID,
		APIInstanceID:   input.APIInstanceID,
		BundleVersion:   input.BundleVersion,
		AgentVersion:    input.AgentVersion,
		XrayVersion:     input.XrayVersion,
		Capabilities:    append([]string(nil), input.Capabilities...),
		Sequence:        input.Sequence,
		RenewedAt:       now,
		ExpiresAt:       now.Add(time.Duration(input.LeaseTTLSeconds) * time.Second),
		LeaseTTLSeconds: input.LeaseTTLSeconds,
	}
	if err := s.leases.PutLease(ctx, lease, time.Duration(input.LeaseTTLSeconds)*time.Second); err != nil {
		return Summary{}, err
	}

	return summaryFrom(record, lease, now), nil
}

func (s *Service) ListNodes(ctx context.Context) ([]Summary, error) {
	records, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	summaries := make([]Summary, 0, len(records))
	for _, record := range records {
		lease, ok, err := s.leases.GetLease(ctx, record.ID)
		if err != nil {
			return nil, err
		}
		if !ok {
			lease = LeaseSnapshot{
				NodeID:        record.ID,
				NodeCode:      record.Code,
				BundleVersion: record.BundleVersion,
				AgentVersion:  record.AgentVersion,
				XrayVersion:   record.XrayVersion,
				Capabilities:  record.Capabilities,
			}
		}
		summaries = append(summaries, summaryFrom(record, lease, now))
	}
	return summaries, nil
}

func summaryFrom(record NodeRecord, lease LeaseSnapshot, now time.Time) Summary {
	status := StatusOffline
	if !lease.ExpiresAt.IsZero() && lease.ExpiresAt.After(now) {
		status = StatusOnline
	}
	return Summary{
		ID:             record.ID,
		Code:           record.Code,
		Status:         status,
		LastOnlineAt:   record.LastOnlineAt,
		BundleVersion:  firstNonEmpty(lease.BundleVersion, record.BundleVersion),
		AgentVersion:   firstNonEmpty(lease.AgentVersion, record.AgentVersion),
		XrayVersion:    firstNonEmpty(lease.XrayVersion, record.XrayVersion),
		APIInstanceID:  lease.APIInstanceID,
		SessionID:      lease.SessionID,
		Capabilities:   append([]string(nil), firstNonNil(lease.Capabilities, record.Capabilities)...),
		LeaseExpiresAt: lease.ExpiresAt,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonNil[T any](values ...[]T) []T {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
