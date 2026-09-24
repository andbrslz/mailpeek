import type { Address, Attachment, Link, MessageData } from "./types.js";

/** A string (matched against link text, then href), a RegExp or a predicate. */
export type LinkMatcher = string | RegExp | ((link: Link) => boolean);

const normalize = (s: string) => s.replace(/\s+/g, " ").trim().toLowerCase();

function matchLinks(links: Link[], matcher: LinkMatcher): Link | undefined {
  if (typeof matcher === "function") return links.find(matcher);
  if (matcher instanceof RegExp) {
    return links.find((l) => {
      matcher.lastIndex = 0;
      if (matcher.test(l.text)) return true;
      matcher.lastIndex = 0;
      return matcher.test(l.href);
    });
  }
  const needle = normalize(matcher);
  return (
    links.find((l) => normalize(l.text) === needle) ??
    links.find((l) => normalize(l.text).includes(needle)) ??
    links.find((l) => l.href.toLowerCase().includes(needle))
  );
}

/**
 * A received email with helpers for tests. All data fields are plain
 * properties, so they work directly with Playwright, Vitest and Jest
 * `expect`.
 */
export class Email {
  readonly id: string;
  readonly messageId?: string;
  readonly from: Address;
  readonly to: Address[];
  readonly cc: Address[];
  readonly replyTo: Address[];
  readonly subject: string;
  readonly text: string;
  readonly html: string;
  readonly headers: Record<string, string[]>;
  readonly attachments: Attachment[];
  readonly links: Link[];
  readonly envelope: { from: string; to: string[] };
  readonly size: number;
  /** Value of the Date header, when present and valid. */
  readonly date?: Date;
  /** When Mailpeek received the message. */
  readonly createdAt: Date;

  constructor(data: MessageData) {
    this.id = data.id;
    if (data.messageId) this.messageId = data.messageId;
    this.from = data.from;
    this.to = data.to ?? [];
    this.cc = data.cc ?? [];
    this.replyTo = data.replyTo ?? [];
    this.subject = data.subject;
    this.text = data.text ?? "";
    this.html = data.html ?? "";
    this.headers = data.headers ?? {};
    this.attachments = data.attachments ?? [];
    this.links = data.links ?? [];
    this.envelope = data.envelope ?? { from: "", to: [] };
    this.size = data.size;
    if (data.date) this.date = new Date(data.date);
    this.createdAt = new Date(data.createdAt);
  }

  /**
   * Returns the first matching link. A string matches link text exactly
   * (case- and whitespace-insensitive), then as a substring, then as a
   * substring of the href. Throws a descriptive error when nothing matches;
   * use `hasLink` for a boolean check.
   */
  findLink(matcher: LinkMatcher): Link {
    const link = matchLinks(this.links, matcher);
    if (link) return link;
    const available = this.links.length
      ? this.links.map((l) => `  - ${JSON.stringify(l.text)} -> ${l.href}`).join("\n")
      : "  (none)";
    const label = typeof matcher === "function" ? "predicate" : String(matcher);
    throw new Error(
      `No link matching ${JSON.stringify(label)} in email ${JSON.stringify(this.subject)} (${this.id}).\nLinks:\n${available}`,
    );
  }

  hasLink(matcher: LinkMatcher): boolean {
    return matchLinks(this.links, matcher) !== undefined;
  }

  /** First value of a header, case-insensitive. */
  getHeader(name: string): string | undefined {
    return this.getHeaders(name)[0];
  }

  /** All values of a header, case-insensitive. */
  getHeaders(name: string): string[] {
    const wanted = name.toLowerCase();
    for (const [key, values] of Object.entries(this.headers)) {
      if (key.toLowerCase() === wanted) return values;
    }
    return [];
  }

  /**
   * Whether the email has an attachment. With a string, matches the
   * filename exactly (case-insensitive); with a RegExp, tests the filename.
   */
  hasAttachment(matcher?: string | RegExp): boolean {
    if (matcher === undefined) return this.attachments.length > 0;
    return this.attachments.some((a) =>
      typeof matcher === "string"
        ? a.filename.toLowerCase() === matcher.toLowerCase()
        : matcher.test(a.filename),
    );
  }

  /**
   * Extracts a verification code (one-time password, PIN, sign-in code).
   * Prefers a 4–10 character code right after words such as "code",
   * "código", "OTP", "PIN" or "verification"; otherwise the first 6-digit
   * number, then the first 4–8 digit number. Pass `pattern` for other
   * formats (its first capture group is returned, or the whole match).
   * Throws when nothing looks like a code.
   */
  findCode(options: { pattern?: RegExp } = {}): string {
    const text = this.text.trim() ? this.text : htmlToText(this.html);
    if (options.pattern) {
      const m = text.match(options.pattern);
      if (m) return m[1] ?? m[0];
    } else {
      const code = guessCode(text);
      if (code) return code;
    }
    const preview = text.replace(/\s+/g, " ").trim().slice(0, 160);
    throw new Error(
      `No verification code found in email ${JSON.stringify(this.subject)} (${this.id}). Text: ${JSON.stringify(preview)}`,
    );
  }

  toJSON() {
    return {
      id: this.id,
      from: this.from,
      to: this.to,
      subject: this.subject,
      text: this.text,
      links: this.links,
      attachments: this.attachments,
    };
  }
}

const keyword =
  /\b(?:verification|verify|confirmation|one[- ]time|passcode|code|c[oó]digo|otp|pin|token)\b/gi;
// Digits, or an uppercase code that contains at least one digit.
const codeToken = /\b(\d{4,10}|(?=[A-Z0-9-]*\d)[A-Z0-9][A-Z0-9-]{2,10}[A-Z0-9])\b/;

function guessCode(text: string): string | undefined {
  for (const k of text.matchAll(keyword)) {
    const after = text.slice((k.index ?? 0) + k[0].length, (k.index ?? 0) + k[0].length + 60);
    const m = after.match(codeToken);
    if (m?.[1]) return m[1];
  }
  return text.match(/\b(\d{6})\b/)?.[1] ?? text.match(/\b(\d{4,8})\b/)?.[1];
}

function htmlToText(html: string): string {
  return html
    .replace(/<(style|script|head)[\s\S]*?<\/\1>/gi, " ")
    .replace(/<br\s*\/?>|<\/(p|div|tr|h\d|li)>/gi, "\n")
    .replace(/<[^>]+>/g, " ")
    .replace(/&nbsp;/g, " ")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'");
}
