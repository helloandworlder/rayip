# RayIP V1 功能需求

> 版本：产品验收 v2
> 日期：2026-04-30
> 范围：RayIP V1 静态家宽 SOCKS5/HTTP 代理销售平台

## 1. 产品定位

RayIP V1 是面向 C 端客户销售静态家宽代理的平台。

V1 只做：

- 静态家宽 IP
- SOCKS5
- HTTP

V1 不做：

- 专线产品
- 动态住宅代理池
- VMess / VLESS / Trojan / Shadowsocks / HY2 对外销售
- 3x-ui 集成
- 节点本地 UI
- 独立 ops-gateway
- 把节点、任务、Runtime、NATS、Worker 等内部概念暴露给客户

核心原则：

> 客户能看到可购买，就必须已经满足可售、可发货、可使用。

管理员可以看到运维状态，但平台不能依赖“客户下单后失败，再让管理员补救”来完成商业闭环。

## 2. V1 覆盖口径

必须覆盖：

| 能力 | RayIP V1 处理方式 |
|---|---|
| 用户注册 / 登录 | Go API + React 用户面板 |
| 钱包、余额、账单流水 | Postgres 事务和不可变流水作为根源事实 |
| 易支付充值 | 充值入账和回调幂等必须落库 |
| 产品、价格、地区、线路 | 静态家宽产品模型，不使用客户可见 `Zone` 概念 |
| 静态家宽下单 | 库存预约 + 钱包冻结 + Runtime ACK 后交付 |
| 订单列表 / 订单详情 | ACK 后才展示可用代理凭据 |
| 续费 / 停用 / 过期 | 平台状态和 Runtime 策略必须同步确认 |
| IP 刷新 | 订单生命周期不变，只替换当前代理 IP |
| 凭据刷新 | Runtime 确认后才展示新凭据 |
| 每订单上下行流量 | XrayCore 账号级统计上报并汇总 |
| 管理后台 | 产品运营、节点可售、订单生命周期和任务追踪 |
| 节点管理 / 节点退役 | 状态机驱动可售判断 |
| 发货 / 下发 / Runtime 状态 | Runtime desired state + NATS JetStream + NodeAgent gRPC 增量下发 |
| 审计日志 | 管理员敏感操作、SSH、钱包、库存、Runtime 操作必须审计 |

后置或不覆盖：

- GoSeaLight 的 `Zone` 作为客户可见产品概念。
- 非静态家宽产品和专线销售。
- 二级租户 / 子客户完整体系。
- 复杂分销体系。

## 3. 用户面板需求

用户面板参考 RocketIP/IPIPD 风格：直接、流畅、商业化，围绕购买和生命周期。

必须包含：

- 注册 / 登录
- 账户概览
- 在线充值
- 静态家宽购买
- 我的静态代理
- 代理详情
- 续费
- 停用
- IP 刷新
- 凭据刷新
- 流量用量
- 账单流水
- 开发者 App
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
8. Runtime ACK 后获得可用代理。

用户面板约束：

- 登录后第一屏是可操作业务面板，不是营销页。
- 客户只看到真实可售的静态家宽选项。
- 客户在 Runtime 确认后才能看到代理凭据。
- 客户界面不出现节点、NATS、Worker、Runtime apply、任务重试等内部概念。

## 4. 管理面板需求

管理面板是产品运营后台，不是失败补救列表。

必须包含：

- 概览
- 用户
- 钱包 / 充值 / 账单
- 订单
- 产品和价格
- 国家 / 城市 / 线路
- 家宽节点
- 候选公网 IP
- IP 库存
- 节点状态和退役
- 节点实时状态面板
- 任务控制台
- Bark / 通知渠道
- 代理生命周期操作
- 优惠券
- 邀请规则
- 工单 / 反馈
- 审计日志
- 系统设置
- 节点列表 / 详情一键 Web SSH

管理端必须展示：

- 哪些地区和线路可售。
- 哪些候选公网 IP 已通过扫描，哪些失败及失败原因。
- 哪些 IP 可售、已售、冻结、禁用、质量差。
- 哪些节点在线、异常、退役中、已退役。
- 哪些订单活跃、过期、续费中、禁用。
- 每个订单消耗的上下行流量。
- 每个产品绑定的限速和连接数策略。
- 所有家宽节点的实时速度、CPU、RAM、连接数、流量、存储和在线状态。

任务控制台必须：

- 按订单、节点、代理账号、任务类型、错误分类筛选。
- 展示从业务动作到 Runtime apply 的事件时间线。
- 支持安全重试、取消、触发对账、标记人工处理。
- 展示补偿动作结果，例如释放库存、解冻余额、恢复节点、重发通知。
- 所有操作必须审计。

## 5. 产品模型

V1 保持小模型：

- 产品：静态家宽代理
- 协议：SOCKS5 / HTTP
- IP 类型：普通 / 原生
- 业务用途：客户选择的使用场景
- 地区：国家 / 城市
- 线路：面向客户的可选线路
- 时长：计费周期
- 数量：购买 IP 数
- 速率策略：带宽上限、连接数上限、优先级
- 候选公网 IP：NodeAgent 发现但尚未必然可售的 IP
- 库存：扫描通过并可售的家宽 IP
- 订单：客户购买和生命周期记录
- 代理账户：实际下发到 XrayCore 的账号和策略
- 开发者 App：每个用户可创建多个 App，每个 App 独立 `AppID + AppSecret`

不为 V1 未销售的产品提前建复杂抽象。

## 6. 真实节点可售验收

家宽节点必须证明每个 IP 可入站、可出站。

NodeAgent 责任：

- 扫描本机网卡、路由、出口，生成 `candidate_public_ips`。
- 上报 Runtime Bundle version、XrayCore version、extension ABI、capabilities、manifest hash、binary hash、账号数、generation 水位、snapshot digest。
- 创建临时探测账号或执行控制面要求的探测资源。

平台扫描 Worker 责任：

- 为每个候选 IP 生成或读取探测凭据。
- 队列化扫描每个 `IP:PORT:user:pass`，例如 `204.42.251.2:9878:testuser1:testpass1`。
- 验证入站端口可达。
- 验证 SOCKS5/HTTP 认证成功。
- 验证出站出口 IP 与候选 IP 一致或符合线路策略。
- 记录延迟、基础吞吐和最近成功时间。
- 对失败项写入 reason code。

不可售 reason code 至少包括：

- `offline`
- `no_candidate_public_ip`
- `private_ip`
- `cgnat`
- `ingress_unreachable`
- `auth_failed`
- `unsupported_protocol`
- `egress_mismatch`
- `scan_timeout`
- `digest_mismatch`
- `unsupported_capability`
- `manual_hold`
- `compliance_hold`
- `bundle_invalid`
- `node_degraded`

进入客户可售目录必须同时满足：

- 节点 `ACTIVE` 且在线。
- node credential 有效。
- Runtime Bundle 签名、hash、channel、minimum version 校验通过。
- 能力协商 `ACCEPTED`。
- Runtime digest 正常。
- 候选公网 IP 扫描通过。
- 库存 `AVAILABLE`。
- 线路启用。
- 价格有效。
- 无 abuse、manual、compliance hold。

## 7. 商业正确性

库存必须真实：

- 只有扫描通过的健康节点 IP 才能展示为可售。
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

- 客户不能在 Runtime ACK 前看到密码、连接串或新凭据。
- 订单详情只有在代理可用后展示 IP、端口、账号、密码。
- 内部重试对客户不可见。
- 最终失败时，客户看到的是未完成购买或退款状态，而不是不可用代理。

生命周期必须可验证：

- 续费同时更新平台过期时间和 Runtime resource version。
- 停用/过期必须同步 remove Runtime resource。
- 凭据刷新必须 Runtime 确认后展示。
- IP 刷新必须保留订单生命周期，只替换当前代理 IP。
- 所有管理员操作必须审计。
- NodeAgent 新安装或重装后，必须能根据 Postgres 已指派订单恢复 Runtime 账号。

## 8. Runtime 要求

Runtime 保持 XrayCore，并做必要扩展。

必须支持：

- SOCKS5 账号管理。
- HTTP 账号管理。
- 按账号统计上下行流量。
- 按账号限速。
- 按账号连接数限制。
- Runtime digest。
- 通过 gRPC Xray API / 扩展 API 运行时下发 / 更新 / 删除账号。
- XrayCore 重启后的策略对账。
- 健康状态和版本上报。
- 节点重装后的账号恢复和策略补齐。

限速必须在改造版 XrayCore 内实现，不以 Linux `tc`、iptables、nftables、外部代理作为主要限速方案。

NodeAgent 与 XrayCore 的主通信方式必须是 gRPC Xray API / 扩展 API，且必须增量下发。配置文件只用于启动基础 Runtime，不作为日常订单下发、续费、停用、凭据刷新、限速变更的主路径。

Runtime Bundle 是供应链资产：

- XrayCore fork 独立仓库维护，RayIP 用 submodule 固定源码指针。
- 生产节点安装签名 artifact，不在节点上临时编译。
- Bundle manifest 必须包含版本、扩展 ABI、capabilities、binary hash、manifest hash 和签名信息。
- NodeAgent 必须校验签名、hash、allowed channel、minimum allowed version。
- 校验失败、版本不兼容、能力缺失、digest 不一致的节点不能销售新库存。

## 9. 状态与幂等

Go Backend 必须无状态：

- API 进程不保存业务事实。
- 任一 API 实例重启后，用户订单、节点状态、任务进度都必须可恢复。

状态归属：

- Postgres：用户、钱包、库存、订单、代理账号、Runtime desired state、幂等结果、审计日志。
- Redis：节点在线、实时指标窗口、session 路由、限流、热缓存。
- NATS JetStream：持久任务队列，必须使用 durable consumer、ack、redelivery。

需要幂等的动作：

- 充值回调。
- 购买。
- 续费。
- 停用。
- 凭据刷新。
- IP 刷新。
- Runtime 下发任务。
- NodeAgent apply ack。

要求：

- 幂等键和最终结果落 Postgres。
- 重复请求返回同一业务结果。
- 不能重复扣款、重复发货、重复释放库存。

## 10. 节点和库存状态

节点状态：

- `ACTIVE`：健康，可售。
- `DEGRADED`：异常，不卖新库存。
- `DRAINING`：退役中，不卖新库存，存量继续续费/迁移/到期。
- `RETIRED`：已退役，无活跃订单。
- `OFFLINE`：心跳丢失。
- `REMOVED`：归档删除。

库存状态：

- `CANDIDATE`：候选公网 IP，尚未扫描通过。
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

## 11. 指标和观测

V1 只做平台内观测：

- 节点心跳。
- Candidate IP 扫描状态。
- NodeAgent / XrayCore 版本。
- XrayCore 进程状态。
- CPU / 内存 / 磁盘摘要。
- 活跃代理数。
- 活跃连接数。
- 每订单上下行流量。
- 当前速度汇总。
- 一段时间内的速度、CPU、RAM、连接数历史。
- 最近下发结果。
- 节点任务历史。
- 节点健康评分和不可售原因。

不引入 Prometheus、OpenTelemetry Collector、Loki、ClickHouse。

## 12. ZTP、Web SSH 和恢复

ZTP 的目标是让节点接入可复制、可审计、可轮换、可恢复。

安装器负责：

- 安装 NodeAgent systemd 服务。
- 安装或更新签名 Runtime Bundle。
- 校验 manifest、binary hash、签名、allowed channel 和 minimum allowed version。
- 完成节点 enrollment。
- 用 bootstrap token 换取短期 node credential / session identity。
- 启动 NodeAgent。
- 回传首次心跳。
- 上报 Runtime capabilities、digest 和 candidate public IP。
- 拉取平台 desired state，恢复该节点已指派的活跃订单。

Web SSH：

- 浏览器通过 WebSocket 连接 API。
- API 校验管理员权限。
- API 通过现有 NodeAgent gRPC 通道打开终端会话。
- NodeAgent 在节点本地启动 PTY。
- 会话有审计、空闲超时、最长时长限制。

自动恢复：

- SSH 凭据是可选能力，没有凭据时只支持手动一键脚本恢复。
- SSH 凭据必须加密保存，并记录 host key fingerprint。
- NodeAgent 离线后，平台按冷却时间和并发限制自动尝试 SSH 探测。
- SSH 可达时，平台执行幂等安装/修复脚本。
- 自动恢复不能跳过节点身份校验，不能把订单接管到错误节点。
- 自动恢复全过程必须审计，失败后触发告警。

## 13. V1 成功标准

- 客户可以注册、充值、购买静态家宽 SOCKS5/HTTP，并在 ACK 后立即使用。
- 平台不会销售离线、退役中、已退役、质量差、扫描失败或 digest 不一致的库存。
- 每个可售 IP 都有逐 IP 入站和出站验证证据。
- 易支付回调不会重复入账。
- 并发下单不会超卖。
- 发货失败不扣款。
- Runtime 未确认不展示凭据。
- 续费、停用、过期、凭据刷新、IP 刷新都有 Runtime 确认。
- 每订单流量对用户和管理员可见。
- XrayCore 内部限速和连接数限制生效。
- NodeAgent 重装后可以恢复已有指派订单。
- 管理员可以退役节点且不再销售新库存。
- 管理员可以看到不可售原因、任务失败原因并安全触发恢复。
- 管理员可以从节点详情打开 Web SSH。
- Test 环境可完成端到端冒烟和故障演练。
