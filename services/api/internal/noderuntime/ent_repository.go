package noderuntime

import (
	"context"

	"entgo.io/ent/dialect/sql"
	apiEnt "github.com/rayip/rayip/services/api/ent"
)

type EntRepository struct {
	client *apiEnt.Client
}

func NewEntRepository(client *apiEnt.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) UpsertStatus(ctx context.Context, status Status) (Status, error) {
	reasons := make([]string, 0, len(status.UnsellableReasons))
	for _, reason := range status.UnsellableReasons {
		reasons = append(reasons, string(reason))
	}
	err := r.client.NodeRuntimeStatus.Create().
		SetID(status.NodeID).
		SetLeaseOnline(status.LeaseOnline).
		SetRuntimeVerdict(string(status.RuntimeVerdict)).
		SetExpectedRevision(status.ExpectedRevision).
		SetCurrentRevision(status.CurrentRevision).
		SetLastGoodRevision(status.LastGoodRevision).
		SetExpectedDigestHash(status.ExpectedDigestHash).
		SetRuntimeDigestHash(status.RuntimeDigestHash).
		SetAccountCount(status.AccountCount).
		SetCapabilities(status.Capabilities).
		SetManifestHash(status.ManifestHash).
		SetBinaryHash(status.BinaryHash).
		SetExtensionAbi(status.ExtensionABI).
		SetBundleChannel(status.BundleChannel).
		SetManualHold(status.ManualHold).
		SetComplianceHold(status.ComplianceHold).
		SetSellable(status.Sellable).
		SetUnsellableReasons(reasons).
		SetUpdatedAt(status.UpdatedAt).
		OnConflict(sql.ConflictColumns("node_id")).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return Status{}, err
	}
	item, err := r.client.NodeRuntimeStatus.Get(ctx, status.NodeID)
	if err != nil {
		return Status{}, err
	}
	return statusFromEnt(item), nil
}

func (r *EntRepository) GetStatus(ctx context.Context, nodeID string) (Status, bool, error) {
	item, err := r.client.NodeRuntimeStatus.Get(ctx, nodeID)
	if apiEnt.IsNotFound(err) {
		return Status{}, false, nil
	}
	if err != nil {
		return Status{}, false, err
	}
	return statusFromEnt(item), true, nil
}

func statusFromEnt(item *apiEnt.NodeRuntimeStatus) Status {
	reasons := make([]UnsellableReason, 0, len(item.UnsellableReasons))
	for _, reason := range item.UnsellableReasons {
		reasons = append(reasons, UnsellableReason(reason))
	}
	return Status{
		NodeID:             item.ID,
		LeaseOnline:        item.LeaseOnline,
		RuntimeVerdict:     RuntimeVerdict(item.RuntimeVerdict),
		ExpectedRevision:   item.ExpectedRevision,
		CurrentRevision:    item.CurrentRevision,
		LastGoodRevision:   item.LastGoodRevision,
		ExpectedDigestHash: item.ExpectedDigestHash,
		RuntimeDigestHash:  item.RuntimeDigestHash,
		AccountCount:       item.AccountCount,
		Capabilities:       item.Capabilities,
		ManifestHash:       item.ManifestHash,
		BinaryHash:         item.BinaryHash,
		ExtensionABI:       item.ExtensionAbi,
		BundleChannel:      item.BundleChannel,
		ManualHold:         item.ManualHold,
		ComplianceHold:     item.ComplianceHold,
		Sellable:           item.Sellable,
		UnsellableReasons:  reasons,
		UpdatedAt:          item.UpdatedAt,
	}
}

var _ Repository = (*EntRepository)(nil)
