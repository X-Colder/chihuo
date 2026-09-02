import { test, before, after } from "node:test";
import assert from "node:assert/strict";
import type { FastifyInstance } from "fastify";
import { buildApp } from "../src/app.js";
import { loadConfig } from "../src/config.js";
import { MemoryStore } from "../src/db/memory.js";

type LoginResult = {
  token: string;
  user: { id: string; role: string; merchantId?: string };
};

const config = loadConfig({ NODE_ENV: "test", JWT_SECRET: "test-secret", HOST: "127.0.0.1", PORT: "0" });
const store = new MemoryStore();
let app: FastifyInstance;

async function login(name: string, role: "CONSUMER" | "MERCHANT" | "RIDER" | "ADMIN", merchantName?: string): Promise<LoginResult> {
  const response = await app.inject({
    method: "POST",
    url: "/v1/auth/demo-login",
    payload: { name, role, ...(merchantName ? { merchantName } : {}) }
  });
  assert.equal(response.statusCode, 200);
  return response.json<{ data: LoginResult }>().data;
}

function auth(token: string): { authorization: string } {
  return { authorization: `Bearer ${token}` };
}

before(async () => {
  app = await buildApp({ config, store });
  await app.ready();
});

after(async () => {
  await app.close();
});

test("health and structured validation errors", async () => {
  const live = await app.inject({ method: "GET", url: "/health/live" });
  assert.equal(live.statusCode, 200);
  assert.equal(live.json().status, "ok");

  const invalid = await app.inject({
    method: "POST",
    url: "/v1/auth/demo-login",
    payload: { role: "INVALID" }
  });
  assert.equal(invalid.statusCode, 422);
  assert.equal(invalid.json().error.code, "VALIDATION_ERROR");
  assert.ok(invalid.json().error.requestId);
});

test("consumer demand aggregation through merchant offer and campaign", async () => {
  const consumer = await login("测试消费者一", "CONSUMER");
  const secondConsumer = await login("测试消费者二", "CONSUMER");
  const admin = await login("测试平台管理员", "ADMIN");
  const merchant = await login("测试商家", "MERCHANT", "测试便当店");

  const merchants = await app.inject({
    method: "GET",
    url: "/v1/admin/merchants",
    headers: auth(admin.token)
  });
  assert.equal(merchants.statusCode, 200);
  const merchantId = merchants.json().data.find((item: { ownerUserId: string }) => item.ownerUserId === merchant.user.id).id;

  const merchantReview = await app.inject({
    method: "PATCH",
    url: `/v1/admin/merchants/${merchantId}/review`,
    headers: auth(admin.token),
    payload: { status: "APPROVED" }
  });
  assert.equal(merchantReview.statusCode, 200);

  const demandPayload = {
    title: "低油定量午餐",
    category: "便当",
    serviceArea: "测试园区",
    servingDate: "2026-09-20",
    servingTime: "12:00",
    budgetMinCents: 2000,
    budgetMaxCents: 2800,
    quantity: 1,
    weightMinGrams: 300,
    weightMaxGrams: 400,
    hardConstraints: ["不含花生"],
    preferences: ["少油"],
    minimumMembers: 2,
    maximumMembers: 20
  };
  const created = await app.inject({
    method: "POST",
    url: "/v1/demands",
    headers: auth(consumer.token),
    payload: demandPayload
  });
  assert.equal(created.statusCode, 201);
  const demandId = created.json().data.demand.id;
  assert.equal(created.json().data.demand.memberCount, 1);
  assert.equal(created.json().data.demand.status, "PENDING_REVIEW");

  const demandReview = await app.inject({
    method: "PATCH",
    url: `/v1/admin/demands/${demandId}/review`,
    headers: auth(admin.token),
    payload: { status: "OPEN" }
  });
  assert.equal(demandReview.statusCode, 200);

  const joined = await app.inject({
    method: "POST",
    url: "/v1/demands",
    headers: auth(secondConsumer.token),
    payload: demandPayload
  });
  assert.equal(joined.statusCode, 200);
  assert.equal(joined.json().data.matched, true);
  assert.equal(joined.json().data.demand.memberCount, 2);
  assert.equal(joined.json().data.demand.status, "READY");

  const offer = await app.inject({
    method: "POST",
    url: "/v1/merchant/offers",
    headers: auth(merchant.token),
    payload: {
      demandId,
      unitPriceCents: 2400,
      productionCapacity: 20,
      weightGrams: 350,
      ingredients: ["鸡胸肉", "西兰花", "糙米"],
      allergens: ["无"],
      oilLevel: "LOW",
      saltLevel: "LOW",
      productionTime: "11:30",
      shelfLifeMinutes: 180,
      storageInstructions: "建议两小时内食用"
    }
  });
  assert.equal(offer.statusCode, 201);
  const offerId = offer.json().data.id;

  const campaign = await app.inject({
    method: "POST",
    url: "/v1/merchant/campaigns",
    headers: auth(merchant.token),
    payload: {
      demandId,
      offerId,
      title: "低油定量鸡肉便当",
      unitPriceCents: 2400,
      deliveryFeeCents: 500,
      platformFeeBps: 500,
      minimumOrders: 1,
      maximumOrders: 20,
      startsAt: "2026-09-18T00:00:00+08:00",
      endsAt: "2026-09-19T23:59:59+08:00",
      pickupPoint: "测试便当店",
      foodSpec: {
        weightGrams: 350,
        ingredients: ["鸡胸肉", "西兰花", "糙米"],
        allergens: ["无"],
        oilLevel: "LOW",
        saltLevel: "LOW",
        productionTime: "11:30",
        shelfLifeMinutes: 180,
        storageInstructions: "建议两小时内食用"
      }
    }
  });
  assert.equal(campaign.statusCode, 201);
  const campaignId = campaign.json().data.id;

  const campaignReview = await app.inject({
    method: "PATCH",
    url: `/v1/admin/campaigns/${campaignId}/review`,
    headers: auth(admin.token),
    payload: { status: "OPEN" }
  });
  assert.equal(campaignReview.statusCode, 200);

  const campaigns = await app.inject({
    method: "GET",
    url: "/v1/campaigns",
    headers: auth(consumer.token)
  });
  assert.equal(campaigns.statusCode, 200);
  assert.ok(campaigns.json().data.some((item: { id: string }) => item.id === campaignId));
});

test("order payment creates rider task and food safety records", async () => {
  const consumer = await login("订单消费者", "CONSUMER");
  const admin = await login("订单管理员", "ADMIN");
  const merchant = await login("订单商家", "MERCHANT", "订单餐厅");
  const rider = await login("订单骑手", "RIDER");

  const merchantRows = await app.inject({
    method: "GET",
    url: "/v1/admin/merchants",
    headers: auth(admin.token)
  });
  const merchantId = merchantRows.json().data.find((item: { ownerUserId: string }) => item.ownerUserId === merchant.user.id).id;
  await app.inject({
    method: "PATCH",
    url: `/v1/admin/merchants/${merchantId}/review`,
    headers: auth(admin.token),
    payload: { status: "APPROVED" }
  });

  const demand = await app.inject({
    method: "POST",
    url: "/v1/demands",
    headers: auth(consumer.token),
    payload: {
      title: "订单午餐",
      category: "便当",
      serviceArea: "订单园区",
      servingDate: "2026-09-21",
      servingTime: "12:00",
      budgetMinCents: 2000,
      budgetMaxCents: 3000,
      quantity: 1,
      weightMinGrams: 300,
      weightMaxGrams: 400,
      hardConstraints: [],
      preferences: [],
      minimumMembers: 1,
      maximumMembers: 10
    }
  });
  const demandId = demand.json().data.demand.id;
  await app.inject({
    method: "PATCH",
    url: `/v1/admin/demands/${demandId}/review`,
    headers: auth(admin.token),
    payload: { status: "OPEN" }
  });
  const offer = await app.inject({
    method: "POST",
    url: "/v1/merchant/offers",
    headers: auth(merchant.token),
    payload: {
      demandId,
      unitPriceCents: 2000,
      productionCapacity: 10,
      weightGrams: 350,
      ingredients: ["米饭", "鸡肉"],
      allergens: [],
      productionTime: "11:30",
      shelfLifeMinutes: 120,
      storageInstructions: "两小时内食用"
    }
  });
  const campaign = await app.inject({
    method: "POST",
    url: "/v1/merchant/campaigns",
    headers: auth(merchant.token),
    payload: {
      demandId,
      offerId: offer.json().data.id,
      title: "订单鸡肉饭",
      unitPriceCents: 2000,
      minimumOrders: 1,
      maximumOrders: 10,
      startsAt: "2026-09-20T00:00:00+08:00",
      endsAt: "2026-09-21T23:59:59+08:00",
      pickupPoint: "订单餐厅",
      foodSpec: {
        weightGrams: 350,
        ingredients: ["米饭", "鸡肉"],
        allergens: [],
        oilLevel: "UNKNOWN",
        saltLevel: "UNKNOWN",
        productionTime: "11:30",
        shelfLifeMinutes: 120,
        storageInstructions: "两小时内食用"
      }
    }
  });
  const campaignId = campaign.json().data.id;
  await app.inject({
    method: "PATCH",
    url: `/v1/admin/campaigns/${campaignId}/review`,
    headers: auth(admin.token),
    payload: { status: "OPEN" }
  });

  const order = await app.inject({
    method: "POST",
    url: `/v1/campaigns/${campaignId}/orders`,
    headers: auth(consumer.token),
    payload: {
      quantity: 1,
      deliveryAddress: "订单园区 A 栋",
      contactName: "订单消费者",
      contactPhone: "13800000000"
    }
  });
  assert.equal(order.statusCode, 201);
  const orderId = order.json().data.id;
  assert.equal(order.json().data.totalCents, 2600);

  const paid = await app.inject({
    method: "POST",
    url: `/v1/orders/${orderId}/pay`,
    headers: auth(consumer.token)
  });
  assert.equal(paid.statusCode, 200);

  for (const status of ["ACCEPTED", "PREPARING", "READY_FOR_PICKUP"] as const) {
    const response = await app.inject({
      method: "PATCH",
      url: `/v1/merchant/orders/${orderId}/status`,
      headers: auth(merchant.token),
      payload: { status }
    });
    assert.equal(response.statusCode, 200);
  }

  const queue = await app.inject({
    method: "GET",
    url: "/v1/rider/tasks/queue",
    headers: auth(rider.token)
  });
  assert.equal(queue.statusCode, 200);
  const taskId = queue.json().data.find((item: { orderId: string }) => item.orderId === orderId).id;
  assert.equal((await app.inject({
    method: "POST",
    url: `/v1/rider/tasks/${taskId}/claim`,
    headers: auth(rider.token)
  })).statusCode, 200);

  for (const status of ["PICKED_UP", "DELIVERING", "COMPLETED"] as const) {
    const response = await app.inject({
      method: "PATCH",
      url: `/v1/rider/tasks/${taskId}`,
      headers: auth(rider.token),
      payload: { status }
    });
    assert.equal(response.statusCode, 200);
  }

  const delivered = await app.inject({
    method: "GET",
    url: `/v1/orders/${orderId}`,
    headers: auth(consumer.token)
  });
  assert.equal(delivered.statusCode, 200);
  assert.equal(delivered.json().data.status, "DELIVERED");

  const incident = await app.inject({
    method: "POST",
    url: "/v1/food-safety/incidents",
    headers: auth(merchant.token),
    payload: {
      merchantId,
      orderId,
      campaignId,
      severity: "LOW",
      title: "演示食品安全记录",
      description: "记录一次待核查的消费者反馈",
      evidenceUrls: []
    }
  });
  assert.equal(incident.statusCode, 201);
  const incidents = await app.inject({
    method: "GET",
    url: "/v1/food-safety/incidents",
    headers: auth(admin.token)
  });
  assert.equal(incidents.statusCode, 200);
  assert.ok(incidents.json().data.some((item: { id: string }) => item.id === incident.json().data.id));
});
