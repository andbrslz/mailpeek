import { describe, expect, it } from "vitest";
import { Email } from "../src/index.js";
import { messageData } from "./fixtures.js";

describe("Email", () => {
  const email = new Email(messageData());

  it("exposes plain data fields", () => {
    expect(email.subject).toBe("Welcome");
    expect(email.text).toContain("Hello John");
    expect(email.from.address).toBe("hello@acme.com");
    expect(email.createdAt).toBeInstanceOf(Date);
    expect(email.date?.toISOString()).toBe("2024-05-01T10:00:00.000Z");
    expect(email.links).toContainEqual(expect.objectContaining({ text: "Activate account" }));
    expect(email).toMatchObject({ subject: "Welcome", to: [{ address: "john@example.com" }] });
  });

  it("findLink prefers exact text, then substring, then href", () => {
    expect(email.findLink("Activate account").href).toBe("http://localhost:3000/activate/abc");
    expect(email.findLink("  activate   ACCOUNT ").href).toBe("http://localhost:3000/activate/abc");
    expect(email.findLink("account now").href).toBe("http://localhost:3000/activate/other");
    expect(email.findLink("unsubscribe").href).toBe("http://localhost:3000/unsubscribe/xyz");
    expect(email.findLink(/activate\/other/).text).toBe("Activate your account now");
    expect(email.findLink((l) => l.text === "").href).toContain("unsubscribe");
  });

  it("findLink with a global RegExp is not affected by lastIndex", () => {
    const re = /activate/g;
    expect(email.findLink(re).href).toContain("/activate/abc");
    expect(email.findLink(re).href).toContain("/activate/abc");
  });

  it("findLink throws a descriptive error", () => {
    expect(() => email.findLink("Reset password")).toThrow(/No link matching "Reset password"/);
    expect(() => email.findLink("Reset password")).toThrow(/activate\/abc/);
  });

  it("hasLink", () => {
    expect(email.hasLink("Activate account")).toBe(true);
    expect(email.hasLink("Reset")).toBe(false);
  });

  it("getHeader is case-insensitive", () => {
    expect(email.getHeader("subject")).toBe("Welcome");
    expect(email.getHeader("x-trace-id")).toBe("t-1");
    expect(email.getHeaders("X-TRACE-ID")).toEqual(["t-1", "t-2"]);
    expect(email.getHeader("missing")).toBeUndefined();
  });

  it("hasAttachment", () => {
    expect(email.hasAttachment()).toBe(true);
    expect(email.hasAttachment("invoice.pdf")).toBe(true);
    expect(email.hasAttachment(/\.pdf$/i)).toBe(true);
    expect(email.hasAttachment("other.pdf")).toBe(false);
    expect(new Email(messageData({ attachments: [] })).hasAttachment()).toBe(false);
  });

  it("serializes compactly", () => {
    expect(JSON.parse(JSON.stringify(email))).toMatchObject({ id: "abc123", subject: "Welcome" });
  });
});

describe("findCode", () => {
  const withText = (text: string, html = "") => new Email(messageData({ text, html }));

  it("prefers a code next to a keyword", () => {
    expect(withText("Order 2024 confirmed. Your verification code is 482913.").findCode()).toBe(
      "482913",
    );
    expect(withText("Seu código de acesso: 7351").findCode()).toBe("7351");
    expect(withText("Use this one-time passcode: AB12-CD34 to sign in").findCode()).toBe(
      "AB12-CD34",
    );
    expect(withText("PIN 0042 expires in 10 minutes").findCode()).toBe("0042");
  });

  it("falls back to a 6-digit number, and reads HTML-only emails", () => {
    expect(withText("Hi! 123456 is what you need.").findCode()).toBe("123456");
    expect(withText("", "<p>Your code:</p><p><b>918273</b></p>").findCode()).toBe("918273");
  });

  it("accepts a custom pattern", () => {
    expect(withText("token=abc-xyz;").findCode({ pattern: /token=([a-z-]+)/ })).toBe("abc-xyz");
  });

  it("throws with a preview when there is no code", () => {
    expect(() => withText("Welcome aboard, John").findCode()).toThrow(
      /No verification code found.*Welcome aboard/,
    );
  });
});
