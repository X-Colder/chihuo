import Fastify, { type FastifyInstance } from "fastify";
import type { AppConfig } from "./config.js";
import { newId } from "./utils.js";
import { registerAuth } from "./auth.js";
import { registerErrorHandler } from "./errors.js";
import { createStore } from "./db/index.js";
import type { Store } from "./db/types.js";
import { registerRoutes } from "./routes.js";

export type BuildAppOptions = {
  config: AppConfig;
  store?: Store;
  logger?: boolean;
};

export async function buildApp(options: BuildAppOptions): Promise<FastifyInstance> {
  const app = Fastify({
    logger: options.logger ?? false,
    requestIdHeader: "x-request-id",
    genReqId: () => newId(),
    bodyLimit: 1_000_000
  });
  const store = options.store ?? createStore(options.config);
  app.decorate("store", store);

  app.addHook("onRequest", async (request, reply) => {
    const origin = request.headers.origin;
    const allowedOrigins = options.config.corsOrigin.split(",").map((value) => value.trim()).filter(Boolean);
    if (origin && (allowedOrigins.includes("*") || allowedOrigins.includes(origin))) {
      reply.header("access-control-allow-origin", allowedOrigins.includes("*") ? "*" : origin);
      reply.header("access-control-allow-credentials", "true");
      reply.header("vary", "Origin");
    }
    reply.header("access-control-allow-headers", "Authorization, Content-Type, X-Request-ID");
    reply.header("access-control-allow-methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS");
    if (request.method === "OPTIONS") {
      return reply.code(204).send();
    }
  });

  await registerAuth(app, options.config.jwtSecret);
  registerErrorHandler(app);

  app.get("/health/live", async () => ({ status: "ok", service: "chihuo-api" }));
  app.get("/healthz", async () => ({ status: "ok", service: "chihuo-api" }));
  app.get("/health/ready", async (_request, reply) => {
    try {
      await store.ping();
      return { status: "ready", service: "chihuo-api", persistence: options.config.databaseUrl ? "postgres" : "memory" };
    } catch {
      return reply.code(503).send({
        error: {
          code: "NOT_READY",
          message: "Persistence is unavailable",
          requestId: reply.request.id
        }
      });
    }
  });
  app.get("/readyz", async (_request, reply) => {
    try {
      await store.ping();
      return { status: "ready", service: "chihuo-api", persistence: options.config.databaseUrl ? "postgres" : "memory" };
    } catch {
      return reply.code(503).send({
        error: {
          code: "NOT_READY",
          message: "Persistence is unavailable",
          requestId: reply.request.id
        }
      });
    }
  });

  await registerRoutes(app, store);
  return app;
}

declare module "fastify" {
  interface FastifyInstance {
    store: Store;
  }
}
