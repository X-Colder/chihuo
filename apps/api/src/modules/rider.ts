import type { FastifyInstance } from "fastify";
import { ListQuerySchema, UpdateRiderTaskInputSchema } from "@chihuo/contracts";
import { requireRole } from "../auth.js";
import { assertFound, parseOrThrow } from "../errors.js";
import type { Store } from "../db/types.js";

export async function registerRiderRoutes(app: FastifyInstance, store: Store): Promise<void> {
  app.get("/v1/rider/tasks", { preHandler: requireRole("RIDER") }, async (request) => {
    const query = parseOrThrow(ListQuerySchema, request.query);
    return { data: await store.listRiderTasks({ riderId: request.user.id, status: query.status }) };
  });

  app.get("/v1/rider/tasks/queue", { preHandler: requireRole("RIDER") }, async (request) => {
    const query = parseOrThrow(ListQuerySchema, request.query);
    return { data: await store.listRiderTasks({ status: query.status ?? "UNASSIGNED" }) };
  });

  app.post("/v1/rider/tasks/:id/claim", { preHandler: requireRole("RIDER") }, async (request) => {
    const params = request.params as { id: string };
    assertFound(await store.getRiderTask(params.id), "Rider task not found");
    return { data: await store.claimRiderTask(params.id, request.user.id) };
  });

  app.patch("/v1/rider/tasks/:id", { preHandler: requireRole("RIDER") }, async (request) => {
    const params = request.params as { id: string };
    const input = parseOrThrow(UpdateRiderTaskInputSchema, request.body);
    return { data: await store.updateRiderTask(params.id, request.user.id, input.status, input.note) };
  });
}
