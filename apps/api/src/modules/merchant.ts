import type { FastifyInstance } from "fastify";
import {
  CreateCampaignInputSchema,
  CreateOfferInputSchema,
  ListQuerySchema
} from "@chihuo/contracts";
import { requireRole } from "../auth.js";
import { assert, assertFound, parseOrThrow } from "../errors.js";
import type { Store } from "../db/types.js";

export async function registerMerchantRoutes(app: FastifyInstance, store: Store): Promise<void> {
  app.get("/v1/merchant/demands", { preHandler: requireRole("MERCHANT") }, async (request) => {
    const query = parseOrThrow(ListQuerySchema, request.query);
    if (query.status) {
      return { data: await store.listDemands({ ...query, status: query.status }) };
    }
    const [open, ready] = await Promise.all([
      store.listDemands({ ...query, status: "OPEN" }),
      store.listDemands({ ...query, status: "READY" })
    ]);
    return { data: [...open, ...ready].sort((left, right) => right.createdAt.localeCompare(left.createdAt)) };
  });

  app.get("/v1/merchant/offers", { preHandler: requireRole("MERCHANT") }, async (request) => {
    const query = request.query as { demandId?: string; status?: string };
    assert(request.user.merchantId, "MERCHANT_PROFILE_REQUIRED", "Merchant profile is required", 403);
    return { data: await store.listOffers({ demandId: query.demandId, status: query.status, merchantId: request.user.merchantId }) };
  });

  app.post("/v1/merchant/offers", { preHandler: requireRole("MERCHANT") }, async (request, reply) => {
    const input = parseOrThrow(CreateOfferInputSchema, request.body);
    assert(request.user.merchantId, "MERCHANT_PROFILE_REQUIRED", "Merchant profile is required", 403);
    const offer = await store.createOffer(request.user.merchantId, input);
    return reply.code(201).send({ data: offer });
  });

  app.post("/v1/merchant/campaigns", { preHandler: requireRole("MERCHANT") }, async (request, reply) => {
    const input = parseOrThrow(CreateCampaignInputSchema, request.body);
    assert(request.user.merchantId, "MERCHANT_PROFILE_REQUIRED", "Merchant profile is required", 403);
    const campaign = await store.createCampaign(request.user.merchantId, input);
    return reply.code(201).send({ data: campaign });
  });

  app.get("/v1/merchant/campaigns", { preHandler: requireRole("MERCHANT") }, async (request) => {
    assert(request.user.merchantId, "MERCHANT_PROFILE_REQUIRED", "Merchant profile is required", 403);
    const query = request.query as { status?: string; demandId?: string };
    return { data: await store.listCampaigns({ merchantId: request.user.merchantId, status: query.status, demandId: query.demandId }) };
  });

  app.get("/v1/merchant/profile", { preHandler: requireRole("MERCHANT") }, async (request) => {
    assert(request.user.merchantId, "MERCHANT_PROFILE_REQUIRED", "Merchant profile is required", 403);
    return { data: assertFound(await store.getMerchant(request.user.merchantId), "Merchant not found") };
  });
}
