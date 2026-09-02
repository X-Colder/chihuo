import type { FastifyReply, FastifyRequest } from "fastify";
import { ZodError } from "zod";

export class ApiError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly statusCode = 400,
    public readonly details?: unknown
  ) {
    super(message);
    this.name = "ApiError";
  }
}

export function assertFound<T>(value: T | undefined | null, message: string, code = "NOT_FOUND"): T {
  if (value === undefined || value === null) {
    throw new ApiError(code, message, 404);
  }
  return value;
}

export function assert(condition: unknown, code: string, message: string, statusCode = 400, details?: unknown): asserts condition {
  if (!condition) {
    throw new ApiError(code, message, statusCode, details);
  }
}

export function parseOrThrow<T>(schema: { parse: (value: unknown) => T }, value: unknown): T {
  try {
    return schema.parse(value);
  } catch (error) {
    if (error instanceof ZodError) {
      throw new ApiError("VALIDATION_ERROR", "Request validation failed", 422, error.flatten());
    }
    throw error;
  }
}

export function registerErrorHandler(app: {
  setErrorHandler: (
    handler: (error: unknown, request: FastifyRequest, reply: FastifyReply) => Promise<unknown>
  ) => void;
}): void {
  app.setErrorHandler(async (error, request, reply) => {
    const requestId = request.id;
    if (error instanceof ApiError) {
      return reply.status(error.statusCode).send({
        error: {
          code: error.code,
          message: error.message,
          ...(error.details === undefined ? {} : { details: error.details }),
          requestId
        }
      });
    }

    request.log.error(error);
    return reply.status(500).send({
      error: {
        code: "INTERNAL_ERROR",
        message: "Internal server error",
        requestId
      }
    });
  });
}
