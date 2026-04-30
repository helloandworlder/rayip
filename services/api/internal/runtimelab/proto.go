package runtimelab

import controlv1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/control/v1"

func ApplyToProto(apply RuntimeApply) *controlv1.RuntimeApply {
	resources := make([]*controlv1.RuntimeResource, 0, len(apply.Resources))
	for _, resource := range apply.Resources {
		resources = append(resources, resourceToProto(resource))
	}
	return &controlv1.RuntimeApply{
		ApplyId:              apply.ApplyID,
		NodeId:               apply.NodeID,
		Mode:                 applyModeToProto(apply.Mode),
		VersionInfo:          apply.VersionInfo,
		Nonce:                apply.Nonce,
		BaseRevision:         apply.BaseRevision,
		TargetRevision:       apply.TargetRevision,
		DeadlineUnixMs:       apply.DeadlineUnixMS,
		Resources:            resources,
		RemovedResourceNames: append([]string(nil), apply.RemovedResourceNames...),
	}
}

func ResultFromProto(ack *controlv1.RuntimeApplyAck) ApplyResult {
	if ack == nil {
		return ApplyResult{}
	}
	return ApplyResult{
		ApplyID:          ack.GetApplyId(),
		NodeID:           ack.GetNodeId(),
		Status:           statusFromProto(ack.GetStatus()),
		VersionInfo:      ack.GetVersionInfo(),
		Nonce:            ack.GetNonce(),
		AppliedRevision:  ack.GetAppliedRevision(),
		LastGoodRevision: ack.GetLastGoodRevision(),
		ErrorDetail:      ack.GetErrorDetail(),
		Digest: Digest{
			AccountCount:  ack.GetDigest().GetAccountCount(),
			EnabledCount:  ack.GetDigest().GetEnabledCount(),
			DisabledCount: ack.GetDigest().GetDisabledCount(),
			MaxGeneration: ack.GetDigest().GetMaxGeneration(),
			Hash:          ack.GetDigest().GetHash(),
		},
	}
}

func resourceToProto(resource RuntimeResource) *controlv1.RuntimeResource {
	return &controlv1.RuntimeResource{
		Name:              resource.Name,
		Kind:              resourceKindToProto(resource.Kind),
		ResourceVersion:   resource.ResourceVersion,
		RuntimeEmail:      resource.RuntimeEmail,
		Protocol:          protocolToProto(resource.Protocol),
		ListenIp:          resource.ListenIP,
		Port:              resource.Port,
		Username:          resource.Username,
		Password:          resource.Password,
		EgressLimitBps:    resource.EgressLimitBPS,
		IngressLimitBps:   resource.IngressLimitBPS,
		MaxConnections:    resource.MaxConnections,
		Priority:          resource.Priority,
		AbuseReportPolicy: resource.AbuseReportPolicy,
		ExpiresAtUnixMs:   resource.ExpiresAtUnixMS,
	}
}

func applyModeToProto(mode ApplyMode) controlv1.RuntimeApplyMode {
	switch mode {
	case ApplyModeSnapshot:
		return controlv1.RuntimeApplyMode_RUNTIME_APPLY_MODE_SNAPSHOT
	case ApplyModeDelta:
		return controlv1.RuntimeApplyMode_RUNTIME_APPLY_MODE_DELTA
	default:
		return controlv1.RuntimeApplyMode_RUNTIME_APPLY_MODE_UNSPECIFIED
	}
}

func resourceKindToProto(kind ResourceKind) controlv1.RuntimeResourceKind {
	switch kind {
	case ResourceKindProxyAccount:
		return controlv1.RuntimeResourceKind_RUNTIME_RESOURCE_KIND_PROXY_ACCOUNT
	default:
		return controlv1.RuntimeResourceKind_RUNTIME_RESOURCE_KIND_UNSPECIFIED
	}
}

func protocolToProto(protocol Protocol) controlv1.RuntimeProtocol {
	switch protocol {
	case ProtocolSOCKS5:
		return controlv1.RuntimeProtocol_RUNTIME_PROTOCOL_SOCKS5
	case ProtocolHTTP:
		return controlv1.RuntimeProtocol_RUNTIME_PROTOCOL_HTTP
	case ProtocolMixed:
		return controlv1.RuntimeProtocol_RUNTIME_PROTOCOL_MIXED
	default:
		return controlv1.RuntimeProtocol_RUNTIME_PROTOCOL_UNSPECIFIED
	}
}

func statusFromProto(status controlv1.RuntimeApplyStatus) ApplyStatus {
	switch status {
	case controlv1.RuntimeApplyStatus_RUNTIME_APPLY_STATUS_ACK:
		return ApplyStatusACK
	case controlv1.RuntimeApplyStatus_RUNTIME_APPLY_STATUS_NACK:
		return ApplyStatusNACK
	case controlv1.RuntimeApplyStatus_RUNTIME_APPLY_STATUS_PARTIAL:
		return ApplyStatusPartial
	default:
		return ApplyStatusFailed
	}
}
