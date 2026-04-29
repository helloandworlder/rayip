# RayIP V1 功能需求

> 版本：草案 v1  
> 日期：2026-04-29  
> 范围：RocketIP / RayIP V1 静态家宽代理销售平台

## 1. 产品定位

RayIP V1 是一个面向 C 端客户销售静态家宽代理的平台。

V1 只做：

- 静态家宽 IP
- SOCKS5
- HTTP

V1 不做：

- 动态住宅代理池
- VMess / VLESS / Trojan / Shadowsocks / HY2 对外销售
- 3x-ui 集成
- 节点本地 UI
- 微服务拆分
- 独立 ops-gateway
- 把节点、任务、下发失败等内部概念暴露给客户

核心原则：

> 客户能看到可购买，就必须已经满足可售、可发货、可使用。

管理员可以看到运维状态，但平台不能依赖“客户下单后失败，再让管理员补救”来完成商业闭环。

## 1.1 GoSeaLight 核心功能覆盖口径

RayIP V1 不是继续优化 GoSeaLight，而是在更简单的技术栈下覆盖 GoSeaLight 对静态家宽销售必需的能力。

必须覆盖：

| GoSeaLight 能力 | RayIP V1 处理方式 |
|---|---|
| 用户注册 / 登录 / 账户资料 | 保留，使用 Go API + React 用户面板重做 |
| 钱包、余额、账单流水 | 保留，Postgres 事务和不可变流水作为根源事实 |
| 易支付充值 | 保留，充值入账和回调幂等必须落库 |
| 产品、价格、地区、线路 | 保留，但不使用 `Zone` 产品概念 |
| 静态家宽下单 | 保留，库存预约 + 钱包冻结 + Runtime 确认后交付 |
| 订单列表 / 订单详情 | 保留，客户只能看到已确认可用的代理凭据 |
| 续费 / 停用 / 过期 | 保留，平台状态和 Runtime 策略必须同步确认 |
| IP 刷新 | 保留，订单生命周期不变，只替换当前代理 IP |
| 凭据修改 / 刷新 | 保留，Runtime 确认后才展示新凭据 |
| 每订单上下行流量 | 保留，由 XrayCore 账号级统计上报并汇总 |
| 优惠券 / 邀请返利 | 保留 V1 基础能力，复杂分销以后再扩展 |
| 工单 / 反馈 | 保留轻量入口，不做复杂客服系统 |
| 管理后台 | 保留并重做，重点是产品运营、节点可售、订单生命周期和任务追踪 |
| 节点管理 / 节点退役 | 保留并加强，状态机驱动可售判断 |
| 发货 / 下发 / Runtime 状态 | 用 Runtime 期望状态 + NATS JetStream + NodeAgent gRPC 增量下发替代 |
| XrayTool 节点控制 | 用无 UI NodeAgent + 魔改 XrayCore Runtime Bundle 替代 |
| 审计日志 | 保留，管理员敏感操作、SSH、钱包、库存、Runtime 操作必须审计 |
| 备份 / 迁移类运维能力 | V1 保留数据库迁移和生产备份要求，不在用户产品面暴露 |

明确不覆盖或后置：

- GoSeaLight 的 `Zone` 作为客户可见产品概念。
- 非静态家宽产品和专线销售。
- 对外销售 VMess / VLESS / Trojan / Shadowsocks / HY2。
- 节点本地 UI。
- 二级租户 / 子客户完整体系；V1 先用开发者 App、邀请返利、API 权限覆盖轻量自动化诉求。

覆盖验收标准：

- GoSeaLight 当前静态家宽客户能完成的“注册、充值、下单、查看代理、续费、刷新 IP、修改凭据、看流量、提交反馈”在 RayIP V1 必须可完成。
- GoSeaLight 管理员当前依赖的“用户、钱包、订单、产品价格、库存、节点、生命周期、通知、审计”在 RayIP V1 必须有对应管理能力。
- GoSeaLight 中容易出问题的“节点不可用仍可售、下单后发货失败、Runtime 状态不一致、续费停用不同步”在 RayIP V1 必须通过状态机和契约前置避免。

## 2. 用户面板

用户面板参考 RocketIP/IPIPD 风格：直接、流畅、商业化，围绕购买和生命周期。

必须包含：

- 注册 / 登录
- 账户概览
- 在线充值：接入易支付
- 静态家宽购买
- 我的静态代理
- 代理详情
- 续费
- IP 刷新
- 凭据修改 / 刷新
- 流量用量
- 账单流水
- 开发者 App：每个 App 独立 `AppID + AppSecret`
- IP 白名单 / 回调地址
- 优惠券
- 邀请返利
- 工单 / 反馈

购买流程必须短：

1. 选择业务用途。
2. 选择 IP 类型：普通 / 原生。
3. 选择国家 / 城市 / 线路。
4. 选择协议：SOCKS5 或 HTTP。
5. 选择时长和数量。
6. 查看最终价格。
7. 使用余额支付。
8. 获得可用代理。

V1 不引入 `Zone` 作为产品概念。985Proxy API 里的 `zone` 只作为研究材料，不进入 RayIP 客户购买模型。

### 2.1 用户故事和前端设计约束

用户面板必须以 `frontend_design/` 下的 RocketIP / IPIPD 风格截图作为硬参考。具体用户故事、路由和页面验收见 [用户故事与前端设计计划](./plans/rayip-v1-user-stories-and-frontend-plan.md)。

核心用户故事：

- 新客户注册后能立即看到余额和充值入口。
- 客户只看到真实可售的静态家宽选项，并能用余额购买。
- 客户在 Runtime 确认后才能看到代理凭据。
- 客户能在“我的静态代理”里复制、续费、停用、刷新 IP、刷新凭据、导出。
- 客户能查看每个代理的上下行流量。
- 客户能创建多个开发者 App，每个 App 独立 `AppID + AppSecret`、IP 白名单和回调地址。
- 客户能使用代理验证工具、优惠券、推广账户和反馈入口。

用户面板实现要求：

- 登录后第一屏是可操作的业务面板，不是营销页。
- 左侧分组导航、顶部余额 / 充值 / 主题 / 语言 / 账户操作必须保留。
- 页面风格保持浅灰背景、白色内容面、蓝色主按钮、紧凑表单和清晰表格。
- 客户界面不出现节点、NATS、Worker、Runtime apply、任务重试等内部概念。

## 3. 管理面板

管理面板是产品运营后台，不是失败补救列表。

必须包含：

- 概览
- 用户
- 钱包 / 充值 / 账单
- 订单
- 产品和价格
- 国家 / 城市 / 线路
- 家宽节点
- IP 库存
- 节点状态和退役
- 节点实时状态面板
- Bark / 通知渠道
- 代理生命周期操作
- 优惠券
- 邀请规则
- 工单 / 反馈
- 审计日志
- 系统设置
- 节点列表 / 详情一键 Web SSH

管理端关注商业状态：

- 哪些地区和线路可售
- 哪些 IP 可售、已售、冻结、禁用、质量差
- 哪些节点在线、异常、退役中、已退役
- 哪些订单活跃、过期、续费中、禁用
- 每个订单消耗的上下行流量
- 每个产品绑定的限速和连接数策略
- 所有家宽节点的实时速度、CPU、RAM、连接数、流量、存储和在线状态

管理端还必须有一个面向运维闭环的任务控制台，但它不是给客户兜底的失败列表：

- 按订单、节点、代理账号、任务类型、错误分类筛选。
- 展示从业务动作到 Runtime apply 的事件时间线。
- 支持安全重试、取消、触发对账、标记人工处理。
- 展示补偿动作结果，例如释放库存、解冻余额、恢复节点、重发通知。
- 所有操作必须审计。

任务控制台的目标是让管理员快速定位和修复异常，同时产品主流程仍然要求“可见即可买，买到即可用”。

## 4. 产品模型

V1 保持小模型：

- 产品：静态家宽代理
- 协议：SOCKS5 / HTTP
- IP 类型：普通 / 原生
- 业务用途：客户选择的使用场景
- 地区：国家 / 城市
- 线路：面向客户的可选线路，不等同于 985Proxy 的 `zone`
- 时长：计费周期
- 数量：购买 IP 数
- 速率策略：带宽上限、连接数上限、优先级
- 库存：可售家宽 IP
- 订单：客户购买和生命周期记录
- 代理账户：实际下发到 XrayCore 的账号和策略
- 开发者 App：允许一个用户创建多个 App，每个 App 独立 `AppID + AppSecret`、IP 白名单、回调地址、权限和启停状态

不为 V1 未销售的产品提前建复杂抽象。

## 4.1 开发者 App

RayIP V1 不使用单个全局 API 密钥。

用户可以创建多个开发者 App：

- 每个 App 有独立 `AppID`。
- 每个 App 有独立 `AppSecret`。
- AppSecret 只在创建或重置时完整展示。
- 每个 App 可配置 IP 白名单。
- 每个 App 可配置回调地址。
- 每个 App 可启用/停用。
- 每个 App 有独立最后调用时间和调用统计。

V1 的开发者 API 先保持小范围：

- 查询我的代理列表。
- 查询代理详情。
- 查询流量用量。
- 触发续费。
- 触发凭据刷新。
- 触发 IP 刷新。

开发者 API 必须先统一协议：

- 使用 `AppID + AppSecret`。
- 请求签名包含 HTTP 方法、路径、时间戳、nonce 和 body hash。
- 时间戳有重放窗口，nonce 落 Redis 热缓存和 Postgres 幂等记录。
- 支持 App 级 IP 白名单、权限范围和启停状态。
- Webhook 回调必须签名、可重试、可查看投递状态。

## 5. 商业正确性

库存必须真实：

- 只有健康节点上的可用 IP 才能展示为可售。
- 异常、离线、退役中、已退役、质量差的节点不能卖新库存。
- 库存冻结必须有超时释放。
- 并发下单不能超卖。
- 外部 API 或第三方资料只能作为参考，不能绕过 RayIP 自己的可售判断。

支付必须闭环：

- 易支付只负责充值入账。
- 购买和续费从 RayIP 钱包余额支付。
- 价格计算顺序固定为：基础价格 -> 用户/渠道价格覆盖 -> 数量/时长折扣 -> 优惠券 -> 钱包冻结 -> Runtime 确认后入账。
- 下单先冻结余额，不立即确认收入。
- 代理确认可用后才确认扣款。
- 发货失败必须释放余额和库存。
- 易支付回调必须防重复入账。
- 幂等结果必须落 Postgres，Redis 只能缓存。

交付必须原子：

- 客户不能在代理未确认前看到凭据。
- 订单详情只有在代理可用后展示 IP、端口、账号、密码。
- 内部重试对客户不可见。
- 最终失败时，客户看到的是未完成购买或退款状态，而不是不可用代理。

生命周期必须可验证：

- 续费同时更新平台过期时间和 Runtime 过期策略。
- 停用/过期必须同步到 Runtime。
- 凭据修改必须 Runtime 确认后展示。
- IP 刷新必须保留订单生命周期，只替换当前代理 IP。
- 所有管理员操作必须审计。
- NodeAgent 新安装或重装后，必须能根据平台已指派订单恢复 Runtime 账号。
- NodeAgent 掉线后，如果平台配置的 SSH 仍可连，平台必须能自动尝试修复或重装 NodeAgent。

## 6. 节点和库存状态

节点状态：

- `ACTIVE`：健康，可售。
- `DEGRADED`：异常，不卖新库存。
- `DRAINING`：退役中，不卖新库存，存量继续续费/迁移/到期。
- `RETIRED`：已退役，无活跃订单。
- `OFFLINE`：心跳丢失。
- `REMOVED`：归档删除。

库存状态：

- `AVAILABLE`：可售。
- `HELD`：下单中冻结。
- `SOLD`：已售。
- `DISABLED`：暂不可售。
- `BAD`：质量差或检测失败。
- `RETIRED`：随节点退役。

订单状态：

- `PENDING_PAYMENT`
- `PAYMENT_HELD`
- `PROVISIONING`
- `ACTIVE`
- `RENEWING`
- `EXPIRED`
- `DISABLED`
- `REFUNDED`
- `FAILED`

任务状态：

- `QUEUED`
- `RUNNING`
- `SUCCEEDED`
- `RETRYING`
- `FAILED`
- `CANCELLED`

## 7. Runtime 要求

Runtime 保持 XrayCore，并做必要魔改。

NodeAgent 与魔改版 XrayCore 深度绑定，作为同一个 Runtime Bundle 管理：

- 一起安装。
- 一起升级。
- 一起回滚。
- 一起做版本兼容校验。
- NodeAgent 通过 manifest 和 XrayCore 扩展 API 自发现 Runtime 版本、扩展 ABI、hash、能力和 digest。
- API 通过能力协商决定节点是否可售。

平台不能出现 NodeAgent 已升级但 XrayCore 能力不兼容、或者 XrayCore 已升级但 NodeAgent 不知道新能力的状态。

必须支持：

- SOCKS5 账号管理
- HTTP 账号管理
- 按账号统计上下行流量
- 按账号限速
- 基于优先级、固定限速和短期流量消耗的智能公平限速
- 按账号连接数限制
- 滥用检测、账号禁用事件、report-only / disable-and-report 动作
- 通过 gRPC Xray API / 扩展 API 运行时下发 / 更新 / 删除账号
- XrayCore 重启后的策略对账
- 健康状态和版本上报
- 节点重装后的账号恢复和策略补齐

限速必须在改造版 XrayCore 内实现，不以 Linux `tc`、iptables、nftables、外部代理作为主要限速方案。

NodeAgent 与 XrayCore 的主通信方式必须是 gRPC Xray API / 扩展 API，且必须增量下发。配置文件只用于启动基础 Runtime，不作为日常订单下发、续费、停用、凭据刷新、限速变更的主路径。

Runtime Bundle 是供应链资产：

- XrayCore fork 独立仓库维护，RayIP 用 submodule 固定源码指针。
- 生产节点安装签名 artifact，不在节点上临时编译。
- Bundle manifest 必须包含版本、扩展 ABI、capabilities、binary hash、manifest hash 和签名信息。
- NodeAgent 必须校验签名、hash、allowed channel、minimum allowed version。
- 校验失败、版本不兼容、能力缺失、digest 不一致的节点不能销售新库存。

## 7.1 幂等要求

V1 必须把幂等作为商业正确性能力，而不是前端防重复点击。

需要幂等的动作：

- 充值回调。
- 购买。
- 续费。
- 停用。
- 凭据修改。
- IP 刷新。
- Runtime 下发任务。
- NodeAgent apply ack。

要求：

- 幂等键和最终结果落 Postgres。
- Redis 只能作为短期缓存。
- 重复请求返回同一业务结果。
- 不能重复扣款、重复发货、重复释放库存。

## 7.2 状态分层

Go Backend 必须无状态：

- API 进程不保存业务事实。
- API 进程内只允许保存当前连接、临时 buffer、Fx 生命周期对象。
- 任一 API 实例重启后，用户订单、节点状态、任务进度都必须可恢复。

状态归属：

- Postgres：用户、钱包、库存、订单、代理账号、Runtime 期望状态、幂等结果、审计日志。
- Redis：节点在线、实时指标窗口、session 路由、限流、热缓存，并开启 AOF/RDB 持久化。
- NATS JetStream：持久任务队列，必须使用 file storage、durable consumer、ack、redelivery。

Redis 和 NATS 都要持久化，但不能替代 Postgres 的根源状态。

## 8. 指标和观测

V1 只做平台内观测：

- 节点心跳
- NodeAgent / XrayCore 版本
- XrayCore 进程状态
- CPU / 内存 / 磁盘摘要
- 活跃代理数
- 活跃连接数
- 每订单上下行流量
- 当前速度汇总
- 一段时间内的速度、CPU、RAM、连接数历史
- 最近下发结果
- 节点任务历史
- 节点健康评分和不可售原因

不引入 Prometheus、OpenTelemetry Collector、Loki、ClickHouse。

## 8.0 节点健康

节点健康决定是否可售。

健康输入：

- 心跳是否正常。
- 节点身份和 node credential 是否有效。
- Runtime Bundle 签名、hash、channel、minimum version 是否通过。
- Runtime Bundle 版本、扩展 ABI 和能力是否兼容。
- XrayCore 是否运行。
- Runtime digest 是否和 Postgres 期望状态一致。
- 节点是否存在 abuse hold、manual hold 或合规限制。
- CPU / RAM / 存储是否超过阈值。
- 当前连接数是否超过策略。
- 最近任务是否连续失败。
- 最近对账是否成功。
- IP 质量检测是否通过。

只有身份有效、供应链校验通过、能力协商 `ACCEPTED`、digest 正常、合规/滥用状态 clean 且状态为 `ACTIVE` 的节点可以销售新库存。

## 8.1 大规模节点通信

V1 需要按数千到上万节点设计，不以单机小规模脚本作为前提。

节点在线判断：

- NodeAgent 通过 gRPC 长连接主动连接 API。
- NodeAgent 定期发送轻量 Lease，不把完整指标塞进每次心跳。
- Lease 超时后节点必须立即停止新销售。
- Redis 保存在线热状态，Postgres 保存状态变化和最后在线时间。

任务下发：

- Postgres 保存 Runtime 期望状态和变更序列。
- NATS JetStream 保存持久任务队列。
- NATS 消息只带最小索引，Worker 回 Postgres 读取当前期望状态。
- Backend 到 NodeAgent 只下发增量 Runtime batch。
- NodeAgent 到 XrayCore 只通过 gRPC Xray API / 扩展 API 增量 apply。
- 任务重复投递时不得重复改 Runtime、重复扣款、重复释放库存。
- 单节点任务需要按账号冲突关系串行或批量合并，不能无序并发写同一账号。
- 不使用胖 `node_tasks` 表承载每个下发帧的大 payload。

单节点 10k+ 订单：

- 日常变更必须走账号级 delta。
- 不能每次改一个账号就重写整份 Xray 配置。
- 恢复和对账允许扫描全量期望状态，但必须分页拆成增量 batch 执行，支持断点续传、分桶 hash。
- NodeAgent 和 XrayCore 必须能承受 10k+ SOCKS5/HTTP 账号的增量管理。

## 8.2 管理员实时节点面板

V1 需要一个管理员专用的实时节点面板，参考现有节点状态图的体验。

面板能力：

- 按节点组/线路分组展示。
- 展示节点在线、异常、退役中等状态。
- 展示 v4/v6 地区、出口 IP、实时上下行速度、开机时长、累计流量。
- 展示 CPU、RAM、存储、连接数。
- 展示最近启动时间、最近在线时间、最后同步时间。
- 支持实时模式和历史模式。
- 支持按 5 分钟、1 小时、24 小时、7 天查看趋势。
- 支持快捷操作：详情、Web SSH、退役、禁止售卖、重启 Runtime、触发对账。

实现原则：

- 浏览器通过 WebSocket 订阅 API 的节点状态流。
- API 聚合 NodeAgent 上报，不让浏览器直接连节点。
- 高频样本先进入 Redis 热数据窗口，低频汇总落 Postgres。
- 面板是管理员工具，不向客户暴露。

## 9. ZTP 和安装

ZTP 的目标是让节点接入可复制、可审计、可轮换、可恢复。它不是把所有 Runtime 信息写进安装脚本。

- 管理员创建节点，填 IP、SSH 端口、root 用户、密码或 SSH Key，点击安装。
- 管理员复制一键安装脚本，在节点手动执行。
- 安装脚本只携带 bootstrap 所需信息：API 地址、node code、一次性或可轮换 enrollment token、Runtime Bundle channel。
- Runtime 版本、能力、限速池、滥用阈值和合规策略不能写进 `.env`，必须来自 Bundle manifest、XrayCore 扩展 API 自发现和控制面策略下发。

安装器负责：

- 安装 NodeAgent systemd 服务。
- 安装或更新签名 Runtime Bundle。
- 校验 manifest、binary hash、签名、allowed channel 和 minimum allowed version。
- 完成节点 enrollment。
- 用 bootstrap token 换取短期 node credential / session identity。
- 启动 NodeAgent。
- 回传首次心跳。
- 上报 Runtime Bundle manifest、XrayCore 扩展 API capabilities、extension ABI、binary hash、runtime digest。
- 拉取平台期望状态，恢复该节点已指派的活跃订单。

节点无 UI。

重装恢复要求：

- 使用稳定 node_id 和 enrollment/claim token 识别节点。
- API 从 Postgres 查询该节点应存在的代理账号。
- NodeAgent 对比本地 Runtime，补齐缺失账号，更新策略，删除不应存在账号。
- 恢复过程幂等，可重复执行。
- 身份无法确认时不能自动接管订单，必须管理员重新绑定。

SSH 自动恢复要求：

- 管理员创建节点时可以保存 SSH Key 或密码，推荐 SSH Key。
- SSH 凭据是可选能力；没有凭据时只支持手动一键脚本恢复。
- SSH 凭据必须加密保存，并记录 host key fingerprint。
- NodeAgent 离线后，平台按冷却时间和并发限制自动尝试 SSH 探测。
- SSH 可达时，平台执行幂等安装/修复脚本，恢复 NodeAgent + XrayCore Runtime Bundle。
- 自动恢复不能跳过节点身份校验，不能把订单接管到错误节点。
- 自动恢复全过程必须审计，失败后触发 Bark 告警。

## 9.1 Bark / 通知

V1 可以接入 Bark 作为轻量管理员告警。

通知事件：

- 节点离线。
- 节点异常或进入 `DEGRADED`。
- Runtime Bundle 不兼容。
- 发货/续费/停用最终失败。
- 易支付回调异常。
- 库存低水位。
- 节点退役卡住或完成。

通知事件必须先落 Postgres，Bark 只是投递渠道。Bark 投递失败不能影响购买、续费、发货等主流程。

## 10. Web SSH

管理员必须能从节点列表/详情打开在线 SSH Terminal。

V1 设计：

- 浏览器通过 WebSocket 连接 API。
- API 校验管理员权限。
- API 通过现有 NodeAgent gRPC 通道打开终端会话。
- NodeAgent 在节点本地启动 PTY。
- 会话有审计、空闲超时、最长时长限制。

不单独引入 ops-gateway。

## 11. V1 成功标准

- 客户可以注册、充值、购买静态家宽 SOCKS5/HTTP，并立即使用。
- 易支付回调不会重复入账。
- 平台不会销售离线、退役中、已退役、质量差的库存。
- 并发下单不会超卖。
- 发货、续费、停用、凭据修改、IP 刷新都有 Runtime 确认。
- 每订单流量对用户和管理员可见。
- XrayCore 内部限速和连接数限制生效。
- NodeAgent 重装后可以恢复已有指派订单。
- 管理员可以退役节点且不再销售新库存。
- 管理员可以收到 Bark 节点/任务告警。
- 管理员可以从节点详情打开 Web SSH。
- Railway `RayIP` / `Test` 环境可完成端到端冒烟。
