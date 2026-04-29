# RayIP V1 985Proxy 静态家宽 API 研究笔记

> 日期：2026-04-29  
> 范围：学习 985Proxy 静态家宽 API 设计，不照抄其产品模型

## 1. 可以学习什么

985Proxy 的静态家宽 API 对 RayIP 有参考价值：

- 业务用途列表
- 按 IP 类型、国家、城市查询库存
- 价格计算
- 购买
- 续费
- IP 列表 / 详情
- 修改代理账号密码
- 可更换 IP 列表
- 更换 IP

重点不是接入或照搬，而是学习一个商业静态家宽平台如何把生命周期 API 做得小而完整。

## 2. RayIP 决策

- 对外开发者身份使用 `AppID + AppSecret`，不是单个 API 密钥。
- 允许一个用户创建多个开发者 App。
- 用户面板的开发者设置展示 AppID、AppSecret、IP 白名单、回调地址、启停、重置、删除。
- V1 不引入 `Zone` 产品概念。
- 客户选择用途、IP 类型、国家/城市、线路、时长、数量、协议。
- 代理详情是客户侧事实来源：IP、端口、账号、密码、地区、到期时间、状态、流量、生命周期记录。
- IP 刷新表示替换当前代理 IP，不创建新订单。
- 凭据刷新必须 Runtime 确认后才展示。

## 3. 不照搬什么

- 不暴露 985Proxy 的 `zone` 要求。
- 不用供应商内部字段命名 RayIP 产品。
- 不让供应商订单号成为客户订单模型。
- 不把外部库存快照直接当作 RayIP 可售库存。
- 不把 RayIP 做成薄薄的转售 API 包装。

## 4. 已查看页面

- https://docs.985proxy.com/414120986e0.md
- https://docs.985proxy.com/414121284e0.md
- https://docs.985proxy.com/414122338e0.md
- https://docs.985proxy.com/414122390e0.md
- https://docs.985proxy.com/414211353e0.md
- https://docs.985proxy.com/414119981e0.md
- https://docs.985proxy.com/414120746e0.md
- https://docs.985proxy.com/414123289e0.md
- https://docs.985proxy.com/414123430e0.md
- https://docs.985proxy.com/414123473e0.md
- https://docs.985proxy.com/8140229m0.md
