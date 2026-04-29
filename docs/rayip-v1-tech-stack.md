# RayIP V1 技术栈

> 版本：草案 v1  
> 日期：2026-04-29

## 1. 后端

语言：

- Go

核心框架：

- 依赖注入 / 生命周期：Uber Fx
- HTTP：GoFiber / Fiber v3
- 数据访问：GORM
- 数据库迁移：Goose
- 数据库：Postgres
- 缓存 / 短期协调：Redis
- 异步任务：NATS JetStream
- 节点控制：gRPC / Protobuf
- 配置：Viper 加载后转为 typed config
- 日志：Zap
- 参数校验：Fiber Bind + go-playground/validator + 显式领域校验

原则：

- Less is More 不是少用成熟库，而是少造无意义框架。
- Fx 只负责组装依赖和生命周期，业务逻辑不依赖 Fx。
- GORM 用于常规 CRUD；钱包、库存、订单、任务状态机必须使用显式事务、行锁、条件更新和幂等键。
- Goose 负责生产 schema 迁移；生产禁用 GORM AutoMigrate。
- OpenAPI 用于稳定的外部 API 文档；不默认强制 oapi-codegen 生成所有 HTTP 代码。
- GoFiber 是 V1 主线，不再以 Gin 作为默认 HTTP 框架。
- Go API 服务必须无状态：业务事实不放进进程内存，连接句柄可丢失，重启后从 Postgres / Redis / NATS 恢复。

建议目录：

```text
services/api
├─ cmd/api
├─ internal/app
├─ internal/config
├─ internal/db
├─ internal/cache
├─ internal/bus
├─ internal/http
├─ internal/grpc
├─ internal/auth
├─ internal/domain
│  ├─ user
│  ├─ wallet
│  ├─ payment
│  ├─ product
│  ├─ inventory
│  ├─ order
│  ├─ node
│  ├─ runtime
│  └─ traffic
└─ internal/worker
```

## 2. NodeAgent

语言：

- Go

NodeAgent 使用 Fx，但比 API 更薄。

职责：

- enrollment
- 主动连接 API gRPC 双向流
- 管理 NodeAgent + XrayCore Runtime Bundle
- 管理 XrayCore 进程
- 通过 gRPC Xray API / 扩展 API 应用账号、限速、连接数策略
- 上报心跳、速度、连接数、CPU、RAM、磁盘、流量
- 上报 Runtime 快照和能力列表
- 新安装或重装后恢复已指派订单
- 执行 Web SSH 终端桥接
- 重启后 Runtime 对账

建议目录：

```text
services/node-agent
├─ cmd/node-agent
├─ internal/app
├─ internal/config
├─ internal/enroll
├─ internal/control
├─ internal/runtime
│  ├─ bundle
│  └─ xray
├─ internal/metrics
├─ internal/terminal
└─ internal/systemd
```

NodeAgent 不导入 API 的用户、订单、钱包、产品包。共享契约只放在 `packages/proto`。

## 3. Runtime Bundle

Runtime Bundle 包含：

- NodeAgent
- 魔改版 XrayCore
- systemd unit
- 本地配置
- Runtime manifest
- 签名和 hash 元数据

版本规则：

- `bundle_version` 标识一组兼容的 NodeAgent + XrayCore。
- API 根据 `bundle_version`、`extension_abi`、manifest hash、binary hash 和 observed capabilities 判断节点是否支持限速、连接数、流量统计、IP 刷新等能力。
- 升级和回滚以 Bundle 为单位，不单独升级其中一部分。
- 不兼容节点进入 `DEGRADED`，停止新销售。
- 生产 Bundle 必须作为签名 artifact 发布，NodeAgent 安装/升级前必须校验签名、hash 和 minimum allowed version。
- RayIP 仓库使用 `third_party/xray-core` submodule 固定 XrayCore fork 源码指针，生产节点不直接从 RayIP 工作树临时编译 Runtime。

Runtime manifest 示例：

```json
{
  "bundle_version": "rayip-runtime-v26.3.27.1",
  "xray_version": "v26.3.27-rayip.1",
  "extension_abi": "rayip.runtime.v1",
  "binary_sha256": "sha256:...",
  "manifest_sha256": "sha256:...",
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

Bootstrap `.env` 只配置：

- node code
- enrollment token
- API gRPC 地址
- Runtime Bundle 目录

Runtime 版本、capabilities、限速池、账号禁用状态、滥用阈值、合规规则不写入 `.env`，必须通过自发现和控制面策略下发获得。

NodeAgent 与 XrayCore：

- NodeAgent 使用 gRPC Xray API / 扩展 API 控制 XrayCore。
- XrayCore gRPC API 只监听本机环回或 Unix socket。
- 托管 XrayCore 的 gRPC API 默认使用 `auto` 随机端口；NodeAgent 每次启动前先探测端口，启动或探活失败就换端口重试，避免与 3x-ui、XrayTool 或其他本机 XrayCore 冲突。
- 所有订单生命周期变更都通过增量 gRPC apply 完成。
- 日常订单下发不通过写配置文件加重启完成。
- 配置文件只用于启动基础 inbound、API listener 和基础 Runtime 参数。
- NodeAgent 通过 XrayCore 扩展 API 获取真实 capabilities、extension ABI、账号 digest 和 abuse events。
- 账号 Disabled、限速、连接数、合规处置和滥用处置由平台云控决策并下发；NodeAgent 与 XrayCore 只执行策略和上报事件，不做本地业务决策。
- Runtime 下发使用 `version_info + nonce + ACK/NACK + last_good_generation` 语义。

## 4. 前端

用户面板：

- React 19
- TypeScript
- TanStack Router
- TanStack Query
- shadcn/ui
- Tailwind CSS
- Zod 做必要的客户端校验
- 视觉和信息结构必须对齐 `frontend_design/` 与 [用户故事与前端设计计划](./plans/rayip-v1-user-stories-and-frontend-plan.md)

管理面板：

- React 19
- TypeScript
- TanStack Query
- Refine 或同类后台框架
- shadcn/ui
- TanStack Table
- xterm.js 用于 Web SSH
- WebSocket 用于实时节点面板
- 图表使用轻量方案，例如 ECharts 或 Recharts

用户面板和管理面板是同一 monorepo 下的两个独立应用。可以共享基础 UI token，不共享大块业务页面。

## 5. 实时节点面板

管理端需要一个独立页面展示所有家宽节点实时状态。

前端：

- WebSocket 订阅节点状态流。
- 表格按节点组/线路分组。
- 展示实时上下行速度、CPU、RAM、存储、连接数、流量、开机时长。
- 支持实时模式和历史模式。
- 支持快捷操作：详情、Web SSH、禁止售卖、退役、重启 Runtime、触发对账。

后端：

- NodeAgent 通过 gRPC 上报指标。
- API 写入 Redis 实时窗口。
- API 按分钟/小时聚合到 Postgres。
- 管理面板只连 API，不直连节点。

不引入外部观测栈。

## 6. NATS JetStream

使用 NATS JetStream 的前提是它承担可靠任务。

用于：

- 发货
- 续费
- 停用 / 过期
- 凭据修改
- IP 刷新
- 节点退役
- 对账
- SSH 自动恢复 NodeAgent
- 通知 fanout

不用 NATS 承载：

- 高频指标样本
- Web SSH 字节流
- NodeAgent 心跳
- 浏览器 API

可靠性规则：

- NATS JetStream 使用 file storage、durable consumer、ack、redelivery。
- 业务事务只写 Postgres 根源状态和 outbox。
- 通过 `outbox_events` 发布 NATS 消息。
- outbox publisher 可重放。
- NATS `Msg-Id` 使用 `outbox_event_id`、`change_id` 或业务幂等键。
- NATS 消息只带最小索引，例如 `change_id`、`batch_id`、`node_id`、`seq_range`。
- Worker 消费后回 Postgres 读取当前期望状态。
- Worker 生成增量 Runtime batch，下发给 NodeAgent。
- Worker 以 generation、apply ack、状态条件更新作为幂等门闩。
- JetStream 不等于 exactly once；消息重投、Worker 重启、重复 publish 都必须靠 Postgres 根源状态、幂等键、`Nats-Msg-Id`、generation 和条件更新吸收。
- Worker 必须有全局、线路、节点级并发预算，避免大面积异常时自我放大。
- NATS backlog 超阈值时，购买和刷新类入口需要降级或暂停，保护已售代理生命周期任务。

## 7. Redis

Redis 用于短期协调：

- 库存预约热缓存
- 预约过期提醒
- 幂等热缓存
- API 限流
- 节点在线状态
- NodeAgent session 路由
- 实时节点指标窗口
- 热点计数
- Web SSH 会话 TTL

Redis 不保存订单、钱包、库存等持久事实。

Redis 需要开启 AOF/RDB 持久化，降低实时状态丢失概率。但 Redis 也不作为库存预约、钱包冻结、幂等结果的唯一依据。Redis 故障时，业务正确性必须能由 Postgres 保证。

## 8. Postgres

核心表：

- users
- admin_users
- wallets
- wallet_ledger
- wallet_holds
- payment_orders
- products
- product_prices
- regions
- cities
- lines
- nodes
- node_runtime_status
- node_agent_sessions
- node_capability_snapshots
- node_metric_rollups
- node_inventory_ips
- inventory_reservations
- proxy_orders
- proxy_accounts
- runtime_account_states
- runtime_change_log
- runtime_apply_results
- rate_policies
- node_jobs
- node_job_attempts
- outbox_events
- traffic_rollups
- notification_channels
- notification_events
- notification_deliveries
- developer_apps
- developer_app_secrets
- developer_app_ip_allowlists
- developer_app_webhooks
- coupons
- referrals
- feedback
- audit_logs

关键约束：

- 钱包、预约、幂等、订单状态变更必须使用 Postgres 事务。
- 库存预约通过 `inventory_reservations`、行锁、唯一约束和条件更新防超卖。
- 钱包冻结使用 `wallet_holds` 或等价冻结流水表示 `FREEZE / CAPTURE / UNFREEZE / REFUND`。
- 账号下发不使用胖 `node_tasks` 表作为主模型；主模型是 Runtime 期望状态 + 变更序列。
- `runtime_change_log.seq` 和 `proxy_accounts.desired_generation` 作为 etcd revision 类似的逻辑版本。
- `runtime_change_log` 可压缩；落后窗口外的 NodeAgent 走快照对账 + 分页增量恢复。
- 高频节点指标不直接写入 Postgres；Redis 保存实时窗口，Postgres 保存分钟/小时聚合。

## 9. 开发前必须冻结的技术契约

这些契约必须在正式写业务代码前定稿，后续可以演进版本，但不能边写边猜。

### 9.1 数据库关键约束

第一版 DDL 不需要追求复杂，但必须有这些不可退让的约束：

| 表 / 模型 | 必须有的约束 |
|---|---|
| `users` | 邮箱或手机号唯一，状态字段有枚举约束 |
| `wallets` | `(user_id, currency)` 唯一 |
| `wallet_ledger` | 不可变流水，关联业务单据和幂等键 |
| `wallet_holds` | `hold_no` 唯一，`FREEZE/CAPTURE/UNFREEZE/REFUND` 状态条件更新 |
| `payment_orders` | 平台充值单号唯一，第三方交易号唯一，回调幂等 |
| `products/product_prices` | 产品、地区、线路、IP 类型、时长维度不能出现重复有效价格 |
| `regions/cities/lines` | 对客户展示的 code 唯一，禁用后不参与新销售 |
| `nodes` | `node_id`、节点编码唯一，节点生命周期状态有枚举约束 |
| `node_inventory_ips` | `(node_id, ip, port, protocol)` 唯一，库存状态条件更新 |
| `inventory_reservations` | 预约号和幂等键唯一，过期释放必须可重放 |
| `proxy_orders` | 订单号唯一，用户幂等键唯一，金额快照不可被后续价格变更影响 |
| `proxy_accounts` | `proxy_account_id` 唯一，`(node_id, runtime_email)` 唯一，记录 `desired_generation` |
| `runtime_account_states` | `(proxy_account_id, desired_generation)` 可追踪，当前期望状态唯一 |
| `runtime_change_log` | `(node_id, seq)` 唯一，支持按节点顺序恢复 |
| `runtime_apply_results` | `batch_id` 唯一，账号级结果可查询 |
| `node_jobs/node_job_attempts` | job 幂等键唯一，attempt 可审计 |
| `outbox_events` | event id 唯一，发布状态可重放 |
| `developer_apps` | `(user_id, app_id)` 唯一，支持启停和权限范围 |
| `audit_logs` | 管理员、目标对象、动作、结果、request id 必填 |

事务边界：

- 购买：库存预约、钱包冻结、订单创建、代理账号创建、Runtime 期望状态、outbox 写入在同一 Postgres 事务里完成。
- 续费：钱包冻结、订单续费记录、代理账号过期时间、Runtime 变更、outbox 在同一事务里完成。
- 凭据刷新 / IP 刷新：旧凭据或旧 IP 在 Runtime 确认前仍然是客户可见版本。
- 充值回调：第三方单号去重、余额入账、流水写入必须原子。

### 9.2 NATS JetStream 默认契约

Stream 先保持少量：

| Stream | Subject | 用途 |
|---|---|---|
| `RAYIP_TASKS` | `rayip.task.*.v1` | 发货、续费、停用、IP/凭据刷新、退役、对账、SSH 恢复 |
| `RAYIP_RUNTIME` | `rayip.runtime.apply.v1` | Runtime apply 调度和重放 |
| `RAYIP_NOTIFY` | `rayip.notify.*.v1` | Bark、Webhook、站内通知 |

默认要求：

- `storage=file`。
- 每类 Worker 使用 durable consumer。
- 手动 ack。
- `ack_wait` 按任务类型配置，普通任务 30-120 秒，SSH/恢复类更长。
- `max_deliver` 和退避策略必须固定。
- 超过重试预算进入 DLQ 或 `FAILED_NEEDS_ATTENTION`，不无限重试。
- 购买/刷新类任务优先级低于续费、停用、过期、恢复类任务。

### 9.3 Redis Key 契约

Redis 只放热状态，key 需要统一前缀：

- `rayip:lease:node:{node_id}`：节点在线 lease。
- `rayip:session:node:{node_id}`：NodeAgent 所属 API instance。
- `rayip:metrics:node:{node_id}`：节点实时指标窗口。
- `rayip:idempotency:{scope}:{key}`：短期幂等热缓存。
- `rayip:ratelimit:{scope}:{id}`：API 限流。
- `rayip:ssh:{session_id}`：Web SSH 会话 TTL。
- `rayip:reservation:expire:{reservation_id}`：预约过期提醒。

Redis key 丢失不能改变 Postgres 中的钱包、库存、订单事实。

### 9.4 Proto 契约

`packages/proto` 至少包含：

- `control.v1`：NodeAgent 双向控制流、lease、任务帧、终端帧。
- `runtime.v1`：Runtime batch、账号 delta、策略、apply result、snapshot digest。
- `metrics.v1`：节点指标、账号流量、聚合采样。
- `common.v1`：错误码、状态枚举、分页 cursor、版本能力。

Protobuf 字段命名和枚举一旦进入测试环境，只允许兼容性演进；删除字段必须走废弃流程。

## 10. API 协议

前端 API：

- HTTP JSON
- Cookie 登录态
- 管理端权限走 RBAC

开发者 API：

- 支持一个用户创建多个 App
- 每个 App 独立 `AppID + AppSecret`
- 支持 IP 白名单
- 支持回调地址
- 支持重置 Secret

NodeAgent API：

- gRPC 双向流
- Protobuf 契约
- NodeAgent 主动连接 API

Web SSH：

- 浏览器到 API：WebSocket
- API 到 NodeAgent：gRPC stream frame

实时节点面板：

- 浏览器到 API：WebSocket
- NodeAgent 到 API：gRPC 指标上报

## 11. 构建和质量

后端：

- 领域服务单元测试
- 数据库 repository 集成测试
- 钱包/库存/订单并发测试
- NodeAgent gRPC 契约测试
- 下单到发货 E2E 冒烟

前端：

- TypeScript 检查
- 构建检查
- 登录、充值、购买、我的代理、节点面板关键流测试

NodeAgent：

- Xray 配置生成测试
- Runtime Bundle 版本兼容测试
- 重连和对账测试
- 新安装/重装恢复已有订单测试
- 指标上报测试

可靠性：

- NATS outbox 重放测试
- 幂等重复请求测试
- Redis 不可用降级测试
- Bark 通知失败不影响主流程测试

发布门槛：

- API 测试通过
- 前端构建通过
- NodeAgent 构建通过
- 测试环境能完成充值模拟、购买、发货、使用代理
