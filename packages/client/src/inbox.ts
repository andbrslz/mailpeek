import type { Email } from "./email.js";
import type {
  FailureOptions,
  MessageFilter,
  MessageSummary,
  RequestOptions,
  SmtpFailure,
  WaitOptions,
} from "./types.js";

/** The subset of the client an Inbox needs. */
export interface InboxClient {
  messages(filter?: MessageFilter, options?: RequestOptions): Promise<MessageSummary[]>;
  latest(filter?: MessageFilter, options?: RequestOptions): Promise<Email | undefined>;
  waitFor(options?: WaitOptions): Promise<Email>;
  waitForEmails(count: number, options?: WaitOptions): Promise<Email[]>;
  clear(filter?: MessageFilter, options?: RequestOptions): Promise<number>;
  failNext(failure?: FailureOptions, options?: RequestOptions): Promise<SmtpFailure>;
  clearFailures(address?: string, options?: RequestOptions): Promise<number>;
}

type InboxFilter = Omit<MessageFilter, "address" | "to">;
type InboxWaitOptions = Omit<WaitOptions, "address" | "to">;

export interface InboxOptions {
  /** Local-part prefix (default "test"). */
  prefix?: string;
  /** Domain (default: the client's inboxDomain, "mailpeek.local"). */
  domain?: string;
}

const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789";

/** Random lowercase alphanumeric string. */
export function randomId(length = 8): string {
  const bytes = new Uint8Array(length);
  globalThis.crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => alphabet[b % alphabet.length]).join("");
}

/**
 * A unique address that isolates one test's emails from every other test.
 * Nothing is created on the server: the inbox is an exact match on the
 * recipient address (To, Cc or the SMTP envelope).
 */
export class Inbox {
  constructor(
    private readonly client: InboxClient,
    readonly address: string,
  ) {}

  messages(filter: InboxFilter = {}, options?: RequestOptions): Promise<MessageSummary[]> {
    return this.client.messages({ ...filter, address: this.address }, options);
  }

  latest(filter: InboxFilter = {}, options?: RequestOptions): Promise<Email | undefined> {
    return this.client.latest({ ...filter, address: this.address }, options);
  }

  /** Waits for an email sent to this inbox. */
  waitForEmail(options: InboxWaitOptions = {}): Promise<Email> {
    return this.client.waitFor({ ...options, address: this.address });
  }

  /** Waits for `count` emails sent to this inbox; returns them in arrival order. */
  waitForEmails(count: number, options: InboxWaitOptions = {}): Promise<Email[]> {
    return this.client.waitForEmails(count, { ...options, address: this.address });
  }

  /**
   * Makes the next `count` deliveries to this inbox fail (default: one, with
   * 451), so the application's retry or error handling can be tested. Other
   * inboxes are not affected.
   */
  failNext(
    failure: Omit<FailureOptions, "address"> = {},
    options?: RequestOptions,
  ): Promise<SmtpFailure> {
    return this.client.failNext({ ...failure, address: this.address }, options);
  }

  /** Deletes this inbox's emails (and pending failures) and returns how many emails were removed. */
  async clear(options?: RequestOptions): Promise<number> {
    const [deleted] = await Promise.all([
      this.client.clear({ address: this.address }, options),
      this.client.clearFailures(this.address, options),
    ]);
    return deleted;
  }

  toString(): string {
    return this.address;
  }
}
