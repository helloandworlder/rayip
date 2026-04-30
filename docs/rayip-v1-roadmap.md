# RayIP V1 产品路线图

> 版本：产品路线图 v3
> 日期：2026-04-30
> 原则：可售真实性优先，首单闭环优先，生命周期其次，运营生产化最后。

详细执行清单以项目根目录 [TODO.md](../TODO.md) 为准。本文档只解释产品成熟度、路线图顺序和 Milestone 退出条件。

## 1. 路线图结论

RayIP V1 不按“后端、前端、数据库、部署”分层推进，也不继续沿旧的 Runtime-first 技术拆解无限扩展。新的主线是：

1. 先证明 Runtime 可控可验收。
2. 再证明真实家宽节点可售。
3. 然后完成首单商业闭环。
4. 接着补齐客户生命周期自助。
5. 最后做生产运营闭环和 Test 环境发布门槛。

## 2. 五个产品闸门

| Gate | 名称 | 产品含义 |
|---|---|---|
| G1 | Runtime 可控可验收 | NodeAgent + XrayCore 能真实创建 SOCKS5/HTTP 账号，限速、连接数、usage、digest 都可复现实测 |
| G2 | 真实节点可售 | NodeAgent 发现候选公网 IP，平台逐 IP 扫描 `IP:PORT:user:pass`，通过后才进入库存 |
| G3 | 首单商业闭环 | 注册、充值、余额下单、钱包冻结、库存预约、Runtime 发货、ACK 后扣款交付 |
| G4 | 生命周期自助 | 我的代理、续费、停用、过期删除、刷新 IP/凭据、流量展示全部以 Runtime 确认为准 |
| G5 | 生产运营闭环 | 管理驾驶舱、任务控制台、告警、Web SSH、退役、恢复、Test 环境 E2E 和故障演练完成 |

客户看到“可买”时，必须已经满足节点在线、候选公网 IP 可用、协议能力匹配、Runtime digest 一致、库存可用、线路启用和价格有效。

## 3. 三个 Milestone

| Milestone | 目标 | Gate | Task |
|---|---|---|---|
| M1：可控可售基础 | Runtime 能控，真实节点能被证明可售 | G1-G2 | T1-T4 |
| M2：首单与生命周期 | 客户能完成真实购买和主要自助操作 | G3-G4 | T5-T8 |
| M3：生产运营 | 管理员能运营、恢复、发布和演练 | G5 | T9-T11 |

## 4. M1：可控可售基础

目标：在进入商业交易前，证明数据面和可售判断成立。

### T1：工程基线与真实 NodeAgent 在线

退出：

- API、NodeAgent、前后台、基础设施可本地启动。
- 管理端能看到真实 NodeAgent 在线/离线。
- API 重启后 NodeAgent 可自动重连。

### T2：Runtime P0

退出：

- 管理员能创建 SOCKS5/HTTP 测试代理，并真实连接。
- 限速、连接数、usage、digest 有可复现实测证据。
- 重复 apply 同一 generation 不产生副作用。
- Bundle manifest、hash、签名、能力协商失败时节点不可售。

### T3：Runtime desired-state + NATS Worker

退出：

- 创建测试代理走 PG -> outbox -> NATS -> Worker -> NodeAgent -> XrayCore。
- NATS 消息只带索引，Worker 回 Postgres 读取事实。
- 重复消息、Worker 重启、NodeAgent 断线重连都不破坏 Runtime。
- NACK 不推进 accepted generation，管理端能看到 last good generation 和错误原因。

### T4：家宽节点可售发现

退出：

- NodeAgent 上报 `candidate_public_ips`。
- 平台队列逐个扫描 `IP:PORT:user:pass`，验证入站可达、出站出口一致、延迟和基础吞吐。
- 未通过扫描、私网、CGNAT、默认出口漂移 IP 不进入可售目录。
- 管理端能看到候选 IP、扫描状态、延迟和不可售 reason code。

## 5. M2：首单与生命周期

目标：当前主线是首单闭环，不是继续扩后台 CRUD。M2 必须从一个客户真实购买成功倒推钱包、库存、订单和 Runtime ACK。

### T5：用户、钱包、充值、审计

退出：

- 用户可注册、登录、充值并看到余额变化。
- 同一充值回调重复投递不会重复入账。
- 钱包流水不可变，只能追加。
- 管理员可审计用户、钱包、充值订单和敏感操作。

### T6：产品、价格、地区线路、库存预约

退出：

- 用户只看到真实可售选项。
- 可售结果必须依赖 G2 扫描通过的库存。
- 并发预约不超卖。
- Redis 丢失不影响 Postgres 库存事实。

### T7：首单下单与 Runtime ACK 交付

退出：

- 用户从 0 开始完成注册、充值、购买 1 个静态家宽 SOCKS5/HTTP。
- ACK 前订单显示发货中且无凭据。
- ACK 后扣款、库存 SOLD、代理凭据可复制且可连接。
- NACK/超时释放 wallet hold 和 inventory reservation。
- 重复下单请求返回同一业务结果，不重复扣款。

### T8：生命周期自助

退出：

- 续费后平台到期时间和 Runtime resource version 同步。
- 停用/过期后 Runtime remove resource，代理连接失败。
- 刷新 IP/凭据必须 Runtime 确认后才展示。
- 客户可查看每个代理上下行流量。

## 6. M3：生产运营

目标：M3 只在首单真实 smoke 后启动。生产运营能力必须服务真实售卖链路，而不是补救一个不可靠的主流程。

### T9：管理运营中心与任务控制台

退出：

- 管理员能看到节点、库存、订单、任务和 Runtime apply 状态。
- 管理员能看到不可售原因和任务失败原因。
- 管理员能安全重试、取消或触发对账。
- Bark 可收到节点离线、DEGRADED、任务最终失败和库存低水位告警。

### T10：节点接入、Web SSH、退役、恢复

退出：

- 管理员可复制一键脚本或通过后台 SSH 安装节点。
- 管理员可从节点详情打开 Web SSH。
- 退役中节点不再销售新库存。
- NodeAgent 重装后能恢复已有指派订单。
- SSH 可达时平台能自动修复离线 NodeAgent。

### T11：Test 环境、E2E、故障演练、发布门槛

退出：

- Test 环境完成注册、充值、购买、发货、连接代理、续费、停用、刷新 IP、刷新凭据。
- 并发下单不超卖。
- 发货失败不扣款。
- Runtime 未确认不展示凭据。
- NodeAgent 掉线后节点不可售。
- Redis、NATS、NodeAgent、Runtime、Bark 故障都有明确降级和恢复行为。

## 7. V1 发布门槛

客户侧必须通过：

- 注册、登录、充值、余额、购买、代理详情、复制凭据、流量、续费、停用、IP 刷新、凭据刷新、反馈。

管理侧必须通过：

- 用户、钱包、账单、充值、产品价格、地区线路、库存、订单、代理账号、节点、退役、实时节点面板、任务、通知、审计、Web SSH。

节点侧必须通过：

- NodeAgent enrollment、候选公网 IP 扫描、Runtime Bundle 检查、增量下发、XrayCore 内限速、连接数限制、流量上报、重连恢复、重装恢复。

正确性必须通过：

- 不可售库存不展示。
- ACK 前不展示凭据。
- 发货失败不扣款。
- 并发下单不超卖。
- 续费、停用、过期、刷新 IP、刷新凭据必须 Runtime 确认。
