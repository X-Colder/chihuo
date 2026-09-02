import type { FastifyInstance } from "fastify";
import { registerAuthRoutes } from "./modules/auth.js";
import { registerDemandRoutes } from "./modules/demands.js";
import { registerMerchantRoutes } from "./modules/merchant.js";
import { registerAdminRoutes } from "./modules/admin.js";
import { registerCampaignRoutes } from "./modules/campaigns.js";
import { registerOrderRoutes } from "./modules/orders.js";
import { registerRiderRoutes } from "./modules/rider.js";
import { registerFoodSafetyRoutes } from "./modules/food-safety.js";
import type { Store } from "./db/types.js";

export async function registerRoutes(app: FastifyInstance, store: Store): Promise<void> {
  await registerAuthRoutes(app, store);
  await registerDemandRoutes(app, store);
  await registerMerchantRoutes(app, store);
  await registerAdminRoutes(app, store);
  await registerCampaignRoutes(app, store);
  await registerOrderRoutes(app, store);
  await registerRiderRoutes(app, store);
  await registerFoodSafetyRoutes(app, store);
}
