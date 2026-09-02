import { Pool, type PoolClient } from "pg";
import type {
  Campaign,
  DemandCluster,
  DemandMember,
  FoodSafetyIncident,
  Merchant,
  Offer,
  Order,
  RiderTask,
  User
} from "@chihuo/contracts";
import { ApiError, assertFound } from "../errors.js";
import { newId, normalizeList, nowIso } from "../utils.js";
import type {
  CreateCampaignRecord,
  CreateDemandRecord,
  CreateIncidentRecord,
  CreateMemberRecord,
  CreateMerchant,
  CreateOfferRecord,
  CreateOrderRecord,
  CreateUser,
  Store
} from "./types.js";

type DbRow = Record<string, any>;

function timestamp(value: unknown): string {
  return new Date(value as string | Date).toISOString();
}

function json<T>(value: unknown, fallback: T): T {
  return value === null || value === undefined ? fallback : (value as T);
}

function sameStringSet(left: string[], right: string[]): boolean {
  const a = normalizeList(left);
  const b = normalizeList(right);
  return a.length === b.length && a.every((item, index) => item === b[index]);
}

function rangesOverlap(aMin: number, aMax: number, bMin: number, bMax: number): boolean {
  return aMin <= bMax && bMin <= aMax;
}

function assertOrderTransition(current: Order["status"], next: Order["status"]): void {
  const allowed: Record<Order["status"], Order["status"][]> = {
    PENDING_PAYMENT: ["PAID", "CANCELLED"],
    PAID: ["ACCEPTED", "CANCELLED", "REFUNDED"],
    ACCEPTED: ["PREPARING", "CANCELLED", "REFUNDED"],
    PREPARING: ["READY_FOR_PICKUP", "CANCELLED", "REFUNDED"],
    READY_FOR_PICKUP: ["PICKED_UP", "CANCELLED", "REFUNDED"],
    PICKED_UP: ["DELIVERING", "CANCELLED"],
    DELIVERING: ["DELIVERED", "CANCELLED"],
    DELIVERED: [],
    CANCELLED: [],
    REFUNDED: []
  };
  if (current !== next && !allowed[current].includes(next)) {
    throw new ApiError("INVALID_STATUS_TRANSITION", `Cannot change order from ${current} to ${next}`, 409);
  }
}

function mapUser(row: DbRow): User {
  return { id: row.id, name: row.name, role: row.role, createdAt: timestamp(row.created_at) };
}

function mapMerchant(row: DbRow): Merchant {
  return {
    id: row.id,
    ownerUserId: row.owner_user_id,
    name: row.name,
    status: row.status,
    license: json(row.license, {}),
    createdAt: timestamp(row.created_at)
  };
}

function mapDemand(row: DbRow): DemandCluster {
  return {
    id: row.id,
    createdBy: row.created_by,
    title: row.title,
    category: row.category,
    serviceArea: row.service_area,
    servingDate: row.serving_date instanceof Date ? row.serving_date.toISOString().slice(0, 10) : String(row.serving_date),
    servingTime: row.serving_time,
    budgetMinCents: row.budget_min_cents,
    budgetMaxCents: row.budget_max_cents,
    quantity: row.quantity,
    weightMinGrams: row.weight_min_grams,
    weightMaxGrams: row.weight_max_grams,
    hardConstraints: json(row.hard_constraints, []),
    preferences: json(row.preferences, []),
    notes: row.notes ?? undefined,
    minimumMembers: row.minimum_members,
    maximumMembers: row.maximum_members,
    memberCount: Number(row.member_count ?? 0),
    status: row.status,
    reviewedBy: row.reviewed_by ?? undefined,
    reviewedAt: row.reviewed_at ? timestamp(row.reviewed_at) : undefined,
    reviewReason: row.review_reason ?? undefined,
    createdAt: timestamp(row.created_at),
    updatedAt: timestamp(row.updated_at)
  };
}

function mapMember(row: DbRow): DemandMember {
  return {
    id: row.id,
    demandId: row.demand_id,
    userId: row.user_id,
    quantity: row.quantity,
    weightGrams: row.weight_grams ?? undefined,
    preferences: json(row.preferences, []),
    notes: row.notes ?? undefined,
    createdAt: timestamp(row.created_at)
  };
}

function mapOffer(row: DbRow): Offer {
  return {
    id: row.id,
    demandId: row.demand_id,
    merchantId: row.merchant_id,
    unitPriceCents: row.unit_price_cents,
    productionCapacity: row.production_capacity,
    weightGrams: row.weight_grams,
    ingredients: json(row.ingredients, []),
    allergens: json(row.allergens, []),
    oilLevel: row.oil_level,
    saltLevel: row.salt_level,
    productionTime: row.production_time,
    shelfLifeMinutes: row.shelf_life_minutes,
    storageInstructions: row.storage_instructions,
    notes: row.notes ?? undefined,
    status: row.status,
    createdAt: timestamp(row.created_at)
  };
}

function mapCampaign(row: DbRow): Campaign {
  return {
    id: row.id,
    demandId: row.demand_id,
    offerId: row.offer_id,
    merchantId: row.merchant_id,
    title: row.title,
    description: row.description ?? undefined,
    unitPriceCents: row.unit_price_cents,
    deliveryFeeCents: row.delivery_fee_cents,
    platformFeeBps: row.platform_fee_bps,
    minimumOrders: row.minimum_orders,
    maximumOrders: row.maximum_orders,
    startsAt: new Date(row.starts_at).toISOString(),
    endsAt: new Date(row.ends_at).toISOString(),
    pickupPoint: row.pickup_point,
    foodSpec: json(row.food_spec, {} as Campaign["foodSpec"]),
    status: row.status,
    currentOrders: Number(row.current_orders ?? 0),
    createdAt: timestamp(row.created_at),
    updatedAt: timestamp(row.updated_at)
  };
}

function mapOrder(row: DbRow): Order {
  return {
    id: row.id,
    campaignId: row.campaign_id,
    consumerId: row.consumer_id,
    quantity: row.quantity,
    deliveryAddress: row.delivery_address,
    contactName: row.contact_name,
    contactPhone: row.contact_phone,
    status: row.status,
    unitPriceCents: row.unit_price_cents,
    subtotalCents: row.subtotal_cents,
    deliveryFeeCents: row.delivery_fee_cents,
    platformFeeCents: row.platform_fee_cents,
    totalCents: row.total_cents,
    createdAt: timestamp(row.created_at),
    updatedAt: timestamp(row.updated_at)
  };
}

function mapTask(row: DbRow): RiderTask {
  return {
    id: row.id,
    orderId: row.order_id,
    riderId: row.rider_id ?? undefined,
    status: row.status,
    pickupPoint: row.pickup_point,
    deliveryAddress: row.delivery_address,
    note: row.note ?? undefined,
    createdAt: timestamp(row.created_at),
    updatedAt: timestamp(row.updated_at)
  };
}

function mapIncident(row: DbRow): FoodSafetyIncident {
  return {
    id: row.id,
    merchantId: row.merchant_id,
    orderId: row.order_id ?? undefined,
    campaignId: row.campaign_id ?? undefined,
    reportedBy: row.reported_by,
    severity: row.severity,
    title: row.title,
    description: row.description,
    evidenceUrls: json(row.evidence_urls, []),
    affectedQuantity: row.affected_quantity ?? undefined,
    status: row.status,
    createdAt: timestamp(row.created_at),
    updatedAt: timestamp(row.updated_at)
  };
}

export class PostgresStore implements Store {
  constructor(private readonly pool: Pool) {}

  async ping(): Promise<boolean> {
    await this.pool.query("SELECT 1");
    return true;
  }

  async close(): Promise<void> {
    await this.pool.end();
  }

  async seed(): Promise<void> {
    const client = await this.pool.connect();
    try {
      await client.query("BEGIN");
      const admin = await this.createOrGetUserWithClient(client, { name: "平台管理员", role: "ADMIN", demoKey: "admin@demo" });
      const consumer = await this.createOrGetUserWithClient(client, { name: "演示消费者", role: "CONSUMER", demoKey: "consumer@demo" });
      await this.createOrGetUserWithClient(client, { name: "演示骑手", role: "RIDER", demoKey: "rider@demo" });
      const merchant = await this.createOrGetMerchantWithClient(client, {
        ownerUserId: admin.id,
        name: "安心便当演示店",
        license: { licenseNo: "DEMO-FOOD-001", foodPermitNo: "DEMO-PERMIT-001", verified: true }
      });
      await client.query("UPDATE merchants SET status = 'APPROVED' WHERE id = $1", [merchant.id]);
      const existingDemand = await client.query("SELECT id FROM demand_clusters WHERE title = $1 LIMIT 1", ["低油定量午餐"]);
      if (existingDemand.rowCount === 0) {
        const demandId = newId();
        const now = nowIso();
        await client.query(
          `INSERT INTO demand_clusters
            (id, created_by, title, category, service_area, serving_date, serving_time, budget_min_cents, budget_max_cents,
             quantity, weight_min_grams, weight_max_grams, hard_constraints, preferences, notes, minimum_members,
             maximum_members, status, created_at, updated_at)
           VALUES ($1, $2, '低油定量午餐', '便当', '演示产业园', '2026-09-15', '12:00', 2000, 3000, 1, 300, 400,
                   $3::jsonb, $4::jsonb, '工作日午餐', 3, 50, 'OPEN', $5, $5)`,
          [demandId, consumer.id, JSON.stringify(["不含花生"]), JSON.stringify(["少油"]), now]
        );
        await client.query(
          `INSERT INTO demand_members (id, demand_id, user_id, quantity, weight_grams, preferences, notes, created_at)
           VALUES ($1, $2, $3, 1, 350, $4::jsonb, '演示数据', $5)`,
          [newId(), demandId, consumer.id, JSON.stringify(["少油"]), now]
        );
        const offerId = newId();
        await client.query(
          `INSERT INTO offers
            (id, demand_id, merchant_id, unit_price_cents, production_capacity, weight_grams, ingredients, allergens,
             oil_level, salt_level, production_time, shelf_life_minutes, storage_instructions, notes, status, created_at)
           VALUES ($1, $2, $3, 2400, 50, 350, $4::jsonb, $5::jsonb, 'LOW', 'LOW', '11:30', 180,
                   '常温避光，建议两小时内食用', '演示报价', 'SUBMITTED', $6)`,
          [offerId, demandId, merchant.id, JSON.stringify(["鸡胸肉", "西兰花", "糙米"]), JSON.stringify(["无"]), now]
        );
        await client.query(
          `INSERT INTO campaigns
            (id, demand_id, offer_id, merchant_id, title, description, unit_price_cents, delivery_fee_cents,
             platform_fee_bps, minimum_orders, maximum_orders, starts_at, ends_at, pickup_point, food_spec,
             status, created_at, updated_at)
           VALUES ($1, $2, $3, $4, '低油定量鸡肉便当', '演示预售商品', 2400, 500, 500, 3, 50,
                   '2026-09-10T00:00:00+08:00', '2026-09-14T23:59:59+08:00', '安心便当演示店', $5::jsonb,
                   'OPEN', $6, $6)`,
          [
            newId(),
            demandId,
            offerId,
            merchant.id,
            JSON.stringify({
              weightGrams: 350,
              ingredients: ["鸡胸肉", "西兰花", "糙米"],
              allergens: ["无"],
              oilLevel: "LOW",
              saltLevel: "LOW",
              productionTime: "11:30",
              shelfLifeMinutes: 180,
              storageInstructions: "常温避光，建议两小时内食用"
            }),
            now
          ]
        );
      }
      await client.query("COMMIT");
    } catch (error) {
      await client.query("ROLLBACK");
      throw error;
    } finally {
      client.release();
    }
  }

  private async createOrGetUserWithClient(client: PoolClient, input: CreateUser): Promise<User> {
    const result = await client.query(
      `INSERT INTO users (id, name, role, demo_key, created_at)
       VALUES ($1, $2, $3, $4, $5)
       ON CONFLICT (demo_key) DO UPDATE SET name = EXCLUDED.name, role = EXCLUDED.role
       RETURNING id, name, role, created_at`,
      [newId(), input.name, input.role, input.demoKey, nowIso()]
    );
    return mapUser(result.rows[0]);
  }

  private async createOrGetMerchantWithClient(client: PoolClient, input: CreateMerchant): Promise<Merchant> {
    const existing = await client.query("SELECT * FROM merchants WHERE owner_user_id = $1 LIMIT 1", [input.ownerUserId]);
    if (existing.rowCount) return mapMerchant(existing.rows[0]);
    const result = await client.query(
      `INSERT INTO merchants (id, owner_user_id, name, status, license, created_at)
       VALUES ($1, $2, $3, 'PENDING', $4::jsonb, $5)
       RETURNING *`,
      [newId(), input.ownerUserId, input.name, JSON.stringify(input.license ?? {}), nowIso()]
    );
    return mapMerchant(result.rows[0]);
  }

  async createOrGetUser(input: CreateUser): Promise<User> {
    const client = await this.pool.connect();
    try {
      return await this.createOrGetUserWithClient(client, input);
    } finally {
      client.release();
    }
  }

  async getUser(id: string): Promise<User | undefined> {
    const result = await this.pool.query("SELECT id, name, role, created_at FROM users WHERE id = $1", [id]);
    return result.rowCount ? mapUser(result.rows[0]) : undefined;
  }

  async createOrGetMerchant(input: CreateMerchant): Promise<Merchant> {
    const client = await this.pool.connect();
    try {
      return await this.createOrGetMerchantWithClient(client, input);
    } finally {
      client.release();
    }
  }

  async getMerchant(id: string): Promise<Merchant | undefined> {
    const result = await this.pool.query("SELECT * FROM merchants WHERE id = $1", [id]);
    return result.rowCount ? mapMerchant(result.rows[0]) : undefined;
  }

  async listMerchants(status?: string): Promise<Merchant[]> {
    const result = await this.pool.query("SELECT * FROM merchants WHERE ($1::text IS NULL OR status = $1) ORDER BY created_at DESC", [status ?? null]);
    return result.rows.map(mapMerchant);
  }

  async reviewMerchant(id: string, status: Merchant["status"], reason?: string): Promise<Merchant> {
    const result = await this.pool.query(
      "UPDATE merchants SET status = $2, license = jsonb_set(license, '{lastReviewReason}', to_jsonb($3::text), true) WHERE id = $1 RETURNING *",
      [id, status, reason ?? ""]
    );
    return mapMerchant(assertFound(result.rows[0], "Merchant not found"));
  }

  async createDemand(input: CreateDemandRecord): Promise<DemandCluster> {
    const id = newId();
    const now = nowIso();
    const result = await this.pool.query(
      `INSERT INTO demand_clusters
       (id, created_by, title, category, service_area, serving_date, serving_time, budget_min_cents, budget_max_cents,
        quantity, weight_min_grams, weight_max_grams, hard_constraints, preferences, notes, minimum_members,
        maximum_members, status, created_at, updated_at)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb, $14::jsonb, $15, $16, $17, 'PENDING_REVIEW', $18, $18)
       RETURNING *, 0::integer AS member_count`,
      [
        id,
        input.createdBy,
        input.spec.title,
        input.spec.category,
        input.spec.serviceArea,
        input.spec.servingDate,
        input.spec.servingTime,
        input.spec.budgetMinCents,
        input.spec.budgetMaxCents,
        input.spec.quantity,
        input.spec.weightMinGrams,
        input.spec.weightMaxGrams,
        JSON.stringify(input.spec.hardConstraints),
        JSON.stringify(input.spec.preferences),
        input.spec.notes ?? null,
        input.minimumMembers,
        input.maximumMembers,
        now
      ]
    );
    return mapDemand(result.rows[0]);
  }

  async findMatchingDemand(input: CreateDemandRecord["spec"]): Promise<DemandCluster | undefined> {
    const candidates = await this.listDemands({ status: "OPEN", limit: 1_000 });
    const match = candidates.find(
      (demand) =>
        demand.category === input.category &&
        demand.serviceArea === input.serviceArea &&
        demand.servingDate === input.servingDate &&
        demand.servingTime === input.servingTime &&
        sameStringSet(demand.hardConstraints, input.hardConstraints) &&
        rangesOverlap(demand.budgetMinCents, demand.budgetMaxCents, input.budgetMinCents, input.budgetMaxCents) &&
        rangesOverlap(demand.weightMinGrams, demand.weightMaxGrams, input.weightMinGrams, input.weightMaxGrams) &&
        demand.memberCount < demand.maximumMembers
    );
    return match;
  }

  async getDemand(id: string): Promise<DemandCluster | undefined> {
    const result = await this.pool.query(
      `SELECT d.*, COUNT(m.id)::integer AS member_count
       FROM demand_clusters d
       LEFT JOIN demand_members m ON m.demand_id = d.id
       WHERE d.id = $1
       GROUP BY d.id`,
      [id]
    );
    return result.rowCount ? mapDemand(result.rows[0]) : undefined;
  }

  async listDemands(filters: { status?: string; limit?: number; offset?: number } = {}): Promise<DemandCluster[]> {
    const result = await this.pool.query(
      `SELECT d.*, COUNT(m.id)::integer AS member_count
       FROM demand_clusters d
       LEFT JOIN demand_members m ON m.demand_id = d.id
       WHERE ($1::text IS NULL OR d.status = $1)
       GROUP BY d.id
       ORDER BY d.created_at DESC
       LIMIT $2 OFFSET $3`,
      [filters.status ?? null, filters.limit ?? 50, filters.offset ?? 0]
    );
    return result.rows.map(mapDemand);
  }

  async reviewDemand(id: string, status: DemandCluster["status"], reviewerId: string, reason?: string): Promise<DemandCluster> {
    const result = await this.pool.query(
      `UPDATE demand_clusters
       SET status = $2, reviewed_by = $3, reviewed_at = $4, review_reason = $5, updated_at = $4
       WHERE id = $1
       RETURNING *, 0::integer AS member_count`,
      [id, status, reviewerId, nowIso(), reason ?? null]
    );
    const demand = assertFound(result.rows[0], "Demand not found");
    return this.getDemand(demand.id).then((value) => assertFound(value, "Demand not found"));
  }

  async addDemandMember(input: CreateMemberRecord): Promise<DemandMember> {
    const client = await this.pool.connect();
    try {
      await client.query("BEGIN");
      const demandResult = await client.query("SELECT * FROM demand_clusters WHERE id = $1 FOR UPDATE", [input.demandId]);
      const demand = assertFound(demandResult.rows[0], "Demand not found");
      const memberCountResult = await client.query("SELECT COUNT(*)::integer AS count FROM demand_members WHERE demand_id = $1", [input.demandId]);
      const memberCount = Number(memberCountResult.rows[0].count);
      if (demand.status === "REJECTED" || demand.status === "CLOSED") throw new ApiError("DEMAND_NOT_OPEN", "Demand is not open for joining", 409);
      if (memberCount >= demand.maximum_members) throw new ApiError("DEMAND_FULL", "Demand has reached its maximum member count", 409);
      const result = await client.query(
        `INSERT INTO demand_members (id, demand_id, user_id, quantity, weight_grams, preferences, notes, created_at)
         VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8)
         RETURNING *`,
        [newId(), input.demandId, input.userId, input.quantity, input.weightGrams ?? null, JSON.stringify(input.preferences), input.notes ?? null, nowIso()]
      );
      if (memberCount + 1 >= demand.minimum_members && demand.status === "OPEN") {
        await client.query("UPDATE demand_clusters SET status = 'READY', updated_at = $2 WHERE id = $1", [input.demandId, nowIso()]);
      }
      await client.query("COMMIT");
      return mapMember(result.rows[0]);
    } catch (error: any) {
      await client.query("ROLLBACK");
      if (error?.code === "23505") throw new ApiError("ALREADY_JOINED", "User has already joined this demand", 409);
      throw error;
    } finally {
      client.release();
    }
  }

  async getDemandMember(demandId: string, userId: string): Promise<DemandMember | undefined> {
    const result = await this.pool.query("SELECT * FROM demand_members WHERE demand_id = $1 AND user_id = $2", [demandId, userId]);
    return result.rowCount ? mapMember(result.rows[0]) : undefined;
  }

  async listDemandMembers(demandId: string): Promise<DemandMember[]> {
    const result = await this.pool.query("SELECT * FROM demand_members WHERE demand_id = $1 ORDER BY created_at", [demandId]);
    return result.rows.map(mapMember);
  }

  async createOffer(merchantId: string, input: CreateOfferRecord): Promise<Offer> {
    const merchant = assertFound(await this.getMerchant(merchantId), "Merchant not found");
    if (merchant.status !== "APPROVED") throw new ApiError("MERCHANT_NOT_APPROVED", "Merchant must be approved before submitting offers", 403);
    assertFound(await this.getDemand(input.demandId), "Demand not found");
    const result = await this.pool.query(
      `INSERT INTO offers
       (id, demand_id, merchant_id, unit_price_cents, production_capacity, weight_grams, ingredients, allergens,
        oil_level, salt_level, production_time, shelf_life_minutes, storage_instructions, notes, status, created_at)
       VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9, $10, $11, $12, $13, $14, 'SUBMITTED', $15)
       RETURNING *`,
      [
        newId(),
        input.demandId,
        merchantId,
        input.unitPriceCents,
        input.productionCapacity,
        input.weightGrams,
        JSON.stringify(input.ingredients),
        JSON.stringify(input.allergens),
        input.oilLevel,
        input.saltLevel,
        input.productionTime,
        input.shelfLifeMinutes,
        input.storageInstructions,
        input.notes ?? null,
        nowIso()
      ]
    );
    return mapOffer(result.rows[0]);
  }

  async getOffer(id: string): Promise<Offer | undefined> {
    const result = await this.pool.query("SELECT * FROM offers WHERE id = $1", [id]);
    return result.rowCount ? mapOffer(result.rows[0]) : undefined;
  }

  async listOffers(filters: { demandId?: string; merchantId?: string; status?: string } = {}): Promise<Offer[]> {
    const result = await this.pool.query(
      `SELECT * FROM offers
       WHERE ($1::uuid IS NULL OR demand_id = $1)
         AND ($2::uuid IS NULL OR merchant_id = $2)
         AND ($3::text IS NULL OR status = $3)
       ORDER BY created_at DESC`,
      [filters.demandId ?? null, filters.merchantId ?? null, filters.status ?? null]
    );
    return result.rows.map(mapOffer);
  }

  async createCampaign(merchantId: string, input: CreateCampaignRecord): Promise<Campaign> {
    const merchant = assertFound(await this.getMerchant(merchantId), "Merchant not found");
    if (merchant.status !== "APPROVED") throw new ApiError("MERCHANT_NOT_APPROVED", "Merchant must be approved before creating campaigns", 403);
    const offer = assertFound(await this.getOffer(input.offerId), "Offer not found");
    if (offer.merchantId !== merchantId || offer.demandId !== input.demandId) throw new ApiError("OFFER_MISMATCH", "Offer does not belong to this merchant and demand", 409);
    const result = await this.pool.query(
      `INSERT INTO campaigns
       (id, demand_id, offer_id, merchant_id, title, description, unit_price_cents, delivery_fee_cents, platform_fee_bps,
        minimum_orders, maximum_orders, starts_at, ends_at, pickup_point, food_spec, status, created_at, updated_at)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, 'PENDING_REVIEW', $16, $16)
       RETURNING *, 0::integer AS current_orders`,
      [
        newId(),
        input.demandId,
        input.offerId,
        merchantId,
        input.title,
        input.description ?? null,
        input.unitPriceCents,
        input.deliveryFeeCents,
        input.platformFeeBps,
        input.minimumOrders,
        input.maximumOrders,
        input.startsAt,
        input.endsAt,
        input.pickupPoint,
        JSON.stringify(input.foodSpec),
        nowIso()
      ]
    );
    return mapCampaign(result.rows[0]);
  }

  async getCampaign(id: string): Promise<Campaign | undefined> {
    const result = await this.pool.query(
      `SELECT c.*, COALESCE(SUM(CASE WHEN o.status NOT IN ('CANCELLED', 'REFUNDED') THEN o.quantity ELSE 0 END), 0)::integer AS current_orders
       FROM campaigns c
       LEFT JOIN orders o ON o.campaign_id = c.id
       WHERE c.id = $1
       GROUP BY c.id`,
      [id]
    );
    return result.rowCount ? mapCampaign(result.rows[0]) : undefined;
  }

  async listCampaigns(filters: { status?: string; merchantId?: string; demandId?: string } = {}): Promise<Campaign[]> {
    const result = await this.pool.query(
      `SELECT c.*, COALESCE(SUM(CASE WHEN o.status NOT IN ('CANCELLED', 'REFUNDED') THEN o.quantity ELSE 0 END), 0)::integer AS current_orders
       FROM campaigns c
       LEFT JOIN orders o ON o.campaign_id = c.id
       WHERE ($1::text IS NULL OR c.status = $1)
         AND ($2::uuid IS NULL OR c.merchant_id = $2)
         AND ($3::uuid IS NULL OR c.demand_id = $3)
       GROUP BY c.id
       ORDER BY c.created_at DESC`,
      [filters.status ?? null, filters.merchantId ?? null, filters.demandId ?? null]
    );
    return result.rows.map(mapCampaign);
  }

  async reviewCampaign(id: string, status: Campaign["status"], _reason?: string): Promise<Campaign> {
    const result = await this.pool.query(
      "UPDATE campaigns SET status = $2, updated_at = $3 WHERE id = $1 RETURNING *",
      [id, status, nowIso()]
    );
    if (!result.rowCount) throw new ApiError("NOT_FOUND", "Campaign not found", 404);
    return assertFound(await this.getCampaign(id), "Campaign not found");
  }

  async incrementCampaignOrders(id: string, quantity: number): Promise<Campaign> {
    const client = await this.pool.connect();
    try {
      await client.query("BEGIN");
      const campaignResult = await client.query(
        `SELECT c.*,
                COALESCE((SELECT SUM(o.quantity) FROM orders o
                          WHERE o.campaign_id = c.id
                            AND o.status NOT IN ('CANCELLED', 'REFUNDED')), 0)::integer AS current_orders
         FROM campaigns c
         WHERE c.id = $1
         FOR UPDATE`,
        [id]
      );
      const campaign = assertFound(campaignResult.rows[0], "Campaign not found");
      if (Number(campaign.current_orders) + quantity > campaign.maximum_orders) throw new ApiError("CAMPAIGN_SOLD_OUT", "Campaign does not have enough remaining capacity", 409);
      if (Number(campaign.current_orders) + quantity >= campaign.minimum_orders && campaign.status === "PENDING_REVIEW") {
        await client.query("UPDATE campaigns SET status = 'OPEN', updated_at = $2 WHERE id = $1", [id, nowIso()]);
      }
      await client.query("COMMIT");
      return assertFound(await this.getCampaign(id), "Campaign not found");
    } catch (error) {
      await client.query("ROLLBACK");
      throw error;
    } finally {
      client.release();
    }
  }

  async createOrder(consumerId: string, input: CreateOrderRecord): Promise<Order> {
    const client = await this.pool.connect();
    try {
      await client.query("BEGIN");
      const campaignResult = await client.query(
        `SELECT c.*,
                COALESCE((SELECT SUM(o.quantity) FROM orders o
                          WHERE o.campaign_id = c.id
                            AND o.status NOT IN ('CANCELLED', 'REFUNDED')), 0)::integer AS current_orders
         FROM campaigns c
         WHERE c.id = $1
         FOR UPDATE`,
        [input.campaignId]
      );
      const campaign = assertFound(campaignResult.rows[0], "Campaign not found");
      if (campaign.status !== "OPEN") throw new ApiError("CAMPAIGN_NOT_OPEN", "Campaign is not open for orders", 409);
      if (Number(campaign.current_orders) + input.quantity > campaign.maximum_orders) throw new ApiError("CAMPAIGN_SOLD_OUT", "Campaign does not have enough remaining capacity", 409);
      const subtotalCents = campaign.unit_price_cents * input.quantity;
      const platformFeeCents = Math.ceil(subtotalCents * campaign.platform_fee_bps / 10_000);
      const totalCents = subtotalCents + campaign.delivery_fee_cents + platformFeeCents;
      const id = newId();
      const now = nowIso();
      const result = await client.query(
        `INSERT INTO orders
         (id, campaign_id, consumer_id, quantity, delivery_address, contact_name, contact_phone, status, unit_price_cents,
          subtotal_cents, delivery_fee_cents, platform_fee_cents, total_cents, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, 'PENDING_PAYMENT', $8, $9, $10, $11, $12, $13, $13)
         RETURNING *`,
        [
          id,
          input.campaignId,
          consumerId,
          input.quantity,
          input.deliveryAddress,
          input.contactName,
          input.contactPhone,
          campaign.unit_price_cents,
          subtotalCents,
          campaign.delivery_fee_cents,
          platformFeeCents,
          totalCents,
          now
        ]
      );
      if (Number(campaign.current_orders) + input.quantity >= campaign.maximum_orders) {
        await client.query("UPDATE campaigns SET status = 'SOLD_OUT', updated_at = $2 WHERE id = $1", [input.campaignId, now]);
      }
      await client.query("COMMIT");
      return mapOrder(result.rows[0]);
    } catch (error) {
      await client.query("ROLLBACK");
      throw error;
    } finally {
      client.release();
    }
  }

  async getOrder(id: string): Promise<Order | undefined> {
    const result = await this.pool.query("SELECT * FROM orders WHERE id = $1", [id]);
    return result.rowCount ? mapOrder(result.rows[0]) : undefined;
  }

  async listOrders(filters: { consumerId?: string; merchantId?: string; riderId?: string; status?: string }): Promise<Order[]> {
    const result = await this.pool.query(
      `SELECT DISTINCT o.*
       FROM orders o
       JOIN campaigns c ON c.id = o.campaign_id
       LEFT JOIN rider_tasks rt ON rt.order_id = o.id
       WHERE ($1::uuid IS NULL OR o.consumer_id = $1)
         AND ($2::uuid IS NULL OR c.merchant_id = $2)
         AND ($3::uuid IS NULL OR rt.rider_id = $3)
         AND ($4::text IS NULL OR o.status = $4)
       ORDER BY o.created_at DESC`,
      [filters.consumerId ?? null, filters.merchantId ?? null, filters.riderId ?? null, filters.status ?? null]
    );
    return result.rows.map(mapOrder);
  }

  async updateOrderStatus(id: string, status: Order["status"]): Promise<Order> {
    const client = await this.pool.connect();
    try {
      await client.query("BEGIN");
      const currentResult = await client.query("SELECT * FROM orders WHERE id = $1 FOR UPDATE", [id]);
      const current = assertFound(currentResult.rows[0], "Order not found");
      assertOrderTransition(current.status, status);
      const now = nowIso();
      const result = await client.query("UPDATE orders SET status = $2, updated_at = $3 WHERE id = $1 RETURNING *", [id, status, now]);
      if (status === "PAID") {
        await client.query(
          `INSERT INTO rider_tasks (id, order_id, status, pickup_point, delivery_address, created_at, updated_at)
           SELECT $1, o.id, 'UNASSIGNED', c.pickup_point, o.delivery_address, $2, $2
           FROM orders o JOIN campaigns c ON c.id = o.campaign_id
           WHERE o.id = $3
           ON CONFLICT (order_id) DO NOTHING`,
          [newId(), now, id]
        );
      }
      await client.query("COMMIT");
      return mapOrder(result.rows[0]);
    } catch (error) {
      await client.query("ROLLBACK");
      throw error;
    } finally {
      client.release();
    }
  }

  async getRiderTask(id: string): Promise<RiderTask | undefined> {
    const result = await this.pool.query("SELECT * FROM rider_tasks WHERE id = $1", [id]);
    return result.rowCount ? mapTask(result.rows[0]) : undefined;
  }

  async listRiderTasks(filters: { riderId?: string; status?: string } = {}): Promise<RiderTask[]> {
    const result = await this.pool.query(
      "SELECT * FROM rider_tasks WHERE ($1::uuid IS NULL OR rider_id = $1) AND ($2::text IS NULL OR status = $2) ORDER BY created_at DESC",
      [filters.riderId ?? null, filters.status ?? null]
    );
    return result.rows.map(mapTask);
  }

  async claimRiderTask(id: string, riderId: string): Promise<RiderTask> {
    const client = await this.pool.connect();
    try {
      await client.query("BEGIN");
      const currentResult = await client.query("SELECT * FROM rider_tasks WHERE id = $1 FOR UPDATE", [id]);
      const current = assertFound(currentResult.rows[0], "Rider task not found");
      if (current.rider_id && current.rider_id !== riderId) throw new ApiError("TASK_ALREADY_ASSIGNED", "Rider task is assigned to another rider", 409);
      if (!["UNASSIGNED", "ASSIGNED"].includes(current.status)) throw new ApiError("TASK_NOT_CLAIMABLE", "Rider task is not claimable", 409);
      const result = await client.query(
        "UPDATE rider_tasks SET rider_id = $2, status = 'ASSIGNED', updated_at = $3 WHERE id = $1 RETURNING *",
        [id, riderId, nowIso()]
      );
      await client.query("COMMIT");
      return mapTask(result.rows[0]);
    } catch (error) {
      await client.query("ROLLBACK");
      throw error;
    } finally {
      client.release();
    }
  }

  async updateRiderTask(id: string, riderId: string, status: RiderTask["status"], note?: string): Promise<RiderTask> {
    const client = await this.pool.connect();
    try {
      await client.query("BEGIN");
      const currentResult = await client.query("SELECT * FROM rider_tasks WHERE id = $1 FOR UPDATE", [id]);
      const current = assertFound(currentResult.rows[0], "Rider task not found");
      if (current.rider_id !== riderId) throw new ApiError("TASK_NOT_ASSIGNED", "Task is not assigned to this rider", 403);
      const result = await client.query(
        "UPDATE rider_tasks SET status = $2, note = $3, updated_at = $4 WHERE id = $1 RETURNING *",
        [id, status, note ?? null, nowIso()]
      );
      const nextOrderStatus: Record<RiderTask["status"], Order["status"] | undefined> = {
        UNASSIGNED: undefined,
        ASSIGNED: undefined,
        PICKED_UP: "PICKED_UP",
        DELIVERING: "DELIVERING",
        COMPLETED: "DELIVERED",
        CANCELLED: "CANCELLED"
      };
      const next = nextOrderStatus[status];
      if (next) {
        const orderResult = await client.query("SELECT * FROM orders WHERE id = $1 FOR UPDATE", [current.order_id]);
        const order = assertFound(orderResult.rows[0], "Order not found");
        if (order.status !== next) {
          assertOrderTransition(order.status, next);
          await client.query("UPDATE orders SET status = $2, updated_at = $3 WHERE id = $1", [current.order_id, next, nowIso()]);
        }
      }
      await client.query("COMMIT");
      return mapTask(result.rows[0]);
    } catch (error) {
      await client.query("ROLLBACK");
      throw error;
    } finally {
      client.release();
    }
  }

  async createIncident(reportedBy: string, input: CreateIncidentRecord): Promise<FoodSafetyIncident> {
    const result = await this.pool.query(
      `INSERT INTO food_safety_incidents
       (id, merchant_id, order_id, campaign_id, reported_by, severity, title, description, evidence_urls,
        affected_quantity, status, created_at, updated_at)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, 'OPEN', $11, $11)
       RETURNING *`,
      [
        newId(),
        input.merchantId,
        input.orderId ?? null,
        input.campaignId ?? null,
        reportedBy,
        input.severity,
        input.title,
        input.description,
        JSON.stringify(input.evidenceUrls),
        input.affectedQuantity ?? null,
        nowIso()
      ]
    );
    return mapIncident(result.rows[0]);
  }

  async listIncidents(filters: { merchantId?: string; status?: string } = {}): Promise<FoodSafetyIncident[]> {
    const result = await this.pool.query(
      `SELECT * FROM food_safety_incidents
       WHERE ($1::uuid IS NULL OR merchant_id = $1)
         AND ($2::text IS NULL OR status = $2)
       ORDER BY created_at DESC`,
      [filters.merchantId ?? null, filters.status ?? null]
    );
    return result.rows.map(mapIncident);
  }
}
