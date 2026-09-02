import type { FastifyInstance } from "fastify";
import {
  DemoLoginInputSchema,
  type Role,
  type SessionUser
} from "@chihuo/contracts";
import { parseOrThrow } from "../errors.js";
import { newId } from "../utils.js";
import { requireAuth } from "../auth.js";
import type { Store } from "../db/types.js";

export async function registerAuthRoutes(app: FastifyInstance, store: Store): Promise<void> {
  app.post("/v1/auth/demo-login", async (request, reply) => {
    const input = parseOrThrow(DemoLoginInputSchema, request.body);
    const demoKey = `${input.role}:${input.name.trim().toLowerCase()}`;
    const user = await store.createOrGetUser({ name: input.name, role: input.role, demoKey });
    let merchantId: string | undefined;
    if (input.role === "MERCHANT") {
      const merchant = await store.createOrGetMerchant({
        ownerUserId: user.id,
        name: input.merchantName ?? `${input.name}的店`,
        license: { verificationRequired: true }
      });
      merchantId = merchant.id;
    }
    const sessionUser: SessionUser = {
      id: user.id,
      name: user.name,
      role: user.role as Role,
      ...(merchantId ? { merchantId } : {})
    };
    const token = await reply.jwtSign(sessionUser, { expiresIn: "7d", jti: newId() });
    return reply.code(200).send({ data: { token, user: sessionUser } });
  });

  app.get("/v1/auth/me", { preHandler: requireAuth }, async (request) => {
    return { data: request.user };
  });
}
