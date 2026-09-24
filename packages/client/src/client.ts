import { explainMismatch } from "./diagnose.js";
import { Email } from "./email.js";
import { Inbox, randomId, type InboxOptions } from "./inbox.js";
import type {
  FailureOptions,
  MailpeekOptions,
  MessageData,
  MessageFilter,
  MessageSummary,
  RequestOptions,
  ServerInfo,
  SmtpFailure,
  WaitOptions,
} from "./types.js";

/** The server holds a wait request for at most 30 seconds. */
const MAX_SERVER_WAIT = 30_000;
const DEFAULT_TIMEOUT = 10_000;
const DEFAULT_REQUEST_TIMEOUT = 10_000;
/** Diagnostics after a timeout must not make a failing test much slower. */
const DIAGNOSE_TIMEOUT = 2_000;

export class MailpeekError extends Error {
  constructor(
    message: string,
    readonly status?: number,
  ) {
    super(message);
    this.name = "MailpeekError";
  }
}

/** What Mailpeek held when a wait gave up, to explain why nothing matched. */
export interface TimeoutDiagnostics {
  /** Newest messages on the server (up to 5), each with why it didn't match. */
  recent: { message: MessageSummary; reasons: string[] }[];
  /** Messages on the server when the wait ended. */
  total: number;
  /** Messages removed because Mailpeek reached max-messages or max-store-size. */
  evicted: number;
  /** Web UI link filtered to this wait. */
  inspectUrl: string;
}

export class MailpeekTimeoutError extends MailpeekError {
  constructor(
    readonly filter: MessageFilter,
    readonly timeout: number,
    readonly diagnostics?: TimeoutDiagnostics,
    /** For waitForEmails: how many arrived versus how many were expected. */
    readonly progress?: { received: number; expected: number },
  ) {
    super(timeoutMessage(filter, timeout, diagnostics, progress));
    this.name = "MailpeekTimeoutError";
  }
}

function describe(filter: MessageFilter): string {
  const parts = Object.entries(filter)
    .filter(([, v]) => v !== undefined && v !== "")
    .map(([k, v]) => `${k}=${JSON.stringify(v instanceof Date ? v.toISOString() : v)}`);
  return parts.length ? `{ ${parts.join(", ")} }` : "any filter";
}

function timeoutMessage(
  filter: MessageFilter,
  timeout: number,
  d?: TimeoutDiagnostics,
  progress?: { received: number; expected: number },
): string {
  const head = progress
    ? `Only ${progress.received} of ${progress.expected} emails matching ${describe(filter)} arrived within ${timeout}ms`
    : `No email matching ${describe(filter)} arrived within ${timeout}ms`;
  if (!d) return head;
  const lines = [head + "."];
  if (d.total === 0) {
    lines.push(
      "Mailpeek holds no messages at all: is the application sending to this Mailpeek's SMTP port?",
    );
  } else {
    lines.push(`Mailpeek holds ${d.total} message${d.total === 1 ? "" : "s"}. Most recent:`);
    for (const { message: m, reasons } of d.recent) {
      const to = [...m.to, ...m.cc].map((a) => a.address).join(", ") || m.envelope?.to.join(", ");
      const when = new Date(m.createdAt).toISOString().slice(11, 23);
      const why = reasons.length
        ? reasons.join("; ")
        : "matches (it arrived after the wait ended?)";
      lines.push(`  - ${JSON.stringify(m.subject)} to ${to || "(none)"} at ${when}: ${why}`);
    }
  }
  if (d.evicted > 0) {
    lines.push(
      `${d.evicted} message(s) were removed because Mailpeek reached its limits ` +
        "(MAILPEEK_MAX_MESSAGES / MAILPEEK_MAX_STORE_SIZE). Raise them if parallel tests need more.",
    );
  }
  lines.push(`Inspect: ${d.inspectUrl}`);
  return lines.join("\n");
}

function envUrl(): string | undefined {
  const proc = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process;
  return proc?.env?.MAILPEEK_URL || undefined;
}

function query(filter: MessageFilter, extra: Record<string, string> = {}): string {
  const params = new URLSearchParams();
  for (const key of ["to", "address", "from", "subject", "body", "q"] as const) {
    const value = filter[key];
    if (value) params.set(key, value);
  }
  if (filter.since !== undefined) {
    const since = filter.since;
    params.set("since", since instanceof Date ? since.toISOString() : String(since));
  }
  for (const [k, v] of Object.entries(extra)) params.set(k, v);
  const qs = params.toString();
  return qs ? `?${qs}` : "";
}

type Init = RequestInit & { timeout?: number };

/**
 * Client for the Mailpeek HTTP API.
 *
 * ```ts
 * const mailpeek = new Mailpeek({ baseUrl: "http://localhost:8026" });
 * const email = await mailpeek.waitFor({ to: "john@example.com", subject: "Welcome" });
 * ```
 */
export class Mailpeek {
  readonly baseUrl: string;
  /** Default waitFor deadline (ms): how long to wait for an email. */
  readonly timeout: number;
  /** Transport timeout (ms) for each HTTP request. */
  readonly requestTimeout: number;
  private readonly inboxDomain: string;
  private readonly fetchImpl: typeof fetch;
  private readonly headers: Record<string, string> = {};

  constructor(options: MailpeekOptions = {}) {
    // fetch rejects URLs with credentials, so they move to a header.
    const url = new URL(options.baseUrl ?? envUrl() ?? "http://localhost:8026");
    const auth =
      options.auth ??
      (url.username
        ? {
            username: decodeURIComponent(url.username),
            password: decodeURIComponent(url.password),
          }
        : undefined);
    url.username = "";
    url.password = "";
    if (auth) this.headers.Authorization = basicAuth(auth.username, auth.password);
    this.baseUrl = url.toString().replace(/\/+$/, "");
    this.timeout = options.timeout ?? DEFAULT_TIMEOUT;
    this.requestTimeout = options.requestTimeout ?? DEFAULT_REQUEST_TIMEOUT;
    this.inboxDomain = options.inboxDomain ?? "mailpeek.local";
    this.fetchImpl = options.fetch ?? ((...args) => globalThis.fetch(...args));
  }

  /** Lists messages, newest first. Summaries do not include bodies; use `get`. */
  async messages(
    filter: MessageFilter = {},
    options: RequestOptions = {},
  ): Promise<MessageSummary[]> {
    const res = await this.request(`/messages${query(filter)}`, options);
    const body = (await res.json()) as { messages: MessageSummary[] };
    return body.messages;
  }

  async get(id: string, options: RequestOptions = {}): Promise<Email> {
    const res = await this.request(`/messages/${encodeURIComponent(id)}`, options);
    return new Email((await res.json()) as MessageData);
  }

  /** The newest message matching the filter, or undefined. */
  async latest(
    filter: MessageFilter = {},
    options: RequestOptions = {},
  ): Promise<Email | undefined> {
    const res = await this.request(`/messages/latest${query(filter)}`, options, [404]);
    if (res.status === 404) return undefined;
    return new Email((await res.json()) as MessageData);
  }

  /**
   * Resolves with the newest matching message, waiting for one to arrive if
   * needed. Rejects with MailpeekTimeoutError after `timeout` milliseconds;
   * the error lists what Mailpeek received and why it didn't match.
   */
  async waitFor(options: WaitOptions = {}): Promise<Email> {
    const { timeout = this.timeout, signal, ...filter } = options;
    const deadline = Date.now() + timeout;
    for (;;) {
      const chunk = Math.max(0, Math.min(deadline - Date.now(), MAX_SERVER_WAIT));
      const res = await this.request(
        `/messages/wait${query(filter, { timeout: String(chunk) })}`,
        { signal, timeout: chunk + this.requestTimeout },
        [204],
      );
      if (res.status === 200) return new Email((await res.json()) as MessageData);
      if (Date.now() >= deadline) {
        throw new MailpeekTimeoutError(filter, timeout, await this.diagnose(filter, signal));
      }
    }
  }

  /**
   * Waits until at least `count` matching messages exist and returns the
   * first `count` of them in the order they arrived. Useful when one action
   * sends several emails (for example a welcome email and a verification code).
   */
  async waitForEmails(count: number, options: WaitOptions = {}): Promise<Email[]> {
    const { timeout = this.timeout, signal, ...filter } = options;
    const deadline = Date.now() + timeout;
    for (;;) {
      const list = await this.messages(filter, { signal });
      if (list.length >= count) {
        const oldestFirst = list.slice(-count).reverse();
        return Promise.all(oldestFirst.map((m) => this.get(m.id, { signal })));
      }
      const remaining = deadline - Date.now();
      if (remaining <= 0) {
        throw new MailpeekTimeoutError(filter, timeout, await this.diagnose(filter, signal), {
          received: list.length,
          expected: count,
        });
      }
      // Wait for anything newer than what we already have.
      const newest = list[0];
      const since = newest ? new Date(Date.parse(newest.createdAt) + 1) : filter.since;
      try {
        await this.waitFor({ ...filter, since, timeout: remaining, signal });
      } catch (err) {
        if (!(err instanceof MailpeekTimeoutError)) throw err;
      }
    }
  }

  async delete(id: string, options: RequestOptions = {}): Promise<void> {
    await this.request(`/messages/${encodeURIComponent(id)}`, { ...options, method: "DELETE" });
  }

  /** Deletes messages matching the filter (all when omitted). Returns the count. */
  async clear(filter: MessageFilter = {}, options: RequestOptions = {}): Promise<number> {
    const res = await this.request(`/messages${query(filter)}`, { ...options, method: "DELETE" });
    const body = (await res.json()) as { deleted: number };
    return body.deleted;
  }

  /**
   * Makes the next `count` matching SMTP deliveries fail with an error reply,
   * to test how the application retries or reports it. Failed deliveries are
   * not stored. Prefer `inbox.failNext()`, which only affects that inbox.
   *
   * ```ts
   * await mailpeek.failNext({ address: "ana@example.com", code: 451 });
   * ```
   */
  async failNext(failure: FailureOptions = {}, options: RequestOptions = {}): Promise<SmtpFailure> {
    const res = await this.request("/smtp/failures", {
      ...options,
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(failure),
    });
    return (await res.json()) as SmtpFailure;
  }

  /** Simulated failures that have not been used up yet, oldest first. */
  async smtpFailures(options: RequestOptions = {}): Promise<SmtpFailure[]> {
    const res = await this.request("/smtp/failures", options);
    return ((await res.json()) as { failures: SmtpFailure[] }).failures;
  }

  /** Removes pending simulated failures (only those for `address` when given). Returns the count. */
  async clearFailures(address?: string, options: RequestOptions = {}): Promise<number> {
    const qs = address ? `?${new URLSearchParams({ address })}` : "";
    const res = await this.request(`/smtp/failures${qs}`, { ...options, method: "DELETE" });
    return ((await res.json()) as { deleted: number }).deleted;
  }

  /** The raw MIME source. */
  async raw(id: string, options: RequestOptions = {}): Promise<string> {
    const res = await this.request(`/messages/${encodeURIComponent(id)}/raw`, options);
    return res.text();
  }

  /** Attachment content. */
  async attachment(
    messageId: string,
    attachmentId: string,
    options: RequestOptions = {},
  ): Promise<Uint8Array> {
    const res = await this.request(
      `/messages/${encodeURIComponent(messageId)}/attachments/${encodeURIComponent(attachmentId)}`,
      options,
    );
    return new Uint8Array(await res.arrayBuffer());
  }

  /** Version, ports, limits and current usage of the server. */
  async info(options: RequestOptions = {}): Promise<ServerInfo> {
    const res = await this.request("/info", options);
    return (await res.json()) as ServerInfo;
  }

  /** Whether the server is reachable and healthy. */
  async health(options: RequestOptions = {}): Promise<boolean> {
    try {
      const res = await this.request("/health", options);
      return ((await res.json()) as { status?: string }).status === "ok";
    } catch {
      return false;
    }
  }

  /** A new inbox with a unique address, e.g. `test-k7x92ab1@mailpeek.local`. */
  createInbox(options: InboxOptions = {}): Inbox {
    const prefix = options.prefix ?? "test";
    const domain = options.domain ?? this.inboxDomain;
    return new Inbox(this, `${prefix}-${randomId()}@${domain}`);
  }

  /** An inbox view for an existing address (matched exactly). */
  inbox(address: string): Inbox {
    return new Inbox(this, address);
  }

  /** Web UI link filtered to what a wait was looking for. */
  inspectUrl(filter: MessageFilter = {}): string {
    const q = filter.address ?? filter.to ?? filter.subject ?? filter.from ?? filter.q;
    return q ? `${this.baseUrl}/?q=${encodeURIComponent(q)}` : `${this.baseUrl}/`;
  }

  /** Best effort: never throws, and gives up quickly. */
  private async diagnose(
    filter: MessageFilter,
    signal?: AbortSignal,
  ): Promise<TimeoutDiagnostics | undefined> {
    if (signal?.aborted) return undefined;
    try {
      const opts = { signal, timeout: DIAGNOSE_TIMEOUT };
      const [all, info] = await Promise.all([
        this.messages({}, opts),
        this.info(opts).catch(() => undefined),
      ]);
      return {
        total: all.length,
        evicted: info?.store?.evicted ?? 0,
        recent: all
          .slice(0, 5)
          .map((message) => ({ message, reasons: explainMismatch(message, filter) })),
        inspectUrl: this.inspectUrl(filter),
      };
    } catch {
      return undefined;
    }
  }

  private async request(path: string, init: Init = {}, allowed: number[] = []): Promise<Response> {
    const url = `${this.baseUrl}/api/v1${path}`;
    const { timeout = this.requestTimeout, signal: userSignal, ...rest } = init;
    const timer = AbortSignal.timeout(timeout);
    const signal = userSignal ? AbortSignal.any([userSignal, timer]) : timer;
    const method = rest.method ?? "GET";
    let res: Response;
    try {
      const headers = { ...this.headers, ...(rest.headers as Record<string, string> | undefined) };
      res = await this.fetchImpl(url, { ...rest, signal, headers });
    } catch (err) {
      if (userSignal?.aborted) throw userSignal.reason ?? err;
      if (timer.aborted) {
        throw new MailpeekError(`Mailpeek did not answer ${method} ${path} within ${timeout}ms`);
      }
      const reason =
        err instanceof Error
          ? err.cause instanceof Error
            ? err.cause.message
            : err.message
          : String(err);
      throw new MailpeekError(
        `Could not reach Mailpeek at ${this.baseUrl} (${reason}). Is it running?`,
      );
    }
    if (res.ok || allowed.includes(res.status)) return res;
    if (res.status === 401) {
      throw new MailpeekError(
        `Mailpeek at ${this.baseUrl} requires a login (it runs with --ui-auth). ` +
          "Pass the credentials in the URL (http://user:password@host:8026) or the auth option.",
        401,
      );
    }
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new MailpeekError(
      `Mailpeek ${method} ${path} failed: ${body.error ?? `HTTP ${res.status}`}`,
      res.status,
    );
  }
}

function basicAuth(username: string, password: string): string {
  const bytes = new TextEncoder().encode(`${username}:${password}`);
  return `Basic ${btoa(String.fromCharCode(...bytes))}`;
}
