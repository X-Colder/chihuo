import { z } from "zod";

export const RoleSchema = z.enum(["CONSUMER", "MERCHANT", "RIDER", "ADMIN"]);
export type Role = z.infer<typeof RoleSchema>;

export const MerchantStatusSchema = z.enum(["PENDING", "APPROVED", "REJECTED", "SUSPENDED"]);
export type MerchantStatus = z.infer<typeof MerchantStatusSchema>;

export const DemandStatusSchema = z.enum(["PENDING_REVIEW", "OPEN", "READY", "CLOSED", "REJECTED"]);
export type DemandStatus = z.infer<typeof DemandStatusSchema>;

export const CampaignStatusSchema = z.enum(["DRAFT", "PENDING_REVIEW", "OPEN", "SOLD_OUT", "CLOSED", "CANCELLED"]);
export type CampaignStatus = z.infer<typeof CampaignStatusSchema>;

export const OfferStatusSchema = z.enum(["SUBMITTED", "ACCEPTED", "REJECTED", "WITHDRAWN"]);
export type OfferStatus = z.infer<typeof OfferStatusSchema>;

export const OrderStatusSchema = z.enum([
  "PENDING_PAYMENT",
  "PAID",
  "ACCEPTED",
  "PREPARING",
  "READY_FOR_PICKUP",
  "PICKED_UP",
  "DELIVERING",
  "DELIVERED",
  "CANCELLED",
  "REFUNDED"
]);
export type OrderStatus = z.infer<typeof OrderStatusSchema>;

export const RiderTaskStatusSchema = z.enum(["UNASSIGNED", "ASSIGNED", "PICKED_UP", "DELIVERING", "COMPLETED", "CANCELLED"]);
export type RiderTaskStatus = z.infer<typeof RiderTaskStatusSchema>;

export const FoodSafetyIncidentSeveritySchema = z.enum(["LOW", "MEDIUM", "HIGH", "CRITICAL"]);
export type FoodSafetyIncidentSeverity = z.infer<typeof FoodSafetyIncidentSeveritySchema>;

export const FoodSafetyIncidentStatusSchema = z.enum(["OPEN", "INVESTIGATING", "RESOLVED", "REPORTED"]);
export type FoodSafetyIncidentStatus = z.infer<typeof FoodSafetyIncidentStatusSchema>;

const nonEmptyText = z.string().trim().min(1).max(500);
const isoDate = z.string().regex(/^\d{4}-\d{2}-\d{2}$/, "must be YYYY-MM-DD");
const isoDateTime = z.string().datetime({ offset: true });

const DemandSpecBaseSchema = z.object({
  category: nonEmptyText.max(80),
  title: nonEmptyText.max(120),
  serviceArea: nonEmptyText.max(120),
  servingDate: isoDate,
  servingTime: z.string().regex(/^\d{2}:\d{2}$/, "must be HH:mm"),
  budgetMinCents: z.number().int().positive().max(100_000),
  budgetMaxCents: z.number().int().positive().max(100_000),
  quantity: z.number().int().positive().max(100),
  weightMinGrams: z.number().int().positive().max(10_000),
  weightMaxGrams: z.number().int().positive().max(10_000),
  hardConstraints: z.array(nonEmptyText.max(80)).max(20).default([]),
  preferences: z.array(nonEmptyText.max(80)).max(20).default([]),
  notes: z.string().trim().max(1_000).optional()
});

export const DemandSpecSchema = DemandSpecBaseSchema.superRefine((value, ctx) => {
  if (value.budgetMinCents > value.budgetMaxCents) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["budgetMinCents"], message: "must not exceed budgetMaxCents" });
  }
  if (value.weightMinGrams > value.weightMaxGrams) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["weightMinGrams"], message: "must not exceed weightMaxGrams" });
  }
});
export type DemandSpec = z.infer<typeof DemandSpecSchema>;

export const CreateDemandInputSchema = DemandSpecBaseSchema.extend({
  minimumMembers: z.number().int().positive().max(1_000).default(10),
  maximumMembers: z.number().int().positive().max(5_000).default(100)
}).superRefine((value, ctx) => {
  if (value.budgetMinCents > value.budgetMaxCents) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["budgetMinCents"], message: "must not exceed budgetMaxCents" });
  }
  if (value.weightMinGrams > value.weightMaxGrams) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["weightMinGrams"], message: "must not exceed weightMaxGrams" });
  }
  if (value.minimumMembers > value.maximumMembers) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["minimumMembers"], message: "must not exceed maximumMembers" });
  }
});
export type CreateDemandInput = z.infer<typeof CreateDemandInputSchema>;

export const JoinDemandInputSchema = z.object({
  quantity: z.number().int().positive().max(100).default(1),
  weightGrams: z.number().int().positive().max(10_000).optional(),
  preferences: z.array(nonEmptyText.max(80)).max(20).default([]),
  notes: z.string().trim().max(1_000).optional()
});
export type JoinDemandInput = z.infer<typeof JoinDemandInputSchema>;

export const DemoLoginInputSchema = z.object({
  name: nonEmptyText.max(80),
  role: RoleSchema.default("CONSUMER"),
  merchantName: z.string().trim().min(1).max(120).optional()
});
export type DemoLoginInput = z.infer<typeof DemoLoginInputSchema>;

export const MerchantReviewInputSchema = z.object({
  status: z.enum(["APPROVED", "REJECTED", "SUSPENDED"]),
  reason: z.string().trim().max(500).optional()
});
export type MerchantReviewInput = z.infer<typeof MerchantReviewInputSchema>;

export const DemandReviewInputSchema = z.object({
  status: z.enum(["OPEN", "REJECTED"]),
  reason: z.string().trim().max(500).optional()
});
export type DemandReviewInput = z.infer<typeof DemandReviewInputSchema>;

export const CreateOfferInputSchema = z.object({
  demandId: z.string().uuid(),
  unitPriceCents: z.number().int().positive().max(100_000),
  productionCapacity: z.number().int().positive().max(10_000),
  weightGrams: z.number().int().positive().max(10_000),
  ingredients: z.array(nonEmptyText.max(100)).max(100),
  allergens: z.array(nonEmptyText.max(100)).max(50).default([]),
  oilLevel: z.enum(["UNKNOWN", "LOW", "MEDIUM", "HIGH"]).default("UNKNOWN"),
  saltLevel: z.enum(["UNKNOWN", "LOW", "MEDIUM", "HIGH"]).default("UNKNOWN"),
  productionTime: z.string().regex(/^\d{2}:\d{2}$/, "must be HH:mm"),
  shelfLifeMinutes: z.number().int().positive().max(10_080),
  storageInstructions: nonEmptyText.max(300),
  notes: z.string().trim().max(1_000).optional()
});
export type CreateOfferInput = z.infer<typeof CreateOfferInputSchema>;

export const CreateCampaignInputSchema = z.object({
  demandId: z.string().uuid(),
  offerId: z.string().uuid(),
  title: nonEmptyText.max(120),
  description: z.string().trim().max(1_000).optional(),
  unitPriceCents: z.number().int().positive().max(100_000),
  deliveryFeeCents: z.number().int().nonnegative().max(100_000).default(500),
  platformFeeBps: z.number().int().min(0).max(2_000).default(500),
  minimumOrders: z.number().int().positive().max(10_000),
  maximumOrders: z.number().int().positive().max(10_000),
  startsAt: isoDateTime,
  endsAt: isoDateTime,
  pickupPoint: nonEmptyText.max(200),
  foodSpec: z.object({
    weightGrams: z.number().int().positive().max(10_000),
    ingredients: z.array(nonEmptyText.max(100)).max(100),
    allergens: z.array(nonEmptyText.max(100)).max(50).default([]),
    oilLevel: z.enum(["UNKNOWN", "LOW", "MEDIUM", "HIGH"]).default("UNKNOWN"),
    saltLevel: z.enum(["UNKNOWN", "LOW", "MEDIUM", "HIGH"]).default("UNKNOWN"),
    productionTime: z.string().regex(/^\d{2}:\d{2}$/, "must be HH:mm"),
    shelfLifeMinutes: z.number().int().positive().max(10_080),
    storageInstructions: nonEmptyText.max(300)
  })
}).superRefine((value, ctx) => {
  if (value.minimumOrders > value.maximumOrders) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["minimumOrders"], message: "must not exceed maximumOrders" });
  }
  if (value.endsAt <= value.startsAt) {
    ctx.addIssue({ code: z.ZodIssueCode.custom, path: ["endsAt"], message: "must be after startsAt" });
  }
});
export type CreateCampaignInput = z.infer<typeof CreateCampaignInputSchema>;

export const CreateOrderInputSchema = z.object({
  quantity: z.number().int().positive().max(100),
  deliveryAddress: nonEmptyText.max(300),
  contactName: nonEmptyText.max(80),
  contactPhone: z.string().trim().min(5).max(30)
});
export type CreateOrderInput = z.infer<typeof CreateOrderInputSchema>;

export const UpdateOrderStatusInputSchema = z.object({
  status: z.enum(["ACCEPTED", "PREPARING", "READY_FOR_PICKUP", "CANCELLED", "REFUNDED"]),
  reason: z.string().trim().max(500).optional()
});
export type UpdateOrderStatusInput = z.infer<typeof UpdateOrderStatusInputSchema>;

export const UpdateRiderTaskInputSchema = z.object({
  status: z.enum(["PICKED_UP", "DELIVERING", "COMPLETED", "CANCELLED"]),
  note: z.string().trim().max(500).optional()
});
export type UpdateRiderTaskInput = z.infer<typeof UpdateRiderTaskInputSchema>;

export const CreateIncidentInputSchema = z.object({
  merchantId: z.string().uuid(),
  orderId: z.string().uuid().optional(),
  campaignId: z.string().uuid().optional(),
  severity: FoodSafetyIncidentSeveritySchema,
  title: nonEmptyText.max(160),
  description: nonEmptyText.max(2_000),
  evidenceUrls: z.array(z.string().url()).max(20).default([]),
  affectedQuantity: z.number().int().positive().max(100_000).optional()
});
export type CreateIncidentInput = z.infer<typeof CreateIncidentInputSchema>;

export const ListQuerySchema = z.object({
  status: z.string().trim().min(1).max(40).optional(),
  limit: z.coerce.number().int().min(1).max(100).default(50),
  offset: z.coerce.number().int().min(0).max(100_000).default(0)
});
export type ListQuery = z.infer<typeof ListQuerySchema>;

export type ApiErrorBody = {
  error: {
    code: string;
    message: string;
    details?: unknown;
    requestId: string;
  };
};

export type SessionUser = {
  id: string;
  name: string;
  role: Role;
  merchantId?: string;
};

export type User = SessionUser & {
  createdAt: string;
};

export type Merchant = {
  id: string;
  ownerUserId: string;
  name: string;
  status: MerchantStatus;
  license: Record<string, unknown>;
  createdAt: string;
};

export type DemandCluster = DemandSpec & {
  id: string;
  status: DemandStatus;
  minimumMembers: number;
  maximumMembers: number;
  memberCount: number;
  reviewedBy?: string;
  reviewedAt?: string;
  reviewReason?: string;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
};

export type DemandMember = {
  id: string;
  demandId: string;
  userId: string;
  quantity: number;
  weightGrams?: number;
  preferences: string[];
  notes?: string;
  createdAt: string;
};

export type Offer = CreateOfferInput & {
  id: string;
  merchantId: string;
  status: OfferStatus;
  createdAt: string;
};

export type Campaign = CreateCampaignInput & {
  id: string;
  merchantId: string;
  status: CampaignStatus;
  currentOrders: number;
  createdAt: string;
  updatedAt: string;
};

export type Order = CreateOrderInput & {
  id: string;
  campaignId: string;
  consumerId: string;
  status: OrderStatus;
  unitPriceCents: number;
  subtotalCents: number;
  deliveryFeeCents: number;
  platformFeeCents: number;
  totalCents: number;
  createdAt: string;
  updatedAt: string;
};

export type RiderTask = {
  id: string;
  orderId: string;
  riderId?: string;
  status: RiderTaskStatus;
  pickupPoint: string;
  deliveryAddress: string;
  note?: string;
  createdAt: string;
  updatedAt: string;
};

export type FoodSafetyIncident = CreateIncidentInput & {
  id: string;
  reportedBy: string;
  status: FoodSafetyIncidentStatus;
  createdAt: string;
  updatedAt: string;
};
