import type { MessageFilter, MessageSummary } from "./types.js";

const includes = (value: string | undefined, needle: string) =>
  (value ?? "").toLowerCase().includes(needle.trim().toLowerCase());

/**
 * Why a message does not match a filter, mirroring the server's rules.
 * An empty list means it matches. Summaries carry no body, so this is meant
 * for messages the server already did not match: when every other filter
 * passes, a `body` filter is reported as the reason.
 */
export function explainMismatch(m: MessageSummary, f: MessageFilter): string[] {
  const reasons: string[] = [];
  const recipients = [...m.to, ...m.cc];
  const envelopeTo = m.envelope?.to ?? [];

  if (f.address) {
    const wanted = f.address.trim().toLowerCase();
    const all = [...recipients.map((a) => a.address), ...envelopeTo];
    if (!all.some((a) => a.toLowerCase() === wanted)) {
      reasons.push(`not sent to ${JSON.stringify(f.address)}`);
    }
  }
  const { to, q, from, subject } = f;
  if (to) {
    const hit =
      recipients.some((a) => includes(a.address, to) || includes(a.name, to)) ||
      envelopeTo.some((a) => includes(a, to));
    if (!hit) reasons.push(`no recipient contains ${JSON.stringify(to)}`);
  }
  if (
    from &&
    !includes(m.from.address, from) &&
    !includes(m.from.name, from) &&
    !includes(m.envelope?.from, from)
  ) {
    reasons.push(`from ${m.from.address || "(none)"}, not ${JSON.stringify(from)}`);
  }
  if (subject && !includes(m.subject, subject)) {
    reasons.push(`subject does not contain ${JSON.stringify(subject)}`);
  }
  if (q) {
    const hit =
      includes(m.subject, q) ||
      includes(m.from.address, q) ||
      includes(m.from.name, q) ||
      recipients.some((a) => includes(a.address, q) || includes(a.name, q));
    if (!hit) reasons.push(`does not match ${JSON.stringify(q)} (subject, from, to or body)`);
  }
  if (f.since !== undefined) {
    const since = f.since instanceof Date ? f.since.getTime() : new Date(f.since).getTime();
    if (Date.parse(m.createdAt) < since) reasons.push("received before `since`");
  }
  if (f.body && reasons.length === 0) {
    reasons.push(`body does not contain ${JSON.stringify(f.body)}`);
  }
  return reasons;
}
