import { randomUUID } from "node:crypto";
import type { Role } from "@chihuo/contracts";

export function nowIso(): string {
  return new Date().toISOString();
}

export function newId(): string {
  return randomUUID();
}

export function normalizeList(values: string[]): string[] {
  return [...new Set(values.map((value) => value.trim().toLowerCase()).filter(Boolean))].sort();
}

export function parseOptionalUuid(value: unknown): string | undefined {
  return typeof value === "string" && value.length > 0 ? value : undefined;
}

export function moneyToNumber(value: unknown): number {
  return typeof value === "number" ? value : Number(value);
}

export function roleIs(role: Role, ...expected: Role[]): boolean {
  return expected.includes(role);
}
