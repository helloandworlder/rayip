package runtimecontrol

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rayip/rayip/services/api/internal/runtimelab"
)

type RuntimeDispatcher interface {
	DispatchRuntimeApply(ctx context.Context, apply runtimelab.RuntimeApply) (runtimelab.ApplyResult, error)
}

type Worker struct {
	service    *Service
	dispatcher RuntimeDispatcher
	now        func() time.Time
}

func NewWorker(service *Service, dispatcher RuntimeDispatcher, now func() time.Time) *Worker {
	if now == nil {
		now = time.Now
	}
	return &Worker{service: service, dispatcher: dispatcher, now: now}
}

func (w *Worker) ProcessNodeChanges(ctx context.Context, nodeID string, afterSeq uint64, limit int) (JobResult, error) {
	changes, err := w.service.ListChanges(ctx, nodeID, afterSeq, limit)
	if err != nil {
		return JobResult{}, err
	}
	if len(changes) == 0 {
		result := w.newJobResult(nodeID, JobStatusSucceeded, afterSeq, afterSeq)
		result.AcceptedRevision = afterSeq
		result.LastGoodRevision = afterSeq
		_ = w.service.SaveJobResult(ctx, result)
		return result, nil
	}

	targetRevision := changes[len(changes)-1].Seq
	apply := w.newDeltaApply(nodeID, afterSeq, targetRevision)
	for _, change := range changes {
		switch change.Action {
		case ChangeActionRemove:
			apply.RemovedResourceNames = append(apply.RemovedResourceNames, change.ResourceName)
		default:
			state, ok, err := w.service.GetResourceByName(ctx, change.ResourceName)
			if err != nil {
				return JobResult{}, err
			}
			if !ok || state.Removed {
				apply.RemovedResourceNames = append(apply.RemovedResourceNames, change.ResourceName)
				continue
			}
			apply.Resources = append(apply.Resources, resourceToRuntime(state))
		}
	}

	dispatchResult, err := w.dispatcher.DispatchRuntimeApply(ctx, apply)
	if dispatchResult.ApplyID == "" {
		dispatchResult.ApplyID = apply.ApplyID
	}
	result := w.jobResultFromApply(nodeID, afterSeq, targetRevision, apply, dispatchResult)
	if err != nil {
		result.Status = JobStatusRetryable
		result.ErrorDetail = err.Error()
		_ = w.service.SaveJobResult(ctx, result)
		return result, err
	}
	if dispatchResult.Status == runtimelab.ApplyStatusACK || dispatchResult.Status == runtimelab.ApplyStatusDuplicate {
		result.Status = JobStatusSucceeded
		result.AcceptedRevision = dispatchResult.AppliedRevision
	} else {
		result.Status = JobStatusFailed
		result.ErrorDetail = dispatchResult.ErrorDetail
	}
	if saveErr := w.service.SaveJobResult(ctx, result); saveErr != nil {
		return result, saveErr
	}
	return result, nil
}

func (w *Worker) newDeltaApply(nodeID string, baseRevision uint64, targetRevision uint64) runtimelab.RuntimeApply {
	applyID := uuid.NewString()
	return runtimelab.RuntimeApply{
		ApplyID:        applyID,
		NodeID:         nodeID,
		Mode:           runtimelab.ApplyModeDelta,
		VersionInfo:    fmt.Sprintf("revision/%d", targetRevision),
		Nonce:          uuid.NewString(),
		BaseRevision:   baseRevision,
		TargetRevision: targetRevision,
		DeadlineUnixMS: w.now().Add(8 * time.Second).UnixMilli(),
	}
}

func (w *Worker) newJobResult(nodeID string, status JobStatus, baseRevision uint64, targetRevision uint64) JobResult {
	now := w.now().UTC()
	return JobResult{
		JobID:          uuid.NewString(),
		NodeID:         nodeID,
		Status:         status,
		BaseRevision:   baseRevision,
		TargetRevision: targetRevision,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func (w *Worker) jobResultFromApply(nodeID string, baseRevision uint64, targetRevision uint64, apply runtimelab.RuntimeApply, dispatchResult runtimelab.ApplyResult) JobResult {
	result := w.newJobResult(nodeID, JobStatusPending, baseRevision, targetRevision)
	result.ApplyID = apply.ApplyID
	result.VersionInfo = apply.VersionInfo
	result.Nonce = apply.Nonce
	result.AcceptedRevision = dispatchResult.AppliedRevision
	result.LastGoodRevision = dispatchResult.LastGoodRevision
	result.ErrorDetail = dispatchResult.ErrorDetail
	return result
}

func resourceToRuntime(state ResourceState) runtimelab.RuntimeResource {
	expiresAt := int64(0)
	if !state.ExpiresAt.IsZero() {
		expiresAt = state.ExpiresAt.UnixMilli()
	}
	return runtimelab.RuntimeResource{
		Name:              state.ResourceName,
		Kind:              runtimelab.ResourceKindProxyAccount,
		ResourceVersion:   state.DesiredRevision,
		RuntimeEmail:      state.RuntimeEmail,
		Protocol:          runtimelab.Protocol(state.Protocol),
		ListenIP:          state.ListenIP,
		Port:              state.Port,
		Username:          state.Username,
		Password:          state.Password,
		EgressLimitBPS:    state.EgressLimitBPS,
		IngressLimitBPS:   state.IngressLimitBPS,
		MaxConnections:    state.MaxConnections,
		Priority:          state.Priority,
		AbuseReportPolicy: "REPORT_ONLY",
		ExpiresAtUnixMS:   expiresAt,
	}
}
