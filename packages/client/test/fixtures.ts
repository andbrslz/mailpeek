import type { MessageData } from "../src/index.js";

export function messageData(overrides: Partial<MessageData> = {}): MessageData {
  return {
    id: "abc123",
    from: { name: "Acme", address: "hello@acme.com" },
    to: [{ address: "john@example.com" }],
    cc: [],
    replyTo: [],
    subject: "Welcome",
    date: "2024-05-01T10:00:00Z",
    text: "Hello John",
    html: "<p>Hello John</p>",
    headers: { Subject: ["Welcome"], "X-Trace-Id": ["t-1", "t-2"] },
    attachments: [
      { id: "1", filename: "Invoice.pdf", contentType: "application/pdf", inline: false, size: 10 },
    ],
    links: [
      { text: "Activate account", href: "http://localhost:3000/activate/abc" },
      { text: "Activate your account now", href: "http://localhost:3000/activate/other" },
      { text: "", href: "http://localhost:3000/unsubscribe/xyz" },
    ],
    envelope: { from: "bounce@acme.com", to: ["john@example.com"] },
    size: 1234,
    createdAt: "2024-05-01T10:00:01Z",
    ...overrides,
  };
}
