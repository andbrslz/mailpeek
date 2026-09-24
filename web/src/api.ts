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

export interface MessageSummary {
  id: string;
  from: Address;
  to: Address[];
  cc: Address[];
  subject: string;
  attachments: number;
  size: number;
  createdAt: string;
}

export interface Message {
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

export interface Info {
  version: string;
  smtpPort: number;
  httpPort: number;
  maxMessages: number;
  maxMessageSize: number;
  smtpAuth: boolean;
  uiAuth: boolean;
}

const base = "/api/v1";

export class ApiError extends Error {
  readonly status: number;
  constructor(status: number, message = `HTTP ${status}`) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

let signingOut = false;

async function call(input: string, init?: RequestInit): Promise<Response> {
  const res = await fetch(input, init);
  if (res.status === 401 && !signingOut) {
    const next = window.location.pathname + window.location.search;
    window.location.assign(`/login?next=${encodeURIComponent(next)}`);
  }
  return res;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await call(base + path, init);
  if (!res.ok) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    throw new ApiError(res.status, body.error);
  }
  return (await res.json()) as T;
}

export interface MessagePage {
  messages: MessageSummary[];
  nextCursor?: string;
}

export function listMessages(
  query: string,
  page: { limit: number; cursor?: string },
  signal?: AbortSignal,
): Promise<MessagePage> {
  const params = new URLSearchParams({ limit: String(page.limit) });
  if (query) params.set("q", query);
  if (page.cursor) params.set("cursor", page.cursor);
  return request<MessagePage>(`/messages?${params}`, { signal });
}

export async function countMessages(query: string, signal?: AbortSignal): Promise<number> {
  const qs = query ? `?q=${encodeURIComponent(query)}` : "";
  return (await request<{ count: number }>(`/messages/count${qs}`, { signal })).count;
}

export const getMessage = (id: string) => request<Message>(`/messages/${encodeURIComponent(id)}`);

export const getInfo = () => request<Info>("/info");

export async function getRaw(id: string): Promise<string> {
  const res = await call(rawUrl(id));
  if (!res.ok) throw new ApiError(res.status);
  return res.text();
}

export async function deleteMessage(id: string): Promise<void> {
  const res = await call(`${base}/messages/${encodeURIComponent(id)}`, { method: "DELETE" });
  if (!res.ok && res.status !== 404) throw new ApiError(res.status);
}

export async function clearMessages(): Promise<void> {
  await request<{ deleted: number }>("/messages", { method: "DELETE" });
}

export async function signOut(): Promise<void> {
  signingOut = true;
  await fetch("/logout", { method: "POST" });
  window.location.assign("/login");
}

export const rawUrl = (id: string, download = false) =>
  `${base}/messages/${encodeURIComponent(id)}/raw${download ? "?download=1" : ""}`;

export const attachmentUrl = (messageId: string, attachmentId: string) =>
  `${base}/messages/${encodeURIComponent(messageId)}/attachments/${encodeURIComponent(attachmentId)}`;

const previewableTypes = new Set([
  "image/png",
  "image/jpeg",
  "image/gif",
  "image/webp",
  "image/avif",
  "image/bmp",
]);

export const canPreview = (a: Attachment) => previewableTypes.has(a.contentType.toLowerCase());

export const attachmentPreviewUrl = (messageId: string, attachmentId: string) =>
  `${attachmentUrl(messageId, attachmentId)}?inline=1`;

export const eventsUrl = `${base}/events`;
