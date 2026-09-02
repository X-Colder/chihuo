import { createStore } from "../db/index.js";
import { loadConfig } from "../config.js";

const config = loadConfig();
if (!config.databaseUrl) {
  console.log("DATABASE_URL is required for database seed");
  process.exit(1);
}

const store = createStore(config);
await store.seed();
await store.close();
console.log("Seed data applied");
