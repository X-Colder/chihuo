import { buildApp } from "./app.js";
import { loadConfig } from "./config.js";
import { createStore } from "./db/index.js";

const config = loadConfig();
const store = createStore(config);
if (!config.databaseUrl) {
  await store.seed();
}
const app = await buildApp({ config, store, logger: true });

const shutdown = async (signal: string): Promise<void> => {
  app.log.info({ signal }, "shutting down");
  await app.close();
  await store.close();
  process.exit(0);
};

process.once("SIGINT", () => void shutdown("SIGINT"));
process.once("SIGTERM", () => void shutdown("SIGTERM"));

try {
  await app.listen({ host: config.host, port: config.port });
  app.log.info({ host: config.host, port: config.port }, "server listening");
} catch (error) {
  app.log.error(error);
  await store.close();
  process.exit(1);
}
