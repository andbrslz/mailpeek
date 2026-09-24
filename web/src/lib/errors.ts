import { ApiError } from "../api";
import type { MessageKey } from "../i18n/messages";

export interface Problem {
  key: MessageKey;
  vars?: Record<string, string | number>;
}

export function describeError(err: unknown): Problem {
  if (err instanceof ApiError) {
    if (err.status === 401) return { key: "error.unauthorized" };
    if (err.status === 404) return { key: "error.notFound" };
    if (err.status >= 500) return { key: "error.server", vars: { status: err.status } };
    return { key: "error.http", vars: { status: err.status } };
  }
  if (err instanceof TypeError) return { key: "error.network" };
  return { key: "error.unknown" };
}
