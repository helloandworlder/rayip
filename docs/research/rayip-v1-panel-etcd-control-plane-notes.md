# RayIP V1 Panel 与 etcd 控制面研究笔记

> 日期：2026-04-30  
> 目的：学习 Panel 与 etcd 的设计思想，收敛 RayIP 的 API / NATS / Worker / Redis / Postgres / NodeAgent 边界。

## 1. Panel 值得借鉴的部分

`refers/panel` 不是 RayIP 的目标架构，但它解决了一部分同类问题。

可以借鉴：

- 角色分离：`backend`、`node`、`scheduler`、`all-in-one`。RayIP 不拆微服务，但可以在同一个 Go API 内保留 API、worker、scheduler 的生命周期边界。
- NATS 解耦：Panel 通过 NATS 把 HTTP 请求和节点执行分离。RayIP 可以借鉴这个方向，但商业任务必须使用 NATS JetStream 持久队列，而不是只依赖临时 command publish。
- Node worker：Panel 的 node worker 持有节点连接、执行节点命令、查状态、拉日志。RayIP 的 stream owner / NodeAgent session router 可以借鉴这个思路。
- 健康检查：Panel 对节点健康检查做并发限制、超时处理、错误分类、状态回写。RayIP 应吸收这一点，避免大规模节点异常时自我放大故障。
- 批量统计写入：Panel 对 usage 做聚合、批量 UPSERT、时间桶、DB 写并发限制。RayIP 的流量汇总和节点指标汇总应该采用类似思路。
- 通知队列：Panel 用 JetStream stream + durable pull consumer 做通知队列。RayIP 的 Bark、Webhook、管理员告警也应按这个模型处理。
- 缓存恢复：Panel 使用 NATS KV 缓存部分 manager 状态。RayIP 不照搬 NATS KV，但可以学习“进程内状态必须能从外部恢复”的原则。

不能照搬：

- Panel 更像节点代理管理面板，不是 toC 静态家宽交易平台。
- Panel 的部分节点命令是 fire-and-forget，不适合作为 RayIP 发货正确性基础。
- Panel 存在全量同步用户的路径，不适合 RayIP 单节点 10k+ 订单场景。
- Panel 没有 RayIP 需要的余额冻结、库存预约、商业发货确认、SSH 自动恢复 NodeAgent。
- Panel 的后端是 Python / FastAPI / SQLAlchemy，不影响 RayIP Go First 的技术栈选择。

## 2. etcd 值得学习的设计哲学

RayIP 不引入 etcd 组件，但学习它的控制面思想。

### 2.1 单一事实源

etcd 把 key-value store 作为一致性事实源。RayIP 对应为 Postgres：

- 订单、钱包、库存、代理账号、Runtime 期望状态以 Postgres 为准。
- Redis 是热状态。
- NATS JetStream 是持久任务队列。
- NodeAgent 和 XrayCore 是执行状态，不是业务事实源。

### 2.2 Revision / Generation

etcd 每次修改都会推进 revision，revision 是全局逻辑时钟。

RayIP 对应：

- `runtime_change_log.seq` 是节点/Runtime 维度的变更序列。
- `proxy_accounts.desired_generation` 是账号维度的期望版本。
- NodeAgent 保存 `applied_generation`。
- 重复下发、乱序下发、Worker 重试时，只应用更高 generation。

### 2.3 Watch / Resume

etcd Watch 可以从指定 revision 开始，断线后按已知 revision 恢复。

RayIP 对应：

- NodeAgent 重连时上报最后已应用 `seq` 和 Runtime 快照摘要。
- API 从 Postgres 查缺失变更，生成分页增量 batch。
- 变更历史被压缩后，NodeAgent 走快照摘要对账，再分页补齐。

### 2.4 Lease

etcd Lease 用 TTL 和 keepalive 判断客户端活性。

RayIP 对应：

- NodeAgent 通过 gRPC stream 发送轻量 Lease。
- Redis 保存在线 Lease 热状态。
- Postgres 只记录状态变化和最后在线时间。
- Lease 过期后节点立即停止新销售。

### 2.5 Transaction / CAS

etcd 事务是 If / Then / Else，适合做 CAS 和并发控制。

RayIP 对应：

- 钱包冻结、库存预约、订单状态变更必须在 Postgres 事务中完成。
- 更新状态时带前置条件，例如 `status = PROVISIONING` 才能转 `ACTIVE`。
- 幂等键必须落 Postgres，重复请求返回同一个业务结果。

### 2.6 Compaction

etcd 会压缩历史 revision，客户端如果落后太多，需要重新同步。

RayIP 对应：

- `runtime_change_log` 保留有限窗口。
- 过旧变更按节点/账号压缩为当前 `runtime_account_states`。
- NodeAgent 落后窗口外时，不重放所有历史；改走快照对账 + 分页增量 apply。

### 2.7 批量和背压

etcd 通过批量提交提升吞吐，并承认磁盘和网络是性能边界。

RayIP 对应：

- Worker 按节点合并变更，生成增量 batch。
- 单节点、单线路、全局都要有并发预算。
- 指标写入按时间桶批量聚合。
- NATS 积压、Redis 不可用、Postgres 慢查询都必须触发限流或暂停销售，而不是无限重试。

## 3. RayIP 最终吸收

RayIP 的控制面收敛为：

```text
Postgres
  -> 根源状态、Runtime 期望状态、变更序列、幂等、审计
NATS JetStream
  -> 持久任务队列、ack、redelivery、durable consumer
Redis
  -> Lease、在线状态、session 路由、实时指标窗口、热缓存
Go API
  -> 无状态 HTTP/gRPC/WebSocket 接入层 + worker/scheduler 生命周期
NodeAgent
  -> 主动 gRPC stream 连接 API，执行增量 Runtime batch
XrayCore
  -> gRPC Xray API / 扩展 API 增量 apply，负责真实数据面
```

核心原则：

- 不引入 etcd。
- 不照搬 Panel。
- 学习 Panel 的 worker、健康检查、批量写入、通知队列。
- 学习 etcd 的 revision、watch、lease、CAS、compaction、背压思想。
- RayIP 自己的事实源仍然是 Postgres。
