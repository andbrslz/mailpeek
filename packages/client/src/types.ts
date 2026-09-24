export interface Address {
  name?: string;
  address: string;
}

export interface Attachment {
  id: string;
  filename: string;
  contentType: string;
  contentId?: string;
  inline: boolean;
  size: number;
}

export interface Link {
  text: string;
  href: string;
}

/** Lightweight message returned by listings. */
export interface MessageSummary {
  id: string;
  from: Address;
  to: Address[];
  cc: Address[];
  subject: string;
  /** SMTP envelope (includes Bcc recipients). */
  envelope: { from: string; to: string[] };
  /** Number of attachments. */
  attachments: number;
  size: number;
  createdAt: string;
}

/** Full message as returned by the REST API. */
export interface MessageData {
  id: string;
  messageId?: string;
  from: Address;
  to: Address[];
  cc: Address[];
  replyTo: Address[];
  subject: string;
  date?: string;
  text: string;
  html: string;
  headers: Record<string, string[]>;
  attachments: Attachment[];
  links: Link[];
  envelope: { from: string; to: string[] };
  size: number;
  createdAt: string;
}

/**
 * Selects messages. All fields are case-insensitive substring matches and
 * are combined with AND.
 */
export interface MessageFilter {
  /** To, Cc or SMTP envelope recipient (so Bcc recipients match too); substring. */
  to?: string;
  /**
   * Exact recipient address (To, Cc or envelope), case-insensitive. Inboxes use
   * this, so "ana@example.com" never matches "joana@example.com".
   */
  address?: string;
  /** From header or SMTP envelope sender. */
  from?: string;
  subject?: string;
  /** Text or HTML body. */
  body?: string;
  /** Matches subject, from, to or body. */
  q?: string;
  /** Only messages received at or after this moment. */
  since?: Date | string | number;
}

/** Makes upcoming SMTP deliveries fail; see `Mailpeek.failNext`. */
export interface FailureOptions {
  /** `"rcpt"` refuses the recipient; `"data"` (default) refuses the message after it was sent. */
  stage?: "rcpt" | "data";
  /** SMTP reply code, 400-599 (default 451, a temporary failure). */
  code?: number;
  /** Reply text after the code (single line). */
  message?: string;
  /** Only deliveries to this exact address (default: any recipient). */
  address?: string;
  /** How many deliveries fail (default 1). */
  count?: number;
}

/** A pending simulated SMTP failure. */
export interface SmtpFailure {
  id: string;
  stage: "rcpt" | "data";
  code: number;
  message: string;
  address?: string;
  /** Deliveries it still applies to; the rule is removed at zero. */
  remaining: number;
}

export interface RequestOptions {
  /** Cancels the operation. */
  signal?: AbortSignal;
}

export interface WaitOptions extends MessageFilter, RequestOptions {
  /** Milliseconds to wait for the email. Defaults to the client timeout (10s). */
  timeout?: number;
}

export interface ServerInfo {
  version: string;
  smtpPort: number;
  httpPort: number;
  maxMessages: number;
  maxMessageSize: number;
  maxStoreSize: number;
  smtpAuth: boolean;
  uiAuth: boolean;
  store: { messages: number; bytes: number; evicted: number };
}

export interface MailpeekOptions {
  /**
   * Mailpeek HTTP address. Defaults to MAILPEEK_URL or http://localhost:8026.
   * Credentials may be embedded: http://user:password@localhost:8026.
   */
  baseUrl?: string;
  /** Credentials when Mailpeek runs with --ui-auth. Overrides any in baseUrl. */
  auth?: { username: string; password: string };
  /** Default waitFor deadline in milliseconds (10000): how long to wait for an email. */
  timeout?: number;
  /** Transport timeout in milliseconds for each HTTP request (10000). */
  requestTimeout?: number;
  /** Domain used by createInbox() (mailpeek.local). */
  inboxDomain?: string;
  /** Custom fetch implementation. */
  fetch?: typeof fetch;
}
