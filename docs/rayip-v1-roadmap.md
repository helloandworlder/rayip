# RayIP V1 路线图

> 版本：草案 v2  
> 日期：2026-04-30  
> 原则：Runtime first，厚任务交付，客户流程和管理员流程一起验收

## 1. 路线图结论

RayIP V1 不按“后端、前端、数据库、部署”分层推进，而按风险和产品闭环推进。

V1 分为 3 个 Milestone、11 个厚 Task：

| Milestone | 目标 | Task |
|---|---|---|
| M1：Runtime 可行性闸门 | 先证明 NodeAgent、魔改 XrayCore、gRPC 增量下发、Xray 内限速、节点可售判断成立 | T1-T4 |
| M2：商业交易闭环 | 在已验证 Runtime 上完成注册、钱包、充值、可售目录、下单发货、生命周期 | T5-T8 |
| M3：生产运营闭环 | 完成管理运营、实时节点面板、Bark、ZTP、Web SSH、节点恢复、Railway Test 发布门槛 | T9-T11 |

详细执行清单以项目根目录 [TODO.md](../TODO.md) 为准。本文档负责说明路线图顺序、风险闸门和每个 Milestone 的退出条件。

## 2. 为什么 Runtime 要放在最前面

RayIP 的商业承诺是：

> 客户能看到可购买，就必须已经满足可售、可发货、可使用。

这个承诺依赖 Runtime 能力，而不是依赖购买页是否好看：

- NodeAgent 必须能稳定连接控制面。
- 魔改 XrayCore 必须能按账号限速、限制连接数、统计流量。
- Runtime Bundle 必须可签名发布、可验签、可回滚，能力必须自发现和协商。
- 节点接入必须符合 ZTP：bootstrap 最小化，注册后使用短期 node credential / session identity。
- gRPC API 必须能增量下发账号、策略、凭据、停用和刷新。
- Postgres 期望状态、NATS JetStream 持久任务、Worker、NodeAgent、XrayCore 必须幂等。
- 节点离线、Bundle 不兼容、签名/hash 校验失败、能力缺失、digest 不一致、合规/滥用 hold 时必须阻止新销售。
- 单节点 10k+ 账号恢复必须可分页、可断点续传。

如果这些能力最后才做，商业流程会建在不确定基础上，容易重演 GoSeaLight 当前的问题：能下单但不一定能发货、发货了不一定能用、异常后只能靠管理员补救。

## 3. M1：Runtime 可行性闸门

目标：在进入完整商业交易前，证明数据面、控制面、节点状态和可售闸门成立。

### T1：薄工程基线 + 第一台节点在线

交付：

- Monorepo 基线：`services/api`、`services/node-agent`、`packages/proto`、`apps/user-web`、`apps/admin-web`。
- API：Fiber、Fx、Zap、Viper，health / ready / version。
- NodeAgent：启动、读取配置、主动连接 API。
- Redis：节点 lease 和 session route。
- DB：nodes、node_agent_sessions、node_capability_snapshots 最小表。
- 管理端：节点列表和在线状态。
- 用户端：按 `frontend_design/` 建立登录和主面板骨架。

退出：

- API、NodeAgent、前后台、基础设施可本地启动。
- 管理端能看到真实 NodeAgent 在线/离线。
- API 重启后 NodeAgent 可自动重连。

### T2：XrayCore Runtime 能力 P0

交付：

- 魔改 XrayCore 的账号级 SOCKS5 / HTTP 管理。
- XrayCore fork 作为独立仓库维护，RayIP 用 submodule 固定源码指针。
- Runtime Bundle manifest、hash、签名、allowed channel、minimum version、原子升级和回滚。
- NodeAgent ZTP bootstrap、Runtime 自发现和能力协商。
- `UpsertAccount`、`DeleteAccount`、`DisableAccount`、`UpdatePolicy`、`GetUsage`、`GetDigest`。
- XrayCore 内账号级限速、智能公平限速、连接数限制、流量统计、滥用检测和 digest。
- NodeAgent 到 XrayCore 的本机 gRPC Xray API / 扩展 API。
- 管理端 Runtime Lab。

退出：

- NodeAgent 不靠 env 声明 Runtime 能力，能力来自 manifest 和 XrayCore 扩展 API。
- API 能对 Bundle、扩展 ABI、hash、capabilities 做准入，失败节点不可售。
- 管理员可创建真实测试代理。
- 测试代理可连接。
- 固定限速、智能公平限速、连接数、流量统计、滥用禁用/上报可验证。
- 同一 generation 重复 apply 不产生副作用。

### T3：可靠增量下发通道

交付：

- Postgres Runtime 期望状态、变更序列、outbox。
- NATS JetStream stream、subject、durable consumer、ack、redelivery、DLQ。
- Worker 从 NATS 收消息后回 Postgres 读取当前期望状态。
- API -> Worker -> NodeAgent -> XrayCore 增量 batch。
- Runtime 下发采用 `version_info + nonce + ACK/NACK + last_good_generation`。
- 管理端任务控制台最小闭环。

退出：

- 创建测试代理走 PG -> outbox -> NATS -> Worker -> NodeAgent -> XrayCore。
- 重复消息、Worker 重启、NodeAgent 断线重连都不破坏 Runtime。
- NACK 不推进 accepted generation，管理端能看到 last good generation 和错误原因。
- 管理端能看到任务时间线和账号级 apply result。

### T4：节点能力、可售闸门和 10k 恢复验证

交付：

- 节点健康评分、node identity、Bundle 签名/hash/channel、能力协商、Runtime digest。
- 可售判断依赖节点状态、身份、供应链校验、能力、库存、价格、digest、合规/滥用状态。
- 单节点 10k+ 测试账号分页 apply、分页 digest、断点续传。
- 管理端节点能力页和不可售原因。

退出：

- OFFLINE、DEGRADED、DRAINING、QUARANTINED、NEEDS_UPGRADE、能力不匹配、签名/hash 校验失败、digest 不一致、合规/滥用 hold 的节点不卖新库存。
- 单节点 10k+ 账号恢复不需要整机重写配置。
- 用户端不可售地区/线路/库存不展示。

## 4. M2：商业交易闭环

目标：基于 M1 已验证 Runtime，完成客户真实购买和生命周期。

### T5：用户、管理员、钱包和充值闭环

交付：

- 用户注册、登录、概览、余额、充值、计费流水。
- 管理员登录、RBAC、用户、钱包、充值订单、审计。
- 易支付充值模拟和正式回调接口。
- 钱包不可变流水、充值回调幂等。

退出：

- 用户可注册、登录、充值并看到余额变化。
- 同一充值回调重复投递不会重复入账。
- 管理员可审计用户、钱包和充值订单。

### T6：真实可售目录、产品价格和库存预约

交付：

- 用户购买页：用途、IP 类型、国家/城市、线路、协议、时长、数量、价格预览、可售数量。
- 管理端：产品、价格、用途、地区、城市、线路、库存 IP、速率策略。
- Postgres 库存预约和并发防超卖。
- Redis 只做热缓存和过期提醒。

退出：

- 用户只看到真实可售选项。
- 并发预约不超卖。
- 禁用线路、异常节点、能力不匹配节点不参与可售。

### T7：首单购买、钱包冻结和 Runtime 发货

交付：

- 下单、钱包冻结、库存预约、订单创建、代理账号创建、Runtime 期望状态和 outbox 同事务。
- 发货 Worker 增量下发正式代理账号。
- 用户端订单处理中和凭据展示。
- 管理端订单详情和发货时间线。

退出：

- 客户可购买一个静态家宽 SOCKS5/HTTP 并实际连接。
- Runtime 未确认前不展示凭据。
- 发货成功后扣款，失败后释放库存和冻结余额。
- 重复下单请求返回同一结果，不重复扣款。

### T8：代理生命周期、IP 刷新、流量和开发者 App

交付：

- 用户端：我的代理、代理详情、续费、停用、IP 刷新、凭据刷新、流量图、开发者 App、Webhook。
- 管理端：生命周期操作、IP 刷新记录、流量汇总、开发者 App、Webhook 投递、优惠券、邀请、反馈。
- Worker：renew、disable、expire、refresh_credential、refresh_ip、traffic_rollup、webhook_delivery、notification。
- 开发者 API：多个 App、`AppID + AppSecret`、签名、防重放、IP 白名单。

退出：

- 续费、停用、过期和 Runtime 状态一致。
- IP 刷新和凭据刷新必须 Runtime 确认后才展示。
- 客户可查看每个代理流量。
- 开发者 App 和 Webhook 可用，失败重试不影响主流程。

## 5. M3：生产运营闭环

目标：让 RayIP V1 具备真实运营能力，而不是只能跑演示流程。

### T9：管理员运营中心、任务控制台和 Bark 告警

交付：

- 管理端概览、实时节点面板、任务控制台、订单时间线、不可售原因、Bark 配置。
- NodeAgent 上报 CPU、RAM、磁盘、速度、连接数、Runtime 状态、账号数。
- Redis 实时窗口，Postgres 分钟/小时聚合。
- Bark 通知事件先落 Postgres，再由 Worker 投递。

退出：

- 管理员能看到所有节点实时速度、CPU、RAM、存储、连接数、流量。
- 管理员能看到节点不可售原因和任务失败原因。
- Bark 可收到节点离线、DEGRADED、任务最终失败、库存低水位告警。

### T10：ZTP、Web SSH、节点退役和自动恢复

交付：

- 管理端节点安装、复制一键脚本、Web SSH、退役流程、恢复记录。
- SSH 凭据加密、host key fingerprint、recovery token。
- SSH 自动恢复 NodeAgent + XrayCore Runtime Bundle。
- NodeAgent 重装恢复已有指派订单。
- 节点退役状态机。

退出：

- 管理员可后台 SSH 安装节点或复制一键脚本。
- 管理员可从节点详情打开 Web SSH。
- NodeAgent 重装后恢复已有订单。
- 退役中节点不卖新库存。
- SSH 可达时平台能自动修复离线 NodeAgent。

### T11：Railway Test、发布门槛和生产硬化

交付：

- Railway 项目 `RayIP`，环境 `Test`。
- Postgres、Redis、NATS、API、user-web、admin-web。
- 至少一台真实 Linux 节点部署 NodeAgent + XrayCore Runtime Bundle。
- E2E 冒烟、并发下单、充值回调、outbox 重放、NodeAgent 重连/重装、Redis/NATS/Bark 故障演练。
- Secret 管理、SSH 凭据加密、管理员审计、开发者 API 防重放、备份和回滚检查表。

退出：

- Test 环境完成注册、充值、购买、发货、使用代理、续费、停用、IP 刷新、凭据刷新。
- 并发下单不超卖。
- 发货失败不扣款。
- Runtime 未确认不展示凭据。
- NodeAgent 掉线后节点不可售。
- Redis / NATS / Bark 故障有明确降级行为。
- 发布检查表通过。

## 6. V1 发布门槛

客户侧必须通过：

- 注册、登录、充值、余额、购买、代理详情、复制凭据、流量、续费、停用、IP 刷新、凭据刷新、优惠券、推广、反馈。

管理侧必须通过：

- 用户、钱包、账单、充值、产品价格、地区线路、库存、订单、代理账号、节点、退役、实时节点面板、任务、通知、审计、Web SSH。

节点侧必须通过：

- ZTP 安装、NodeAgent enrollment、Bundle 检查、增量下发、XrayCore 内限速、连接数限制、流量上报、重连恢复、重装恢复。

正确性必须通过：

- 不可售库存不展示。
- 发货失败不扣款。
- Runtime 未确认不展示凭据。
- 续费、停用、过期同步 Runtime。
- 并发下单不超卖。
- NodeAgent 掉线后停止新销售。

如果某个 GoSeaLight 静态家宽核心功能没有覆盖，必须明确标为“V1 不做”或“后置版本”，不能以遗漏形式进入开发。
