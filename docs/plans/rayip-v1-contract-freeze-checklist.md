# RayIP V1 开发前契约冻结清单

> 日期：2026-04-30  
> 原则：薄冻结、按 Task 冻结、先冻结边界再写代码  
> 目标：让 API、前端、NodeAgent、魔改 XrayCore、Worker 不靠猜接口并行开发

## 1. 冻结方式

RayIP 不做一个庞大的前置设计阶段。契约冻结按 Task 进行：

- T1 开工前，只冻结能让 API、NodeAgent、前后台、本地基础设施跑起来的最小契约。
- T2/T3 开工前，冻结 Runtime 和可靠任务通道契约。
- T5-T8 开工前，冻结钱包、库存、订单、生命周期、开发者 API 契约。
- T9-T11 开工前，冻结管理运营、ZTP、Web SSH、发布门槛契约。

每个 Task 开工前必须确认：

- HTTP JSON 请求、响应、错误码。
- DB 表、关键字段、唯一约束、索引、状态枚举。
- Redis key、TTL、是否允许丢失。
- NATS subject、consumer、ack、重试、DLQ。
- gRPC proto 消息、错误码、幂等字段、版本兼容。
- 前端页面入口、状态、空态、错误态、成功态。
- 管理端可见性和审计点。

如果某项契约还不能冻结，必须先缩小 Task 范围，不能边写边猜。

## 2. T1 契约：薄工程基线 + 第一台节点在线

必须冻结：

- API health / ready / version 响应结构。
- NodeAgent enrollment / connect / lease 最小 proto。
- `nodes`、`node_agent_sessions`、`node_capability_snapshots` 最小字段。
- Redis `rayip:lease:node:{node_id}`、`rayip:session:node:{node_id}` TTL。
- 管理端节点列表字段：节点编码、状态、最后在线、Bundle 版本、API instance。
- 用户面板和管理面板导航骨架，必须对齐 `frontend_design/` 的布局节奏。

不在 T1 冻结：

- 完整订单、钱包、库存 DDL。
- 完整 Runtime apply 协议。
- 完整开发者 API。

## 3. T2 契约：XrayCore Runtime 能力 P0

必须冻结：

- XrayCore fork 和 Runtime Bundle 供应链：
  - `third_party/xray-core` submodule 指针
  - fork release tag
  - bundle manifest schema
  - binary hash
  - manifest hash
  - signature
  - allowed channel
  - minimum allowed version
  - last good version
- ZTP bootstrap 契约：
  - bootstrap token 作用域
  - node code
  - node public key / machine fingerprint
  - enrollment audit
  - 短期 node credential / session identity
- Runtime discovery 契约：
  - NodeAgent 读取 manifest 的字段
  - XrayCore 扩展 API 返回的 capabilities
  - extension ABI
  - XrayCore version
  - runtime digest
- Capability negotiation 契约：
  - `ACCEPTED`
  - `NEEDS_UPGRADE`
  - `QUARANTINED`
  - `UNSUPPORTED_CAPABILITY`
  - `DIGEST_MISMATCH`
  - 错误码和管理端展示字段
- NodeAgent 到 XrayCore 的本机 gRPC Xray API / 扩展 API 方法：
  - `UpsertAccount`
  - `DeleteAccount`
  - `DisableAccount`
  - `UpdatePolicy`
  - `GetUsage`
  - `GetDigest`
  - `GetCapabilities`
  - `GetAbuseEvents`
- Runtime 账号主键：`proxy_account_id` / runtime email。
- 策略字段：协议、IP、端口、账号、密码、过期时间、固定限速、公平限速权重、短期流量窗口、连接数、滥用阈值、合规 action、状态、`policy_version`、`desired_generation`。
- XrayCore apply 幂等规则：同账号同 generation 重复 apply 返回相同结果。
- Runtime Lab HTTP API 和管理端页面字段。

必须验证：

- NodeAgent 不从 env 声明 Runtime capabilities。
- Runtime manifest 和 XrayCore 扩展 API 的能力发现一致。
- Bundle hash / 签名 / minimum version 不满足时节点不可售。
- SOCKS5 / HTTP 账号可连接。
- 限速在 XrayCore 内生效。
- 智能公平限速按优先级、固定限速和短期流量消耗生效。
- 连接数限制在 XrayCore 内生效。
- 上下行流量能归属到账号。
- 滥用事件可上报，并能触发禁用、限速、report-only 或人工审核。
- digest 能用于对账。

## 4. T3 契约：可靠增量下发通道

必须冻结：

- `runtime_account_states`、`runtime_change_log`、`runtime_apply_results`、`outbox_events`、`node_jobs`、`node_job_attempts` 关键字段。
- `runtime_change_log` 按 `(node_id, seq)` 唯一递增。
- `proxy_accounts.desired_generation` 和 `policy_version` 的递增规则。
- NATS JetStream：
  - Stream：`RAYIP_TASKS`、`RAYIP_RUNTIME`
  - Subject：`rayip.task.*.v1`、`rayip.runtime.apply.v1`
  - storage：file
  - durable consumer 命名
  - manual ack
  - `ack_wait`
  - `max_deliver`
  - backoff
  - DLQ / `FAILED_NEEDS_ATTENTION`
- Worker 消息 payload 只带索引：`change_id`、`batch_id`、`node_id`、`seq_range`。
- Apply result 的账号级状态：成功、失败、跳过、重复、部分成功。
- Runtime 下发协议：
  - `resource_type`
  - `version_info`
  - `nonce`
  - `seq_range`
  - `desired_generation`
  - `last_good_generation`
  - ACK
  - NACK
  - PARTIAL
  - `error_code`
  - `error_message`
- 管理端任务时间线字段：任务、attempt、错误分类、账号级结果、关联订单/节点/账号。

必须验证：

- outbox 可重放。
- NATS 重投不重复修改 Runtime。
- 重复 `version_info + nonce` 下发返回一致结果。
- NACK 后 API 不推进 desired accepted generation，管理端展示 last good generation 和错误原因。
- Worker 重启不丢任务。
- NodeAgent 断线重连可继续处理。

## 5. T4 契约：节点能力、可售闸门和 10k 恢复

必须冻结：

- 节点健康输入：lease、Bundle 版本、XrayCore 状态、CPU、RAM、存储、连接数、最近任务、最近对账、IP 质量。
- 节点供应链输入：node identity、credential 状态、bundle channel、manifest hash、binary hash、signature status、minimum allowed version。
- 节点合规输入：abuse hold、manual hold、notice/action 状态、地区限制、支付风控状态。
- 节点不可售原因枚举。
- digest 结构：账号数、generation 水位、分桶 hash、异常账号列表。
- 分页恢复 cursor、page size、断点续传字段。
- 可售判断函数输入和输出：产品、线路、库存、节点、Runtime 能力、digest。

必须验证：

- OFFLINE / DEGRADED / DRAINING / RETIRED 不卖新库存。
- Bundle 能力不匹配不卖新库存。
- `NEEDS_UPGRADE` / `QUARANTINED` / 签名校验失败 / 合规 hold 不卖新库存。
- digest 不一致不卖新库存。
- 单节点 10k+ 账号分页恢复可重复执行。

## 6. T5 契约：用户、管理员、钱包和充值

必须冻结：

- 用户注册、登录、session / token 策略。
- 管理员 RBAC 权限点。
- `wallets`、`wallet_ledger`、`wallet_holds`、`payment_orders`、`audit_logs` 字段和约束。
- 钱包流水类型：充值、冻结、确认扣款、解冻、退款、人工调整。
- 易支付回调验签、第三方交易号去重、重复回调结果。
- 金额表示：数据库和 API 禁止 float 作为事实金额。

必须验证：

- 充值回调重复投递不重复入账。
- 钱包流水不可变。
- 管理员人工调整必须审计。

## 7. T6 契约：产品价格和库存预约

必须冻结：

- 产品维度：用途、IP 类型、国家、城市、线路、协议、时长、数量。
- 价格结算顺序：基础价 -> 用户/渠道价覆盖 -> 数量/时长折扣 -> 优惠券 -> 钱包冻结。
- 订单金额快照字段，后续价格变化不影响历史订单。
- `products`、`product_prices`、`regions`、`cities`、`lines`、`rate_policies`、`node_inventory_ips`、`inventory_reservations` 关键字段和唯一约束。
- 库存预约超时释放规则。

必须验证：

- 并发预约不超卖。
- Redis 丢失不影响 Postgres 库存事实。
- 用户购买页只展示真实可售选项。

## 8. T7 契约：首单购买和 Runtime 发货

必须冻结：

- 购买 API 的 `Idempotency-Key` 作用域。
- 购买事务边界：库存预约、钱包冻结、订单、代理账号、Runtime 期望状态、outbox 同事务。
- 订单状态迁移：
  - `PENDING_PAYMENT`
  - `PAYMENT_HELD`
  - `PROVISIONING`
  - `ACTIVE`
  - `FAILED`
  - `REFUNDED`
- 发货成功后的钱包 capture 和凭据展示规则。
- 发货最终失败后的库存释放和钱包 unfreeze 规则。

必须验证：

- Runtime 未确认不展示凭据。
- 发货失败不扣款。
- 重复购买请求返回同一业务结果。

## 9. T8 契约：生命周期、流量、开发者 App

必须冻结：

- 续费、停用、过期、IP 刷新、凭据刷新 API。
- 客户可见版本和 Runtime 待确认版本的字段。
- 流量采样、分钟/小时聚合、保留周期。
- 开发者 App：
  - `AppID`
  - `AppSecret`
  - Secret 存储方式
  - IP 白名单
  - 权限范围
  - 启停状态
  - 最后调用时间
- 签名输入：HTTP method、path、query、timestamp、nonce、body hash。
- Webhook 事件、签名、重试、投递状态。

必须验证：

- IP 或凭据刷新 Runtime 确认后才展示。
- 开发者 API 防重放生效。
- Webhook 失败可重试，不影响主流程。

## 10. T9 契约：管理员运营中心和 Bark

必须冻结：

- 实时节点 WebSocket 消息结构。
- Redis 实时指标窗口 key 和 TTL。
- Postgres 分钟/小时指标聚合字段。
- 任务控制台筛选维度：订单、节点、账号、任务类型、错误分类、状态。
- 管理端安全操作：重试、取消、触发对账、标记人工处理。
- Bark 通知事件、投递状态和失败重试规则。

必须验证：

- 管理端可以从概览下钻到节点、订单、任务和审计。
- Bark 失败不影响主业务。

## 11. T10 契约：ZTP、Web SSH、退役和恢复

必须冻结：

- 节点创建字段：IP、SSH 端口、用户、SSH Key / 密码、host key fingerprint、自动恢复开关。
- 安装脚本参数和 recovery token。
- SSH 凭据加密和审计字段。
- Web SSH WebSocket frame 和 NodeAgent terminal gRPC frame。
- 终端会话 TTL、空闲超时、最大时长。
- 节点退役状态机和禁止新销售规则。
- SSH 自动恢复并发、冷却、失败熔断。

必须验证：

- Web SSH 有权限和审计。
- 重装恢复不能接管错误节点。
- 退役中节点不卖新库存。

## 12. T11 契约：Test 环境和发布门槛

必须冻结：

- Railway `RayIP` / `Test` 服务清单和环境变量。
- Secret 管理方式。
- 数据库迁移和回滚流程。
- 备份策略。
- 维护模式：暂停购买和刷新，允许登录、查看、续费、停用。
- 发布检查表。
- 故障演练清单：Redis、NATS、Postgres、NodeAgent、XrayCore、Bark、易支付。

必须验证：

- Test 环境端到端通过。
- Redis / NATS / Bark 故障有明确降级行为。
- 生产发布前检查表全部通过。
