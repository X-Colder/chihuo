import { resolve } from "node:path";
import { runMigrations } from "../db/migrations.js";

const databaseUrl = process.env.DATABASE_URL;
if (!databaseUrl) {
  console.error("DATABASE_URL is required for migrations");
  process.exit(1);
}

await runMigrations(databaseUrl, resolve(process.cwd(), "migrations"));
console.log("Migrations applied");
