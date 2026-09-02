FROM node:22-alpine AS build

WORKDIR /workspace
RUN corepack enable
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY apps/api/package.json apps/api/package.json
COPY packages/contracts/package.json packages/contracts/package.json
RUN pnpm install --frozen-lockfile --ignore-scripts

COPY apps/api apps/api
COPY packages/contracts packages/contracts
RUN pnpm --filter @chihuo/contracts build
RUN pnpm --filter @chihuo/api build

FROM node:22-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
RUN corepack enable
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY apps/api/package.json apps/api/package.json
COPY packages/contracts/package.json packages/contracts/package.json
RUN pnpm install --prod --frozen-lockfile --ignore-scripts
COPY --from=build /workspace/apps/api/dist apps/api/dist
COPY --from=build /workspace/packages/contracts/dist packages/contracts/dist
COPY apps/api/migrations apps/api/migrations

WORKDIR /app/apps/api
EXPOSE 4000
CMD ["sh", "-c", "node dist/cli/migrate.js && node dist/cli/seed.js && node dist/server.js"]
