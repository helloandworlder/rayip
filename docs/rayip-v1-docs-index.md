# RayIP V1 文档索引

> 日期：2026-04-30

## 主文档

- [功能需求](./rayip-v1-requirements.md)
- [架构设计](./rayip-v1-architecture.md)
- [技术栈](./rayip-v1-tech-stack.md)
- [路线图](./rayip-v1-roadmap.md)

## 计划文档

- [用户故事与前端设计计划](./plans/rayip-v1-user-stories-and-frontend-plan.md)
- [开发前契约冻结清单](./plans/rayip-v1-contract-freeze-checklist.md)
- [GoSeaLight 自运营迁移方案](./plans/gosealight-self-migration-plan.md)

## 研究笔记

- [Go 技术栈取舍](./research/rayip-v1-go-stack-decisions.md)
- [sx-core 限速与 Fiber 研究](./research/rayip-v1-sx-core-fiber-research.md)
- [985Proxy 静态家宽 API 研究](./research/rayip-v1-985proxy-api-notes.md)
- [易支付与钱包结算笔记](./research/rayip-v1-yipay-billing-notes.md)
- [状态机与不变量笔记](./research/rayip-v1-state-machines-notes.md)
- [前台 / 后台 UX 笔记](./research/rayip-v1-frontend-ux-notes.md)
- [Panel 与 etcd 控制面研究](./research/rayip-v1-panel-etcd-control-plane-notes.md)
- [生产化架构研究笔记](./research/rayip-v1-production-readiness-research.md)

## 附录

- [AI 项目规划与编码交付方法论](./plans/ai-project-delivery-methodology.md)

## 当前共识

- V1 只做静态家宽 IP。
- V1 只对外销售 SOCKS5 / HTTP。
- V1 是 toC 商业平台，不是节点运维面板。
- 客户能下单，就必须已经满足可售、可发货、可使用。
- V1 必须覆盖 GoSeaLight 静态家宽销售核心闭环，但不照搬 GoSeaLight 的技术实现和 `Zone` 等过宽产品概念。
- Go First：后端 Go，NodeAgent Go。
- HTTP 框架使用 GoFiber。
- 使用 Uber Fx 做依赖注入和生命周期管理。
- 前端：用户面板 React 19 + TanStack + shadcn/ui；管理面板 React 19 + 后台框架/数据表格/xterm.js。
- 数据库：Postgres；缓存/短期协调：Redis。
- Go Backend 无状态；进程内只保存可丢失的连接句柄和临时 buffer。
- Postgres 是订单、钱包、库存预约、幂等、Runtime 期望状态的根源状态。
- Redis 保存实时状态、在线状态、session 路由和热缓存，并启用持久化，但不做根源状态。
- NATS JetStream 是持久任务队列；只有在承担可靠异步任务时使用，不做装饰性引入。
- NATS 通过 Postgres outbox 发布，必须可重放。
- NodeAgent 主动通过 gRPC 双向流连接控制面。
- 节点任务下发采用 Runtime 期望状态 + 变更序列 + NATS 最小消息，不使用胖任务 payload 作为主模型。
- 正式开发前必须冻结 HTTP JSON、开发者 API、gRPC Proto、DB 关键约束、NATS Stream、Redis Key、状态机等最小契约。
- Runtime 保持改造版 XrayCore。
- NodeAgent 通过 gRPC Xray API / 扩展 API 控制改造版 XrayCore。
- gRPC API 必须增量下发；恢复和对账也要拆成分页增量 batch。
- NodeAgent `.env` 只保留 bootstrap 必要项；Runtime 版本、能力、限速池、滥用阈值和合规策略来自自发现、能力协商和控制面动态下发。
- Runtime Bundle 是签名供应链资产；XrayCore fork 用 submodule 固定源码指针，生产节点安装签名 artifact 并校验 hash / 签名 / minimum allowed version。
- Runtime 下发采用 `version_info + nonce + ACK/NACK + last_good_generation` 语义。
- NodeAgent 新安装或重装后，必须能从 Postgres 期望状态恢复已指派订单。
- 限速和连接数限制必须落在 XrayCore 内。
- 观测只做平台内指标，不引入 Prometheus / OpenTelemetry Collector / Loki / ClickHouse。
- 支持 Bark 作为轻量管理员告警渠道，但通知事件先落 Postgres，Bark 只负责投递。
- 测试部署优先 Railway 项目 `RayIP`，环境 `Test`。
- 路线图采用 Runtime-first 的 3 个 Milestone、11 个厚 Task，以根目录 `TODO.md` 和 [路线图](./rayip-v1-roadmap.md) 为准。
- 用户面板必须对齐 `frontend_design/` 下的 RocketIP / IPIPD 风格截图，具体见 [用户故事与前端设计计划](./plans/rayip-v1-user-stories-and-frontend-plan.md)。
- 正式编码前按 Task 执行薄契约冻结，具体见 [开发前契约冻结清单](./plans/rayip-v1-contract-freeze-checklist.md)。
- 支持用户创建多个开发者 App；每个 App 独立 `AppID + AppSecret`、IP 白名单、回调地址和启停状态。
- V1 不引入 `Zone` 作为产品概念；客户只选择用途、IP 类型、国家/城市、线路、协议、时长、数量。
- 985Proxy API 是研究参考，不照抄其产品模型。
- 易支付用于在线充值；购买/续费从 RayIP 钱包余额扣款，并通过冻结/确认/释放保证商业正确性。
