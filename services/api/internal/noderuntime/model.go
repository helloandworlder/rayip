package noderuntime

import "time"

type RuntimeVerdict string

const (
	RuntimeVerdictAccepted     RuntimeVerdict = "ACCEPTED"
	RuntimeVerdictDegraded     RuntimeVerdict = "DEGRADED"
	RuntimeVerdictDraining     RuntimeVerdict = "DRAINING"
	RuntimeVerdictQuarantined  RuntimeVerdict = "QUARANTINED"
	RuntimeVerdictNeedsUpgrade RuntimeVerdict = "NEEDS_UPGRADE"
)

type UnsellableReason string

const (
	UnsellableOffline               UnsellableReason = "OFFLINE"
	UnsellableDegraded              UnsellableReason = "DEGRADED"
	UnsellableDraining              UnsellableReason = "DRAINING"
	UnsellableNeedsUpgrade          UnsellableReason = "NEEDS_UPGRADE"
	UnsellableQuarantined           UnsellableReason = "QUARANTINED"
	UnsellableUnsupportedCapability UnsellableReason = "UNSUPPORTED_CAPABILITY"
	UnsellableDigestMismatch        UnsellableReason = "DIGEST_MISMATCH"
	UnsellableRuntimeLagging        UnsellableReason = "RUNTIME_LAGGING"
	UnsellableNoCandidatePublicIP   UnsellableReason = "NO_CANDIDATE_PUBLIC_IP"
	UnsellableManualHold            UnsellableReason = "MANUAL_HOLD"
	UnsellableComplianceHold        UnsellableReason = "COMPLIANCE_HOLD"
)

type StatusInput struct {
	NodeID               string
	LeaseOnline          bool
	RuntimeVerdict       RuntimeVerdict
	ExpectedRevision     uint64
	CurrentRevision      uint64
	LastGoodRevision     uint64
	ExpectedDigestHash   string
	RuntimeDigestHash    string
	AccountCount         uint64
	Capabilities         []string
	CandidatePublicIPs   []string
	RequiredCapabilities []string
	ManifestHash         string
	BinaryHash           string
	ExtensionABI         string
	BundleChannel        string
	ManualHold           bool
	ComplianceHold       bool
}

type RuntimeAckInput struct {
	NodeID           string
	Status           string
	AppliedRevision  uint64
	LastGoodRevision uint64
	DigestHash       string
	AccountCount     uint64
}

type Status struct {
	NodeID             string
	LeaseOnline        bool
	RuntimeVerdict     RuntimeVerdict
	ExpectedRevision   uint64
	CurrentRevision    uint64
	LastGoodRevision   uint64
	ExpectedDigestHash string
	RuntimeDigestHash  string
	AccountCount       uint64
	Capabilities       []string
	CandidatePublicIPs []string
	ManifestHash       string
	BinaryHash         string
	ExtensionABI       string
	BundleChannel      string
	ManualHold         bool
	ComplianceHold     bool
	Sellable           bool
	UnsellableReasons  []UnsellableReason
	UpdatedAt          time.Time
}
