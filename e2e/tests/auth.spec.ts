import { Mailpeek, MailpeekError } from "@mailpeek-dev/client";
import { expect, test } from "@playwright/test";
import nodemailer from "nodemailer";
import { liveEvents } from "./live";
import { authHttpPort, authSmtpPort, smtpCredentials, uiCredentials } from "./ports";

const base = `http://127.0.0.1:${authHttpPort}`;
const { username, password } = uiCredentials;
const mailpeek = new Mailpeek({
  baseUrl: `http://${username}:${password}@127.0.0.1:${authHttpPort}`,
});

function transport(auth?: { user: string; pass: string }) {
  return nodemailer.createTransport({ host: "127.0.0.1", port: authSmtpPort, secure: false, auth });
}

test("SMTP rejects missing or wrong credentials and accepts the right ones", async () => {
  const inbox = mailpeek.createInbox();
  const message = { from: "app@acme.test", to: inbox.address, subject: "Locked", text: "x" };

  await expect(transport().sendMail(message)).rejects.toThrow(/530/);
  await expect(transport({ user: "app", pass: "wrong" }).sendMail(message)).rejects.toThrow(/535/);
  await transport(smtpCredentials).sendMail(message);

  const email = await inbox.waitForEmail({ subject: "Locked" });
  expect(email.text).toContain("x");
  expect(await inbox.messages()).toHaveLength(1);
});

test("API requires credentials, except the health check", async ({ request }) => {
  const anonymous = new Mailpeek({ baseUrl: base });
  const err = await anonymous.messages().catch((e: unknown) => e);
  expect(err).toBeInstanceOf(MailpeekError);
  expect((err as MailpeekError).status).toBe(401);
  expect(await anonymous.health()).toBe(true);

  const wrong = new Mailpeek({ baseUrl: base, auth: { username, password: "nope" } });
  await expect(wrong.messages()).rejects.toThrow(/requires a login/);

  const denied = await request.get(`${base}/api/v1/messages`);
  expect(denied.status()).toBe(401);
  expect(denied.headers()["www-authenticate"]).toBeUndefined(); // never the browser's dialog
  expect(Array.isArray(await mailpeek.messages())).toBe(true);
});

test("Web UI login screen, live updates behind it, and sign out", async ({ page }) => {
  const inbox = mailpeek.createInbox();

  // Any page sends the browser to the login screen (no native Basic dialog).
  await page.goto(`${base}/?from=link`);
  await expect(page).toHaveURL(/\/login\?next=/);
  await expect(page.getByRole("heading", { name: "Sign in" })).toBeVisible();

  await page.getByLabel("Username").fill(username);
  await page.getByLabel("Password").fill("wrong");
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page.getByRole("alert")).toHaveText(/Wrong username or password/);
  await expect(page.getByLabel("Username")).toHaveValue(username);

  await page.getByLabel("Password").fill(password);
  const live = liveEvents(page);
  await page.getByRole("button", { name: "Sign in" }).click();
  await expect(page).toHaveURL(`${base}/?from=link`); // back where we started
  await live;
  await page.getByRole("searchbox", { name: "Search emails" }).fill(inbox.address);

  await transport(smtpCredentials).sendMail({
    from: "app@acme.test",
    to: inbox.address,
    subject: "Behind the login",
    text: "y",
  });
  await expect(
    page.getByRole("listbox", { name: "Inbox" }).getByRole("option", { name: /Behind the login/ }),
  ).toBeVisible();
  await expect(page.getByRole("status")).toHaveText("Live");
  await expect(page.getByRole("img", { name: "SMTP login required" })).toBeVisible();

  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/login$/);
  expect((await page.request.get(`${base}/api/v1/messages`)).status()).toBe(401);
});
