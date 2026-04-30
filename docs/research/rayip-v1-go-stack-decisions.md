# RayIP V1 Go 技术栈取舍研究笔记

> 日期：2026-04-29

## 1. 当前主线

RayIP V1 主线采用成熟 Go 技术栈：

- GoFiber / Fiber v3
- Ent
- Atlas / Ent schema migration
- Zap
- Viper
- Uber Fx
- gRPC / Protobuf
- Redis
- NATS JetStream
- Postgres

原因很实际：RayIP 是商业平台，重点是正确性、可维护性、开发效率和后续团队接手成本，不是极限性能。

Less is More 的含义是：

- 不拆无意义服务。
- 不造自研框架。
- 不把内部运维失败包装成产品功能。
- 不为 V1 不卖的产品建抽象。

不是有成熟库不用。

## 2. GoFiber / Gin / chi

当前建议：

- V1 主线使用 GoFiber / Fiber v3。
- Gin 不再作为默认选择。
- chi 不作为首选。

理由：

- GoFiber 的路由、binding、中间件、限流、idempotency、SSE 等能力更贴近 RayIP 管理后台和 API 开发。
- GoFiber 性能和内存占用适合 Railway / 容器环境。
- `fasthttp` 生态和对象复用需要团队纪律，但这是可控成本。
- Gin 仍然成熟可用，但不作为 RayIP V1 主线。
- chi 很薄，容易让项目自己发明太多约定。

结论：

> V1 使用 GoFiber / Fiber v3。

## 3. Ent / GORM / Bun / pgx/sqlc

当前建议：

- V1 使用 Ent 作为 schema-as-code 和应用数据访问主线。
- 开发期使用 Ent schema migration，生产发布前使用 Atlas migration directory 固化 DDL。
- 关键交易路径允许通过 Ent SQL driver/raw SQL 执行行锁、条件更新和幂等门闩。
- GORM 不作为 RayIP V1 主线。
- Bun 是 Ent 不适合表达某些查询时的备选。
- pgx/sqlc 不作为 V1 默认。

理由：

- RayIP 后续会有钱包、库存、订单、Runtime desired state、outbox 和审计日志，schema 演进成本高，Ent 的 schema-as-code 更适合长期维护。
- Ent codegen 带来的流程成本在 V1 早期可以接受，且比后期从 GORM 迁移更低。
- 钱包、库存、订单状态机用显式事务和锁，不依赖 ORM 魔法。
- Atlas/Ent migration 能把 schema 变更固化为可审查 DDL，适合生产发布门槛。
- pgx/sqlc 最显式，但会增加大量 SQL 和生成代码维护。

## 4. Atlas / Ent migration 的作用

Ent schema 是业务表结构的源头。迁移策略分两层：

- 开发期：`client.Schema.Create` 用于本地/Test 快速同步 schema。
- 生产期：发布前用 Atlas migration directory 生成、审查和执行版本化 DDL。

应用数据访问默认走 Ent Client。生产环境不允许只靠运行时自动迁移完成不可回滚的 schema 变更。

## 5. OpenAPI / oapi-codegen

V1 不默认把所有 HTTP 接口交给 oapi-codegen。

建议：

- 稳定的公开 API 用 OpenAPI 文档。
- 开发者 API 需要长期契约，可以再考虑生成部分类型。
- 管理后台早期接口用手写 DTO + Fiber Bind 更快。
- gRPC / Protobuf 仍然必须用于 NodeAgent 和 Runtime 控制契约。

## 6. Zap / Viper / Fx

Zap：

- 用于结构化日志。
- 关键字段：`request_id`、`user_id`、`admin_id`、`order_id`、`node_id`、`task_id`。

Viper：

- 启动时读取 env / config file。
- 立即转成 typed config。
- 业务代码不直接读全局 Viper。

Fx：

- 负责构造 config、logger、db、redis、nats、HTTP、gRPC、worker。
- 不承载业务逻辑。
