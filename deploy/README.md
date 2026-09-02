# 吃货部署说明

本目录提供两种部署方式：

- Docker Compose：本地开发、联调和单机演示。
- Kubernetes + Kustomize：集群部署，包含 Postgres、API、消费者端、商家端、骑手端、管理后台和 Ingress。

部署配置只假设 workspace 中存在以下包：

```text
apps/api       @chihuo/api
apps/consumer  @chihuo/consumer
apps/merchant  @chihuo/merchant
apps/rider     @chihuo/rider
apps/admin     @chihuo/admin
```

前端构建产物默认位于各应用的 `dist/`，API 默认入口为
`apps/api/dist/index.js`。如 API 使用其他入口，可以通过 `API_ENTRYPOINT`
覆盖。

## 健康接口约定

API 应提供：

- `GET /healthz`：进程存活，不要求数据库业务查询。
- `GET /readyz`：服务已就绪，建议检查数据库连接和必要依赖。

四个前端由 Nginx 提供：

- `GET /healthz`：返回 `200 ok`。
- 其他路径：回退到 `index.html`，支持 SPA 路由。

## Docker Compose

复制本地环境变量文件并修改密码：

```bash
cp deploy/.env.example deploy/.env
```

检查 Compose 配置：

```bash
docker compose \
  --env-file deploy/.env \
  -f deploy/docker-compose.yml \
  config --quiet
```

构建并启动 API、Postgres 和四个前端：

```bash
docker compose \
  --env-file deploy/.env \
  -f deploy/docker-compose.yml \
  up --build -d
```

默认访问地址：

| 服务 | 地址 |
| --- | --- |
| API | `http://localhost:4000/healthz` |
| 消费者端 | `http://localhost:5173` |
| 商家端 | `http://localhost:5174` |
| 骑手端 | `http://localhost:5175` |
| 管理后台 | `http://localhost:5176` |
| Postgres | `localhost:5432` |

查看状态和日志：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml ps
docker compose --env-file deploy/.env -f deploy/docker-compose.yml logs -f api
```

停止服务但保留数据库卷：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml down
```

只有明确需要清空本地数据库时才删除卷：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml down -v
```

Compose 会生成以下本地镜像，Kubernetes 默认也使用这些标签：

```text
chihuo/api:local
chihuo/consumer:local
chihuo/merchant:local
chihuo/rider:local
chihuo/admin:local
```

## Kubernetes

### 预览渲染结果

```bash
kubectl kustomize deploy/k8s
```

### 本地集群

为 Kubernetes 构建本地镜像时，前端 API 地址应编译为 Ingress 下的 `/api`，不要沿用
Compose 本地开发用的 `http://localhost:4000/api`：

```bash
VITE_API_BASE_URL=/api docker compose \
  --env-file deploy/.env.example \
  -f deploy/docker-compose.yml \
  build api consumer merchant rider admin
```

如果使用 Kind，将镜像加载到集群：

```bash
kind load docker-image \
  chihuo/api:local \
  chihuo/consumer:local \
  chihuo/merchant:local \
  chihuo/rider:local \
  chihuo/admin:local
```

如果使用 Minikube，使用：

```bash
minikube image load chihuo/api:local
minikube image load chihuo/consumer:local
minikube image load chihuo/merchant:local
minikube image load chihuo/rider:local
minikube image load chihuo/admin:local
```

应用清单：

```bash
kubectl apply -k deploy/k8s
kubectl -n chihuo get pods,svc,ingress
```

等待发布完成：

```bash
kubectl -n chihuo rollout status statefulset/postgres
kubectl -n chihuo rollout status deployment/api
kubectl -n chihuo rollout status deployment/consumer
kubectl -n chihuo rollout status deployment/merchant
kubectl -n chihuo rollout status deployment/rider
kubectl -n chihuo rollout status deployment/admin
```

### 本地访问方式

Ingress 示例使用以下域名：

```text
consumer.chihuo.local
merchant.chihuo.local
rider.chihuo.local
admin.chihuo.local
api.chihuo.local
```

将这些域名指向 Ingress 控制器地址，或临时写入 `/etc/hosts`。如果只需要快速验证服务，不使用 Ingress 也可以端口转发：

```bash
kubectl -n chihuo port-forward svc/consumer 5173:80
kubectl -n chihuo port-forward svc/merchant 5174:80
kubectl -n chihuo port-forward svc/rider 5175:80
kubectl -n chihuo port-forward svc/admin 5176:80
kubectl -n chihuo port-forward svc/api 4000:4000
```

健康检查：

```bash
curl -fsS http://localhost:4000/healthz
curl -fsS http://localhost:4000/readyz
curl -fsS http://localhost:5173/healthz
```

### 生产镜像

默认镜像标签是 `local`，适用于本地集群。生产环境应把
`deploy/k8s/kustomization.yaml` 中的 `images` 块替换为实际可访问的镜像仓库和不可变版本标签，例如：

```yaml
images:
  - name: chihuo/api
    newName: registry.example.com/chihuo/api
    newTag: "2026.09.02-abc123"
```

四个前端和 API 都应使用同一版本发布。生产部署前还应：

1. 将 `deploy/k8s/secret.example.yaml` 替换为集群外部 Secret 管理方案，或在应用前生成同名 Secret。
2. 修改 `API_ALLOWED_ORIGINS` 为实际前端域名。
3. 为 Ingress 配置 TLS，并将 `ingressClassName` 调整为集群实际使用的控制器。
4. 为 Postgres 配置可靠的备份、恢复和存储类。
5. 确认 API 的迁移流程是幂等的，并在发布流程中单独执行迁移。

示例 Secret 仅用于本地演示，不能直接用于生产环境。Kubernetes Secret 默认不会因为写在 YAML 中而自动安全，生产环境应使用 Secret Manager、External Secrets 或等效方案。

## 回滚

Compose 回滚到之前的源码版本后重新构建并启动：

```bash
docker compose --env-file deploy/.env -f deploy/docker-compose.yml up --build -d
```

Kubernetes 回滚最近一次 Deployment：

```bash
kubectl -n chihuo rollout undo deployment/api
kubectl -n chihuo rollout undo deployment/consumer
kubectl -n chihuo rollout undo deployment/merchant
kubectl -n chihuo rollout undo deployment/rider
kubectl -n chihuo rollout undo deployment/admin
```

Postgres 的 StatefulSet 和 PVC 不应作为普通回滚动作删除。数据库卷需要单独备份和恢复。

## 当前仓库状态

当前 workspace 仅包含根级 workspace 文件，`apps/*` 尚未生成时，镜像构建会因缺少应用包而失败。这是预期行为；应用实现完成并满足上面的包名、构建产物和健康接口约定后，部署配置可直接复用。
