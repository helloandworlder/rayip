package runtimecontrol

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rayip/rayip/services/api/internal/runtimelab"
)

type ReconcilePlanner struct {
	service *Service
	now     func() time.Time
}

func NewReconcilePlanner(service *Service, now func() time.Time) *ReconcilePlanner {
	if now == nil {
		now = time.Now
	}
	return &ReconcilePlanner{service: service, now: now}
}

func (p *ReconcilePlanner) BuildSnapshotApply(ctx context.Context, nodeID string, offset int, limit int, baseRevision uint64, targetRevision uint64) (runtimelab.RuntimeApply, error) {
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	states, err := p.service.ListResourcesByNode(ctx, nodeID, false, offset, limit)
	if err != nil {
		return runtimelab.RuntimeApply{}, err
	}
	resources := make([]runtimelab.RuntimeResource, 0, len(states))
	for _, state := range states {
		resources = append(resources, resourceToRuntime(state))
	}
	return runtimelab.RuntimeApply{
		ApplyID:        uuid.NewString(),
		NodeID:         nodeID,
		Mode:           runtimelab.ApplyModeSnapshot,
		VersionInfo:    fmt.Sprintf("revision/%d", targetRevision),
		Nonce:          uuid.NewString(),
		BaseRevision:   baseRevision,
		TargetRevision: targetRevision,
		DeadlineUnixMS: p.now().Add(30 * time.Second).UnixMilli(),
		Resources:      resources,
	}, nil
}
