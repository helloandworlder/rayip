package node

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	apiEnt "github.com/rayip/rayip/services/api/ent"
	entNode "github.com/rayip/rayip/services/api/ent/node"
)

type EntRepository struct {
	client *apiEnt.Client
}

func NewEntRepository(client *apiEnt.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) UpsertLease(ctx context.Context, input LeaseInput, now time.Time) (NodeRecord, error) {
	capabilities, capabilitiesHash, err := normalizeCapabilities(input.Capabilities)
	if err != nil {
		return NodeRecord{}, err
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return NodeRecord{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	record, err := tx.Node.Query().Where(entNode.Code(input.NodeCode)).Only(ctx)
	if apiEnt.IsNotFound(err) {
		record, err = tx.Node.Create().
			SetID(firstNonEmpty(input.NodeID, uuid.NewString())).
			SetCode(input.NodeCode).
			SetStatus(string(StatusOnline)).
			SetBundleVersion(input.BundleVersion).
			SetAgentVersion(input.AgentVersion).
			SetXrayVersion(input.XrayVersion).
			SetCapabilities(capabilities).
			SetPublicIP(input.PublicIP).
			SetCandidatePublicIps(input.CandidatePublicIPs).
			SetScanHost(input.ScanHost).
			SetProbePort(input.ProbePort).
			SetProbeProtocols(input.ProbeProtocols).
			SetLastOnlineAt(now).
			SetCreatedAt(now).
			SetUpdatedAt(now).
			Save(ctx)
		if err == nil && !input.ProbeCheckedAt.IsZero() {
			record, err = tx.Node.UpdateOneID(record.ID).SetProbeCheckedAt(input.ProbeCheckedAt).Save(ctx)
		}
	}
	if err != nil {
		return NodeRecord{}, err
	}
	if record.Code == input.NodeCode && !record.CreatedAt.Equal(now) {
		update := tx.Node.UpdateOneID(record.ID).
			SetStatus(string(StatusOnline)).
			SetBundleVersion(input.BundleVersion).
			SetAgentVersion(input.AgentVersion).
			SetXrayVersion(input.XrayVersion).
			SetCapabilities(capabilities).
			SetPublicIP(input.PublicIP).
			SetCandidatePublicIps(input.CandidatePublicIPs).
			SetScanHost(input.ScanHost).
			SetProbePort(input.ProbePort).
			SetProbeProtocols(input.ProbeProtocols).
			SetLastOnlineAt(now).
			SetUpdatedAt(now)
		if !input.ProbeCheckedAt.IsZero() {
			update.SetProbeCheckedAt(input.ProbeCheckedAt)
		}
		record, err = update.Save(ctx)
		if err != nil {
			return NodeRecord{}, err
		}
	}

	if err := tx.NodeAgentSession.Create().
		SetID(input.SessionID).
		SetNodeID(record.ID).
		SetAPIInstanceID(input.APIInstanceID).
		SetStatus("CONNECTED").
		SetBundleVersion(input.BundleVersion).
		SetConnectedAt(now).
		SetLastSeenAt(now).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		OnConflict(sql.ConflictColumns("session_id")).
		UpdateNewValues().
		Exec(ctx); err != nil {
		return NodeRecord{}, err
	}

	if err := tx.NodeCapabilitySnapshot.Create().
		SetID(uuid.NewString()).
		SetNodeID(record.ID).
		SetBundleVersion(input.BundleVersion).
		SetAgentVersion(input.AgentVersion).
		SetXrayVersion(input.XrayVersion).
		SetCapabilities(capabilities).
		SetCapabilitiesHash(capabilitiesHash).
		SetCapturedAt(now).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		OnConflict(sql.ConflictColumns("node_id", "bundle_version", "agent_version", "xray_version", "capabilities_hash")).
		Update(func(u *apiEnt.NodeCapabilitySnapshotUpsert) {
			u.SetCapturedAt(now)
			u.SetUpdatedAt(now)
		}).
		Exec(ctx); err != nil {
		return NodeRecord{}, err
	}

	if err := tx.Commit(); err != nil {
		return NodeRecord{}, err
	}
	committed = true
	return nodeRecordFromEnt(record), nil
}

func (r *EntRepository) Get(ctx context.Context, nodeID string) (NodeRecord, bool, error) {
	item, err := r.client.Node.Get(ctx, nodeID)
	if apiEnt.IsNotFound(err) {
		return NodeRecord{}, false, nil
	}
	if err != nil {
		return NodeRecord{}, false, err
	}
	return nodeRecordFromEnt(item), true, nil
}

func (r *EntRepository) List(ctx context.Context) ([]NodeRecord, error) {
	items, err := r.client.Node.Query().Order(apiEnt.Asc(entNode.FieldCode)).All(ctx)
	if err != nil {
		return nil, err
	}
	records := make([]NodeRecord, 0, len(items))
	for _, item := range items {
		records = append(records, nodeRecordFromEnt(item))
	}
	return records, nil
}

func (r *EntRepository) SaveScanResult(ctx context.Context, nodeID string, result ScanResult) error {
	update := r.client.Node.UpdateOneID(nodeID).
		SetLastScanStatus(result.Status).
		SetLastScanError(result.Error).
		SetLastScanReasonCode(string(result.ReasonCode)).
		SetLastScanLatencyMs(result.LatencyMs).
		SetUpdatedAt(result.ScannedAt)
	if !result.ScannedAt.IsZero() {
		update.SetLastScanAt(result.ScannedAt)
	}
	return update.Exec(ctx)
}

func nodeRecordFromEnt(item *apiEnt.Node) NodeRecord {
	lastOnlineAt := time.Time{}
	if item.LastOnlineAt != nil {
		lastOnlineAt = *item.LastOnlineAt
	}
	record := NodeRecord{
		ID:                 item.ID,
		Code:               item.Code,
		BundleVersion:      item.BundleVersion,
		AgentVersion:       item.AgentVersion,
		XrayVersion:        item.XrayVersion,
		Capabilities:       append([]string(nil), item.Capabilities...),
		PublicIP:           item.PublicIP,
		CandidatePublicIPs: append([]string(nil), item.CandidatePublicIps...),
		ScanHost:           item.ScanHost,
		ProbePort:          item.ProbePort,
		ProbeProtocols:     append([]string(nil), item.ProbeProtocols...),
		LastScanStatus:     item.LastScanStatus,
		LastScanError:      item.LastScanError,
		LastScanReasonCode: ScanReasonCode(item.LastScanReasonCode),
		LastScanLatency:    time.Duration(item.LastScanLatencyMs) * time.Millisecond,
		LastOnlineAt:       lastOnlineAt,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
	}
	if item.ProbeCheckedAt != nil {
		record.ProbeCheckedAt = *item.ProbeCheckedAt
	}
	if item.LastScanAt != nil {
		record.LastScanAt = *item.LastScanAt
	}
	return record
}

func normalizeCapabilities(capabilities []string) ([]string, string, error) {
	if capabilities == nil {
		capabilities = []string{}
	}
	payload, err := json.Marshal(capabilities)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(payload)
	return capabilities, hex.EncodeToString(sum[:]), nil
}
