import type { FastifyInstance } from "fastify";
import { CreateIncidentInputSchema } from "@chihuo/contracts";
import { requireRole } from "../auth.js";
import { assert, assertFound, parseOrThrow } from "../errors.js";
import type { Store } from "../db/types.js";

export async function registerFoodSafetyRoutes(app: FastifyInstance, store: Store): Promise<void> {
  app.post("/v1/food-safety/incidents", { preHandler: requireRole("MERCHANT", "ADMIN") }, async (request, reply) => {
    const input = parseOrThrow(CreateIncidentInputSchema, request.body);
    const merchant = assertFound(await store.getMerchant(input.merchantId), "Merchant not found");
    if (request.user.role === "MERCHANT") {
      assert(request.user.merchantId === merchant.id, "FORBIDDEN", "Merchant can only report incidents for its own store", 403);
    }
    const incident = await store.createIncident(request.user.id, input);
    return reply.code(201).send({ data: incident });
  });

  app.get("/v1/food-safety/incidents", { preHandler: requireRole("MERCHANT", "ADMIN") }, async (request) => {
    const query = request.query as { merchantId?: string; status?: string };
    const merchantId = request.user.role === "MERCHANT" ? request.user.merchantId : query.merchantId;
    assert(request.user.role === "ADMIN" || merchantId, "MERCHANT_PROFILE_REQUIRED", "Merchant profile is required", 403);
    return { data: await store.listIncidents({ merchantId, status: query.status }) };
  });
}
