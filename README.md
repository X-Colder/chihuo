# 吃货 Chihuo

需求驱动的本地化预售餐饮平台。当前仓库同时保留业务原型和生产版第一阶段。

## 第一版工作区

- `apps/api`：统一后端 API
- `apps/consumer`：消费者端
- `apps/merchant`：商家端
- `apps/rider`：骑手端
- `apps/admin`：平台管理后台
- `packages/contracts`：跨端 API 类型和领域契约
- `deploy`：Docker Compose、容器和 Kubernetes 配置
- `backend-go`：生产版 Go API
- `miniprograms/consumer`：微信原生消费者小程序
- `miniprograms/merchant`：微信原生商家小程序
- `miniprograms/rider`：微信原生骑手小程序
- `deploy/go`：Go API 的 Docker Compose 和 Kubernetes 配置

## 核心闭环

消费者发布需求 -> 用户聚合需求 -> 商家报价 -> 平台审核 -> 预售下单 -> 生产批次 -> 骑手配送 -> 售后与食品安全记录。

## 本地开发

```bash
pnpm install
pnpm dev
```

各端默认端口：

- API: `4000`
- 消费者端: `5173`
- 商家端: `5174`
- 骑手端: `5175`
- 管理后台: `5176`

## Go API

```bash
cd backend-go
export JWT_SECRET="$(openssl rand -hex 32)"
export DEV_LOGIN_ENABLED=true
go run ./cmd/server
```

配置 `DATABASE_URL` 后，Go API 会使用 PostgreSQL 并在启动时执行幂等迁移；未配置时使用内存存储。

## 微信开发者工具

分别导入以下目录：

```text
miniprograms/consumer
miniprograms/merchant
miniprograms/rider
```

三端均为原生 WXML/WXSS/JavaScript 项目，可直接用微信开发者工具打开。详细说明见 [`docs/native-miniprogram.md`](docs/native-miniprogram.md)。

## 部署

```bash
docker compose -f deploy/docker-compose.yml up --build
kubectl apply -k deploy/k8s
```

Go 版部署：

```bash
docker compose --env-file deploy/go/.env -f deploy/go/docker-compose.yml up --build
kubectl apply -k deploy/go/k8s
```
