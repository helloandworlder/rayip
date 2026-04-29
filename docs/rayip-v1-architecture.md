# RayIP V1 架构设计

> 版本：草案 v1  
> 日期：2026-04-29  
> 原则：Go First，Less is More，产品正确性优先

## 1. 总体形态

RayIP V1 使用一个小型 monorepo：

```text
RayIP
├─ apps/user-web        用户面板
├─ apps/admin-web       管理面板，包含实时节点面板
├─ services/api         Go 控制面
├─ services/node-agent  Go 节点代理
├─ packages/proto       gRPC / Protobuf 契约
└─ docs                 文档
```

运行链路：

```text
浏览器
  -> HTTP JSON
Go API
  -> Postgres / Redis / NATS JetStream
  -> gRPC 双向流
Go NodeAgent
  -> Runtime Bundle 管理
NodeAgent + 改造版 XrayCore
```

V1 不拆微服务。`api` 负责产品、用户、钱包、订单、库存、任务、节点控制、管理后台；`node-agent` 只负责节点本地 Runtime、主机指标、任务执行和 Web SSH。

Go Backend 必须无状态运行。API 进程内只能保存当前 gRPC/WebSocket 连接、临时 batch buffer 等可丢失句柄；任何影响订单、库存、钱包、节点可售、Runtime 期望状态的事实都必须能从 Postgres、Redis、NATS 恢复。

## 2. API 职责

Go API 是控制面。

职责：

- 用户认证
- 管理员认证和 RBAC
- 钱包、易支付充值、账单流水
- 产品和价格
- 国家 / 城市 / 线路
- 库存预约和销售
- 订单生命周期
- NodeAgent 会话管理
- 任务编排和 Worker
- NATS JetStream 发布/消费
- Web SSH WebSocket
- 管理员实时节点状态 WebSocket
- 前端 HTTP JSON API
- NodeAgent gRPC Server

API 不应该把 Xray 配置细节暴露为产品概念。

## 3. NodeAgent 职责

NodeAgent 是安装在每台家宽节点上的无 UI Go 二进制。

职责：

- 通过 ZTP bootstrap 主动接入 API。
- 注册后维护短期 node credential / session identity，而不是长期依赖 enrollment token。
- 主动连接 API 的 gRPC 双向流。
- 管理改造版 XrayCore 进程。
- 管理 NodeAgent + XrayCore Runtime Bundle 的版本、升级、回滚和兼容性。
- 读取 Runtime Bundle manifest，调用 XrayCore 扩展 API 做能力自发现。
- 上报 Bundle 签名/hash、XrayCore 版本、扩展 ABI、capabilities、runtime digest。
- 通过 gRPC Xray API / 扩展 API 下发、更新、删除代理账户。
- 上报心跳、版本、主机指标。
- 上报每账号流量。
- 执行发货、续费、停用、凭据修改、IP 刷新。
- 承载 Web SSH 终端帧转发。
- 重启后和 API 对账 Runtime 状态。

NodeAgent 不包含用户、订单、钱包、价格等业务模型。

## 4. Runtime Bundle

NodeAgent 和魔改版 XrayCore 是深度绑定的一组 Runtime Bundle。

Bundle 规则：

- 安装时一起安装。
- 升级时一起升级。
- 回滚时一起回滚。
- Runtime Bundle 以签名 artifact 形式发布，包含 manifest、XrayCore、启动基础配置、systemd 单元和校验信息。
- API 只下发与当前已协商 Bundle 能力兼容的任务。
- NodeAgent 每次 hello / lease 上报自身版本、XrayCore 版本、Bundle 版本、扩展 ABI、能力列表、manifest hash、binary hash。
- 能力来自 Runtime 自发现，不来自 `.env` 人工声明。
- Runtime 能力不匹配时，节点进入 `DEGRADED`，停止新销售。

这样可以避免节点侧出现“控制面认为支持限速/连接数，但实际 XrayCore 不支持”的商业事故。

### 4.1 Runtime 供应链

Runtime Bundle 是 RayIP 的数据面供应链资产，不是普通安装脚本产物。

要求：

- XrayCore fork 独立仓库维护；RayIP 仓库用 submodule 固定源码指针，便于审计和复现。
- 生产节点安装签名 bundle artifact，不在节点上临时编译源码。
- Bundle manifest 至少包含 `bundle_version`、`xray_version`、`extension_abi`、`binary_sha256`、`manifest_sha256`、`capabilities`、签名信息。
- NodeAgent 启动时校验 manifest、binary hash、签名和 minimum allowed version。
- 升级使用 `releases/<version>` 目录和 `current` 原子切换；失败自动回滚到 last good version。
- API 可按 channel 控制节点允许的 bundle 版本：`stable`、`canary`、`blocked`。

### 4.2 Bootstrap 配置边界

NodeAgent `.env` 只保留 bootstrap 必需项：

```env
RAYIP_AGENT_NODE_CODE=local-home-001
RAYIP_AGENT_ENROLLMENT_TOKEN=one-time-or-rotatable-bootstrap-token
RAYIP_AGENT_API_GRPC_ADDR=api.example.com:9090
RAYIP_AGENT_RUNTIME_BUNDLE_DIR=/opt/rayip/runtime
```

不在 `.env` 中放 Runtime 版本、capabilities、限速池、滥用阈值、合规策略。这些属于自发现状态或控制面期望策略，必须由 Runtime discovery 和 API 动态下发产生。

## 5. XrayCore 职责

XrayCore 是数据面 Runtime。

必须改造：

- 按账号限速。
- 按账号限制连接数。
- 按账号统计流量。
- 通过 gRPC Xray API / 扩展 API 暴露 NodeAgent 所需的运行时控制能力。

NodeAgent 把 XrayCore 当作 Runtime 引擎，不把它当业务数据库。NodeAgent 管理 XrayCore 的主路径是 gRPC Xray API / 扩展 API，不以拼接配置文件、重启进程作为日常账号变更路径。

## 6. 通信模型

前端到 API：

- HTTP JSON
- Cookie 登录态
- 开发者 API 支持多个 App，每个 App 使用独立 `AppID + AppSecret` 签名或换取 token
- OpenAPI 只用于稳定公共 API 文档，不默认强制 oapi-codegen
- 管理员实时节点面板通过 WebSocket 订阅 API 聚合后的节点状态流

API 到 NodeAgent：

- gRPC 双向流
- NodeAgent 主动连接
- 一条流承载 enrollment / hello、lease、能力协商、任务、结果、流量上报、终端帧

NodeAgent 到 XrayCore：

- gRPC Xray API / 扩展 API
- 本机环回或 Unix socket
- 必须增量 apply 账号、限速策略、连接数策略、流量读取、健康检查
- 配置文件只用于启动基础 Runtime，不用于日常订单下发

为什么用 gRPC 双向流：

- 家宽节点经常在 NAT 或不稳定网络后面。
- 控制面不需要访问节点入站端口。
- Protobuf 契约适合节点控制。

### 6.0 ZTP、能力发现和版本协商

RayIP 的节点接入按 ZTP 设计。

首次接入：

- NodeAgent 使用 bootstrap token、node code、机器指纹、公钥和 Runtime manifest digest 向 API 发起 enrollment。
- API 校验 token、安装窗口、允许的线路/区域和机器指纹，创建或绑定 node identity。
- API 返回短期 node credential、allowed channel、minimum bundle version、初始策略和下一步协商要求。
- 后续 gRPC stream 使用短期 credential 或 mTLS 身份，不长期依赖 enrollment token。

能力发现：

- NodeAgent 读取 Runtime Bundle manifest。
- NodeAgent 调 XrayCore 扩展 API 获取真实 capabilities、extension ABI、runtime digest。
- NodeAgent 计算 XrayCore binary hash 和 manifest hash。
- API 只相信 observed capabilities，不相信 `.env` 人工声明。

版本协商：

- NodeAgent hello / lease 上报 `agent_version`、`bundle_version`、`xray_version`、`extension_abi`、`capabilities`、`binary_sha256`、`manifest_sha256`、`runtime_digest`、`last_good_generation`。
- API 返回节点 runtime verdict：`ACCEPTED`、`NEEDS_UPGRADE`、`QUARANTINED`、`UNSUPPORTED_CAPABILITY`、`DIGEST_MISMATCH`。
- 只有 `ACCEPTED` 节点可进入可售池。

### 6.0.1 xDS-like Runtime 下发

Runtime 变更下发采用类似 xDS 的版本语义。

下发帧包含：

- `resource_type`
- `node_id`
- `batch_id`
- `version_info`
- `nonce`
- `seq_range`
- `desired_generation`
- `deadline`

NodeAgent 返回：

- `ACK`：已应用 version / generation。
- `NACK`：拒绝原因、错误字段、last good version。
- `PARTIAL`：账号级成功、失败、重复、跳过明细。

API 必须保存 last good generation。NodeAgent 重连后带上 current applied generation 和 last good generation，API 再决定补发 delta、触发 digest 对账或隔离节点。

### 6.1 大规模节点通信与 Lease

RayIP V1 需要按“数千到上万节点、单节点 10k+ 代理账号”设计通信边界。

核心原则：

- Lease 判断节点是否活着。
- Postgres 保存根源状态和 Runtime 期望状态。
- NATS JetStream 保存持久任务队列。
- gRPC stream 只承载当前在线节点的实时控制流。
- Redis 保存在线热状态和实时窗口，不保存根源事实。
- NodeAgent 不直接连接 NATS，只主动连接 API。

节点 Lease：

- NodeAgent 通过 gRPC stream 发送轻量 lease renewal。
- Lease payload 只包含 `node_id`、`session_id`、`bundle_version`、`sequence`、`renewed_at`、摘要状态。
- API 将热 lease 写入 Redis，TTL 建议 30-60 秒。
- Postgres 只记录状态变化、最后在线时间、最后会话、最后能力快照，不为每次心跳写库。
- 全量状态按变化触发或低频周期上报；高频速度、CPU、RAM、连接数进入 Redis 实时窗口。

持久化规则：

- Postgres 是根源状态，保存订单、库存、钱包、代理账号、节点、Runtime 期望状态。
- Redis 开启 AOF/RDB 持久化，但仍然只作为实时状态层；Redis 丢失不能造成订单或库存错误。
- NATS JetStream 使用 file storage 和 durable consumer，作为持久任务队列。
- Go API 实例可以随时重启；NodeAgent 会重连，Worker 会重新消费 NATS，业务状态从 Postgres 恢复。

任务模型：

- 不把 Postgres 设计成主任务队列。
- Postgres 只保存可重建的期望状态、变更序列、outbox、安全审计。
- NATS JetStream 才是任务队列，消息持久化、ack、redelivery。
- NATS 消息只带 `change_id`、`batch_id`、`node_id`、`seq_range` 等最小索引。
- Worker 收到消息后回 Postgres 读取当前期望状态，再合并下发。
- 重复消息、重复 Worker、重复 gRPC 下发都靠 generation 和 apply ack 幂等处理。

多 API 实例时：

- 每个 NodeAgent gRPC stream 归属于一个 API instance。
- `node_agent_sessions` 记录 `node_id -> session_id -> api_instance_id`。
- Worker 如果不在 stream owner 上，通过内部控制 subject 路由到 stream owner；这个路由只转发在线控制帧，不保存业务事实。
- NodeAgent 仍然只连接 API，不直接连接 NATS。

控制帧不承载大 payload。常规下发帧只需要：

- `batch_id` 或 `change_id`
- `node_id`
- `seq_range`
- `deadline`
- `lease_token`

账号明细通过 API 从 Postgres 当前期望状态生成，并按页传给 NodeAgent。NodeAgent 必须持久化最近批次结果和每个账号的已应用 generation。重复下发同一批次时，NodeAgent 返回同一结果，不重复修改 Runtime。

下发硬约束：

- Backend 到 NodeAgent 只下发增量 Runtime batch。
- NodeAgent 到 XrayCore 只通过 gRPC Xray API / 扩展 API 做增量 apply。
- 日常购买、续费、停用、凭据刷新、IP 刷新、限速调整都不能触发整份配置重写。
- 冷启动恢复、节点重装、对账修复可以扫描全量期望状态，但必须拆成分页增量 batch 执行。
- 全量快照只能用于校验和恢复计划，不作为一次性下发 payload。

### 6.2 单节点 10k+ 账号同步

单节点 10k+ 订单不能靠每次重写整份 Xray 配置解决。

Runtime 同步规则：

- 账号使用稳定 `proxy_account_id` / email 作为 Runtime 主键。
- 每个账号有独立 `policy_version` 和 `desired_generation`。
- 所有变更走 delta：新增、更新策略、刷新凭据、刷新 IP、停用、删除。
- 同一节点的多次变更先按账号和 generation 合并，旧 generation 不再下发。
- 批量续费、批量停用、节点恢复可以按账号分页批量下发。
- NodeAgent 对每批 apply 返回成功、失败、跳过、重复任务等结构化结果。
- XrayCore 改造 gRPC API 必须支持账号级增量更新，避免重启或全量重载造成存量代理抖动。

对账方式：

- NodeAgent 上报 Runtime 快照摘要，而不是每次上传 10k+ 明细。
- 摘要包含账号数量、generation 水位、分桶 hash、异常账号列表。
- API 发现摘要不一致时，再按 bucket / cursor 拉取明细并下发修复 delta。
- 新安装或重装恢复时，API 通过 cursor 分页下发期望账号，单页建议 500-1000 条。
- 恢复过程允许断点续传，NodeAgent 回传 last applied cursor 和 snapshot hash。

这样 V1 可以支持大节点，同时不把 Redis、NATS 或 gRPC stream 当业务数据库。

### 6.3 Panel 与 etcd 的吸收原则

RayIP 不照搬 `refers/panel`，也不引入 etcd，但吸收两者的控制面经验。

从 Panel 吸收：

- HTTP API 不直接长时间阻塞在节点操作上。
- Worker 持有节点执行能力，API 只提交任务和读取状态。
- 节点健康检查必须有超时、并发上限、错误分类、状态回写。
- usage / metrics 写入必须批量聚合、时间分桶、限制 DB 并发。
- 通知使用持久队列和独立消费者，失败不影响主业务。

从 etcd 吸收：

- 单一事实源：Postgres 是根源状态。
- Revision：`runtime_change_log.seq` 和 `desired_generation` 是逻辑版本。
- Watch：NodeAgent 断线后按 last applied seq 恢复缺失变更。
- Lease：NodeAgent 在线状态用轻量 TTL lease 表示。
- CAS：钱包、库存、订单、Runtime apply ack 都必须带前置状态条件。
- Compaction：变更历史可以压缩为当前 Runtime 期望状态；落后太多时走快照对账。
- Backpressure：NATS 积压、DB 慢、节点大面积掉线时必须限流或暂停销售。

这些是设计思想，不是新增组件。

### 6.4 开发前必须统一的契约

正式编码前必须先冻结一组最小契约。契约先行不是为了代码生成，而是为了让 Go API、React 前端、NodeAgent、魔改 XrayCore 在同一套边界上开发。

HTTP JSON 契约：

- 统一错误结构：`code`、`message`、`request_id`、`details`。
- 统一分页：`page`、`page_size`、`total`、`items`。
- 统一金额：数据库用最小货币单位或 decimal，API 明确字符串或整数，不用 float 作为金额事实。
- 统一时间：API 使用 RFC3339，数据库使用 `timestamptz`。
- 统一幂等头：购买、续费、凭据刷新、IP 刷新使用 `Idempotency-Key`。
- 管理端 API 按资源组织：`users`、`wallets`、`orders`、`proxy-accounts`、`inventory`、`nodes`、`runtime-jobs`、`payments`、`audit`、`settings`。

开发者 API 契约：

- App 鉴权使用 `AppID + AppSecret`。
- 签名输入固定为：HTTP method、path、query、timestamp、nonce、body hash。
- 重放窗口、nonce 去重、IP 白名单、App 权限、Webhook 签名和重试规则必须在第一版协议里确定。
- AppSecret 只保存 hash 或加密密文，完整 Secret 只在创建或重置时展示。

NodeAgent gRPC 契约：

- 连接握手：`node_id`、`session_id`、`enrollment_token`、`bundle_version`、能力列表。
- Lease 续约：`node_id`、`session_id`、`sequence`、`renewed_at`、摘要状态。
- Runtime 下发：`batch_id`、`node_id`、`seq_range`、`deadline`、`lease_token`、分页账号 delta。
- 账号 delta：`proxy_account_id`、Runtime email、协议、IP、端口、账号、密码、过期时间、`desired_generation`、`policy_version`、限速、连接数上限、状态。
- Apply 结果：成功、失败、跳过、重复、部分成功；必须返回账号级错误码和当前已应用 generation。
- 指标上报：实时样本走 Redis 窗口，分钟/小时聚合落 Postgres。
- 终端帧：Web SSH 只传 PTY 字节帧和 resize/control frame，不进入 NATS。

NodeAgent 到 XrayCore 契约：

- 只通过本机 gRPC Xray API / 扩展 API 做增量操作。
- 必须有 `UpsertAccount`、`DeleteAccount`、`UpdatePolicy`、`DisableAccount`、`GetUsage`、`GetDigest` 类能力。
- 账号级限速、连接数限制、流量统计必须在 XrayCore 内生效。
- 同一账号同一 `desired_generation` 重复 apply 必须幂等。

状态机契约：

- 订单、库存、节点、任务、Runtime apply 的允许状态迁移必须在开发前列成表。
- 所有跨状态更新必须带前置状态条件，不能直接覆盖。
- 任何状态迁移失败都必须返回可分类错误：可重试、不可重试、需人工、已过期、版本冲突。

任务和消息契约：

- NATS message payload 只带索引，不带业务事实大 payload。
- `Msg-Id` 使用 outbox event 或业务幂等键。
- Worker 必须先读 Postgres 当前期望状态，再生成 Runtime batch。
- DLQ、重试次数、退避、优先级和并发预算需要在开发前固定默认值。

## 7. NATS JetStream

NATS 要么不用，要用就承担真正的可靠异步任务。

V1 使用 NATS JetStream 处理：

- 发货
- 续费
- 停用 / 过期
- 凭据修改
- IP 刷新
- 节点退役
- 对账任务
- 通知投递，例如 Bark 管理员告警
- SSH 自动恢复 NodeAgent

规则：

- Postgres 是业务事实来源。
- NATS JetStream 是持久任务队列，必须使用 file storage、durable consumer、ack 和 redelivery。
- 业务事务只写 Postgres 根源状态、Runtime 变更序列和 `outbox_events`。
- 事务提交后由 outbox publisher 发布 JetStream。
- JetStream `Msg-Id` 使用 `outbox_event_id`、`change_id` 或业务幂等键，避免重复发布。
- Worker 收到消息后必须读取 Postgres 当前期望状态，不能只相信消息 payload。
- JetStream 负责持久排队和重投；任务是否仍需执行由 Postgres 期望状态决定。
- 发布失败可以从 `outbox_events` 重放。
- NodeAgent 不直接连接 NATS。

示例 Subject：

```text
rayip.task.provision.v1
rayip.task.renew.v1
rayip.task.disable.v1
rayip.task.refresh_credential.v1
rayip.task.refresh_ip.v1
rayip.task.retire_node.v1
rayip.task.reconcile.v1
rayip.task.ssh_recover_node.v1
rayip.runtime.apply.v1
```

不走 NATS：

- 高频原始流量样本
- Web SSH 字节流
- 节点心跳流
- 浏览器请求响应

## 8. 数据归属

Postgres 保存持久业务事实：

- 用户
- 管理员
- 钱包和流水
- 充值订单
- 产品和价格
- 国家 / 城市 / 线路
- 节点
- 库存
- 库存预约
- 订单
- 代理账户
- Runtime 期望状态
- Runtime 变更序列
- Runtime apply 结果
- 钱包冻结和流水
- 速率策略
- 节点恢复/退役/对账 Job
- outbox 事件
- 流量汇总
- 节点指标汇总
- 节点能力快照
- NodeAgent 会话记录
- 开发者 App：`AppID + AppSecret`、IP 白名单、回调地址、权限、启停状态
- 通知事件和通知渠道
- 优惠券
- 邀请返利
- 工单
- 审计日志

Redis 保存短期辅助状态：

- 库存预约热缓存和过期提醒
- 幂等热缓存
- API 限流
- 节点在线状态
- NodeAgent session 路由
- 热点统计
- 节点实时指标窗口
- Web SSH 会话 TTL

Redis 必须开启持久化和合理的 maxmemory 策略，但不保存订单、钱包、库存预约、幂等结果等最终事实。Redis 丢失后，系统必须能从 Postgres 恢复。

NATS JetStream 保存持久任务队列和投递状态，不保存业务根源事实。NATS 消息可以重放，Worker 必须幂等。

NodeAgent 保存本地 Runtime 缓存、enrollment 配置和最近应用快照。

## 9. 幂等、预约和钱包冻结

RayIP 的正确性以 Postgres 为准。

幂等规则：

- 用户购买、续费、IP 刷新、凭据刷新必须带幂等键。
- 幂等键落 Postgres，作用域至少包含用户、动作、业务目标。
- Redis 可以缓存幂等结果，但不能作为唯一判断依据。
- 重复请求返回同一个业务结果，不重复扣款、不重复创建订单、不重复下发 Runtime。

库存预约规则：

- `inventory_reservations` 是预约事实表。
- 预约记录包含库存 ID、订单 ID、用户 ID、过期时间、状态、幂等键。
- 通过行锁、唯一约束、条件更新防止超卖。
- Redis TTL 只用于快速过期提醒，不能替代 Postgres 预约状态。

钱包规则：

- 钱包余额和冻结金额由 Postgres 事务保护。
- 可用余额 = `balance - frozen_balance`。
- 下单先 `FREEZE`，Runtime 确认后 `CAPTURE`，失败后 `UNFREEZE`。
- 退款使用 `REFUND`。
- 所有变动必须写不可变流水。

## 10. 发货流程

```text
客户提交订单
  -> API 校验产品、线路、库存、余额
  -> API 在 Postgres 中预约库存
  -> API 在 Postgres 中冻结钱包余额
  -> API 创建订单和代理账户，状态 PROVISIONING
  -> API 写 Runtime 期望状态和变更序列
  -> API 写 outbox 事件
  -> outbox publisher 发布发货任务到 NATS
  -> Worker 消费 NATS，读取 Postgres 当前期望状态
  -> Worker 合并同节点变更，按 batch 通过 gRPC 下发
  -> NodeAgent 通过 gRPC Xray API 更新 XrayCore
  -> XrayCore 确认账号、限速、连接数策略生效
  -> NodeAgent 回传成功
  -> API CAPTURE 冻结金额
  -> API 标记订单 ACTIVE
  -> 客户看到 IP、端口、账号、密码
```

失败处理：

- 任务在预算内重试。
- 重试期间库存不卖给其他人。
- 最终失败释放库存和冻结余额。
- 客户不会拿到不可用凭据。

## 11. 节点健康和可售判断

节点可售不是单个在线布尔值，而是健康状态计算结果。

健康输入：

- NodeAgent 心跳。
- 节点身份和 credential 状态。
- Runtime Bundle 签名、hash、channel、minimum version。
- Runtime Bundle 版本、扩展 ABI 和能力。
- XrayCore 进程状态。
- Runtime digest 与 Postgres 期望状态是否一致。
- 合规 / 滥用 hold 状态。
- 当前连接数。
- 速度、CPU、RAM、磁盘。
- 最近任务成功/失败。
- 最近对账结果。
- IP 质量检测结果。

可售规则：

- 只有 `ACTIVE`、身份有效、供应链校验通过、能力协商 `ACCEPTED`、digest 正常、合规 / 滥用状态 clean 且健康检查通过的节点可售。
- `DEGRADED`、`OFFLINE`、`DRAINING`、`RETIRED`、`QUARANTINED`、`NEEDS_UPGRADE` 不卖新库存。
- Redis 可保存实时在线状态；Postgres 保存生命周期、最后在线时间、最后能力快照、最后同步状态。

## 12. 对账

对账是为了维持产品正确性，不是正常流程的补丁。

对账类型：

- NodeAgent 重连对账
- XrayCore 账号对账
- 订单过期对账
- 库存和节点生命周期对账
- 流量汇总对账
- NodeAgent 重装恢复对账

对账影响可售状态：

- 节点离线，不卖新库存。
- 节点异常，不卖新库存。
- 库存不一致，相关 IP 不可售。
- 退役中节点不接新订单。

## 13. NodeAgent ZTP 和重装恢复

NodeAgent 新安装或重装后，必须能自动恢复其上已指派订单。

恢复原则：

- API 的 Postgres 记录是期望状态来源。
- NodeAgent 本地状态只是 Runtime 快照。
- 代理账号使用稳定 `proxy_account_id` / email 作为 Runtime 标识。
- 每次下发携带策略版本 `policy_version`。
- NodeAgent 应用任务必须幂等；同一账号同一版本重复下发不产生副作用。

恢复流程：

```text
NodeAgent 安装或重装
  -> 使用 bootstrap / recovery token 连接 API
  -> 上报 node_id、bundle manifest、binary hash、extension ABI、capabilities、Runtime 快照
  -> API 校验节点身份、token、Bundle 签名/hash、allowed channel、minimum version
  -> API 查询该节点所有 ACTIVE/PROVISIONING/RENEWING 代理账户
  -> API 生成期望 Runtime 配置
  -> API 按 version_info + nonce 下发恢复任务
  -> NodeAgent 对比本地 Runtime，补齐缺失账号，更新策略，删除不应存在账号
  -> NodeAgent 回传 ACK/NACK/PARTIAL、apply_result 和 runtime_snapshot
  -> API 更新最后同步状态
```

如果节点磁盘全新但 node_id 不变，仍然从 Postgres 恢复所有已指派订单。若节点身份无法确认，必须进入人工重新绑定流程，不能自动接管订单。

### 13.1 SSH 自动恢复 NodeAgent

平台可以保存节点 SSH 凭据，但它是恢复能力，不是正常控制链路。

管理员创建节点时可以配置：

- IP / SSH 端口
- root 或指定 sudo 用户
- SSH Key，推荐
- 密码，可选
- Host key fingerprint，推荐绑定
- 是否允许自动恢复

自动恢复触发条件：

- NodeAgent lease 超时，节点进入 `OFFLINE` 或 `RECOVERING`。
- 节点不是 `DRAINING` / `RETIRED` / `REMOVED`。
- 该节点开启 `auto_recover_enabled`。
- 存在可用 SSH 凭据。
- 未命中恢复冷却时间和失败熔断。

恢复流程：

```text
NodeAgent lease 超时
  -> API 停止该节点新销售
  -> recovery scheduler 创建 ssh_recovery_job
  -> Worker 按全局/线路/节点并发预算执行 SSH 探测
  -> SSH 可达且 host key 校验通过
  -> 上传或执行一键安装脚本，携带一次性 recovery token
  -> 安装/修复 NodeAgent + XrayCore Runtime Bundle
  -> systemd 启动 NodeAgent
  -> NodeAgent 主动重连 API
  -> API 校验 recovery token 和 node_id
  -> 按 Postgres 期望状态恢复 Runtime 账号
  -> 更新节点最后同步状态
```

安全和限流规则：

- SSH 凭据必须加密保存，默认优先使用 SSH Key。
- 生产恢复不能使用跳过 host key 校验的 SSH 参数。
- 每次 SSH 恢复必须写审计日志：谁配置的凭据、哪个 Worker 执行、执行结果、耗时、错误摘要。
- 全局、每线路、每节点都要有恢复并发限制，避免大面积断连时打爆 SSH 或控制面。
- 连续失败后进入冷却，并触发 Bark 告警。
- SSH 不可达时只标记需要人工处理，不反复暴力尝试。

恢复脚本必须幂等：

- 已安装且版本正确时只确保 systemd 运行。
- 版本不兼容时按 Runtime Bundle 升级或回滚。
- 本地 token 存在时优先保留。
- 磁盘全新时使用一次性 recovery token 重新绑定原 node_id。
- 不自动把订单迁移到其他节点。

如果 XrayCore 仍在运行但 NodeAgent 掉线，恢复脚本应尽量不打断现有代理流量；只有 Bundle 不兼容或 Runtime 已不可控时，才进入受控重启。

## 14. Bark 和通知

V1 可以接入 Bark 作为轻量管理员告警渠道。

通知事件落 Postgres，Bark 只是投递渠道。

适合 Bark 的事件：

- 节点离线。
- 节点进入 `DEGRADED`。
- Runtime Bundle 版本不兼容。
- 发货/续费/停用任务最终失败。
- 易支付回调异常。
- 库存低水位。
- 节点退役完成或卡住。

实现：

- API 写 `notification_events`。
- outbox publisher 投递通知任务。
- 通知 Worker 调 Bark HTTP API。
- 投递失败可重试，不能影响主业务事务。

## 15. 管理员实时节点面板

实时节点面板服务于管理员快速判断家宽节点是否好用、是否可售、是否需要退役。

数据链路：

```text
NodeAgent
  -> gRPC 流上报心跳、速度、连接数、CPU、RAM、磁盘、Runtime 状态
API
  -> 写 Redis 实时窗口
  -> 周期汇总到 Postgres
管理面板
  -> WebSocket 订阅节点状态
```

面板展示：

- 节点组 / 线路
- 在线状态
- v4/v6 地区
- 出口 IP
- 实时上下行速度
- 开机时长
- 累计上下行流量
- CPU / RAM / 存储
- 活跃连接数
- 最近启动、最近在线、最后同步
- 快捷操作：详情、Web SSH、禁止售卖、退役、重启 Runtime、触发对账

历史数据：

- Redis 保留短窗口实时样本。
- Postgres 保留分钟级/小时级聚合。
- 不引入 Prometheus、Loki、ClickHouse。

## 16. Web SSH

```text
管理员打开终端
  -> 浏览器 WebSocket 连接 API
  -> API 校验权限
  -> API 创建审计会话
  -> API 通过 gRPC 请求 NodeAgent 打开终端
  -> NodeAgent 启动本地 PTY
  -> 终端帧在 gRPC 和 WebSocket 之间转发
  -> 超时、断线或管理员关闭后结束
```

V1 审计会话元数据即可：谁、何时、打开哪个节点、持续多久、关闭原因。

## 17. 故障边界

API 不可用：

- 用户和管理面板不可用。
- 已有 XrayCore 代理继续运行。
- NodeAgent 重连后上报快照。

NATS 不可用：

- 新异步任务不能可靠派发。
- 下单在冻结余额前失败关闭。
- 已有代理继续运行。

Redis 不可用：

- 实时节点面板、限流、热缓存、Web SSH TTL 能力降级。
- 下单正确性仍由 Postgres 保证。
- 如果部署策略要求 Redis 参与削峰，可临时关闭高并发购买，但这不是因为订单事实丢失。
- 已有代理继续运行。

Postgres 不可用：

- 业务 API 失败关闭。
- 已有代理继续运行。

NodeAgent 断开：

- 节点超时后不可售。
- 已有 XrayCore 代理继续运行。
- 生命周期任务等待重试或进入退役处理。

XrayCore 下发失败：

- NodeAgent 返回结构化失败。
- API 在安全范围内重试。
- 订单、库存、钱包状态保持受保护。

## 18. 部署形态

开发 / 测试优先：

- Railway 项目：`RayIP`
- 环境：`Test`
- 服务：Postgres、Redis、NATS、API、user-web、admin-web
- NodeAgent 部署在真实或虚拟 Linux 节点

V1 不需要 Kubernetes。
