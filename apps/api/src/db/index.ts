import { Pool } from "pg";
import type { AppConfig } from "../config.js";
import { MemoryStore } from "./memory.js";
import { PostgresStore } from "./postgres.js";
import type { Store } from "./types.js";

export function createStore(config: AppConfig): Store {
  if (!config.databaseUrl) return new MemoryStore();
  return new PostgresStore(
    new Pool({
      connectionString: config.databaseUrl,
      max: Number(process.env.DB_POOL_MAX ?? 10),
      idleTimeoutMillis: 30_000,
      connectionTimeoutMillis: 5_000
    })
  );
}
