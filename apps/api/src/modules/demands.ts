import type { FastifyInstance } from "fastify";
import {
  CreateDemandInputSchema,
  DemandReviewInputSchema,
  JoinDemandInputSchema,
  ListQuerySchema
} from "@chihuo/contracts";
import { requireRole } from "../auth.js";
import { ApiError, assert, assertFound, parseOrThrow } from "../errors.js";
import { normalizeList } from "../utils.js";
import type { Store } from "../db/types.js";

export async function registerDemandRoutes(app: FastifyInstance, store: Store): Promise<void> {
  app.post("/v1/demands", { preHandler: requireRole("CONSUMER") }, async (request, reply) => {
    const input = parseOrThrow(CreateDemandInputSchema, request.body);
    const normalized = {
      ...input,
      hardConstraints: normalizeList(input.hardConstraints),
      preferences: normalizeList(input.preferences)
    };
    const matching = await store.findMatchingDemand(normalized);
    if (matching) {
      const member = await store.addDemandMember({
        demandId: matching.id,
        userId: request.user.id,
        quantity: input.quantity,
        weightGrams: input.weightMinGrams === input.weightMaxGrams ? input.weightMinGrams : undefined,
        preferences: normalized.preferences,
        notes: input.notes
      });
      return reply.code(200).send({ data: { demand: await store.getDemand(matching.id), member, matched: true } });
    }

    const demand = await store.createDemand({
      createdBy: request.user.id,
      minimumMembers: input.minimumMembers,
      maximumMembers: input.maximumMembers,
      spec: normalized
    });
    const member = await store.addDemandMember({
      demandId: demand.id,
      userId: request.user.id,
      quantity: input.quantity,
      weightGrams: input.weightMinGrams === input.weightMaxGrams ? input.weightMinGrams : undefined,
      preferences: normalized.preferences,
      notes: input.notes
    });
    return reply.code(201).send({ data: { demand: await store.getDemand(demand.id), member, matched: false } });
  });

  app.get("/v1/demands", { preHandler: requireRole("CONSUMER", "MERCHANT", "ADMIN") }, async (request) => {
    const query = parseOrThrow(ListQuerySchema, request.query);
    return { data: await store.listDemands(query) };
  });

  app.get("/v1/demands/:id", { preHandler: requireRole("CONSUMER", "MERCHANT", "ADMIN") }, async (request) => {
    const params = request.params as { id: string };
    const demand = assertFound(await store.getDemand(params.id), "Demand not found");
    return { data: demand };
  });

  app.post("/v1/demands/:id/join", { preHandler: requireRole("CONSUMER") }, async (request, reply) => {
    const params = request.params as { id: string };
    const input = parseOrThrow(JoinDemandInputSchema, request.body);
    const demand = assertFound(await store.getDemand(params.id), "Demand not found");
    assert(demand.status !== "REJECTED" && demand.status !== "CLOSED", "DEMAND_NOT_OPEN", "Demand is not open for joining", 409);
    const member = await store.addDemandMember({
      demandId: demand.id,
      userId: request.user.id,
      quantity: input.quantity,
      weightGrams: input.weightGrams,
      preferences: normalizeList(input.preferences),
      notes: input.notes
    });
    return reply.code(201).send({ data: { demand: await store.getDemand(demand.id), member } });
  });

  app.get("/v1/demands/:id/members", { preHandler: requireRole("MERCHANT", "ADMIN") }, async (request) => {
    const params = request.params as { id: string };
    assertFound(await store.getDemand(params.id), "Demand not found");
    const members = await store.listDemandMembers(params.id);
    return {
      data: members.map(({ userId: _userId, ...member }) => member)
    };
  });

  app.patch("/v1/admin/demands/:id/review", { preHandler: requireRole("ADMIN") }, async (request) => {
    const params = request.params as { id: string };
    const input = parseOrThrow(DemandReviewInputSchema, request.body);
    const demand = await store.reviewDemand(params.id, input.status, request.user.id, input.reason);
    return { data: demand };
  });

  app.get("/v1/admin/demands", { preHandler: requireRole("ADMIN") }, async (request) => {
    const query = parseOrThrow(ListQuerySchema, request.query);
    return { data: await store.listDemands(query) };
  });
}
