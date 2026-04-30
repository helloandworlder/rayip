package httpapi

import (
	"context"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/rayip/rayip/services/api/internal/config"
	"github.com/rayip/rayip/services/api/internal/grpcapi"
	"github.com/rayip/rayip/services/api/internal/node"
	"github.com/rayip/rayip/services/api/internal/noderuntime"
	"github.com/rayip/rayip/services/api/internal/runtimelab"
)

type NodeAgentRoutesParams struct {
	Config      config.Config
	Nodes       *node.Service
	NodeRuntime *noderuntime.Service
	Runtime     *grpcapi.RuntimeDispatcher
	Lab         *runtimelab.Service
}

type nodeAgentHelloRequest struct {
	NodeCode        string                      `json:"node_code"`
	EnrollmentToken string                      `json:"enrollment_token"`
	AgentVersion    string                      `json:"agent_version"`
	XrayVersion     string                      `json:"xray_version"`
	BundleVersion   string                      `json:"bundle_version"`
	Capabilities    []string                    `json:"capabilities"`
	Sequence        uint64                      `json:"sequence"`
	LeaseTTLSeconds int                         `json:"lease_ttl_seconds"`
	Runtime         nodeAgentRuntimeObservation `json:"runtime"`
	Probe           nodeAgentProbeObservation   `json:"probe"`
}

type nodeAgentLeaseRequest struct {
	NodeID          string                      `json:"node_id"`
	NodeCode        string                      `json:"node_code"`
	SessionID       string                      `json:"session_id"`
	AgentVersion    string                      `json:"agent_version"`
	XrayVersion     string                      `json:"xray_version"`
	BundleVersion   string                      `json:"bundle_version"`
	Capabilities    []string                    `json:"capabilities"`
	Sequence        uint64                      `json:"sequence"`
	LeaseTTLSeconds int                         `json:"lease_ttl_seconds"`
	Runtime         nodeAgentRuntimeObservation `json:"runtime"`
	Probe           nodeAgentProbeObservation   `json:"probe"`
}

type nodeAgentRuntimeObservation struct {
	AgentVersion     string   `json:"agent_version"`
	XrayVersion      string   `json:"xray_version"`
	BundleVersion    string   `json:"bundle_version"`
	ExtensionABI     string   `json:"extension_abi"`
	Capabilities     []string `json:"capabilities"`
	BinarySHA256     string   `json:"binary_sha256"`
	ManifestSHA256   string   `json:"manifest_sha256"`
	RuntimeDigest    string   `json:"runtime_digest"`
	LastGoodRevision uint64   `json:"last_good_revision"`
}

type nodeAgentProbeObservation struct {
	PublicIP           string   `json:"public_ip"`
	ProbePort          uint32   `json:"probe_port"`
	ProbeProtocols     []string `json:"probe_protocols"`
	CheckedAtUnixMilli int64    `json:"checked_at_unix_milli"`
	ScanHost           string   `json:"scan_host"`
	CandidatePublicIPs []string `json:"candidate_public_ips"`
}

type nodeAgentWelcome struct {
	NodeID          string `json:"node_id"`
	SessionID       string `json:"session_id"`
	LeaseTTLSeconds int    `json:"lease_ttl_seconds"`
	APIInstanceID   string `json:"api_instance_id"`
}

func RegisterNodeAgentRoutes(app *fiber.App, p NodeAgentRoutesParams) {
	app.Post("/api/node-agent/hello", func(c fiber.Ctx) error {
		var req nodeAgentHelloRequest
		if err := c.Bind().Body(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if req.EnrollmentToken != p.Config.Node.EnrollmentToken {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid enrollment token")
		}
		sessionID := uuid.NewString()
		leaseTTL := req.LeaseTTLSeconds
		if leaseTTL <= 0 {
			leaseTTL = p.Config.Node.LeaseTTLSeconds
		}
		summary, err := p.Nodes.RegisterLease(c.Context(), node.LeaseInput{
			NodeCode:           req.NodeCode,
			SessionID:          sessionID,
			APIInstanceID:      p.Config.Service.InstanceID,
			BundleVersion:      firstNonEmpty(req.Runtime.BundleVersion, req.BundleVersion),
			AgentVersion:       firstNonEmpty(req.Runtime.AgentVersion, req.AgentVersion),
			XrayVersion:        firstNonEmpty(req.Runtime.XrayVersion, req.XrayVersion),
			Capabilities:       firstNonNil(req.Runtime.Capabilities, req.Capabilities),
			PublicIP:           req.Probe.PublicIP,
			CandidatePublicIPs: req.Probe.CandidatePublicIPs,
			ScanHost:           req.Probe.ScanHost,
			ProbePort:          req.Probe.ProbePort,
			ProbeProtocols:     req.Probe.ProbeProtocols,
			ProbeCheckedAt:     unixMilli(req.Probe.CheckedAtUnixMilli),
			Sequence:           req.Sequence,
			LeaseTTLSeconds:    leaseTTL,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		recordRuntimeObservation(c.Context(), p.NodeRuntime, summary.ID, req.Runtime, req.Probe.CandidatePublicIPs)
		p.Runtime.RegisterHTTP(summary.ID)
		return c.JSON(nodeAgentWelcome{
			NodeID:          summary.ID,
			SessionID:       sessionID,
			LeaseTTLSeconds: leaseTTL,
			APIInstanceID:   p.Config.Service.InstanceID,
		})
	})

	app.Post("/api/node-agent/lease", func(c fiber.Ctx) error {
		var req nodeAgentLeaseRequest
		if err := c.Bind().Body(&req); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if req.NodeID == "" || req.SessionID == "" {
			return fiber.NewError(fiber.StatusBadRequest, "node_id and session_id are required")
		}
		leaseTTL := req.LeaseTTLSeconds
		if leaseTTL <= 0 {
			leaseTTL = p.Config.Node.LeaseTTLSeconds
		}
		_, err := p.Nodes.RegisterLease(c.Context(), node.LeaseInput{
			NodeID:             req.NodeID,
			NodeCode:           req.NodeCode,
			SessionID:          req.SessionID,
			APIInstanceID:      p.Config.Service.InstanceID,
			BundleVersion:      firstNonEmpty(req.Runtime.BundleVersion, req.BundleVersion),
			AgentVersion:       firstNonEmpty(req.Runtime.AgentVersion, req.AgentVersion),
			XrayVersion:        firstNonEmpty(req.Runtime.XrayVersion, req.XrayVersion),
			Capabilities:       firstNonNil(req.Runtime.Capabilities, req.Capabilities),
			PublicIP:           req.Probe.PublicIP,
			CandidatePublicIPs: req.Probe.CandidatePublicIPs,
			ScanHost:           req.Probe.ScanHost,
			ProbePort:          req.Probe.ProbePort,
			ProbeProtocols:     req.Probe.ProbeProtocols,
			ProbeCheckedAt:     unixMilli(req.Probe.CheckedAtUnixMilli),
			Sequence:           req.Sequence,
			LeaseTTLSeconds:    leaseTTL,
		})
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		recordRuntimeObservation(c.Context(), p.NodeRuntime, req.NodeID, req.Runtime, req.Probe.CandidatePublicIPs)
		return c.JSON(fiber.Map{"ok": true})
	})

	app.Get("/api/node-agent/nodes/:id/poll", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 25*time.Second)
		defer cancel()
		apply, ok, err := p.Runtime.PollHTTP(ctx, c.Params("id"))
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return fiber.NewError(fiber.StatusNotFound, err.Error())
		}
		if !ok || err != nil {
			return c.JSON(fiber.Map{"runtime_apply": nil})
		}
		return c.JSON(fiber.Map{"runtime_apply": apply})
	})

	app.Post("/api/node-agent/runtime-apply-ack", func(c fiber.Ctx) error {
		var result runtimelab.ApplyResult
		if err := c.Bind().Body(&result); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if result.CreatedAt.IsZero() {
			result.CreatedAt = time.Now().UTC()
		}
		p.Runtime.HandleResult(result)
		if p.Lab != nil {
			_ = p.Lab.SaveApplyResult(c.Context(), result)
		}
		_, _ = p.NodeRuntime.RecordRuntimeAck(c.Context(), noderuntime.RuntimeAckInput{
			NodeID:           result.NodeID,
			Status:           string(result.Status),
			AppliedRevision:  result.AppliedRevision,
			LastGoodRevision: result.LastGoodRevision,
			DigestHash:       result.Digest.Hash,
			AccountCount:     result.Digest.AccountCount,
		})
		return c.JSON(fiber.Map{"ok": true})
	})
}

func recordRuntimeObservation(ctx context.Context, service *noderuntime.Service, nodeID string, obs nodeAgentRuntimeObservation, candidates []string) {
	_, _ = service.UpsertStatus(ctx, noderuntime.StatusInput{
		NodeID:             nodeID,
		LeaseOnline:        true,
		RuntimeVerdict:     noderuntime.RuntimeVerdictAccepted,
		ExpectedRevision:   obs.LastGoodRevision,
		CurrentRevision:    obs.LastGoodRevision,
		LastGoodRevision:   obs.LastGoodRevision,
		ExpectedDigestHash: obs.RuntimeDigest,
		RuntimeDigestHash:  obs.RuntimeDigest,
		Capabilities:       obs.Capabilities,
		CandidatePublicIPs: candidates,
		ManifestHash:       obs.ManifestSHA256,
		BinaryHash:         obs.BinarySHA256,
		ExtensionABI:       obs.ExtensionABI,
		BundleChannel:      obs.BundleVersion,
	})
}

func unixMilli(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(value).UTC()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonNil(primary []string, fallback []string) []string {
	if primary != nil {
		return append([]string(nil), primary...)
	}
	return append([]string(nil), fallback...)
}
