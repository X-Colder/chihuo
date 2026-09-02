import type { FastifyInstance } from "fastify";
import { CreateOrderInputSchema, UpdateOrderStatusInputSchema } from "@chihuo/contracts";
import { requireRole } from "../auth.js";
import { assert, assertFound, parseOrThrow } from "../errors.js";
import type { Store } from "../db/types.js";

export async function registerOrderRoutes(app: FastifyInstance, store: Store): Promise<void> {
  app.post("/v1/campaigns/:id/orders", { preHandler: requireRole("CONSUMER") }, async (request, reply) => {
    const params = request.params as { id: string };
    const input = parseOrThrow(CreateOrderInputSchema, request.body);
    assertFound(await store.getCampaign(params.id), "Campaign not found");
    const order = await store.createOrder(request.user.id, { ...input, campaignId: params.id });
    return reply.code(201).send({ data: order });
  });

  app.post("/v1/orders/:id/pay", { preHandler: requireRole("CONSUMER") }, async (request) => {
    const params = request.params as { id: string };
    const order = assertFound(await store.getOrder(params.id), "Order not found");
    assert(order.consumerId === request.user.id, "FORBIDDEN", "You can only pay for your own order", 403);
    return { data: await store.updateOrderStatus(order.id, "PAID") };
  });

  app.get("/v1/orders", { preHandler: requireRole("CONSUMER", "MERCHANT", "RIDER", "ADMIN") }, async (request) => {
    const query = request.query as { status?: string };
    const filters =
      request.user.role === "CONSUMER"
        ? { consumerId: request.user.id, status: query.status }
        : request.user.role === "MERCHANT"
          ? { merchantId: request.user.merchantId, status: query.status }
          : request.user.role === "RIDER"
            ? { riderId: request.user.id, status: query.status }
            : { status: query.status };
    return { data: await store.listOrders(filters) };
  });

  app.get("/v1/orders/:id", { preHandler: requireRole("CONSUMER", "MERCHANT", "RIDER", "ADMIN") }, async (request) => {
    const params = request.params as { id: string };
    const order = assertFound(await store.getOrder(params.id), "Order not found");
    await assertOrderVisible(store, request.user, order.id);
    return { data: order };
  });

  app.patch("/v1/merchant/orders/:id/status", { preHandler: requireRole("MERCHANT") }, async (request) => {
    const params = request.params as { id: string };
    const input = parseOrThrow(UpdateOrderStatusInputSchema, request.body);
    assert(request.user.merchantId, "MERCHANT_PROFILE_REQUIRED", "Merchant profile is required", 403);
    const order = assertFound(await store.getOrder(params.id), "Order not found");
    const campaigns = await store.listCampaigns({ merchantId: request.user.merchantId });
    assert(campaigns.some((campaign) => campaign.id === order.campaignId), "FORBIDDEN", "You can only update your own merchant orders", 403);
    return { data: await store.updateOrderStatus(order.id, input.status) };
  });

  app.post("/v1/admin/orders/:id/refund", { preHandler: requireRole("ADMIN") }, async (request) => {
    const params = request.params as { id: string };
    const order = assertFound(await store.getOrder(params.id), "Order not found");
    return { data: await store.updateOrderStatus(order.id, "REFUNDED") };
  });
}

async function assertOrderVisible(
  store: Store,
  user: { id: string; role: string; merchantId?: string },
  orderId: string
): Promise<void> {
  if (user.role === "ADMIN") return;
  const orders = await store.listOrders(
    user.role === "CONSUMER"
      ? { consumerId: user.id }
      : user.role === "RIDER"
        ? { riderId: user.id }
        : { merchantId: user.merchantId }
  );
  assert(orders.some((order) => order.id === orderId), "FORBIDDEN", "You do not have access to this order", 403);
}
