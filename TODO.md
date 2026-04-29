# RayIP V1 项目交付计划

> 规划原则：Runtime first，但不是纯技术 spike。每个 Task 都必须形成可运行、可验收的产品能力，至少有管理端可见性、API/数据/任务链路、节点或 Runtime 验证、测试证据。商业目标放在需求优先级最前，技术实现顺序先打掉 NodeAgent + 魔改 XrayCore + gRPC 增量下发 + Xray 内限速这些最大未知数。

## 0. 项目目标

RayIP V1 是一个可商业运营的 toC 静态家宽代理销售平台。

V1 只做：

- 静态家宽 IP
- SOCKS5
- HTTP

核心验收：

- 客户能注册、充值、购买、获得可用代理、续费、停用、刷新 IP、刷新凭据、查看流量。
- 管理员能管理产品、价格、库存、节点、订单、任务、通知、Web SSH、退役和恢复。
- 客户看到可购买，就必须已经满足可售、可发货、可使用。
- Runtime 未确认前，客户不能看到代理凭据。
- Postgres 是根源状态，Redis 是实时状态，NATS JetStream 是持久任务队列。
- NodeAgent 与魔改 XrayCore 作为 Runtime Bundle 一起安装、升级、回滚。
- NodeAgent `.env` 只保留 bootstrap 必要项；Runtime 版本、能力、限速池、账号禁用状态、滥用阈值和合规策略必须通过自发现、协商和控制面动态下发获得。
- 托管 XrayCore gRPC 端口必须采用 `auto` 随机模式：每次启动先探测本机环回端口，启动或探活失败就换端口重试，避免与 3x-ui、旧 XrayTool 或其他 XrayCore 冲突。
- 账号 Disabled、限速、连接数、合规和滥用处置必须由平台云控决策并下发；NodeAgent/XrayCore 只执行策略和上报事件，不做本地业务决策。
- Runtime Bundle 是签名供应链资产；生产节点必须校验 manifest、hash、签名和 minimum allowed version。
- 用户面板必须对齐 `frontend_design/` 的 RocketIP / IPIPD 风格：左侧分组导航、顶部余额和充值、紧凑购买页、强复制/导出/续费操作。

## 1. 交付结构

RayIP V1 分 3 个 Milestone、11 个厚 Task。

| Milestone | 目标 | Task |
|---|---|---|
| M1：Runtime 可行性闸门 | 证明节点能接、Xray 能控、策略能生效、任务能可靠送达，避免把商业流程建在不确定 Runtime 上 | T1-T4 |
| M2：商业交易闭环 | 在已验证 Runtime 上完成可售目录、钱包充值、首单购买、生命周期和客户自助 | T5-T8 |
| M3：生产运营闭环 | 完成管理运营、节点恢复、Web SSH、发布硬化和 Test 环境验收 | T9-T11 |

Task 完成定义：

- 有明确用户价值或管理员价值。
- 有前端入口或管理端入口。
- 有 API、领域逻辑、数据库迁移和关键约束。
- 涉及异步动作时，必须有 outbox / NATS / Worker 和幂等处理。
- 涉及 Runtime 时，必须打通 API -> NodeAgent -> XrayCore，并有结构化 apply result。
- 有管理端可见性：状态、结果、错误原因、审计或任务时间线。
- 有单元、集成、契约或 E2E 验收。
- 涉及用户面板时，必须按 [用户故事与前端设计计划](./docs/plans/rayip-v1-user-stories-and-frontend-plan.md) 做设计和流程验收。
- 开工前必须按 [开发前契约冻结清单](./docs/plans/rayip-v1-contract-freeze-checklist.md) 冻结本 Task 的最小契约。
- 不满足上述条件，不能算 Task 完成。

## 2. Milestone 与 Task

## M1：Runtime 可行性闸门

目标：先证明 RayIP 的数据面和控制面成立。没有这个闸门，不进入正式销售闭环。

### T1：薄工程基线 + 第一台节点在线

状态：已完成本地验收，等待后续提交后跑远端 CI。

价值：

- 项目可以端到端运行，管理员能看到真实 NodeAgent 在线/离线，而不是只有空服务。

交付范围：

- Monorepo 基线：`services/api`、`services/node-agent`、`packages/proto`、`apps/user-web`、`apps/admin-web`。
- API：Fiber + Fx + Zap + Viper，提供 health、ready、version。
- NodeAgent：Fx + Zap + Viper，能启动、读取配置、连接 API。
- Proto：control/runtime/metrics/common 的最小消息，能生成 Go 代码。
- 管理面板：节点列表、节点详情、节点在线状态。
- 用户面板：登录壳和购买入口壳即可，不做商业逻辑。
- 基础设施：本地 Postgres / Redis / NATS compose。
- Redis：节点 lease 和 session route。
- DB：nodes、node_agent_sessions、node_capability_snapshots 的最小版本。
- CI：Go test/build、前端 typecheck/build。

验收：

- [x] 本地一条命令启动 Postgres / Redis / NATS。
- [x] API health / ready 可访问。
- [x] NodeAgent 可连接 API 并保持 lease。
- [x] 管理端能看到真实 NodeAgent 在线/离线。
- [x] API 重启后 NodeAgent 可自动重连。
- [x] 前后台可构建，Proto 可生成 Go 代码，CI 工作流已落地，本地同等命令通过。

### T2：XrayCore Runtime 能力 P0

价值：

- 证明魔改 XrayCore 可以兑现 RayIP 产品承诺：SOCKS5/HTTP、账号级限速、连接数限制、流量统计和 digest。

交付范围：

- XrayCore fork：作为独立仓库维护，RayIP 用 `third_party/xray-core` submodule 固定源码指针。
- Runtime Bundle Supply Chain：manifest、binary hash、签名、allowed channel、minimum allowed version、原子切换和回滚。
- ZTP：bootstrap token 只用于首次注册，注册后使用短期 node credential / session identity。
- Capability Negotiation：NodeAgent 从 manifest + XrayCore 扩展 API 自发现能力，API 返回 `ACCEPTED / NEEDS_UPGRADE / QUARANTINED / UNSUPPORTED_CAPABILITY / DIGEST_MISMATCH`。
- XrayCore：实现或验证 `UpsertAccount`、`DeleteAccount`、`DisableAccount`、`UpdatePolicy`、`GetUsage`、`GetDigest`。
- XrayCore：实现账号级 token bucket、智能公平限速、连接数限制、usage stats、abuse event、runtime digest。
- NodeAgent：本机 gRPC Xray API / 扩展 API client，管理 XrayCore 进程、随机 gRPC 端口探测/重试和 Runtime health。
- API：Runtime Lab 接口，允许管理员创建测试账号、更新策略、禁用账号、读取流量和 digest。
- 管理面板：Runtime Lab 页面，展示测试代理、策略、流量、digest、apply 结果。
- DB：runtime lab 测试记录、runtime_apply_results、node_runtime_capabilities、runtime_bundle_versions 的最小结构。
- 测试：账号增量创建/删除、重复 apply 幂等、限速、连接数限制、流量统计、digest。

验收：

- [ ] 管理员可在 Runtime Lab 创建 SOCKS5/HTTP 测试代理。
- [ ] NodeAgent 不依赖 env 声明 Runtime 能力，能力来自 manifest 和 XrayCore 扩展 API 自发现。
- [ ] API 对 Runtime Bundle 和能力协商做准入，不兼容节点进入 DEGRADED / QUARANTINED。
- [ ] Runtime Bundle manifest、binary hash、签名校验失败时节点不可售。
- [ ] 测试代理可真实连接。
- [ ] XrayCore 内账号级限速生效。
- [ ] XrayCore 内基于优先级、固定限速和短期流量消耗的智能公平限速生效。
- [ ] XrayCore 内账号级连接数限制生效。
- [ ] XrayCore 可返回账号级上下行流量。
- [ ] XrayCore 可上报滥用事件，但不本地禁用或改限速；API 可按策略禁用、限速、上报或人工审核并通过 RuntimeCommand 下发。
- [ ] NodeAgent 启动托管 XrayCore 时使用随机 loopback gRPC 端口，端口占用或探活失败会换端口重试。
- [ ] 重复 apply 同一 generation 不产生副作用。

### T3：可靠增量下发通道

价值：

- 证明任务可以从 Postgres 期望状态可靠送到 NodeAgent/XrayCore，重复投递、断线、重试都不会破坏 Runtime。

交付范围：

- API：Runtime 期望状态、change seq、outbox 事件、手动测试代理任务。
- NATS JetStream：stream、subject、durable consumer、ack、redelivery、DLQ。
- Worker：读取 outbox/NATS 消息，回 Postgres 读取当前期望状态，生成 Runtime batch。
- NodeAgent：接收 batch，按账号/generation apply，返回账号级成功、失败、跳过、重复。
- XrayCore：继续使用 T2 的增量 API。
- Runtime 下发：采用 `version_info + nonce + ACK/NACK + last_good_generation`，NACK 必须带错误细节。
- 管理面板：最小任务控制台，展示任务状态、attempt、错误分类、apply result。
- DB：runtime_account_states、runtime_change_log、outbox_events、runtime_apply_results、node_jobs、node_job_attempts 的最小闭环。
- 测试：重复消息、Worker 重启、NodeAgent 断线重连、apply 部分失败、outbox 重放。

验收：

- [ ] 管理员创建测试代理时，链路是 PG -> outbox -> NATS -> Worker -> NodeAgent -> XrayCore。
- [ ] NATS 消息只带索引，不带业务事实大 payload。
- [ ] Worker 必须回 Postgres 读取当前期望状态。
- [ ] 重复消息不会重复修改 Runtime。
- [ ] NodeAgent 对同一 `version_info + nonce` 的重复下发返回一致 ACK/NACK，不重复改 Runtime。
- [ ] NACK 后 API 保留 last good generation，并在管理端展示错误原因。
- [ ] NodeAgent 断线重连后能继续处理未完成变更。
- [ ] 管理端能看到任务时间线和账号级 apply result。

### T4：节点能力、可售闸门和 10k 恢复验证

价值：

- 证明“可售”不是后台手填数量，而是由节点健康、Bundle 能力、Runtime digest 和恢复能力共同决定。

交付范围：

- API：节点健康评分、身份校验、Bundle 签名/hash/channel/能力协商、可售闸门、Runtime digest 对账。
- API：节点身份、Bundle 签名/hash、channel、minimum version、合规 hold、滥用状态纳入可售判断。
- NodeAgent：上报 bundle_version、XrayCore version、extension ABI、capabilities、manifest hash、binary hash、账号数、generation 水位、snapshot digest。
- XrayCore：digest 分桶、账号数量、异常账号列表。
- 管理面板：节点能力页、不可售原因、digest 对账结果。
- DB：node_runtime_status、node_capability_snapshots、runtime digest 对账记录。
- 压测/恢复：单节点 10k+ 测试账号分页 apply、分页 digest、断点续传。
- 测试：OFFLINE、DEGRADED、能力不匹配、digest 不一致时停止新销售。

验收：

- [ ] 节点 OFFLINE / DEGRADED / 能力不匹配时不可售。
- [ ] 节点 QUARANTINED / NEEDS_UPGRADE / 签名校验失败 / 合规 hold 时不可售。
- [ ] Runtime digest 不一致时相关库存不可售并提示原因。
- [ ] 单节点 10k+ 账号可分页 apply 和 digest 对账。
- [ ] 恢复过程支持断点续传，不依赖整份配置重写。
- [ ] 管理端能看到节点为什么不可售。

## M2：商业交易闭环

目标：在已验证 Runtime 上完成用户、钱包、可售目录、首单发货和主要生命周期能力。

### T5：用户、管理员、钱包和充值闭环

价值：

- 客户可以注册登录、查看余额、完成充值；管理员可以审计用户、钱包和充值流水。

交付范围：

- 用户面板：注册、登录、账户概览、余额、充值记录。
- 管理面板：用户列表、用户详情、钱包流水、充值订单、审计记录。
- API：用户认证、管理员认证、RBAC、钱包、不可变流水、充值订单。
- 易支付：充值模拟和易支付回调接口，回调必须幂等。
- DB：users、admin_users、wallets、wallet_ledger、payment_orders、audit_logs。
- 安全：密码 hash、Cookie/session 或 token 策略、管理员操作审计。
- 测试：登录、充值入账、重复回调不重复入账、钱包流水不可变。

验收：

- [ ] 用户可注册、登录、查看余额。
- [ ] 用户可发起充值模拟并入账。
- [ ] 同一充值回调重复投递不会重复入账。
- [ ] 所有充值、购买、续费入口必须有 Idempotency-Key 或第三方交易号幂等门闩。
- [ ] 管理员可查看用户、钱包流水、充值订单。
- [ ] 钱包流水不可修改，只能追加。

### T6：真实可售目录、产品价格和库存预约

价值：

- 客户能看到真实可售的地区、线路、协议、价格和数量；不可发货的库存不会展示。

交付范围：

- 用户面板：购买页的用途、IP 类型、国家/城市、线路、协议、时长、数量、价格预览、可售数量。
- 管理面板：产品、价格、用途、地区、城市、线路、库存 IP、rate policy、可售开关。
- API：产品目录、价格计算、可售库存查询、库存预约预检查。
- DB：products、product_prices、regions、cities、lines、node_inventory_ips、rate_policies、inventory_reservations。
- Redis：可售库存热缓存和预约过期提醒，但不能作为根源事实。
- 规则：节点必须 ACTIVE、身份有效、Bundle 签名/hash 通过、channel 允许、能力匹配、digest 一致、合规/滥用状态 clean、库存 AVAILABLE、线路启用、价格有效才可售。
- 测试：价格计算、不可售过滤、库存状态条件更新、并发预约防超卖。

验收：

- [ ] 管理员可配置产品、价格、地区、线路、库存和速率策略。
- [ ] 用户购买页只展示真实可售地区、线路和数量。
- [ ] 禁用线路、禁用库存、异常节点、能力不匹配节点不会出现在可售结果中。
- [ ] 并发库存预约不超卖。
- [ ] Redis 丢失不影响 Postgres 库存事实。

### T7：首单购买、钱包冻结和 Runtime 发货

价值：

- 客户购买 1 个静态家宽 SOCKS5/HTTP 后，能拿到真实可用代理。

交付范围：

- 用户面板：提交订单、余额支付、订单处理中、代理凭据展示。
- 管理面板：订单详情、代理账号详情、发货任务时间线。
- API：库存预约、钱包冻结、订单创建、代理账号创建、Runtime 期望状态、outbox。
- NATS/Worker：发货任务，消息只带索引，Worker 回 Postgres 读当前期望状态。
- NodeAgent/XrayCore：复用 M1 的可靠增量 apply，创建正式代理账号。
- DB：wallet_holds、proxy_orders、proxy_accounts、runtime_account_states、runtime_change_log、runtime_apply_results、outbox_events。
- 测试：并发下单不超卖、重复请求幂等、发货失败释放库存和冻结余额。

验收：

- [ ] 客户可购买一个静态家宽 SOCKS5/HTTP。
- [ ] Runtime 未确认前，客户看不到代理凭据。
- [ ] 发货成功后扣款并展示凭据。
- [ ] 发货失败后释放库存和冻结余额。
- [ ] 代理可实际连接并产生账号级流量。
- [ ] 重复下单请求返回同一业务结果，不重复扣款。

### T8：代理生命周期、IP 刷新、流量和开发者 App

价值：

- 客户可以管理已购买代理，平台状态和 Runtime 状态保持一致，并支持基础自动化调用。

交付范围：

- 用户面板：我的代理、代理详情、续费、停用、IP 刷新、凭据刷新、流量图、开发者 App、IP 白名单、Webhook。
- 管理面板：生命周期操作、IP 刷新记录、流量汇总、开发者 App、Webhook 投递、优惠券、邀请、反馈。
- API：续费冻结/确认、停用、过期任务、凭据刷新、IP 刷新、流量查询、开发者 API 签名、nonce、防重放、Webhook。
- NATS/Worker：renew、disable、expire、refresh_credential、refresh_ip、traffic_rollup、webhook_delivery、notification。
- NodeAgent/XrayCore：账号策略更新、禁用、凭据更新、IP 刷新 apply、账号流量读取、Runtime digest。
- DB：traffic_rollups、developer_apps、developer_app_secrets、developer_app_ip_allowlists、developer_app_webhooks、notification_events、coupons、referrals、feedback。
- 测试：续费同步 Runtime、停用后不可用、凭据刷新确认后展示、IP 刷新确认后展示、开发者 API 签名和防重放。

验收：

- [ ] 续费成功后平台过期时间和 Runtime 策略一致。
- [ ] 停用/过期后代理不可用。
- [ ] 凭据刷新 Runtime 确认后才展示新凭据。
- [ ] IP 刷新确认后才展示新 IP。
- [ ] 客户可查看每个代理上下行流量。
- [ ] 客户可创建多个开发者 App，AppID/AppSecret 签名、防重放、IP 白名单生效。
- [ ] Webhook 投递失败可重试，不影响主流程。

## M3：生产运营闭环

目标：完成商业运营所需的管理员体验、节点恢复能力和 Test 发布门槛。

### T9：管理员运营中心、任务控制台和 Bark 告警

价值：

- 管理员能知道系统是否可售、节点是否健康、任务是否成功，不需要靠客户投诉发现问题。

交付范围：

- 管理面板：概览、实时节点面板、任务控制台、订单时间线、不可售原因、Bark 配置。
- API：节点健康评分、任务查询、任务重试/取消/对账、通知事件。
- Redis：实时指标窗口。
- Postgres：分钟/小时指标聚合、任务历史、通知投递。
- NATS/Worker：Bark 通知、任务 DLQ、对账触发。
- NodeAgent：CPU、RAM、磁盘、连接数、速度、Runtime 状态、账号数上报。
- 测试：节点离线告警、任务失败告警、通知失败不影响主业务。

验收：

- [ ] 管理员能看到所有节点实时速度、CPU、RAM、存储、连接数、流量。
- [ ] 管理员能看到节点不可售原因。
- [ ] 管理员能查看任务时间线并安全重试或触发对账。
- [ ] Bark 可收到节点离线、DEGRADED、任务最终失败、库存低水位告警。

### T10：ZTP、Web SSH、节点退役和自动恢复

价值：

- 管理员可以方便接入、诊断、退役和恢复节点；节点故障不会变成客户侧不可解释事故。

交付范围：

- 管理面板：节点安装、复制一键脚本、Web SSH、退役流程、恢复记录。
- API：SSH 凭据加密、host key fingerprint、recovery token、退役状态机、恢复 Job。
- NATS/Worker：ssh_recover_node、retire_node、reconcile。
- NodeAgent：Web SSH PTY 桥接、重装恢复、last applied seq、snapshot digest。
- XrayCore：digest 对账、账号级分页恢复、重复 apply 幂等。
- DB：node_jobs、node_job_attempts、terminal audit、retirement state、recovery audit。
- 测试：SSH 自动恢复、NodeAgent 重装恢复、10k+ 账号分页恢复、退役中不卖新库存。

验收：

- [ ] 管理员可从后台 SSH 安装节点或复制一键脚本。
- [ ] 管理员可从节点详情打开 Web SSH。
- [ ] 退役中节点不再销售新库存。
- [ ] NodeAgent 重装后能恢复已有指派订单。
- [ ] SSH 可达时平台能自动修复离线 NodeAgent。
- [ ] 单节点 10k+ 账号恢复不需要整机重写配置。

### T11：Railway Test、发布门槛和生产硬化

价值：

- 项目可以在 Test 环境完整验收，具备进入生产运营的基本质量门槛。

交付范围：

- Railway：项目 `RayIP`，环境 `Test`，Postgres、Redis、NATS、API、user-web、admin-web。
- 节点：至少一个真实 Linux 节点部署 NodeAgent + XrayCore Runtime Bundle。
- 自动化：E2E 冒烟、并发下单、充值回调、outbox 重放、NodeAgent 重连/重装、Redis 降级、NATS 故障、Bark 失败。
- 安全：Secret 管理、SSH 凭据加密、管理员审计、开发者 API 防重放。
- 运维：备份策略、迁移策略、日志级别、错误码、发布回滚。
- 文档：部署说明、运维手册、发布检查表。

验收：

- [ ] Test 环境完成注册、充值模拟、购买、发货、使用代理、续费、停用、IP 刷新、凭据刷新。
- [ ] 并发下单不超卖。
- [ ] 发货失败不扣款。
- [ ] Runtime 未确认不展示凭据。
- [ ] NodeAgent 掉线后节点不可售。
- [ ] NodeAgent 重装恢复已有订单。
- [ ] Redis / NATS / Bark 故障场景有明确降级行为。
- [ ] 发布检查表全部通过。

## 3. 风险登记

| 风险 | 影响 | 前置处理 |
|---|---|---|
| 商业流程先于 Runtime | 订单、库存、钱包建在不确定数据面上，后期大返工 | M1 先完成 Runtime 可行性闸门 |
| XrayCore 魔改限速/连接数能力不足 | 产品策略无法兑现 | T2 必须真实验证账号级限速和连接数 |
| XrayCore 增量 API 不完整 | 日常发货、续费、停用、刷新只能靠重写配置 | T2 必须覆盖 upsert/delete/disable/update policy/usage/digest |
| 可靠任务链路不成立 | 重复投递、断线、重试导致 Runtime 状态错乱 | T3 固定 PG 期望状态 + outbox + NATS + generation + apply result |
| 可售库存是手工判断 | 客户能买到不可发货或不可用库存 | T4 让可售依赖节点健康、Bundle 能力和 Runtime digest |
| 钱包/库存事务边界错误 | 超卖、重复扣款、退款错误 | T6/T7 必须包含并发和幂等测试 |
| NodeAgent 重装恢复不幂等 | 已售代理丢失或重复下发 | T4/T10 必须完成 generation、snapshot digest、分页恢复 |
| 管理面板变成失败补救列表 | 主流程不可靠 | 客户动作必须以 Runtime 确认后对客户可见为验收 |
| Redis 职责越界 | 热状态丢失影响订单事实 | 钱包、库存、订单、幂等结果只以 Postgres 为准 |

## 4. 发布前总验收

- [ ] 客户侧：注册、登录、充值、余额、购买、代理详情、流量、续费、停用、IP 刷新、凭据刷新、反馈。
- [ ] 管理侧：用户、钱包、账单、充值、产品价格、地区线路、库存、订单、代理账号、节点、退役、任务、通知、审计。
- [ ] 节点侧：ZTP、enrollment、Bundle 检查、增量下发、流量上报、重连恢复、重装恢复、Web SSH。
- [ ] Runtime：XrayCore 内限速、连接数限制、流量统计、digest、账号级增量 apply 生效。
- [ ] 正确性：不可售库存不展示、发货失败不扣款、Runtime 未确认不展示凭据、续费/停用/过期同步 Runtime、并发下单不超卖。
- [ ] 运维性：管理员能看到节点健康、不可售原因、任务失败原因，并能安全触发重试、对账、退役、SSH。
