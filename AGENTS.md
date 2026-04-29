# RayIP 项目指引

RayIP 是一个全新的静态家宽代理销售平台项目，不是继续修补 GoSea-Light + XrayTool。

## 产品定位

RayIP V1 是面向 C 端客户销售静态家宽 IP 的商业平台。

V1 只做：

- 静态家宽 IP
- SOCKS5
- HTTP

V1 不做：

- 动态住宅代理池
- 专线销售
- VMess / VLESS / Trojan / Shadowsocks / HY2 对外销售
- 3x-ui 集成
- 节点本地 UI
- 微服务拆分

核心要求：

> 客户能看到可购买，就必须已经满足可售、可发货、可使用。

## 技术栈共识

- 后端：Go
- NodeAgent：Go
- HTTP：GoFiber / Fiber v3
- 依赖注入：Uber Fx
- ORM：GORM
- 迁移：Goose
- 配置：Viper 加载后转 typed config
- 日志：Zap
- 数据库：Postgres
- 实时状态 / 热缓存：Redis
- 持久任务队列：NATS JetStream
- 节点控制：gRPC / Protobuf
- 用户面板：React 19 + TypeScript + TanStack + shadcn/ui
- 管理面板：React 19 + TypeScript + TanStack Table + shadcn/ui + xterm.js

Less is More 的含义是避免过度拆分和无意义抽象，不是拒绝成熟工具。

## 前端设计硬约束

`apps/user-web` 用户面板必须对齐本项目的视觉和交互参考：

```text
frontend_design/
```

这些截图是 RayIP V1 用户面板的硬参考，不是随手参考。实现用户面板时必须保持：

- 左侧分组导航、顶部余额 / 充值 / 主题 / 语言 / 账户操作。
- 浅灰页面背景、白色内容面、蓝色主按钮、清晰表格、紧凑表单、低干扰状态色。
- 登录、概览、静态家宽购买、已购 IP 列表、API 密钥 / 开发者 App、代理验证工具、计费、优惠券、推广、反馈、充值等页面的信息结构。
- 购买页必须是实际可下单工具，不做营销式空页面。
- 代理列表必须强化复制、导出、续费、IP 刷新、凭据刷新等高频动作。
- 卡片圆角保持克制，默认不超过 8px；不要做大面积渐变、装饰插画、漂浮营销卡片。
- 文案少解释、多动作，客户不应看到节点、下发、任务失败等内部概念。
- 移动端可以折叠菜单，但不能牺牲购买、复制凭据、续费和充值的主流程。

`apps/admin-web` 可以采用更高信息密度的后台布局，但视觉语言要和用户面板一致：克制、清晰、表格强、筛选强、批量操作明确。管理员可以看到节点、任务、Runtime、Web SSH 等内部状态，但这些状态不能泄漏到客户面板。

## 目录规划

```text
RayIP
├─ apps/user-web        用户面板
├─ apps/admin-web       管理面板
├─ services/api         Go 控制面
├─ services/node-agent  Go 节点代理
├─ packages/proto       gRPC / Protobuf 契约
└─ docs                 需求、架构、技术栈、路线图、研究笔记
```

当前优先级是先冻结需求和契约，再进入代码实现。

## 状态归属

- Postgres 是根源状态：用户、钱包、订单、库存、代理账号、幂等结果、Runtime 期望状态、审计日志。
- Redis 是实时状态和热缓存：节点在线、NodeAgent session 路由、实时指标窗口、限流、Web SSH TTL。
- NATS JetStream 是持久任务队列：发货、续费、停用、IP/凭据刷新、节点退役、对账、SSH 自动恢复、通知。
- Go API 必须无状态运行，进程内只保存可丢失连接句柄和临时 buffer。

Redis 和 NATS 都可以持久化，但不能替代 Postgres 的根源事实。

## Runtime 原则

- Runtime 保持魔改版 XrayCore。
- NodeAgent 与魔改版 XrayCore 作为同一个 Runtime Bundle 安装、升级、回滚。
- NodeAgent 通过 gRPC Xray API / 扩展 API 控制 XrayCore。
- 所有账号、限速、连接数、流量统计都必须账号级管理。
- 日常购买、续费、停用、凭据刷新、IP 刷新、限速调整必须增量下发。
- 冷启动恢复和重装恢复可以扫描全量期望状态，但必须分页拆成增量 batch。
- 限速和连接数限制必须在 XrayCore 内实现，不以 `tc`、iptables、nftables 作为主方案。

## 开发前必须统一的契约

正式开发前必须先冻结：

- HTTP JSON 错误、分页、金额、时间、幂等头格式。
- 开发者 API 的 `AppID + AppSecret` 签名、nonce、防重放、IP 白名单、Webhook 签名和重试。
- gRPC Proto：NodeAgent 控制流、Runtime batch、账号 delta、apply result、指标、终端帧。
- 数据库关键表、唯一约束、状态枚举、事务边界。
- NATS Stream、Subject、durable consumer、ack wait、重试、DLQ、优先级。
- Redis key 前缀和 TTL 语义。
- 订单、库存、节点、任务、Runtime apply 状态机。

契约冻结按 Task 保持薄而明确，执行时参考 `docs/plans/rayip-v1-contract-freeze-checklist.md`。不要在契约未定时大规模铺代码。

## 文档规则

- 所有项目文档使用中文。
- 主文档保持少而清晰：需求、架构、技术栈、路线图。
- 研究材料放在 `docs/research/`。
- 计划材料放在 `docs/plans/`。
- 不要为每个想法新增根文档，先合并进现有主文档或研究目录。

## 编码约定

- 遵循现有目录和文档中的架构，不擅自引入新服务拆分。
- 业务正确性优先于局部性能微优化。
- 钱包、库存、订单、Runtime 状态机必须使用 Postgres 事务、行锁、唯一约束、条件更新和幂等键。
- 生产环境禁用 GORM AutoMigrate，使用 Goose 管理迁移。
- NATS 消息只带最小索引，Worker 必须回 Postgres 读取当前期望状态。
- NodeAgent 不导入 API 的用户、订单、钱包、产品包；共享契约只放在 `packages/proto`。
- Web SSH 经由浏览器 WebSocket -> API -> NodeAgent gRPC -> 本地 PTY，不引入独立 ops-gateway。

## Git 约定

这是独立 Git 仓库。不要把外层工作区的 `GoSea-Light/`、`xraytool/`、`refers/` 加入本项目。

提交前至少检查：

- `git status`
- Go 测试或构建
- 前端 typecheck/build
- 文档链接是否仍然有效

当前仍处于规划和初始化阶段；未明确要求时，不要提前实现大量业务代码。
