import { afterEach, describe, expect, it, vi } from "vitest";
import { explainMismatch, Mailpeek, MailpeekError, MailpeekTimeoutError } from "../src/index.js";
import { messageData } from "./fixtures.js";

type Handler = (url: URL, init: RequestInit) => Response | Promise<Response>;

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { "Content-Type": "application/json" } });

function fakeFetch(handler: Handler) {
  const calls: { url: URL; init: RequestInit }[] = [];
  const fn = vi.fn(async (input: string | URL | Request, init: RequestInit = {}) => {
    const url = new URL(String(input));
    calls.push({ url, init });
    return handler(url, init);
  });
  return { fetch: fn as unknown as typeof fetch, calls };
}

describe("Mailpeek", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("uses MAILPEEK_URL, then the default", () => {
    vi.stubEnv("MAILPEEK_URL", "");
    expect(new Mailpeek().baseUrl).toBe("http://localhost:8026");
    vi.stubEnv("MAILPEEK_URL", "http://mailpeek:9000/");
    expect(new Mailpeek().baseUrl).toBe("http://mailpeek:9000");
    expect(new Mailpeek({ baseUrl: "http://x:1" }).baseUrl).toBe("http://x:1");
  });

  it("lists messages with filters", async () => {
    const { fetch, calls } = fakeFetch(() => json({ messages: [{ id: "1" }], count: 1 }));
    const client = new Mailpeek({ fetch });
    const since = new Date("2024-01-01T00:00:00Z");
    const list = await client.messages({ to: "john@example.com", subject: "Welcome", since });
    expect(list).toEqual([{ id: "1" }]);
    const url = calls[0]!.url;
    expect(url.pathname).toBe("/api/v1/messages");
    expect(url.searchParams.get("to")).toBe("john@example.com");
    expect(url.searchParams.get("subject")).toBe("Welcome");
    expect(url.searchParams.get("since")).toBe("2024-01-01T00:00:00.000Z");
  });

  it("latest returns an Email or undefined", async () => {
    let found = true;
    const { fetch } = fakeFetch(() => (found ? json(messageData()) : json({ error: "none" }, 404)));
    const client = new Mailpeek({ fetch });
    const email = await client.latest({ to: "john" });
    expect(email?.findLink("Activate account").href).toContain("/activate/abc");
    found = false;
    expect(await client.latest()).toBeUndefined();
  });

  it("waitFor returns the message", async () => {
    const { fetch, calls } = fakeFetch(() => json(messageData()));
    const email = await new Mailpeek({ fetch }).waitFor({
      to: "john@example.com",
      subject: "Welcome",
      timeout: 5000,
    });
    expect(email.subject).toBe("Welcome");
    const url = calls[0]!.url;
    expect(url.pathname).toBe("/api/v1/messages/wait");
    expect(url.searchParams.get("timeout")).toBe("5000");
  });

  it("waitFor splits long timeouts into server-sized chunks", async () => {
    let n = 0;
    const { fetch, calls } = fakeFetch(() =>
      n++ === 0 ? new Response(null, { status: 204 }) : json(messageData()),
    );
    const email = await new Mailpeek({ fetch }).waitFor({ timeout: 45_000 });
    expect(email.id).toBe("abc123");
    expect(calls[0]!.url.searchParams.get("timeout")).toBe("30000");
    expect(calls).toHaveLength(2);
  });

  it("waitFor throws MailpeekTimeoutError on 204 at the deadline", async () => {
    const { fetch } = fakeFetch(() => new Response(null, { status: 204 }));
    const err = await new Mailpeek({ fetch })
      .waitFor({ to: "nobody@example.com", timeout: 0 })
      .catch((e: unknown) => e);
    expect(err).toBeInstanceOf(MailpeekTimeoutError);
    expect(String(err)).toMatch(/to="nobody@example.com".*0ms/);
  });

  it("clear, delete, raw, attachment and health", async () => {
    const { fetch, calls } = fakeFetch((url, init) => {
      if (init.method === "DELETE" && url.pathname === "/api/v1/messages")
        return json({ deleted: 3 });
      if (init.method === "DELETE") return new Response(null, { status: 204 });
      if (url.pathname.endsWith("/raw")) return new Response("From: a@b\r\n\r\nhi");
      if (url.pathname.includes("/attachments/")) return new Response(new Uint8Array([1, 2, 3]));
      return json({ status: "ok" });
    });
    const client = new Mailpeek({ fetch });
    expect(await client.clear({ to: "x@y" })).toBe(3);
    expect(calls[0]!.url.search).toBe("?to=x%40y");
    await client.delete("a/b");
    expect(calls[1]!.url.pathname).toBe("/api/v1/messages/a%2Fb");
    expect(await client.raw("1")).toContain("From: a@b");
    expect(Array.from(await client.attachment("1", "2"))).toEqual([1, 2, 3]);
    expect(await client.health()).toBe(true);
  });

  it("wraps API errors and connection failures", async () => {
    const notFound = fakeFetch(() => json({ error: "message not found" }, 404));
    const err = await new Mailpeek({ fetch: notFound.fetch }).get("x").catch((e: unknown) => e);
    expect(err).toBeInstanceOf(MailpeekError);
    expect((err as MailpeekError).status).toBe(404);
    expect(String(err)).toContain("message not found");

    const down = fakeFetch(() => {
      throw new TypeError("fetch failed", {
        cause: new Error("connect ECONNREFUSED 127.0.0.1:8026"),
      });
    });
    const client = new Mailpeek({ fetch: down.fetch });
    await expect(client.messages()).rejects.toThrow(
      /Could not reach Mailpeek at http:\/\/localhost:8026 \(connect ECONNREFUSED/,
    );
    expect(await client.health()).toBe(false);
  });
});

describe("credentials", () => {
  it("moves credentials from the URL into an Authorization header", async () => {
    const { fetch, calls } = fakeFetch(() => json({ messages: [] }));
    const client = new Mailpeek({ baseUrl: "http://admin:p%40ss%3Aw0rd@mailpeek:8026/", fetch });
    expect(client.baseUrl).toBe("http://mailpeek:8026");
    await client.messages();
    expect(calls[0]!.url.href).toBe("http://mailpeek:8026/api/v1/messages");
    const header = (calls[0]!.init.headers as Record<string, string>).Authorization;
    expect(header).toBe(`Basic ${btoa("admin:p@ss:w0rd")}`);
  });

  it("accepts an auth option, including non-ASCII passwords", async () => {
    const { fetch, calls } = fakeFetch(() => json({ messages: [] }));
    await new Mailpeek({ auth: { username: "joão", password: "sênha" }, fetch }).messages();
    const header = (calls[0]!.init.headers as Record<string, string>).Authorization!;
    const decoded = new TextDecoder().decode(
      Uint8Array.from(atob(header.slice(6)), (c) => c.charCodeAt(0)),
    );
    expect(decoded).toBe("joão:sênha");
  });

  it("sends no header without credentials and explains a 401", async () => {
    const { fetch, calls } = fakeFetch(() => json({ error: "authentication required" }, 401));
    const err = await new Mailpeek({ fetch }).messages().catch((e: unknown) => e);
    expect(calls[0]!.init.headers).toEqual({});
    expect(err).toBeInstanceOf(MailpeekError);
    expect(String(err)).toMatch(/requires a login.*http:\/\/user:password@host:8026/);
  });
});

describe("Inbox", () => {
  it("generates unique addresses", () => {
    const client = new Mailpeek({ fetch: fakeFetch(() => json({})).fetch });
    const a = client.createInbox();
    const b = client.createInbox({ prefix: "signup", domain: "example.test" });
    expect(a.address).toMatch(/^test-[a-z0-9]{8}@mailpeek\.local$/);
    expect(b.address).toMatch(/^signup-[a-z0-9]{8}@example\.test$/);
    expect(a.address).not.toBe(client.createInbox().address);
    expect(`${a}`).toBe(a.address);
  });

  it("scopes every call to its address", async () => {
    const { fetch, calls } = fakeFetch((url, init) => {
      if (init.method === "DELETE") return json({ deleted: 1 });
      if (url.pathname.endsWith("/messages")) return json({ messages: [] });
      return json(messageData());
    });
    const inbox = new Mailpeek({ fetch }).createInbox();
    await inbox.waitForEmail({ subject: "Reset your password" });
    await inbox.messages();
    await inbox.latest();
    await inbox.clear();
    for (const { url } of calls) {
      expect(url.searchParams.get("address")).toBe(inbox.address); // exact match
      expect(url.searchParams.get("to")).toBeNull();
    }
    expect(calls[0]!.url.searchParams.get("subject")).toBe("Reset your password");
  });
});

describe("timeouts and cancellation", () => {
  it("applies the transport timeout to every request", async () => {
    const hang: typeof fetch = (_input, init) =>
      new Promise((_resolve, reject) => {
        init?.signal?.addEventListener("abort", () => reject(init.signal?.reason));
      });
    const client = new Mailpeek({ fetch: hang, requestTimeout: 50 });
    await expect(client.messages()).rejects.toThrow(/did not answer GET \/messages within 50ms/);
    await expect(client.clear()).rejects.toThrow(/DELETE \/messages/);
  });

  it("lets callers cancel any operation", async () => {
    const hang: typeof fetch = (_input, init) =>
      new Promise((_resolve, reject) => {
        init?.signal?.addEventListener("abort", () => reject(init.signal?.reason));
      });
    const ac = new AbortController();
    const pending = new Mailpeek({ fetch: hang }).get("x", { signal: ac.signal });
    ac.abort(new Error("stop"));
    await expect(pending).rejects.toThrow("stop");
  });

  it("gives waitFor a transport deadline longer than the wait itself", async () => {
    const { fetch, calls } = fakeFetch(() => json(messageData()));
    await new Mailpeek({ fetch, requestTimeout: 1234 }).waitFor({ timeout: 5000 });
    expect(calls[0]!.url.searchParams.get("timeout")).toBe("5000");
  });
});

describe("timeout diagnostics", () => {
  const summary = (subject: string, to: string, createdAt = "2024-05-01T10:00:00Z") => ({
    id: subject,
    from: { address: "app@acme.test" },
    to: [{ address: to }],
    cc: [],
    subject,
    envelope: { from: "app@acme.test", to: [to] },
    attachments: 0,
    size: 1,
    createdAt,
  });

  it("explains what arrived and why it did not match", async () => {
    const { fetch } = fakeFetch((url) => {
      if (url.pathname.endsWith("/wait")) return new Response(null, { status: 204 });
      if (url.pathname.endsWith("/info"))
        return json({ store: { messages: 2, bytes: 1, evicted: 3 } });
      return json({
        messages: [
          summary("Reset your password", "ana@x.test"),
          summary("Welcome", "joana@x.test"),
        ],
      });
    });
    const err = (await new Mailpeek({ fetch })
      .waitFor({ address: "ana@x.test", subject: "Welcome", timeout: 0 })
      .catch((e: unknown) => e)) as MailpeekTimeoutError;
    expect(err).toBeInstanceOf(MailpeekTimeoutError);
    expect(err.diagnostics?.total).toBe(2);
    expect(err.message).toContain('"Reset your password" to ana@x.test');
    expect(err.message).toContain('subject does not contain "Welcome"');
    expect(err.message).toContain('not sent to "ana@x.test"'); // joana@ is not ana@
    expect(err.message).toContain("3 message(s) were removed");
    expect(err.message).toContain("Inspect: http://localhost:8026/?q=ana%40x.test");
  });

  it("says so when Mailpeek received nothing at all", async () => {
    const { fetch } = fakeFetch((url) =>
      url.pathname.endsWith("/wait") ? new Response(null, { status: 204 }) : json({ messages: [] }),
    );
    const err = await new Mailpeek({ fetch }).waitFor({ timeout: 0 }).catch((e: unknown) => e);
    expect(String(err)).toMatch(/holds no messages at all/);
  });

  it("explainMismatch mirrors the server filters", () => {
    const m = summary("Invoice", "john@x.test", "2024-01-01T00:00:00Z");
    expect(explainMismatch(m, { to: "john" })).toEqual([]);
    expect(explainMismatch(m, { from: "billing" })[0]).toMatch(/from app@acme.test/);
    expect(explainMismatch(m, { since: new Date("2025-01-01") })).toEqual([
      "received before `since`",
    ]);
    expect(explainMismatch(m, { q: "nothing" })[0]).toMatch(/does not match/);
  });
});

describe("waitForEmails", () => {
  it("returns the first N matching emails in arrival order", async () => {
    let listCalls = 0;
    const { fetch, calls } = fakeFetch((url) => {
      if (url.pathname.endsWith("/messages")) {
        listCalls++;
        const newest = { id: "b", createdAt: "2024-05-01T10:00:02Z" };
        const oldest = { id: "a", createdAt: "2024-05-01T10:00:01Z" };
        return json({ messages: listCalls === 1 ? [oldest] : [newest, oldest] });
      }
      if (url.pathname.endsWith("/wait")) return json(messageData({ id: "b" }));
      return json(messageData({ id: url.pathname.split("/").pop()!, subject: url.pathname }));
    });
    const emails = await new Mailpeek({ fetch }).waitForEmails(2, {
      address: "x@y",
      timeout: 1000,
    });
    expect(emails.map((e) => e.id)).toEqual(["a", "b"]);
    const wait = calls.find((c) => c.url.pathname.endsWith("/wait"))!;
    expect(wait.url.searchParams.get("since")).toBe("2024-05-01T10:00:01.001Z");
  });

  it("reports how many arrived on timeout", async () => {
    const { fetch } = fakeFetch((url) => {
      if (url.pathname.endsWith("/wait")) return new Response(null, { status: 204 });
      if (url.pathname.endsWith("/info")) return json({ store: { evicted: 0 } });
      return json({
        messages: [
          { id: "a", createdAt: "2024-05-01T10:00:01Z", from: {}, to: [], cc: [], subject: "one" },
        ],
      });
    });
    const err = await new Mailpeek({ fetch })
      .waitForEmails(3, { timeout: 0 })
      .catch((e: unknown) => e);
    expect(err).toBeInstanceOf(MailpeekTimeoutError);
    expect(String(err)).toMatch(/Only 1 of 3 emails/);
  });
});
