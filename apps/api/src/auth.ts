import type { FastifyInstance, FastifyReply, FastifyRequest } from "fastify";
import fastifyJwt from "@fastify/jwt";
import type { Role, SessionUser } from "@chihuo/contracts";
import { ApiError, assert } from "./errors.js";

declare module "@fastify/jwt" {
  interface FastifyJWT {
    user: SessionUser;
  }
}

export type AuthenticatedRequest = FastifyRequest & {
  user: SessionUser;
};

export async function requireAuth(request: FastifyRequest): Promise<SessionUser> {
  try {
    await request.jwtVerify();
  } catch {
    throw new ApiError("UNAUTHORIZED", "Authentication required", 401);
  }
  return request.user;
}

export function requireRole(...roles: Role[]) {
  return async (request: FastifyRequest, _reply: FastifyReply): Promise<void> => {
    const user = await requireAuth(request);
    assert(roles.includes(user.role), "FORBIDDEN", "You do not have permission for this resource", 403, {
      requiredRoles: roles
    });
  };
}

export async function registerAuth(app: FastifyInstance, secret: string): Promise<void> {
  await app.register(fastifyJwt, { secret });
}
