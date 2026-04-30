package noderuntime

import (
	"context"
	"errors"
	"time"
)

type Repository interface {
	UpsertStatus(ctx context.Context, status Status) (Status, error)
	GetStatus(ctx context.Context, nodeID string) (Status, bool, error)
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

func (s *Service) UpsertStatus(ctx context.Context, input StatusInput) (Status, error) {
	if input.NodeID == "" {
		return Status{}, errors.New("node_id is required")
	}
	if current, ok, err := s.repo.GetStatus(ctx, input.NodeID); err != nil {
		return Status{}, err
	} else if ok {
		input.ExpectedRevision = maxUint64(input.ExpectedRevision, current.ExpectedRevision)
		input.CurrentRevision = maxUint64(input.CurrentRevision, current.CurrentRevision)
		input.LastGoodRevision = maxUint64(input.LastGoodRevision, current.LastGoodRevision)
		if input.ExpectedDigestHash == "" {
			input.ExpectedDigestHash = current.ExpectedDigestHash
		}
		if input.AccountCount == 0 && input.RuntimeDigestHash != "" && input.RuntimeDigestHash == current.RuntimeDigestHash {
			input.AccountCount = current.AccountCount
		}
		input.ManualHold = input.ManualHold || current.ManualHold
		input.ComplianceHold = input.ComplianceHold || current.ComplianceHold
	}
	status := Status{
		NodeID:             input.NodeID,
		LeaseOnline:        input.LeaseOnline,
		RuntimeVerdict:     input.RuntimeVerdict,
		ExpectedRevision:   input.ExpectedRevision,
		CurrentRevision:    input.CurrentRevision,
		LastGoodRevision:   input.LastGoodRevision,
		ExpectedDigestHash: input.ExpectedDigestHash,
		RuntimeDigestHash:  input.RuntimeDigestHash,
		AccountCount:       input.AccountCount,
		Capabilities:       append([]string(nil), input.Capabilities...),
		CandidatePublicIPs: append([]string(nil), input.CandidatePublicIPs...),
		ManifestHash:       input.ManifestHash,
		BinaryHash:         input.BinaryHash,
		ExtensionABI:       input.ExtensionABI,
		BundleChannel:      input.BundleChannel,
		ManualHold:         input.ManualHold,
		ComplianceHold:     input.ComplianceHold,
		UpdatedAt:          s.now().UTC(),
	}
	status.UnsellableReasons = evaluateUnsellableReasons(input)
	status.Sellable = len(status.UnsellableReasons) == 0
	return s.repo.UpsertStatus(ctx, status)
}

func (s *Service) GetStatus(ctx context.Context, nodeID string) (Status, bool, error) {
	if nodeID == "" {
		return Status{}, false, errors.New("node_id is required")
	}
	return s.repo.GetStatus(ctx, nodeID)
}

func (s *Service) RecordRuntimeAck(ctx context.Context, input RuntimeAckInput) (Status, error) {
	if input.NodeID == "" {
		return Status{}, errors.New("node_id is required")
	}
	current, ok, err := s.repo.GetStatus(ctx, input.NodeID)
	if err != nil {
		return Status{}, err
	}
	if !ok {
		current = Status{
			NodeID:         input.NodeID,
			LeaseOnline:    true,
			RuntimeVerdict: RuntimeVerdictAccepted,
		}
	}
	current.CurrentRevision = input.AppliedRevision
	current.LastGoodRevision = input.LastGoodRevision
	current.RuntimeDigestHash = input.DigestHash
	current.AccountCount = input.AccountCount
	if input.AppliedRevision > current.ExpectedRevision {
		current.ExpectedRevision = input.AppliedRevision
	}
	if current.ExpectedDigestHash == "" {
		current.ExpectedDigestHash = input.DigestHash
	}
	current.UpdatedAt = s.now().UTC()
	current.UnsellableReasons = evaluateUnsellableReasons(StatusInput{
		NodeID:               current.NodeID,
		LeaseOnline:          current.LeaseOnline,
		RuntimeVerdict:       current.RuntimeVerdict,
		ExpectedRevision:     current.ExpectedRevision,
		CurrentRevision:      current.CurrentRevision,
		LastGoodRevision:     current.LastGoodRevision,
		ExpectedDigestHash:   current.ExpectedDigestHash,
		RuntimeDigestHash:    current.RuntimeDigestHash,
		Capabilities:         current.Capabilities,
		RequiredCapabilities: current.Capabilities,
		ManualHold:           current.ManualHold,
		ComplianceHold:       current.ComplianceHold,
	})
	current.Sellable = len(current.UnsellableReasons) == 0
	return s.repo.UpsertStatus(ctx, current)
}

func evaluateUnsellableReasons(input StatusInput) []UnsellableReason {
	reasons := []UnsellableReason{}
	if !input.LeaseOnline {
		reasons = append(reasons, UnsellableOffline)
	}
	switch input.RuntimeVerdict {
	case RuntimeVerdictAccepted:
	case RuntimeVerdictDraining:
		reasons = append(reasons, UnsellableDraining)
	case RuntimeVerdictQuarantined:
		reasons = append(reasons, UnsellableQuarantined)
	case RuntimeVerdictNeedsUpgrade:
		reasons = append(reasons, UnsellableNeedsUpgrade)
	default:
		reasons = append(reasons, UnsellableDegraded)
	}
	if missingCapability(input.Capabilities, input.RequiredCapabilities) {
		reasons = append(reasons, UnsellableUnsupportedCapability)
	}
	if input.ExpectedDigestHash != "" && input.RuntimeDigestHash != "" && input.ExpectedDigestHash != input.RuntimeDigestHash {
		reasons = append(reasons, UnsellableDigestMismatch)
	}
	if input.CurrentRevision < input.ExpectedRevision || input.LastGoodRevision < input.ExpectedRevision {
		reasons = append(reasons, UnsellableRuntimeLagging)
	}
	if len(input.CandidatePublicIPs) == 0 {
		reasons = append(reasons, UnsellableNoCandidatePublicIP)
	}
	if input.ManualHold {
		reasons = append(reasons, UnsellableManualHold)
	}
	if input.ComplianceHold {
		reasons = append(reasons, UnsellableComplianceHold)
	}
	return reasons
}

func missingCapability(actual []string, required []string) bool {
	if len(required) == 0 {
		return false
	}
	set := map[string]struct{}{}
	for _, capability := range actual {
		set[capability] = struct{}{}
	}
	for _, capability := range required {
		if _, ok := set[capability]; !ok {
			return true
		}
	}
	return false
}

func maxUint64(a uint64, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
