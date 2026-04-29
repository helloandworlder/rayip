# RayIP V1 sx-core 限速与 Fiber 研究笔记

> 日期：2026-04-29  
> 来源：`/Users/yuxi/SynexIM/Next-Proxy/sx-core`、`/Users/yuxi/SynexIM-Project/Xray-Tool/refers/fiber`

## 1. 为什么研究

RayIP V1 有两个关键技术点：

- 限速必须在魔改 XrayCore 内完成。
- HTTP 框架要在开发效率和维护成本之间取平衡。

## 2. sx-core 限速发现

`sx-core` 已经有一个实用的 XrayCore 侧限速实现：

- 按用户 email 做限速 key。
- 区分上行和下行 token bucket。
- 支持 gRPC 动态 set/get/list/remove。
- dispatcher 包装读写链路。
- 有当前速度统计。
- 对限速用户绕开 raw splice，保证统计和限速可见。

这个实现适合作为 RayIP V1 账号级限速的起点。

## 3. 仍需补齐

RayIP 还需要：

- 按账号连接数限制。
- 节点总容量保护。
- 保留 1% 带宽余量，避免家宽线路打满后延迟暴涨。
- 多账号公平限速。
- 优先级权重。
- 延迟/容量反馈。
- NodeAgent 重启和 XrayCore 重启后的策略对账。

建议分两期：

1. 第一阶段：账号级上下行限速 + 连接数限制。
2. 第二阶段：节点级公平调度和 1% 余量保护。

## 4. Runtime 策略模型

账号策略：

```text
rate_policy
  upload_bps
  download_bps
  max_connections
  priority_class
  weight
```

节点策略：

```text
node_capacity_policy
  measured_upload_bps
  measured_download_bps
  reserve_ratio = 0.01
  latency_target_ms
  adaptive_enabled
```

最终限速：

```text
effective_account_limit = min(product_limit, node_fair_share)
```

## 5. Fiber 研究结论

Fiber v3 的优点：

- 路由和 middleware 体验完整。
- binding、validator、limiter、idempotency、SSE 等能力丰富。
- 性能和内存占用有优势。

风险：

- 基于 `fasthttp`，不是原生 `net/http`。
- `fiber.Ctx` 值复用，需要注意不能跨 handler 保存引用。
- 如果使用大量 `net/http` 生态包，会有适配成本。

结论：

- V1 主线使用 GoFiber / Fiber v3。
- 不默认使用 oapi-codegen 生成所有 HTTP 代码。
- 团队需要遵守 `fasthttp` / `fiber.Ctx` 对象复用规则，不跨 handler 保存未复制的上下文引用。
