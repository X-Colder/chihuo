import type {
  Campaign,
  DemandCluster,
  DemandMember,
  FoodSafetyIncident,
  Merchant,
  Offer,
  Order,
  RiderTask,
  SessionUser,
  User
} from "@chihuo/contracts";

export type CreateUser = {
  name: string;
  role: SessionUser["role"];
  demoKey: string;
};

export type CreateMerchant = {
  ownerUserId: string;
  name: string;
  license?: Record<string, unknown>;
};

export type CreateDemandRecord = {
  createdBy: string;
  minimumMembers: number;
  maximumMembers: number;
  spec: Omit<DemandCluster, "id" | "status" | "minimumMembers" | "maximumMembers" | "memberCount" | "createdBy" | "createdAt" | "updatedAt">;
};

export type CreateMemberRecord = {
  demandId: string;
  userId: string;
  quantity: number;
  weightGrams?: number;
  preferences: string[];
  notes?: string;
};

export type CreateOfferRecord = Omit<Offer, "id" | "merchantId" | "status" | "createdAt">;
export type CreateCampaignRecord = Omit<Campaign, "id" | "merchantId" | "status" | "currentOrders" | "createdAt" | "updatedAt">;
export type CreateOrderRecord = Omit<
  Order,
  "id" | "consumerId" | "status" | "unitPriceCents" | "subtotalCents" | "deliveryFeeCents" | "platformFeeCents" | "totalCents" | "createdAt" | "updatedAt"
>;
export type CreateIncidentRecord = Omit<FoodSafetyIncident, "id" | "reportedBy" | "status" | "createdAt" | "updatedAt">;

export type Store = {
  ping(): Promise<boolean>;
  close(): Promise<void>;
  seed(): Promise<void>;

  createOrGetUser(input: CreateUser): Promise<User>;
  getUser(id: string): Promise<User | undefined>;
  createOrGetMerchant(input: CreateMerchant): Promise<Merchant>;
  getMerchant(id: string): Promise<Merchant | undefined>;
  listMerchants(status?: string): Promise<Merchant[]>;
  reviewMerchant(id: string, status: Merchant["status"], reason?: string): Promise<Merchant>;

  createDemand(input: CreateDemandRecord): Promise<DemandCluster>;
  findMatchingDemand(input: CreateDemandRecord["spec"]): Promise<DemandCluster | undefined>;
  getDemand(id: string): Promise<DemandCluster | undefined>;
  listDemands(filters?: { status?: string; limit?: number; offset?: number }): Promise<DemandCluster[]>;
  reviewDemand(id: string, status: DemandCluster["status"], reviewerId: string, reason?: string): Promise<DemandCluster>;
  addDemandMember(input: CreateMemberRecord): Promise<DemandMember>;
  getDemandMember(demandId: string, userId: string): Promise<DemandMember | undefined>;
  listDemandMembers(demandId: string): Promise<DemandMember[]>;

  createOffer(merchantId: string, input: CreateOfferRecord): Promise<Offer>;
  getOffer(id: string): Promise<Offer | undefined>;
  listOffers(filters?: { demandId?: string; merchantId?: string; status?: string }): Promise<Offer[]>;

  createCampaign(merchantId: string, input: CreateCampaignRecord): Promise<Campaign>;
  getCampaign(id: string): Promise<Campaign | undefined>;
  listCampaigns(filters?: { status?: string; merchantId?: string; demandId?: string }): Promise<Campaign[]>;
  reviewCampaign(id: string, status: Campaign["status"], reason?: string): Promise<Campaign>;
  incrementCampaignOrders(id: string, quantity: number): Promise<Campaign>;

  createOrder(consumerId: string, input: CreateOrderRecord): Promise<Order>;
  getOrder(id: string): Promise<Order | undefined>;
  listOrders(filters: { consumerId?: string; merchantId?: string; riderId?: string; status?: string }): Promise<Order[]>;
  updateOrderStatus(id: string, status: Order["status"]): Promise<Order>;

  getRiderTask(id: string): Promise<RiderTask | undefined>;
  listRiderTasks(filters?: { riderId?: string; status?: string }): Promise<RiderTask[]>;
  claimRiderTask(id: string, riderId: string): Promise<RiderTask>;
  updateRiderTask(id: string, riderId: string, status: RiderTask["status"], note?: string): Promise<RiderTask>;

  createIncident(reportedBy: string, input: CreateIncidentRecord): Promise<FoodSafetyIncident>;
  listIncidents(filters?: { merchantId?: string; status?: string }): Promise<FoodSafetyIncident[]>;
};
