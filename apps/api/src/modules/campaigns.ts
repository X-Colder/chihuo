import type { FastifyInstance } from "fastify";
import { ListQuerySchema } from "@chihuo/contracts";
import { requireRole } from "../auth.js";
import { assertFound, parseOrThrow } from "../errors.js";
import type { Store } from "../db/types.js";

export async function registerCampaignRoutes(app: FastifyInstance, store: Store): Promise<void> {
  app.get("/v1/campaigns", { preHandler: requireRole("CONSUMER", "MERCHANT", "ADMIN") }, async (request) => {
    const query = parseOrThrow(ListQuerySchema, request.query);
    return { data: await store.listCampaigns({ status: query.status ?? "OPEN" }) };
  });

  app.get("/v1/campaigns/:id", { preHandler: requireRole("CONSUMER", "MERCHANT", "ADMIN") }, async (request) => {
    const params = request.params as { id: string };
    return { data: assertFound(await store.getCampaign(params.id), "Campaign not found") };
  });
}
