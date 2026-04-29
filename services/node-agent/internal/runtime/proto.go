package runtime

import controlv1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/control/v1"

func ObservationToProto(info DiscoveryInfo) *controlv1.RuntimeObservation {
	return &controlv1.RuntimeObservation{
		AgentVersion:       info.AgentVersion,
		XrayVersion:        info.XrayVersion,
		BundleVersion:      info.BundleVersion,
		ExtensionAbi:       info.ExtensionABI,
		Capabilities:       append([]string(nil), info.Capabilities...),
		BinarySha256:       info.BinarySHA256,
		ManifestSha256:     info.ManifestSHA256,
		RuntimeDigest:      info.RuntimeDigest,
		LastGoodGeneration: info.LastGoodGeneration,
	}
}

func CommandFromProto(cmd *controlv1.RuntimeCommand) Command {
	if cmd == nil {
		return Command{}
	}
	return Command{
		CommandID:         cmd.GetCommandId(),
		NodeID:            cmd.GetNodeId(),
		Operation:         operationFromProto(cmd.GetOperation()),
		Account:           accountFromProto(cmd.GetAccount()),
		DesiredGeneration: cmd.GetDesiredGeneration(),
		DeadlineUnixMS:    cmd.GetDeadlineUnixMs(),
	}
}

func ResultToProto(result Result) *controlv1.RuntimeResult {
	return &controlv1.RuntimeResult{
		CommandId:         result.CommandID,
		Status:            resultStatusToProto(result.Status),
		ErrorCode:         result.ErrorCode,
		ErrorMessage:      result.ErrorMessage,
		AppliedGeneration: result.AppliedGeneration,
		Usage: &controlv1.RuntimeUsage{
			ProxyAccountId:    result.Usage.ProxyAccountID,
			RuntimeEmail:      result.Usage.RuntimeEmail,
			RxBytes:           result.Usage.RxBytes,
			TxBytes:           result.Usage.TxBytes,
			ActiveConnections: result.Usage.ActiveConnections,
			RxBytesPerSecond:  result.Usage.RxBytesPerSecond,
			TxBytesPerSecond:  result.Usage.TxBytesPerSecond,
		},
		Digest: &controlv1.RuntimeDigest{
			AccountCount:  result.Digest.AccountCount,
			EnabledCount:  result.Digest.EnabledCount,
			DisabledCount: result.Digest.DisabledCount,
			MaxGeneration: result.Digest.MaxGeneration,
			Hash:          result.Digest.Hash,
		},
	}
}

func accountFromProto(account *controlv1.RuntimeAccount) Account {
	if account == nil {
		return Account{}
	}
	return Account{
		ProxyAccountID:    account.GetProxyAccountId(),
		RuntimeEmail:      account.GetRuntimeEmail(),
		Protocol:          protocolFromProto(account.GetProtocol()),
		ListenIP:          account.GetListenIp(),
		Port:              account.GetPort(),
		Username:          account.GetUsername(),
		Password:          account.GetPassword(),
		ExpiresAtUnixMS:   account.GetExpiresAtUnixMs(),
		EgressLimitBPS:    account.GetEgressLimitBps(),
		IngressLimitBPS:   account.GetIngressLimitBps(),
		MaxConnections:    account.GetMaxConnections(),
		Status:            accountStatusFromProto(account.GetStatus()),
		PolicyVersion:     account.GetPolicyVersion(),
		DesiredGeneration: account.GetDesiredGeneration(),
	}
}

func operationFromProto(operation controlv1.RuntimeOperation) Operation {
	switch operation {
	case controlv1.RuntimeOperation_RUNTIME_OPERATION_UPSERT:
		return OperationUpsert
	case controlv1.RuntimeOperation_RUNTIME_OPERATION_DELETE:
		return OperationDelete
	case controlv1.RuntimeOperation_RUNTIME_OPERATION_DISABLE:
		return OperationDisable
	case controlv1.RuntimeOperation_RUNTIME_OPERATION_UPDATE_POLICY:
		return OperationUpdatePolicy
	case controlv1.RuntimeOperation_RUNTIME_OPERATION_GET_USAGE:
		return OperationGetUsage
	case controlv1.RuntimeOperation_RUNTIME_OPERATION_GET_DIGEST:
		return OperationGetDigest
	case controlv1.RuntimeOperation_RUNTIME_OPERATION_PROBE:
		return OperationProbe
	default:
		return ""
	}
}

func protocolFromProto(protocol controlv1.RuntimeProtocol) Protocol {
	switch protocol {
	case controlv1.RuntimeProtocol_RUNTIME_PROTOCOL_SOCKS5:
		return ProtocolSOCKS5
	case controlv1.RuntimeProtocol_RUNTIME_PROTOCOL_HTTP:
		return ProtocolHTTP
	default:
		return ""
	}
}

func accountStatusFromProto(status controlv1.RuntimeAccountStatus) AccountStatus {
	switch status {
	case controlv1.RuntimeAccountStatus_RUNTIME_ACCOUNT_STATUS_ENABLED:
		return AccountStatusEnabled
	case controlv1.RuntimeAccountStatus_RUNTIME_ACCOUNT_STATUS_DISABLED:
		return AccountStatusDisabled
	case controlv1.RuntimeAccountStatus_RUNTIME_ACCOUNT_STATUS_DELETED:
		return AccountStatusDeleted
	default:
		return ""
	}
}

func resultStatusToProto(status ResultStatus) controlv1.RuntimeResultStatus {
	switch status {
	case ResultSuccess:
		return controlv1.RuntimeResultStatus_RUNTIME_RESULT_STATUS_SUCCESS
	case ResultFailed:
		return controlv1.RuntimeResultStatus_RUNTIME_RESULT_STATUS_FAILED
	case ResultSkipped:
		return controlv1.RuntimeResultStatus_RUNTIME_RESULT_STATUS_SKIPPED
	case ResultDuplicate:
		return controlv1.RuntimeResultStatus_RUNTIME_RESULT_STATUS_DUPLICATE
	default:
		return controlv1.RuntimeResultStatus_RUNTIME_RESULT_STATUS_UNSPECIFIED
	}
}
