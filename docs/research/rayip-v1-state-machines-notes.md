# RayIP V1 状态机与不变量研究笔记

> 日期：2026-04-29

## 1. 核心不变量

RayIP V1 的核心不变量：

> 客户能购买，说明平台已经判断可售；平台标记已交付，说明 Runtime 已确认可用。

管理员可以看到内部原因，但客户流程不能依赖阅读内部失败状态。

## 2. 订单状态

```text
DRAFT
  -> PAYMENT_HELD
  -> PROVISIONING
  -> ACTIVE
  -> RENEWING
  -> EXPIRED
  -> DISABLING
  -> DISABLED
  -> REFUNDING
  -> REFUNDED
  -> FAILED
```

规则：

- `PAYMENT_HELD` 是余额冻结，不是收入确认。
- `ACTIVE` 必须有可用 endpoint、凭据、过期时间和 Runtime 策略。
- 发货前失败必须释放余额和库存。
- 退款必须有钱包流水证据。

## 3. 代理账号状态

```text
CREATED
  -> APPLYING
  -> ACTIVE
  -> UPDATING_CREDENTIALS
  -> REFRESHING_IP
  -> RENEWING
  -> DISABLING
  -> DISABLED
  -> ERROR
```

规则：

- 凭据只在 Runtime 确认后展示。
- 同一代理账号的续费、停用、凭据修改、IP 刷新必须串行。
- 账号策略版本必须和 Runtime 确认版本一致。

## 4. 库存状态

```text
AVAILABLE
  -> HELD
  -> SOLD
  -> DISABLED
  -> BAD
  -> RETIRED
```

规则：

- 只有 `AVAILABLE` 可预约。
- `HELD` 必须有 TTL。
- `SOLD` 必须关联订单项。
- `BAD` 不可售，直到检测或管理员解除。

## 5. 节点状态

```text
REGISTERED
  -> INSTALLING
  -> ACTIVE
  -> DEGRADED
  -> DRAINING
  -> RETIRED
  -> OFFLINE
  -> REMOVED
```

规则：

- 只有 `ACTIVE` 节点可售。
- `DEGRADED`、`OFFLINE`、`DRAINING`、`RETIRED`、`REMOVED` 都不卖新库存。
- Runtime Bundle 能力不匹配时进入 `DEGRADED`。

## 6. 任务状态

```text
QUEUED
  -> RUNNING
  -> SUCCEEDED
  -> RETRYING
  -> FAILED
  -> CANCELLED
```

规则：

- 每个异步任务都有 Postgres 行。
- 每条 NATS 消息引用任务 ID 和幂等键。
- Worker 用 Postgres 状态作为执行保护。
- 最终失败必须触发领域补偿。

## 7. 发布前必须验证

- 并发下单不超卖。
- 易支付回调不重复入账。
- 购买不重复确认扣款。
- 活跃订单一定有 endpoint 和凭据。
- 过期订单最终不可用。
- IP 刷新失败时旧 IP 仍保持可用。
- 凭据刷新失败时旧凭据仍保持可用。
- 节点异常会停止新销售。

