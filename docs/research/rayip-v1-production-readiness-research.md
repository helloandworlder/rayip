# RayIP V1 生产化架构研究笔记

> 日期：2026-04-30  
> 目标：把 RayIP 设计成可商业运营的稳定平台，而不是普通代理面板或手工节点脚本集合。

## 1. 结论

RayIP 的长期架构应按“零信任节点接入 + Runtime 供应链 + 控制面协商 + 可靠状态机 + 风控合规”的商业平台设计。

不推荐：

- 在 `.env` 中手工声明 `xray_version`、`bundle_version`、`capabilities`、限速池、滥用阈值。
- 节点安装后直接相信人工配置的能力。
- Runtime 账号变更靠重写整份配置或重启 XrayCore。
- NATS / Redis 承载业务事实。
- 客户下单后才发现节点能力、digest、库存或合规状态不满足交付。

推荐：

- `.env` 只保留 bootstrap 必需项。
- NodeAgent 启动后自发现 Runtime Bundle，并通过 XrayCore 扩展 API 验证能力。
- API 对节点能力、Bundle 版本、扩展 ABI、二进制 hash 做协商和准入。
- 策略、限速、公平调度、滥用阈值、合规规则全部由控制面动态下发。
- Runtime 下发采用类似 xDS 的 `version_info + nonce + ACK/NACK + last_good_generation`。
- Runtime Bundle 作为供应链资产发布，必须签名、校验 hash、防回滚。
- Postgres 是订单、钱包、库存、Runtime 期望状态和审计的根源事实。

## 2. 参考模型

### SPIFFE / SPIRE：节点身份和 attestation

SPIRE 的模式是 agent 通过 node attestation 向 server 证明节点身份，再获得后续 workload identity。RayIP 不需要照搬 SPIRE，但应吸收两个原则：

- enrollment token 只用于 bootstrap，不长期代表节点身份。
- 节点注册后应换成短期凭证、可轮换凭证或 mTLS 身份。

参考：https://spiffe.io/docs/latest/spire-about/spire-concepts/

### Envoy xDS：版本化配置和 ACK/NACK

xDS 的核心价值是：

- 客户端上报已知资源版本。
- 服务端下发新版本和 nonce。
- 客户端 ACK 或 NACK。
- NACK 必须带错误细节，服务端保留 last good version。

RayIP Runtime 下发应采用同类语义，避免断线、重放、部分失败时状态漂移。

参考：https://www.envoyproxy.io/docs/envoy/latest/api-docs/xds_protocol.html

### TUF / Sigstore：安全更新供应链

Runtime Bundle 是节点数据面的核心供应链，必须防篡改、防错误版本、防回滚。RayIP 应采用：

- artifact hash
- manifest
- 签名
- allowed channel
- minimum allowed version
- 原子切换和失败回滚

参考：

- https://theupdateframework.github.io/specification/draft/
- https://docs.sigstore.dev/cosign/verifying/verify/

### Kubernetes Device Plugin / Node Feature Discovery：能力自发现

Kubernetes 不靠人工写“这个节点有什么硬件”，而是由 agent/plugin 发现并上报资源或特征。RayIP 节点能力也应由 NodeAgent + XrayCore 实测上报，而不是依赖 env。

参考：

- https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/
- https://kubernetes-sigs.github.io/node-feature-discovery/stable/get-started/introduction.html

### Stripe Idempotency：商业入口幂等

充值、购买、续费、刷新凭据、刷新 IP 都是高风险商业入口，必须有幂等键和可重放结果。Stripe 的幂等模型可作为参考。

参考：https://docs.stripe.com/api/idempotent_requests

### OWASP API Security：API 风险基线

RayIP 的用户 API、管理 API、开发者 API 都会面对 BOLA、BFLA、资源滥用和敏感业务流风险。安全约束应从 T5 前进入契约，不等上线后补。

参考：https://owasp.org/API-Security/editions/2023/en/0x11-t10/

### NATS JetStream：可靠异步不是 exactly once

JetStream 可以持久化、ack、redelivery、去重，但商业正确性仍必须靠业务幂等、Postgres 根源状态和条件更新保证。

参考：https://docs.nats.io/using-nats/developer/develop_jetstream/model_deep_dive

### OpenTelemetry：生产可观测性语义

RayIP V1 可以不部署完整外部观测栈，但代码和事件必须保留 trace id / request id / job id / order id / node id / account id 这些关联字段，后续才能接入 OpenTelemetry 或其他观测系统。

参考：https://opentelemetry.io/docs/

## 3. RayIP 生产化设计要求

### 3.1 Bootstrap env 最小化

NodeAgent env 只保留：

```env
RAYIP_AGENT_NODE_CODE=local-home-001
RAYIP_AGENT_ENROLLMENT_TOKEN=one-time-or-rotatable-bootstrap-token
RAYIP_AGENT_API_GRPC_ADDR=api.example.com:9090
RAYIP_AGENT_RUNTIME_BUNDLE_DIR=/opt/rayip/runtime
```

不放：

- `RAYIP_AGENT_RUNTIME_XRAY_VERSION`
- `RAYIP_AGENT_RUNTIME_BUNDLE_VERSION`
- `RAYIP_AGENT_RUNTIME_CAPABILITIES`
- `RAYIP_AGENT_RUNTIME_FAIR_POOL_BPS`
- `RAYIP_AGENT_RUNTIME_ABUSE_BYTES_PER_MIN`

这些属于 discovered state 或 desired policy，不属于 bootstrap config。

### 3.2 Runtime Bundle Manifest

Bundle 内必须有 manifest，例如：

```json
{
  "bundle_version": "rayip-runtime-v26.3.27.1",
  "xray_version": "v26.3.27-rayip.1",
  "extension_abi": "rayip.runtime.v1",
  "build_id": "2026-04-30T00:00:00Z",
  "binary_sha256": "sha256:...",
  "manifest_sha256": "sha256:...",
  "signature": "cosign-or-tuf-signature",
  "capabilities": [
    "socks5",
    "http",
    "account-rate-limit",
    "smart-fair-limit",
    "connection-limit",
    "usage-stats",
    "abuse-detection",
    "runtime-digest"
  ]
}
```

NodeAgent 还必须通过 XrayCore 扩展 API 读取真实 capability，并与 manifest 交叉校验。

### 3.3 ZTP 注册

节点首次启动：

1. NodeAgent 使用 bootstrap token 连接 API。
2. API 校验 token、node code、机器指纹和允许的安装窗口。
3. API 创建或绑定 node identity。
4. API 返回短期 node credential、allowed channel、minimum bundle version、初始策略。
5. 后续 gRPC 使用短期 credential 或 mTLS，不长期依赖 bootstrap token。

### 3.4 能力协商

NodeAgent hello / lease 上报：

- agent version
- runtime bundle version
- XrayCore fork version
- extension ABI
- capabilities
- binary digest
- manifest digest
- current runtime digest
- last good generation

API 返回准入结果：

- `ACCEPTED`
- `NEEDS_UPGRADE`
- `QUARANTINED`
- `UNSUPPORTED_CAPABILITY`
- `DIGEST_MISMATCH`

只有 `ACCEPTED` 节点可进入可售池。

### 3.5 xDS-like Runtime 下发

Runtime 下发帧必须包含：

- `resource_type`
- `node_id`
- `batch_id`
- `version_info`
- `nonce`
- `seq_range`
- `desired_generation`
- `deadline`

NodeAgent / XrayCore 返回：

- `ACK`：已应用 version / generation
- `NACK`：拒绝原因、错误字段、last good version
- `PARTIAL`：账号级成功/失败/重复/跳过明细

### 3.6 可售闸门

客户看到可购买前必须同时满足：

- 节点在线。
- 节点身份有效。
- Bundle 签名和 hash 通过。
- Bundle channel 被允许。
- capability 满足产品要求。
- Runtime digest 与 Postgres 期望状态一致或在允许窗口内。
- 节点健康分达标。
- 线路、库存、价格有效。
- 合规/滥用状态没有 hold。

否则用户面板不展示可售项，管理端展示不可售原因。

### 3.7 钱包、库存和订单

商业事实只在 Postgres：

- 钱包使用 immutable ledger。
- 金额事实禁止 float。
- 购买/续费/充值使用 `Idempotency-Key`。
- 库存预约使用事务、唯一约束、条件更新和过期释放。
- Runtime 未确认前不展示代理凭据。
- 发货失败必须释放库存和冻结余额。

### 3.8 滥用检测和合规

滥用检测不是简单“超量就封”，而是一套生产流程：

- XrayCore 产生账号级 usage、active connection、abuse event。
- NodeAgent 上报 abuse event。
- API 根据产品、客户、节点、历史行为决定 report-only、临时限速、禁用账号、禁售节点或人工审核。
- 管理端需要 notice/action、证据、处置、恢复和审计。

合规策略必须动态下发到节点，不能靠节点 env。

## 4. 当前文档和实现需要调整

- `.env.example` 只保留 bootstrap 字段和 `RUNTIME_BUNDLE_DIR`。
- `RuntimeConfig` 不再承载 `xray_version`、`bundle_version`、`capabilities`，这些来自 discovery。
- 新增 Runtime manifest 示例。
- T2 拆出 Runtime Bundle Supply Chain、ZTP、Capability Negotiation 作为 P0 组成部分。
- T3 下发协议改成 ACK/NACK + version/nonce。
- T4 可售闸门必须加入供应链、身份、合规、digest。
- T5-T8 商业闭环必须把幂等、钱包 ledger、库存事务、Runtime confirmation 作为硬验收。
