# RayIP V1 产品交付主线

> 版本：产品主线 v3
> 日期：2026-04-30
> 口径：可售真实性优先，首单闭环优先，生命周期其次，运营生产化最后。

`TODO.md` 是 RayIP V1 后续执行的唯一主线文档。`docs/rayip-v1-roadmap.md` 负责解释路线图成熟度，`docs/rayip-v1-requirements.md` 负责约束功能和验收口径；执行状态以本文为准。

## 0. V1 产品目标

RayIP V1 是面向 C 端客户销售静态家宽代理的平台。

V1 只销售：

- 静态家宽 IP
- SOCKS5
- HTTP

V1 不做：

- 专线产品
- 动态住宅池
- VMess / VLESS / Trojan / Shadowsocks / HY2 对外销售
- 节点本地 UI
- 3x-ui 集成
- 多租户分销体系

客户必须能完成：注册、充值、购买静态家宽 SOCKS5/HTTP，Runtime ACK 后拿到真实可用代理，并能续费、停用、刷新 IP/凭据、查看流量。

管理员必须能完成：管理节点、库存、订单、任务、生命周期、故障恢复和 Test 环境验收。

## 1. 产品不可变规则

- 客户看到可买，必须已经满足：节点在线、候选公网 IP 可用、协议能力匹配、Runtime digest 一致、库存可用、线路启用、价格有效。
- Runtime ACK 前不展示代理密码、连接串或新凭据。
- 钱包、订单、库存、Runtime desired state 以 Postgres 为根源事实。
- Redis 只做在线态、实时指标、短期缓存和限流，不承载根源事实。
- NATS JetStream 只承载持久任务队列；消息只带索引，Worker 必须回 Postgres 读事实。
- 健康检查、延迟、测速统一归入节点扫描 Worker，不拆独立测速模块。
- 管理端可以展示 Runtime/NATS/Worker 等内部概念，客户界面不能暴露这些概念。
- 任何充值、下单、续费、停用、刷新、Runtime apply 都必须幂等，最终结果落 Postgres。

## 2. 当前真实状态

### 已完成

- Monorepo 基线已存在：`services/api`、`services/node-agent`、`packages/proto`、`apps/user-web`、`apps/admin-web`。
- T1 本地工程基线已完成：API health/ready、NodeAgent lease、管理端节点在线/离线、基础 compose、前后台构建链路。
- Runtime desired state 基座已存在：`runtime_account_states`、`runtime_change_log`、`outbox_events`。
- Runtime outbox、NATS JetStream subject、durable worker、job/attempt 记录和手动 process 管理入口已有实现基础。
- `node_runtime_status` 和 `noderuntime` 服务已有，能计算节点可售状态和不可售 reason code。
- Control gRPC 已能把 RuntimeObservation / RuntimeApplyAck 写入节点 runtime 状态。
- snapshot/reconcile planner 已支持按 desired state 分页构建 `SNAPSHOT` apply。
- 商业骨架已有部分落地：用户、钱包、价格、库存、订单、代理账号等 Ent 模型和商业服务骨架已经在工作树中出现。
- NodeAgent 已有候选公网 IP 发现、控制面探测和节点扫描队列相关代码基础。

### 部分完成

- T2 Runtime P0 已有 NodeAgent/XrayCore 管理代码和测试骨架，但真实 XrayCore 内账号级限速、连接数、usage、digest 仍需实测闭环。
- T3 desired-state + NATS Worker 已有主体链路，但必须用真实 Runtime ACK/NACK 和断线重连场景补齐验证。
- T4 可售判断已有状态模型，但必须改成“候选公网 IP -> 平台逐 IP 探测 -> 通过后入库存”的产品主线。
- T5-T7 商业模型已有基础，但首单从注册、充值、购买、钱包冻结、库存预约、Runtime ACK、扣款交付的真实 smoke 尚未完成。

### 未完成

- 真实 Linux 节点上的 SOCKS5/HTTP 入站和出站逐 IP 验证。
- 平台扫描样例 `IP:PORT:user:pass` 的队列化健康探测、延迟、失败 reason code 和入库闸门。
- Runtime ACK 前后客户凭据可见性闸门。
- NACK/超时后 wallet hold、inventory reservation 和 proxy account 的补偿释放。
- 续费、停用、过期删除、刷新 IP、刷新凭据和流量展示的真实 Runtime smoke。
- 管理运营中心、任务控制台、Web SSH、退役、恢复和 Test 环境故障演练。

### 必须真实 smoke

- 管理员创建测试 SOCKS5/HTTP 代理并真实连接。
- NodeAgent 扫描网卡公网 IP，平台逐个扫描 `204.42.251.2:9878:testuser1:testpass1` 这种代理候选。
- 候选 IP 必须证明可入站、可出站，未通过、私网、CGNAT、默认出口漂移 IP 不进入可售目录。
- 用户从 0 开始注册、充值、购买 1 个静态家宽代理，ACK 后可复制连接串并真实连接。
- 停用/过期后 Runtime remove resource，原代理连接失败。

## 3. 产品闸门

| Gate | 名称 | 退出标准 |
|---|---|---|
| G1 | Runtime 可控可验收 | NodeAgent + XrayCore 能创建 SOCKS5/HTTP 账号，限速、连接数、流量、digest 有实测证据，重复 apply 不重复改 Runtime |
| G2 | 真实节点可售 | NodeAgent 上报 `candidate_public_ips`，平台队列逐 IP 探测代理连通性和延迟，通过后才进入库存 |
| G3 | 首单商业闭环 | 注册、充值、余额下单、钱包冻结、库存预约、Runtime 发货、ACK 后扣款交付，NACK/超时可补偿 |
| G4 | 生命周期自助 | 我的代理、续费、停用、过期删除、刷新 IP/凭据、流量展示全部以 Runtime 确认为准 |
| G5 | 生产运营闭环 | 管理驾驶舱、任务控制台、告警、Web SSH、退役、恢复、Test 环境 E2E 和故障演练完成 |

## 4. 交付结构

| Milestone | 产品成熟度 | Gate | Task |
|---|---|---|---|
| M1：可控可售基础 | Runtime 能控，真实节点能被证明可售 | G1-G2 | T1-T4 |
| M2：首单与生命周期 | 客户能完成真实购买和主要自助操作 | G3-G4 | T5-T8 |
| M3：生产运营 | 管理员能运营、恢复、发布和演练 | G5 | T9-T11 |

## 5. Task 主线

### T1：工程基线与真实 NodeAgent 在线

Status: 已完成本地验收，后续提交前仍需跑对应 CI。

Value:
- 管理员能看到真实 NodeAgent 在线/离线，项目能端到端启动。

Why now:
- 所有 Runtime、库存和商业链路都依赖节点在线态和基础服务可运行。

Scope:
- Monorepo 基线、API health/ready/version、NodeAgent enrollment/lease、Redis 在线态、Postgres 节点表、管理端节点列表、前后台构建、compose。

Acceptance:
- [x] 本地一条命令启动 Postgres / Redis / NATS。
- [x] API health / ready 可访问。
- [x] NodeAgent 可连接 API 并保持 lease。
- [x] 管理端能看到真实 NodeAgent 在线/离线。
- [x] API 重启后 NodeAgent 可自动重连。
- [x] 前后台可构建，Proto 可生成 Go 代码。

Evidence:
- 已有本地验收记录；提交前补远端 CI 结果。

### T2：Runtime P0：SOCKS5/HTTP、限速、连接数、usage、digest

Status: 部分完成，真实 XrayCore 行为仍是 G1 阻塞项。

Value:
- 管理员能创建测试代理，并证明 RayIP 真实能兑现 SOCKS5/HTTP、限速、连接数、流量统计和 digest。

Why now:
- Runtime 是商业发货的最大技术风险；不证明 Runtime，就不能推进首单闭环。

Scope:
- NodeAgent 托管 XrayCore，随机 loopback gRPC 端口探测/重试。
- XrayCore 扩展 API：`UpsertAccount`、`DeleteAccount`、`DisableAccount`、`UpdatePolicy`、`GetUsage`、`GetDigest`。
- SOCKS5/HTTP 测试账号真实创建、连接、删除。
- 账号级限速、连接数限制、usage stats、digest。
- Runtime Bundle manifest、hash、签名、capabilities、extension ABI。
- 管理端 Runtime Lab 和 apply result。

Acceptance:
- [ ] 管理员可在 Runtime Lab 创建 SOCKS5 和 HTTP 测试代理。
- [ ] 测试代理可真实连接。
- [ ] 账号级限速有可复现实测证据。
- [ ] 账号级连接数限制有可复现实测证据。
- [ ] 账号级上下行 usage 可读取并和连接测试对应。
- [ ] Runtime digest 可复现，和期望状态一致。
- [ ] 重复 apply 同一 generation 不产生副作用。
- [ ] Bundle manifest、hash、签名或能力校验失败时节点不可售。

Evidence:
- 必须保留命令、日志、curl/socks/http 连接结果、usage 前后值和 digest 对比。

### T3：Runtime desired-state + NATS Worker 可靠下发

Status: 部分完成，需接入真实 Runtime ACK/NACK 验证。

Value:
- 业务动作可以可靠变成 Runtime 期望状态，并通过 NATS Worker 幂等下发到 NodeAgent/XrayCore。

Why now:
- 首单、续费、停用、刷新都依赖同一条可靠 apply 通道。

Scope:
- Postgres desired state、change log、outbox。
- NATS JetStream stream/subject/durable consumer/ack/redelivery/DLQ。
- Worker 收索引消息后回 Postgres 读取事实，组装 Runtime batch。
- NodeAgent 以 `version_info + nonce + ACK/NACK + last_good_generation` 回执。
- 管理端任务控制台最小闭环。

Acceptance:
- [ ] NATS 消息只带索引，不带业务事实大 payload。
- [ ] Worker 必须回 Postgres 读取当前 desired state。
- [ ] 重复消息不会重复修改 Runtime。
- [ ] NodeAgent 对重复 `version_info + nonce` 返回一致 ACK/NACK。
- [ ] NACK 后 API 保留 last good generation，并展示错误原因。
- [ ] NodeAgent 断线重连后能继续处理未完成变更。
- [ ] 管理端能看到任务时间线和账号级 apply result。

Evidence:
- 单测覆盖重复消息、Worker 重启、断线重连、部分失败；集成 smoke 覆盖 PG -> outbox -> NATS -> Worker -> NodeAgent -> Runtime。

### T4：家宽节点可售发现：候选公网 IP、健康扫描、不可售原因

Status: 部分完成，是 G2 当前主线。

Value:
- 客户看到的库存来自真实可连接的家宽 IP，而不是人工填数或只看节点在线。

Why now:
- 首单闭环前必须证明“可售”真实，否则会出现能下单但不可用。

Scope:
- NodeAgent 扫描网卡、路由和出口，生成 `candidate_public_ips`。
- 平台节点扫描 Worker 为每个候选 IP 创建临时 SOCKS5/HTTP 探测凭据。
- 平台队列逐条扫描 `IP:PORT:user:pass`，验证入站可达、出站出口一致、延迟和基础吞吐。
- 私网、CGNAT、默认出口漂移、端口不可达、认证失败、协议不匹配、digest mismatch 全部写 reason code。
- 通过扫描的 IP 才进入 `node_inventory_ips` 的 `AVAILABLE` 候选，未通过不进入客户可售目录。
- 管理端展示候选 IP、扫描状态、延迟、失败原因、最近成功时间。

Acceptance:
- [ ] NodeAgent 上报 `candidate_public_ips`。
- [ ] 平台队列能逐个扫描 `IP:PORT:user:pass`。
- [ ] 每个入库 IP 都证明可入站、可出站。
- [ ] 未通过扫描、私网、CGNAT、默认出口漂移 IP 不进入可售目录。
- [ ] OFFLINE、DEGRADED、capability mismatch、digest mismatch、manual/compliance hold 的节点不卖新库存。
- [ ] 管理端能看到不可售 reason code。

Evidence:
- 真实 Linux 节点 smoke：候选 IP 列表、扫描任务记录、成功/失败样例、入库结果和用户可售目录对比。

### T5：用户、钱包、充值、审计

Status: 部分商业骨架已有，仍需闭环验证。

Value:
- 客户可注册登录、充值入账、查看余额；管理员可审计钱包和充值流水。

Why now:
- 首单闭环必须先有正确的钱包根源事实和幂等充值。

Scope:
- 用户注册/登录、管理员登录/RBAC、钱包、不可变流水、充值订单、易支付模拟/回调、审计日志。
- Idempotency-Key 或第三方交易号门闩。

Acceptance:
- [ ] 用户可注册、登录、查看余额。
- [ ] 用户可发起充值模拟并入账。
- [ ] 同一充值回调重复投递不会重复入账。
- [ ] 钱包流水不可修改，只能追加。
- [ ] 管理员可查看用户、钱包流水、充值订单和审计日志。

Evidence:
- 单测覆盖充值回调幂等、钱包流水不可变；集成测试覆盖充值事务。

### T6：产品、价格、地区线路、库存预约

Status: 部分商业骨架已有，需和 G2 可售扫描合并。

Value:
- 客户只看到真实可售的地区、线路、协议、价格和数量；并发下单不会超卖。

Why now:
- 这是首单前的售卖目录和库存事务闸门。

Scope:
- 产品、价格、用途、国家/城市、线路、协议、时长、数量、rate policy。
- 库存状态、库存预约、预约过期释放。
- 可售查询必须同时满足节点在线、候选公网 IP 扫描通过、协议能力匹配、digest 一致、线路启用、价格有效、库存可用。

Acceptance:
- [ ] 管理员可配置产品、价格、地区、线路、库存和速率策略。
- [ ] 用户购买页只展示真实可售地区、线路和数量。
- [ ] 禁用线路、禁用库存、异常节点、能力不匹配节点不会出现在可售结果中。
- [ ] 并发库存预约不超卖。
- [ ] Redis 丢失不影响 Postgres 库存事实。

Evidence:
- 单测覆盖价格计算、可售过滤、reason code、并发预约；集成测试覆盖预约事务。

### T7：首单下单与 Runtime ACK 交付

Status: 未完成，是 G3 核心主线。

Value:
- 客户从 0 开始购买 1 个静态家宽 SOCKS5/HTTP，ACK 后拿到真实可用代理。

Why now:
- 这是 V1 第一个商业闭环；完成前不继续扩后台 CRUD。

Scope:
- 注册用户余额下单、钱包冻结、库存预约、订单创建、代理账号创建、Runtime desired state、outbox 同事务。
- Runtime ACK 后扣款、库存 SOLD、订单 ACTIVE、凭据可复制。
- Runtime ACK 前订单显示发货中，不展示代理密码、连接串或新凭据。
- NACK/超时释放 wallet hold、inventory reservation，并把订单推进失败/退款状态。
- 管理端订单详情和发货时间线。

Acceptance:
- [ ] 用户可购买一个静态家宽 SOCKS5/HTTP。
- [ ] ACK 前订单显示发货中且无凭据。
- [ ] ACK 后扣款、库存 SOLD、代理凭据可复制且可连接。
- [ ] NACK/超时释放 wallet hold 和 inventory reservation。
- [ ] 代理可实际连接并产生账号级流量。
- [ ] 重复下单请求返回同一业务结果，不重复扣款。

Evidence:
- E2E smoke 录制或日志：注册 -> 充值 -> 购买 -> ACK -> 连接代理 -> usage 增量。

### T8：生命周期：续费、停用、过期、刷新 IP/凭据、流量

Status: 未完成，是 G4 主线。

Value:
- 客户能自助管理代理，平台状态和 Runtime 状态保持一致。

Why now:
- 首单之后必须证明订单不是一次性交付，而是可运营生命周期。

Scope:
- 我的代理、代理详情、续费、停用、过期删除、刷新 IP、刷新凭据、流量展示。
- Runtime desired state 更新、ACK 后才更新客户可见凭据/IP。
- 过期扫描 Worker、流量 rollup Worker、Webhook/通知基础。

Acceptance:
- [ ] 续费后平台到期时间和 Runtime resource version 同步。
- [ ] 停用/过期后 Runtime remove resource，代理连接失败。
- [ ] 刷新 IP 必须 Runtime 确认后才展示新 IP。
- [ ] 刷新凭据必须 Runtime 确认后才展示新凭据。
- [ ] 客户可查看每个代理上下行流量。

Evidence:
- 生命周期 smoke：续费、停用、过期删除、刷新 IP、刷新凭据、usage 展示。

### T9：管理运营中心与任务控制台

Status: 未完成。

Value:
- 管理员能看到节点、库存、订单、任务、Runtime apply 和失败原因。

Why now:
- 生产运营需要发现问题、定位问题和安全重试，但不能替代主流程正确性。

Scope:
- 管理驾驶舱、实时节点面板、任务控制台、订单时间线、不可售原因、Bark 告警。
- CPU/RAM/磁盘/连接数/速度/账号数/Runtime 状态上报。
- 任务重试、取消、对账、DLQ、通知事件。

Acceptance:
- [ ] 管理员能看到节点、库存、订单、任务和 Runtime apply 状态。
- [ ] 管理员能看到每个不可售原因和任务失败原因。
- [ ] 管理员能安全重试、取消或触发对账。
- [ ] Bark 可收到节点离线、DEGRADED、任务最终失败、库存低水位告警。

Evidence:
- 管理端截图/API smoke、告警投递记录、任务失败和重试样例。

### T10：节点接入、Web SSH、退役、恢复

Status: 未完成。

Value:
- 管理员可以接入、诊断、退役和恢复节点，节点故障不变成客户侧不可解释事故。

Why now:
- 真实运营必须具备节点维护和恢复能力。

Scope:
- 节点安装脚本、ZTP enrollment、Web SSH、SSH 凭据加密、host key fingerprint、recovery token。
- 节点退役状态机、退役中不卖新库存。
- SSH 自动恢复 NodeAgent + XrayCore Runtime Bundle。
- NodeAgent 重装后根据 Postgres desired state 恢复已有订单。

Acceptance:
- [ ] 管理员可复制一键脚本或通过后台 SSH 安装节点。
- [ ] 管理员可从节点详情打开 Web SSH。
- [ ] 退役中节点不再销售新库存。
- [ ] NodeAgent 重装后能恢复已有指派订单。
- [ ] SSH 可达时平台能自动修复离线 NodeAgent。

Evidence:
- 一键安装日志、Web SSH 审计、退役 smoke、重装恢复 smoke。

### T11：Test 环境、E2E、故障演练、发布门槛

Status: 未完成。

Value:
- RayIP V1 可以在 Test 环境完整验收，具备进入生产运营的基本质量门槛。

Why now:
- 生产发布前必须证明端到端链路和故障降级，而不是只证明本地 happy path。

Scope:
- Railway/Dokploy/Test 环境：Postgres、Redis、NATS、API、user-web、admin-web。
- 至少一台真实 Linux 节点部署 NodeAgent + XrayCore Runtime Bundle。
- E2E smoke：注册、充值、购买、连接、续费、停用、刷新 IP/凭据、流量。
- 故障演练：Redis、NATS、NodeAgent 断线、Runtime NACK、Bark 失败、节点重装。
- 发布检查表、备份、回滚、Secret 管理、审计。

Acceptance:
- [ ] Test 环境完成完整 E2E。
- [ ] 并发下单不超卖。
- [ ] 发货失败不扣款。
- [ ] Runtime 未确认不展示凭据。
- [ ] NodeAgent 掉线后节点不可售。
- [ ] Redis/NATS/NodeAgent 断线有明确降级与恢复行为。
- [ ] 发布检查表全部通过。

Evidence:
- E2E 报告、故障演练记录、发布检查表。

## 6. 测试计划

### 单元测试

- 钱包流水幂等。
- 充值回调幂等。
- Idempotency-Key 下单幂等。
- 库存预约并发不超卖。
- 可售 reason code：offline、digest mismatch、unsupported capability、manual/compliance hold、no candidate public IP、scan failed、egress mismatch。
- 节点扫描 Worker 指数退避、最大重试、NATS 消息索引化。

### 集成测试

- Postgres 事务下的钱包 hold、库存 reservation、订单状态推进。
- Runtime desired state outbox 到 NATS Worker，再到 RuntimeApply ACK/NACK。
- NodeAgent 重连后继续处理未完成变更。
- NACK/超时后补偿释放 wallet hold 和 inventory reservation。

### E2E smoke

- Docker Compose 本地商业链路 smoke。
- 真实 Linux 节点 smoke：NodeAgent 扫网卡公网 IP，平台逐 IP SOCKS5/HTTP 验证。
- 首单真实代理连接 smoke。
- 停用后连接失败 smoke。

## 7. 发布前总验收

- [ ] G1 Runtime 可控可验收。
- [ ] G2 真实节点可售。
- [ ] G3 首单商业闭环。
- [ ] G4 生命周期自助。
- [ ] G5 生产运营闭环。
- [ ] 客户侧：注册、登录、充值、余额、购买、代理详情、流量、续费、停用、IP 刷新、凭据刷新、反馈。
- [ ] 管理侧：用户、钱包、账单、充值、产品价格、地区线路、库存、订单、代理账号、节点、退役、任务、通知、审计。
- [ ] 节点侧：ZTP、enrollment、Bundle 检查、候选公网 IP 扫描、增量下发、流量上报、重连恢复、重装恢复、Web SSH。
- [ ] 正确性：不可售库存不展示、发货失败不扣款、Runtime 未确认不展示凭据、续费/停用/过期同步 Runtime、并发下单不超卖。
