# Chihuo API

模块化单体 Fastify API，覆盖第一版的需求聚合、商家报价、预售、订单、骑手任务和食品安全事件基础流程。

## 运行模式

- `DATABASE_URL` 存在时使用 PostgreSQL。
- 未配置 `DATABASE_URL` 时使用进程内内存仓储，适合本地快速开发和 API 测试。
- 生产和容器环境必须使用 PostgreSQL，并在启动前执行迁移。

## 本地启动

```bash
pnpm install
pnpm --filter @chihuo/contracts build
pnpm --filter @chihuo/api migrate
pnpm --filter @chihuo/api seed
pnpm --filter @chihuo/api dev
```

默认监听 `0.0.0.0:4000`。最小环境变量：

```bash
PORT=4000
HOST=0.0.0.0
JWT_SECRET=replace-this-in-production
DATABASE_URL=postgres://chihuo:chihuo@localhost:5432/chihuo
```

## Docker Compose

仅启动 API 和 PostgreSQL：

```bash
docker compose -f apps/api/deploy/docker-compose.yml up --build
```

Compose 会先执行迁移和种子脚本，再启动 API。健康检查地址是 `GET /health/ready`。

## Kubernetes

先构建本地镜像：

```bash
docker build -f apps/api/Dockerfile -t chihuo-api:local .
```

再应用清单：

```bash
kubectl apply -k apps/api/deploy/k8s
kubectl -n chihuo rollout status deployment/chihuo-api
kubectl -n chihuo port-forward svc/chihuo-api 4000:4000
```

示例清单包含单实例 PostgreSQL、Secret、ConfigMap、迁移 Job、Deployment、Service 和探针。生产环境应替换为托管 PostgreSQL、外部 Secret 管理和持久化存储。

## 认证

第一版提供演示登录，不接短信或第三方 OAuth：

```bash
curl -s http://localhost:4000/v1/auth/demo-login \
  -H 'content-type: application/json' \
  -d '{"name":"小明","role":"CONSUMER"}'
```

将返回的 `token` 作为 `Authorization: Bearer <token>` 访问受保护接口。演示角色支持 `CONSUMER`、`MERCHANT`、`RIDER`、`ADMIN`。

## 关键接口

| 方法 | 路径 | 角色 |
| --- | --- | --- |
| GET | `/health/live` | 公开 |
| GET | `/health/ready` | 公开 |
| POST | `/v1/auth/demo-login` | 公开 |
| POST | `/v1/demands` | 消费者 |
| GET | `/v1/demands` | 登录用户 |
| POST | `/v1/demands/:id/join` | 消费者 |
| GET | `/v1/merchant/demands` | 商家 |
| POST | `/v1/merchant/offers` | 商家 |
| PATCH | `/v1/admin/demands/:id/review` | 平台管理员 |
| GET | `/v1/admin/merchants` | 平台管理员 |
| PATCH | `/v1/admin/merchants/:id/review` | 平台管理员 |
| POST | `/v1/merchant/campaigns` | 商家 |
| GET | `/v1/campaigns` | 登录用户 |
| POST | `/v1/campaigns/:id/orders` | 消费者 |
| POST | `/v1/orders/:id/pay` | 消费者 |
| GET | `/v1/orders` | 登录用户 |
| GET | `/v1/rider/tasks` | 骑手 |
| POST | `/v1/rider/tasks/:id/claim` | 骑手 |
| PATCH | `/v1/rider/tasks/:id` | 骑手 |
| POST | `/v1/food-safety/incidents` | 商家/管理员 |
| GET | `/v1/food-safety/incidents` | 商家/管理员 |

所有错误返回统一结构：

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Request validation failed",
    "details": {},
    "requestId": "req-..."
  }
}
```

## 迁移与种子

迁移脚本读取 `apps/api/migrations/*.sql`，按文件名顺序执行并记录到 `schema_migrations`，可重复执行：

```bash
pnpm --filter @chihuo/api migrate
pnpm --filter @chihuo/api seed
```

种子数据包括一个管理员、一个已审核商家、一个消费者、一个骑手、一个开放需求和一组演示报价/预售活动。种子使用固定演示邮箱，重复执行不会产生重复用户。

## 测试

默认测试使用内存仓储，不需要 PostgreSQL：

```bash
pnpm --filter @chihuo/api typecheck
pnpm --filter @chihuo/api test
pnpm --filter @chihuo/api build
```
