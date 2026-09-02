import type { FastifyInstance } from "fastify";
import type { AppConfig } from "./config.js";
import type { Store } from "./db/types.js";

export type AppContext = {
  app: FastifyInstance;
  store: Store;
  config: AppConfig;
};
