# Go API Deployment

This directory contains deployment assets for the production-oriented Go API.
It does not modify or depend on the current Node API deployment.

## Go API contract

The Dockerfile expects the Go source tree at `backend-go` by default:

```text
backend-go/
├── go.mod
└── cmd/
    └── server/
        └── main.go
```

The Dockerfile builds `cmd/server` into `/usr/local/bin/chihuo-api`.
The release flow runs the idempotent PostgreSQL migration in a Kubernetes Job
before rolling out the API. The binary still performs the same idempotent
migration during startup as a compatibility fallback.

The API must listen on `0.0.0.0:4000` by default and expose:

```text
GET /health/live
GET /health/ready
```

The current Go configuration uses `HTTP_ADDR` for the listen address and
`CORS_ALLOWED_ORIGINS` for CORS.

## Docker Compose

Create a local environment file and replace all local-only values:

```bash
cp deploy/go/.env.example deploy/go/.env
docker compose \
  --env-file deploy/go/.env \
  -f deploy/go/docker-compose.yml \
  up --build
```

The PostgreSQL data volume is named `chihuo-go-postgres`; do not remove it
when troubleshooting.

When `DATABASE_URL` is present, the API connects to PostgreSQL and uses durable
database writes. The release Job runs the idempotent migration before the API
rollout. Without `DATABASE_URL`, the API uses the in-memory store for local
tests.

Redis is optional. Enable the Compose profile and configure the API URL:

```bash
REDIS_ENABLED=true \
REDIS_URL=redis://redis:6379/0 \
docker compose \
  --env-file deploy/go/.env \
  --profile redis \
  -f deploy/go/docker-compose.yml \
  up --build
```

The included Redis service is suitable for local integration testing. Use a
managed Redis-compatible service with authentication and TLS for production.
The first phase uses Redis for distributed fixed-window rate limiting when
`REDIS_URL` is configured. Without it, each API replica uses a local limiter;
production multi-replica deployments should configure Redis.

## Build the local API image

Run this from the repository root:

```bash
docker build \
  -f deploy/go/Dockerfile \
  --build-arg GO_API_DIR=backend-go \
  --build-arg GO_VERSION=1.24 \
  -t chihuo-go-api:local .
```

For a different source location, replace `GO_API_DIR`. For a registry image,
set `API_IMAGE` in the Compose environment or replace the Kustomize image
using `kustomize edit set image`.

## Kubernetes

The base Kustomize target includes PostgreSQL, the API, the API Service, and
Ingress. Redis is available as a separate overlay.

Create the required Secret out of band. Do not commit a real Secret:

```bash
kubectl -n chihuo-go create secret generic chihuo-go-api-secret \
  --from-literal=JWT_SECRET="$JWT_SECRET" \
  --from-literal=DATABASE_URL="$DATABASE_URL" \
  --from-literal=POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
  --from-literal=REDIS_PASSWORD="${REDIS_PASSWORD:-}" \
  --dry-run=client -o yaml | kubectl apply -f -
```

The command above assumes that `chihuo-go` already exists. Apply the base
manifests first if necessary:

```bash
kubectl apply -f deploy/go/k8s/base/namespace.yaml
```

For a local image, load it into the cluster runtime before applying:

```bash
kind load docker-image chihuo-go-api:local
```

Render and apply the base:

```bash
kubectl kustomize deploy/go/k8s
kubectl apply -k deploy/go/k8s
kubectl -n chihuo-go rollout status \
  deployment/chihuo-go-api --timeout=10m
```

Enable the optional in-cluster Redis StatefulSet and patch the API flag:

```bash
kubectl kustomize deploy/go/k8s/with-redis
kubectl apply -k deploy/go/k8s/with-redis
```

The base API image is `chihuo-go-api:local`. Replace it for a registry or
CI artifact without editing the Deployment:

```bash
kustomize edit set image \
  chihuo-go-api=registry.example.com/chihuo/api:2026-09-02
```

Run the command with `deploy/go/k8s` as the working directory, or update the
image field in a private overlay. Set `imagePullPolicy: Always` in that
overlay for mutable tags.

### Migration status

The base Kustomize target includes `chihuo-go-migrate`, a one-shot Job using the
same API image as the Deployment. It executes the binary's dedicated
`migrate` command and exits only after the idempotent migration succeeds.

Apply the non-API resources and migration Job first. The label selectors below
keep the API Deployment, Service, and Ingress out of the first step:

```bash
kubectl apply -k deploy/go/k8s \
  -l 'app.kubernetes.io/name!=chihuo-go-api'
kubectl -n chihuo-go wait \
  --for=condition=complete \
  job/chihuo-go-migrate \
  --timeout=10m
kubectl apply -k deploy/go/k8s \
  -l 'app.kubernetes.io/name=chihuo-go-api'
kubectl -n chihuo-go rollout status \
  deployment/chihuo-go-api \
  --timeout=10m
```

On a later release, a Kubernetes Job is immutable. Delete only the completed
migration Job before applying the new image or migration version:

```bash
kubectl -n chihuo-go delete job/chihuo-go-migrate --ignore-not-found
```

Use a release-specific Job name in a private overlay when retaining migration
history is required. Do not run a plain `kubectl apply -k deploy/go/k8s` in the
production release sequence, because it applies the API resources before the
explicit migration gate.

The API Deployment no longer runs schema migrations on startup. This prevents
three replicas from racing on schema changes. Compose uses a completed
`migrate` service before starting the API:

```yaml
command: ["/usr/local/bin/chihuo-api"]
args: ["migrate"]
```

Do not silently run incompatible schema changes from multiple API replicas.

For production, replace the in-cluster PostgreSQL and Redis resources with
managed services, use an external Secret manager, add TLS to the Ingress, and
set resource requests/limits based on measured traffic.

## Health checks

Local API check:

```bash
curl -fsS http://localhost:4000/health/live
curl -fsS http://localhost:4000/health/ready
```

Kubernetes checks:

```bash
kubectl -n chihuo-go get pods
kubectl -n chihuo-go describe pod -l app.kubernetes.io/name=chihuo-go-api
kubectl -n chihuo-go logs deployment/chihuo-go-api
```
