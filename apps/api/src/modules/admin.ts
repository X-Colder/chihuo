import type { FastifyInstance } from "fastify";
import { CampaignStatusSchema, ListQuerySchema, MerchantReviewInputSchema } from "@chihuo/contracts";
import { requireRole } from "../auth.js";
import { parseOrThrow } from "../errors.js";
import type { Store } from "../db/types.js";

export async function registerAdminRoutes(app: FastifyInstance, store: Store): Promise<void> {
  app.get("/v1/admin/merchants", { preHandler: requireRole("ADMIN") }, async (request) => {
    const query = request.query as { status?: string };
    return { data: await store.listMerchants(query.status) };
  });

  app.patch("/v1/admin/merchants/:id/review", { preHandler: requireRole("ADMIN") }, async (request) => {
    const params = request.params as { id: string };
    const input = parseOrThrow(MerchantReviewInputSchema, request.body);
    return { data: await store.reviewMerchant(params.id, input.status, input.reason) };
  });

  app.get("/v1/admin/campaigns", { preHandler: requireRole("ADMIN") }, async (request) => {
    const query = parseOrThrow(ListQuerySchema, request.query);
    return { data: await store.listCampaigns({ status: query.status }) };
  });

  app.patch("/v1/admin/campaigns/:id/review", { preHandler: requireRole("ADMIN") }, async (request) => {
    const params = request.params as { id: string };
    const body = request.body as { status?: unknown; reason?: unknown };
    const status = parseOrThrow(CampaignStatusSchema, body.status);
    return { data: await store.reviewCampaign(params.id, status, typeof body.reason === "string" ? body.reason : undefined) };
  });
}
