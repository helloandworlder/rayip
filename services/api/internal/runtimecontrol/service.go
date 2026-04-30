package runtimecontrol

import (
	"context"
	"errors"
	"time"
)

type Repository interface {
	UpsertResource(ctx context.Context, input ResourceInput, now time.Time) (MutationResult, error)
	RemoveResource(ctx context.Context, proxyAccountID string, now time.Time) (MutationResult, error)
	ListChanges(ctx context.Context, nodeID string, afterSeq uint64, limit int) ([]ChangeLogEntry, error)
	GetResourceByName(ctx context.Context, resourceName string) (ResourceState, bool, error)
	ListResourcesByNode(ctx context.Context, nodeID string, includeRemoved bool, offset int, limit int) ([]ResourceState, error)
	ListOutbox(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkOutboxPublished(ctx context.Context, eventID string, publishedAt time.Time) error
	SaveJobResult(ctx context.Context, result JobResult) error
}

type OutboxPublisher interface {
	PublishRuntimeApply(ctx context.Context, event OutboxEvent) error
}

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, now: now}
}

func (s *Service) UpsertProxyAccount(ctx context.Context, input ResourceInput) (MutationResult, error) {
	if input.ProxyAccountID == "" {
		return MutationResult{}, errors.New("proxy_account_id is required")
	}
	if input.NodeID == "" {
		return MutationResult{}, errors.New("node_id is required")
	}
	if input.Protocol != ProtocolSOCKS5 && input.Protocol != ProtocolHTTP {
		return MutationResult{}, errors.New("protocol must be SOCKS5 or HTTP")
	}
	if input.RuntimeEmail == "" {
		input.RuntimeEmail = input.ProxyAccountID
	}
	if input.ListenIP == "" {
		input.ListenIP = "127.0.0.1"
	}
	if input.Port == 0 {
		return MutationResult{}, errors.New("port is required")
	}
	if input.Username == "" || input.Password == "" {
		return MutationResult{}, errors.New("username and password are required")
	}
	if input.Priority == 0 {
		input.Priority = 1
	}
	return s.repo.UpsertResource(ctx, input, s.now().UTC())
}

func (s *Service) RemoveProxyAccount(ctx context.Context, proxyAccountID string) (MutationResult, error) {
	if proxyAccountID == "" {
		return MutationResult{}, errors.New("proxy_account_id is required")
	}
	return s.repo.RemoveResource(ctx, proxyAccountID, s.now().UTC())
}

func (s *Service) ListChanges(ctx context.Context, nodeID string, afterSeq uint64, limit int) ([]ChangeLogEntry, error) {
	if nodeID == "" {
		return nil, errors.New("node_id is required")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.ListChanges(ctx, nodeID, afterSeq, limit)
}

func (s *Service) ListOutbox(ctx context.Context, limit int) ([]OutboxEvent, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	return s.repo.ListOutbox(ctx, limit)
}

func (s *Service) PublishPendingOutbox(ctx context.Context, publisher OutboxPublisher, limit int) (int, error) {
	if publisher == nil {
		return 0, errors.New("publisher is required")
	}
	events, err := s.ListOutbox(ctx, limit)
	if err != nil {
		return 0, err
	}
	published := 0
	now := s.now().UTC()
	for _, event := range events {
		if !event.PublishedAt.IsZero() {
			continue
		}
		if err := publisher.PublishRuntimeApply(ctx, event); err != nil {
			return published, err
		}
		if err := s.repo.MarkOutboxPublished(ctx, event.ID, now); err != nil {
			return published, err
		}
		published++
	}
	return published, nil
}

func (s *Service) GetResourceByName(ctx context.Context, resourceName string) (ResourceState, bool, error) {
	if resourceName == "" {
		return ResourceState{}, false, errors.New("resource_name is required")
	}
	return s.repo.GetResourceByName(ctx, resourceName)
}

func (s *Service) ListResourcesByNode(ctx context.Context, nodeID string, includeRemoved bool, offset int, limit int) ([]ResourceState, error) {
	if nodeID == "" {
		return nil, errors.New("node_id is required")
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	return s.repo.ListResourcesByNode(ctx, nodeID, includeRemoved, offset, limit)
}

func (s *Service) SaveJobResult(ctx context.Context, result JobResult) error {
	if result.NodeID == "" {
		return errors.New("node_id is required")
	}
	if result.CreatedAt.IsZero() {
		result.CreatedAt = s.now().UTC()
	}
	if result.UpdatedAt.IsZero() {
		result.UpdatedAt = result.CreatedAt
	}
	return s.repo.SaveJobResult(ctx, result)
}
