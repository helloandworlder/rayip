package node

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) UpsertLease(ctx context.Context, input LeaseInput, now time.Time) (NodeRecord, error) {
	capabilitiesJSON, capabilitiesHash, err := marshalCapabilities(input.Capabilities)
	if err != nil {
		return NodeRecord{}, err
	}

	var model nodeModel
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("code = ?", input.NodeCode).First(&model).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			model = nodeModel{
				ID:               firstNonEmpty(input.NodeID, uuid.NewString()),
				Code:             input.NodeCode,
				Status:           string(StatusOnline),
				BundleVersion:    input.BundleVersion,
				AgentVersion:     input.AgentVersion,
				XrayVersion:      input.XrayVersion,
				CapabilitiesJSON: string(capabilitiesJSON),
				LastOnlineAt:     &now,
				CreatedAt:        now,
				UpdatedAt:        now,
			}
			if err := tx.Create(&model).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
			updates := map[string]any{
				"status":         string(StatusOnline),
				"bundle_version": input.BundleVersion,
				"agent_version":  input.AgentVersion,
				"xray_version":   input.XrayVersion,
				"capabilities":   string(capabilitiesJSON),
				"last_online_at": now,
				"updated_at":     now,
			}
			if err := tx.Model(&model).Updates(updates).Error; err != nil {
				return err
			}
			if err := tx.Where("code = ?", input.NodeCode).First(&model).Error; err != nil {
				return err
			}
		}

		session := nodeAgentSessionModel{
			SessionID:     input.SessionID,
			NodeID:        model.ID,
			APIInstanceID: input.APIInstanceID,
			Status:        "CONNECTED",
			BundleVersion: input.BundleVersion,
			ConnectedAt:   now,
			LastSeenAt:    now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "session_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"node_id":         model.ID,
				"api_instance_id": input.APIInstanceID,
				"status":          "CONNECTED",
				"bundle_version":  input.BundleVersion,
				"last_seen_at":    now,
				"updated_at":      now,
			}),
		}).Create(&session).Error; err != nil {
			return err
		}

		snapshot := nodeCapabilitySnapshotModel{
			ID:               uuid.NewString(),
			NodeID:           model.ID,
			BundleVersion:    input.BundleVersion,
			AgentVersion:     input.AgentVersion,
			XrayVersion:      input.XrayVersion,
			CapabilitiesJSON: string(capabilitiesJSON),
			CapabilitiesHash: capabilitiesHash,
			CapturedAt:       now,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&snapshot).Error
	})
	if err != nil {
		return NodeRecord{}, err
	}

	return model.toRecord()
}

func (r *GormRepository) List(ctx context.Context) ([]NodeRecord, error) {
	var models []nodeModel
	if err := r.db.WithContext(ctx).Order("code ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	records := make([]NodeRecord, 0, len(models))
	for _, model := range models {
		record, err := model.toRecord()
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, nil
}

type nodeModel struct {
	ID               string     `gorm:"column:id;type:uuid;primaryKey"`
	Code             string     `gorm:"column:code"`
	Status           string     `gorm:"column:status"`
	BundleVersion    string     `gorm:"column:bundle_version"`
	AgentVersion     string     `gorm:"column:agent_version"`
	XrayVersion      string     `gorm:"column:xray_version"`
	CapabilitiesJSON string     `gorm:"column:capabilities;type:jsonb"`
	LastOnlineAt     *time.Time `gorm:"column:last_online_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (nodeModel) TableName() string { return "nodes" }

func (m nodeModel) toRecord() (NodeRecord, error) {
	var capabilities []string
	if m.CapabilitiesJSON != "" {
		if err := json.Unmarshal([]byte(m.CapabilitiesJSON), &capabilities); err != nil {
			return NodeRecord{}, err
		}
	}

	lastOnlineAt := time.Time{}
	if m.LastOnlineAt != nil {
		lastOnlineAt = *m.LastOnlineAt
	}
	return NodeRecord{
		ID:            m.ID,
		Code:          m.Code,
		BundleVersion: m.BundleVersion,
		AgentVersion:  m.AgentVersion,
		XrayVersion:   m.XrayVersion,
		Capabilities:  capabilities,
		LastOnlineAt:  lastOnlineAt,
		CreatedAt:     m.CreatedAt,
		UpdatedAt:     m.UpdatedAt,
	}, nil
}

type nodeAgentSessionModel struct {
	SessionID     string    `gorm:"column:session_id;primaryKey"`
	NodeID        string    `gorm:"column:node_id;type:uuid"`
	APIInstanceID string    `gorm:"column:api_instance_id"`
	Status        string    `gorm:"column:status"`
	BundleVersion string    `gorm:"column:bundle_version"`
	ConnectedAt   time.Time `gorm:"column:connected_at"`
	LastSeenAt    time.Time `gorm:"column:last_seen_at"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (nodeAgentSessionModel) TableName() string { return "node_agent_sessions" }

type nodeCapabilitySnapshotModel struct {
	ID               string    `gorm:"column:id;type:uuid;primaryKey"`
	NodeID           string    `gorm:"column:node_id;type:uuid"`
	BundleVersion    string    `gorm:"column:bundle_version"`
	AgentVersion     string    `gorm:"column:agent_version"`
	XrayVersion      string    `gorm:"column:xray_version"`
	CapabilitiesJSON string    `gorm:"column:capabilities;type:jsonb"`
	CapabilitiesHash string    `gorm:"column:capabilities_hash"`
	CapturedAt       time.Time `gorm:"column:captured_at"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (nodeCapabilitySnapshotModel) TableName() string { return "node_capability_snapshots" }

func marshalCapabilities(capabilities []string) ([]byte, string, error) {
	if capabilities == nil {
		capabilities = []string{}
	}
	payload, err := json.Marshal(capabilities)
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(payload)
	return payload, hex.EncodeToString(sum[:]), nil
}
