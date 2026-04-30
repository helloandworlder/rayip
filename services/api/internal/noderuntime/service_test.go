package noderuntime_test

import (
	"context"
	"testing"
	"time"

	"github.com/rayip/rayip/services/api/internal/noderuntime"
)

func TestSellableGateAllowsAcceptedAlignedCapableNode(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	svc := noderuntime.NewService(noderuntime.NewMemoryRepository(), func() time.Time { return now })
	status, err := svc.UpsertStatus(context.Background(), noderuntime.StatusInput{
		NodeID:             "node-1",
		LeaseOnline:        true,
		RuntimeVerdict:     noderuntime.RuntimeVerdictAccepted,
		ExpectedRevision:   12,
		CurrentRevision:    12,
		LastGoodRevision:   12,
		ExpectedDigestHash: "digest-a",
		RuntimeDigestHash:  "digest-a",
		Capabilities:       []string{"PROXY_ACCOUNT", "SOCKS5", "HTTP"},
		CandidatePublicIPs: []string{"204.42.251.2"},
		RequiredCapabilities: []string{
			"PROXY_ACCOUNT",
			"SOCKS5",
		},
		ManifestHash:  "manifest-a",
		BinaryHash:    "binary-a",
		ExtensionABI:  "rayip-runtime-v1",
		BundleChannel: "stable",
	})
	if err != nil {
		t.Fatalf("UpsertStatus() error = %v", err)
	}
	if !status.Sellable || len(status.UnsellableReasons) != 0 {
		t.Fatalf("status = %#v", status)
	}
}

func TestSellableGateBlocksDigestMismatchAndRuntimeLag(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	svc := noderuntime.NewService(noderuntime.NewMemoryRepository(), func() time.Time { return now })
	status, err := svc.UpsertStatus(context.Background(), noderuntime.StatusInput{
		NodeID:             "node-1",
		LeaseOnline:        true,
		RuntimeVerdict:     noderuntime.RuntimeVerdictAccepted,
		ExpectedRevision:   12,
		CurrentRevision:    10,
		LastGoodRevision:   10,
		ExpectedDigestHash: "digest-a",
		RuntimeDigestHash:  "digest-b",
		Capabilities:       []string{"PROXY_ACCOUNT", "SOCKS5"},
		CandidatePublicIPs: []string{"204.42.251.2"},
		RequiredCapabilities: []string{
			"PROXY_ACCOUNT",
			"SOCKS5",
		},
		ManifestHash:  "manifest-a",
		BinaryHash:    "binary-a",
		ExtensionABI:  "rayip-runtime-v1",
		BundleChannel: "stable",
	})
	if err != nil {
		t.Fatalf("UpsertStatus() error = %v", err)
	}
	if status.Sellable {
		t.Fatalf("digest mismatch node should not be sellable: %#v", status)
	}
	assertHasReason(t, status.UnsellableReasons, noderuntime.UnsellableDigestMismatch)
	assertHasReason(t, status.UnsellableReasons, noderuntime.UnsellableRuntimeLagging)
}

func TestSellableGateBlocksOfflineUnsupportedCapabilityAndHolds(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	svc := noderuntime.NewService(noderuntime.NewMemoryRepository(), func() time.Time { return now })
	status, err := svc.UpsertStatus(context.Background(), noderuntime.StatusInput{
		NodeID:             "node-1",
		LeaseOnline:        false,
		RuntimeVerdict:     noderuntime.RuntimeVerdictDegraded,
		ExpectedRevision:   12,
		CurrentRevision:    12,
		LastGoodRevision:   12,
		ExpectedDigestHash: "digest-a",
		RuntimeDigestHash:  "digest-a",
		Capabilities:       []string{"PROXY_ACCOUNT"},
		CandidatePublicIPs: []string{"204.42.251.2"},
		RequiredCapabilities: []string{
			"PROXY_ACCOUNT",
			"HTTP",
		},
		ManualHold:     true,
		ComplianceHold: true,
		ManifestHash:   "manifest-a",
		BinaryHash:     "binary-a",
		ExtensionABI:   "rayip-runtime-v1",
		BundleChannel:  "stable",
	})
	if err != nil {
		t.Fatalf("UpsertStatus() error = %v", err)
	}
	if status.Sellable {
		t.Fatalf("blocked node should not be sellable: %#v", status)
	}
	for _, reason := range []noderuntime.UnsellableReason{
		noderuntime.UnsellableOffline,
		noderuntime.UnsellableDegraded,
		noderuntime.UnsellableUnsupportedCapability,
		noderuntime.UnsellableManualHold,
		noderuntime.UnsellableComplianceHold,
	} {
		assertHasReason(t, status.UnsellableReasons, reason)
	}
}

func TestLeaseObservationDoesNotRegressAckedRevision(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	svc := noderuntime.NewService(noderuntime.NewMemoryRepository(), func() time.Time { return now })
	if _, err := svc.UpsertStatus(context.Background(), noderuntime.StatusInput{
		NodeID:             "node-1",
		LeaseOnline:        true,
		RuntimeVerdict:     noderuntime.RuntimeVerdictAccepted,
		ExpectedRevision:   0,
		CurrentRevision:    0,
		LastGoodRevision:   0,
		RuntimeDigestHash:  "digest-empty",
		Capabilities:       []string{"SOCKS5", "HTTP"},
		CandidatePublicIPs: []string{"204.42.251.2"},
		BundleChannel:      "unknown",
	}); err != nil {
		t.Fatalf("initial UpsertStatus() error = %v", err)
	}
	if _, err := svc.RecordRuntimeAck(context.Background(), noderuntime.RuntimeAckInput{
		NodeID:           "node-1",
		AppliedRevision:  3,
		LastGoodRevision: 3,
		DigestHash:       "digest-applied",
		AccountCount:     2,
	}); err != nil {
		t.Fatalf("RecordRuntimeAck() error = %v", err)
	}
	status, err := svc.UpsertStatus(context.Background(), noderuntime.StatusInput{
		NodeID:             "node-1",
		LeaseOnline:        true,
		RuntimeVerdict:     noderuntime.RuntimeVerdictAccepted,
		ExpectedRevision:   1,
		CurrentRevision:    1,
		LastGoodRevision:   1,
		RuntimeDigestHash:  "digest-applied",
		AccountCount:       2,
		Capabilities:       []string{"SOCKS5", "HTTP"},
		CandidatePublicIPs: []string{"204.42.251.2"},
		BundleChannel:      "unknown",
	})
	if err != nil {
		t.Fatalf("lease UpsertStatus() error = %v", err)
	}
	if status.ExpectedRevision != 3 || status.CurrentRevision != 3 || status.LastGoodRevision != 3 {
		t.Fatalf("status revisions regressed: %#v", status)
	}
	if status.AccountCount != 2 || status.RuntimeDigestHash != "digest-applied" {
		t.Fatalf("status digest/account not preserved from live observation: %#v", status)
	}
}

func TestSellableGateBlocksNodesWithoutCandidatePublicIP(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	svc := noderuntime.NewService(noderuntime.NewMemoryRepository(), func() time.Time { return now })
	status, err := svc.UpsertStatus(context.Background(), noderuntime.StatusInput{
		NodeID:             "node-1",
		LeaseOnline:        true,
		RuntimeVerdict:     noderuntime.RuntimeVerdictAccepted,
		ExpectedRevision:   12,
		CurrentRevision:    12,
		LastGoodRevision:   12,
		ExpectedDigestHash: "digest-a",
		RuntimeDigestHash:  "digest-a",
		Capabilities:       []string{"PROXY_ACCOUNT", "SOCKS5"},
		RequiredCapabilities: []string{
			"PROXY_ACCOUNT",
			"SOCKS5",
		},
	})
	if err != nil {
		t.Fatalf("UpsertStatus() error = %v", err)
	}
	if status.Sellable {
		t.Fatalf("node without candidate public ip should not be sellable: %#v", status)
	}
	assertHasReason(t, status.UnsellableReasons, noderuntime.UnsellableNoCandidatePublicIP)
}

func assertHasReason(t *testing.T, reasons []noderuntime.UnsellableReason, expected noderuntime.UnsellableReason) {
	t.Helper()
	for _, reason := range reasons {
		if reason == expected {
			return
		}
	}
	t.Fatalf("reasons %v missing %s", reasons, expected)
}
