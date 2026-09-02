# Chihuo 生产版架构

## 目标

生产版采用 Go 后端和微信原生小程序。当前 React/Vite 版本保留为业务原型，Go API 和原生小程序是后续生产主线。

“百万并发”在落地前必须拆成可压测指标。第一阶段将 100 万在线连接、10 万读请求/秒、1 万写请求/秒作为容量验证目标，而不是未经压测的承诺。

## 服务边界

第一阶段使用模块化 Go 单体，保持以下独立边界：

- auth：微信登录、会话和角色权限。
- demand：需求发布、规范化、相似需求聚合、成员加入。
- merchant：商家资料、资质、报价。
- campaign：商家预售方案和商品规格。
- order：下单、幂等、订单状态。
- delivery：骑手任务和配送状态。
- safety：食品安全事件、证据和处置。
- notification：订阅消息和异步通知。

当单个模块出现独立扩容、发布或数据隔离需求时，再拆成独立服务。

## 基础设施

- PostgreSQL 或 MySQL 兼容数据库：订单、支付、资质和食品安全记录。
- Redis Cluster：会话、热点需求计数、限流和短期缓存。
- Kafka：需求聚合、订单、通知和审计事件。
- OpenSearch：需求和商品检索。
- 对象存储：证照、批次文件和直播录像。
- Kubernetes：多副本、滚动发布、自动扩缩容。

关键写操作需要 `Idempotency-Key`。订单、支付和库存变化以数据库事务为准，事件通过 Outbox 发布，避免重复扣款和丢事件。

## 小程序

- `miniprograms/consumer`：消费者原生小程序。
- `miniprograms/merchant`：商家原生小程序。
- `miniprograms/rider`：骑手原生小程序。
- `apps/admin`：平台 PC Web 后台，暂保留 React 版本。

每个小程序都必须包含 `project.config.json`、`app.json`、`app.js`、`app.wxss` 和页面目录，可直接导入微信开发者工具。
