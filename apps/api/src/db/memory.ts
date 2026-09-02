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

function clone<T>(value: T): T {
  return structuredClone(value);
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

export class MemoryStore implements Store {
  private readonly users = new Map<string, User & { demoKey: string }>();
  private readonly merchants = new Map<string, Merchant>();
  private readonly demands = new Map<string, DemandCluster>();
  private readonly members = new Map<string, DemandMember>();
  private readonly offers = new Map<string, Offer>();
  private readonly campaigns = new Map<string, Campaign>();
  private readonly orders = new Map<string, Order>();
  private readonly tasks = new Map<string, RiderTask>();
  private readonly incidents = new Map<string, FoodSafetyIncident>();

  async ping(): Promise<boolean> {
    return true;
  }

  async close(): Promise<void> {}

  async seed(): Promise<void> {
    const admin = await this.createOrGetUser({ name: "平台管理员", role: "ADMIN", demoKey: "admin@demo" });
    const consumer = await this.createOrGetUser({ name: "演示消费者", role: "CONSUMER", demoKey: "consumer@demo" });
    await this.createOrGetUser({ name: "演示骑手", role: "RIDER", demoKey: "rider@demo" });
    const merchant = await this.createOrGetMerchant({
      ownerUserId: admin.id,
      name: "安心便当演示店",
      license: { licenseNo: "DEMO-FOOD-001", foodPermitNo: "DEMO-PERMIT-001", verified: true }
    });
    await this.reviewMerchant(merchant.id, "APPROVED", "seed");
    const demand = await this.createDemand({
      createdBy: consumer.id,
      minimumMembers: 3,
      maximumMembers: 50,
      spec: {
        title: "低油定量午餐",
        category: "便当",
        serviceArea: "演示产业园",
        servingDate: "2026-09-15",
        servingTime: "12:00",
        budgetMinCents: 2000,
        budgetMaxCents: 3000,
        quantity: 1,
        weightMinGrams: 300,
        weightMaxGrams: 400,
        hardConstraints: ["不含花生"],
        preferences: ["少油"],
        notes: "工作日午餐"
      }
    });
    await this.addDemandMember({
      demandId: demand.id,
      userId: consumer.id,
      quantity: 1,
      weightGrams: 350,
      preferences: ["少油"],
      notes: "演示数据"
    });
    const offer = await this.createOffer(merchant.id, {
      demandId: demand.id,
      unitPriceCents: 2400,
      productionCapacity: 50,
      weightGrams: 350,
      ingredients: ["鸡胸肉", "西兰花", "糙米"],
      allergens: ["无"],
      oilLevel: "LOW",
      saltLevel: "LOW",
      productionTime: "11:30",
      shelfLifeMinutes: 180,
      storageInstructions: "常温避光，建议两小时内食用",
      notes: "演示报价"
    });
    await this.createCampaign(merchant.id, {
      demandId: demand.id,
      offerId: offer.id,
      title: "低油定量鸡肉便当",
      description: "演示预售商品",
      unitPriceCents: 2400,
      deliveryFeeCents: 500,
      platformFeeBps: 500,
      minimumOrders: 3,
      maximumOrders: 50,
      startsAt: "2026-09-10T00:00:00+08:00",
      endsAt: "2026-09-14T23:59:59+08:00",
      pickupPoint: "安心便当演示店",
      foodSpec: {
        weightGrams: 350,
        ingredients: ["鸡胸肉", "西兰花", "糙米"],
        allergens: ["无"],
        oilLevel: "LOW",
        saltLevel: "LOW",
        productionTime: "11:30",
        shelfLifeMinutes: 180,
        storageInstructions: "常温避光，建议两小时内食用"
      }
    });
    await this.reviewDemand(demand.id, "OPEN", admin.id, "seed");
  }

  async createOrGetUser(input: CreateUser): Promise<User> {
    const existing = [...this.users.values()].find((user) => user.demoKey === input.demoKey && user.role === input.role);
    if (existing) return clone(existing);
    const user: User & { demoKey: string } = {
      id: newId(),
      name: input.name,
      role: input.role,
      createdAt: nowIso(),
      demoKey: input.demoKey
    };
    this.users.set(user.id, user);
    return clone(user);
  }

  async getUser(id: string): Promise<User | undefined> {
    const value = this.users.get(id);
    return value ? clone(value) : undefined;
  }

  async createOrGetMerchant(input: CreateMerchant): Promise<Merchant> {
    const existing = [...this.merchants.values()].find((merchant) => merchant.ownerUserId === input.ownerUserId);
    if (existing) return clone(existing);
    const merchant: Merchant = {
      id: newId(),
      ownerUserId: input.ownerUserId,
      name: input.name,
      status: "PENDING",
      license: input.license ?? {},
      createdAt: nowIso()
    };
    this.merchants.set(merchant.id, merchant);
    return clone(merchant);
  }

  async getMerchant(id: string): Promise<Merchant | undefined> {
    const value = this.merchants.get(id);
    return value ? clone(value) : undefined;
  }

  async listMerchants(status?: string): Promise<Merchant[]> {
    return clone([...this.merchants.values()].filter((merchant) => !status || merchant.status === status));
  }

  async reviewMerchant(id: string, status: Merchant["status"], _reason?: string): Promise<Merchant> {
    const merchant = assertFound(this.merchants.get(id), "Merchant not found");
    merchant.status = status;
    this.merchants.set(id, merchant);
    return clone(merchant);
  }

  async createDemand(input: CreateDemandRecord): Promise<DemandCluster> {
    const timestamp = nowIso();
    const demand: DemandCluster = {
      ...clone(input.spec),
      id: newId(),
      status: "PENDING_REVIEW",
      minimumMembers: input.minimumMembers,
      maximumMembers: input.maximumMembers,
      memberCount: 0,
      createdBy: input.createdBy,
      createdAt: timestamp,
      updatedAt: timestamp
    };
    this.demands.set(demand.id, demand);
    return clone(demand);
  }

  async findMatchingDemand(input: CreateDemandRecord["spec"]): Promise<DemandCluster | undefined> {
    const match = [...this.demands.values()].find((demand) => {
      if (!["OPEN", "READY"].includes(demand.status)) return false;
      if (demand.category !== input.category || demand.serviceArea !== input.serviceArea) return false;
      if (demand.servingDate !== input.servingDate || demand.servingTime !== input.servingTime) return false;
      if (!sameStringSet(demand.hardConstraints, input.hardConstraints)) return false;
      return rangesOverlap(demand.budgetMinCents, demand.budgetMaxCents, input.budgetMinCents, input.budgetMaxCents)
        && rangesOverlap(demand.weightMinGrams, demand.weightMaxGrams, input.weightMinGrams, input.weightMaxGrams)
        && demand.memberCount < demand.maximumMembers;
    });
    return match ? clone(match) : undefined;
  }

  async getDemand(id: string): Promise<DemandCluster | undefined> {
    const value = this.demands.get(id);
    return value ? clone(value) : undefined;
  }

  async listDemands(filters: { status?: string; limit?: number; offset?: number } = {}): Promise<DemandCluster[]> {
    return clone(
      [...this.demands.values()]
        .filter((demand) => !filters.status || demand.status === filters.status)
        .sort((a, b) => b.createdAt.localeCompare(a.createdAt))
        .slice(filters.offset ?? 0, (filters.offset ?? 0) + (filters.limit ?? 50))
    );
  }

  async reviewDemand(id: string, status: DemandCluster["status"], reviewerId: string, reason?: string): Promise<DemandCluster> {
    const demand = assertFound(this.demands.get(id), "Demand not found");
    demand.status = status;
    demand.reviewedBy = reviewerId;
    demand.reviewedAt = nowIso();
    demand.reviewReason = reason;
    demand.updatedAt = nowIso();
    this.demands.set(id, demand);
    return clone(demand);
  }

  async addDemandMember(input: CreateMemberRecord): Promise<DemandMember> {
    const demand = assertFound(this.demands.get(input.demandId), "Demand not found");
    if (demand.memberCount >= demand.maximumMembers) {
      throw new ApiError("DEMAND_FULL", "Demand has reached its maximum member count", 409);
    }
    const existing = await this.getDemandMember(input.demandId, input.userId);
    if (existing) throw new ApiError("ALREADY_JOINED", "User has already joined this demand", 409);
    const member: DemandMember = {
      id: newId(),
      demandId: input.demandId,
      userId: input.userId,
      quantity: input.quantity,
      weightGrams: input.weightGrams,
      preferences: input.preferences,
      notes: input.notes,
      createdAt: nowIso()
    };
    this.members.set(member.id, member);
    demand.memberCount += 1;
    if (demand.memberCount >= demand.minimumMembers && demand.status === "OPEN") demand.status = "READY";
    demand.updatedAt = nowIso();
    this.demands.set(demand.id, demand);
    return clone(member);
  }

  async getDemandMember(demandId: string, userId: string): Promise<DemandMember | undefined> {
    const value = [...this.members.values()].find((member) => member.demandId === demandId && member.userId === userId);
    return value ? clone(value) : undefined;
  }

  async listDemandMembers(demandId: string): Promise<DemandMember[]> {
    return clone([...this.members.values()].filter((member) => member.demandId === demandId));
  }

  async createOffer(merchantId: string, input: CreateOfferRecord): Promise<Offer> {
    const merchant = assertFound(this.merchants.get(merchantId), "Merchant not found");
    if (merchant.status !== "APPROVED") throw new ApiError("MERCHANT_NOT_APPROVED", "Merchant must be approved before submitting offers", 403);
    assertFound(this.demands.get(input.demandId), "Demand not found");
    const offer: Offer = { ...clone(input), id: newId(), merchantId, status: "SUBMITTED", createdAt: nowIso() };
    this.offers.set(offer.id, offer);
    return clone(offer);
  }

  async getOffer(id: string): Promise<Offer | undefined> {
    const value = this.offers.get(id);
    return value ? clone(value) : undefined;
  }

  async listOffers(filters: { demandId?: string; merchantId?: string; status?: string } = {}): Promise<Offer[]> {
    return clone(
      [...this.offers.values()].filter(
        (offer) =>
          (!filters.demandId || offer.demandId === filters.demandId) &&
          (!filters.merchantId || offer.merchantId === filters.merchantId) &&
          (!filters.status || offer.status === filters.status)
      )
    );
  }

  async createCampaign(merchantId: string, input: CreateCampaignRecord): Promise<Campaign> {
    const merchant = assertFound(this.merchants.get(merchantId), "Merchant not found");
    if (merchant.status !== "APPROVED") throw new ApiError("MERCHANT_NOT_APPROVED", "Merchant must be approved before creating campaigns", 403);
    const demand = assertFound(this.demands.get(input.demandId), "Demand not found");
    const offer = assertFound(this.offers.get(input.offerId), "Offer not found");
    if (offer.merchantId !== merchantId || offer.demandId !== demand.id) throw new ApiError("OFFER_MISMATCH", "Offer does not belong to this merchant and demand", 409);
    const campaign: Campaign = {
      ...clone(input),
      id: newId(),
      merchantId,
      status: "PENDING_REVIEW",
      currentOrders: 0,
      createdAt: nowIso(),
      updatedAt: nowIso()
    };
    this.campaigns.set(campaign.id, campaign);
    return clone(campaign);
  }

  async getCampaign(id: string): Promise<Campaign | undefined> {
    const value = this.campaigns.get(id);
    return value ? clone(value) : undefined;
  }

  async listCampaigns(filters: { status?: string; merchantId?: string; demandId?: string } = {}): Promise<Campaign[]> {
    return clone(
      [...this.campaigns.values()].filter(
        (campaign) =>
          (!filters.status || campaign.status === filters.status) &&
          (!filters.merchantId || campaign.merchantId === filters.merchantId) &&
          (!filters.demandId || campaign.demandId === filters.demandId)
      )
    );
  }

  async reviewCampaign(id: string, status: Campaign["status"], _reason?: string): Promise<Campaign> {
    const campaign = assertFound(this.campaigns.get(id), "Campaign not found");
    campaign.status = status;
    campaign.updatedAt = nowIso();
    this.campaigns.set(id, campaign);
    return clone(campaign);
  }

  async incrementCampaignOrders(id: string, quantity: number): Promise<Campaign> {
    const campaign = assertFound(this.campaigns.get(id), "Campaign not found");
    if (campaign.currentOrders + quantity > campaign.maximumOrders) throw new ApiError("CAMPAIGN_SOLD_OUT", "Campaign does not have enough remaining capacity", 409);
    campaign.currentOrders += quantity;
    if (campaign.currentOrders >= campaign.maximumOrders) campaign.status = "SOLD_OUT";
    campaign.updatedAt = nowIso();
    this.campaigns.set(id, campaign);
    return clone(campaign);
  }

  async createOrder(consumerId: string, input: CreateOrderRecord): Promise<Order> {
    const campaign = assertFound(this.campaigns.get(input.campaignId), "Campaign not found");
    if (campaign.status !== "OPEN") throw new ApiError("CAMPAIGN_NOT_OPEN", "Campaign is not open for orders", 409);
    const subtotalCents = campaign.unitPriceCents * input.quantity;
    const platformFeeCents = Math.ceil(subtotalCents * campaign.platformFeeBps / 10_000);
    const totalCents = subtotalCents + campaign.deliveryFeeCents + platformFeeCents;
    await this.incrementCampaignOrders(campaign.id, input.quantity);
    const order: Order = {
      ...clone(input),
      id: newId(),
      consumerId,
      status: "PENDING_PAYMENT",
      unitPriceCents: campaign.unitPriceCents,
      subtotalCents,
      deliveryFeeCents: campaign.deliveryFeeCents,
      platformFeeCents,
      totalCents,
      createdAt: nowIso(),
      updatedAt: nowIso()
    };
    this.orders.set(order.id, order);
    return clone(order);
  }

  async getOrder(id: string): Promise<Order | undefined> {
    const value = this.orders.get(id);
    return value ? clone(value) : undefined;
  }

  async listOrders(filters: { consumerId?: string; merchantId?: string; riderId?: string; status?: string }): Promise<Order[]> {
    return clone(
      [...this.orders.values()].filter((order) => {
        if (filters.consumerId && order.consumerId !== filters.consumerId) return false;
        if (filters.status && order.status !== filters.status) return false;
        if (filters.merchantId || filters.riderId) {
          const campaign = this.campaigns.get(order.campaignId);
          if (filters.merchantId && campaign?.merchantId !== filters.merchantId) return false;
          if (filters.riderId) {
            const task = [...this.tasks.values()].find((item) => item.orderId === order.id);
            if (task?.riderId !== filters.riderId) return false;
          }
        }
        return true;
      })
    );
  }

  async updateOrderStatus(id: string, status: Order["status"]): Promise<Order> {
    const order = assertFound(this.orders.get(id), "Order not found");
    assertOrderTransition(order.status, status);
    order.status = status;
    order.updatedAt = nowIso();
    this.orders.set(id, order);
    if (status === "PAID" && ![...this.tasks.values()].some((task) => task.orderId === order.id)) {
      const campaign = assertFound(this.campaigns.get(order.campaignId), "Campaign not found");
      const task: RiderTask = {
        id: newId(),
        orderId: order.id,
        status: "UNASSIGNED",
        pickupPoint: campaign.pickupPoint,
        deliveryAddress: order.deliveryAddress,
        createdAt: nowIso(),
        updatedAt: nowIso()
      };
      this.tasks.set(task.id, task);
    }
    return clone(order);
  }

  async getRiderTask(id: string): Promise<RiderTask | undefined> {
    const value = this.tasks.get(id);
    return value ? clone(value) : undefined;
  }

  async listRiderTasks(filters: { riderId?: string; status?: string } = {}): Promise<RiderTask[]> {
    return clone(
      [...this.tasks.values()].filter((task) => (!filters.riderId || task.riderId === filters.riderId) && (!filters.status || task.status === filters.status))
    );
  }

  async claimRiderTask(id: string, riderId: string): Promise<RiderTask> {
    const task = assertFound(this.tasks.get(id), "Rider task not found");
    if (task.riderId && task.riderId !== riderId) throw new ApiError("TASK_ALREADY_ASSIGNED", "Rider task is assigned to another rider", 409);
    if (!["UNASSIGNED", "ASSIGNED"].includes(task.status)) throw new ApiError("TASK_NOT_CLAIMABLE", "Rider task is not claimable", 409);
    task.riderId = riderId;
    task.status = "ASSIGNED";
    task.updatedAt = nowIso();
    this.tasks.set(id, task);
    return clone(task);
  }

  async updateRiderTask(id: string, riderId: string, status: RiderTask["status"], note?: string): Promise<RiderTask> {
    const task = assertFound(this.tasks.get(id), "Rider task not found");
    if (task.riderId !== riderId) throw new ApiError("TASK_NOT_ASSIGNED", "Task is not assigned to this rider", 403);
    task.status = status;
    task.note = note;
    task.updatedAt = nowIso();
    this.tasks.set(id, task);
    const orderStatus: Record<RiderTask["status"], Order["status"] | undefined> = {
      UNASSIGNED: undefined,
      ASSIGNED: undefined,
      PICKED_UP: "PICKED_UP",
      DELIVERING: "DELIVERING",
      COMPLETED: "DELIVERED",
      CANCELLED: "CANCELLED"
    };
    const nextOrderStatus = orderStatus[status];
    if (nextOrderStatus) {
      const order = assertFound(this.orders.get(task.orderId), "Order not found");
      if (order.status !== nextOrderStatus) {
        try {
          await this.updateOrderStatus(order.id, nextOrderStatus);
        } catch (error) {
          if (!(error instanceof ApiError && error.code === "INVALID_STATUS_TRANSITION")) throw error;
        }
      }
    }
    return clone(task);
  }

  async createIncident(reportedBy: string, input: CreateIncidentRecord): Promise<FoodSafetyIncident> {
    const incident: FoodSafetyIncident = {
      ...clone(input),
      id: newId(),
      reportedBy,
      status: "OPEN",
      createdAt: nowIso(),
      updatedAt: nowIso()
    };
    this.incidents.set(incident.id, incident);
    return clone(incident);
  }

  async listIncidents(filters: { merchantId?: string; status?: string } = {}): Promise<FoodSafetyIncident[]> {
    return clone(
      [...this.incidents.values()].filter(
        (incident) => (!filters.merchantId || incident.merchantId === filters.merchantId) && (!filters.status || incident.status === filters.status)
      )
    );
  }
}
