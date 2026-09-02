export type AppConfig = {
  nodeEnv: string;
  host: string;
  port: number;
  jwtSecret: string;
  databaseUrl?: string;
  corsOrigin: string;
};

export function loadConfig(env: NodeJS.ProcessEnv = process.env): AppConfig {
  const port = Number(env.PORT ?? 4000);
  if (!Number.isInteger(port) || port < 0 || port > 65_535) {
    throw new Error("PORT must be a valid TCP port");
  }

  return {
    nodeEnv: env.NODE_ENV ?? "development",
    host: env.HOST ?? "0.0.0.0",
    port,
    jwtSecret: env.JWT_SECRET ?? "chihuo-demo-secret-change-me",
    databaseUrl: env.DATABASE_URL,
    corsOrigin: env.CORS_ORIGIN ?? env.API_ALLOWED_ORIGINS ?? "*"
  };
}
