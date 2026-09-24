// Regenerates the screenshots in docs/. Run with `make screenshots`, which
// starts Mailpeek on the default ports (1026/8026) so the images show the
// addresses a new user will see.
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";

const require = createRequire(import.meta.url);
const { chromium } = require("@playwright/test");
const nodemailer = require("nodemailer");

const DOCS = fileURLToPath(new URL("../docs", import.meta.url));
const BASE = process.env.MAILPEEK_URL ?? "http://localhost:8026";
const t = nodemailer.createTransport({
  host: "127.0.0.1",
  port: Number(process.env.MAILPEEK_SMTP_PORT ?? 1026),
  secure: false,
});
const send = (o) => t.sendMail({ from: '"Acme" <hello@acme.com>', to: "john@example.com", ...o });

const welcome = `<div style="font-family:-apple-system,Segoe UI,Helvetica,sans-serif;max-width:520px;margin:32px auto;padding:32px;border:1px solid #e4e4e7;border-radius:12px;background:#fff">
  <div style="font-weight:700;font-size:15px;margin-bottom:24px">▲ Acme</div>
  <h1 style="margin:0 0 12px;font-size:22px;color:#18181b">Welcome to Acme</h1>
  <p style="color:#3f3f46;line-height:1.6;margin:0 0 8px">Hello John, thanks for signing up. Confirm your email address to activate your account.</p>
  <p style="margin:24px 0"><a href="http://localhost:3000/activate/abc" style="background:#18181b;color:#fff;padding:10px 16px;border-radius:8px;text-decoration:none;font-size:14px">Activate account</a></p>
  <p style="color:#71717a;font-size:13px;line-height:1.5">If you didn't create an account, you can ignore this email.<br><a href="http://localhost:3000/unsubscribe/xyz" style="color:#71717a">Unsubscribe</a></p></div>`;

async function seed() {
  await fetch(`${BASE}/api/v1/messages`, { method: "DELETE" });
  await send({
    subject: "Your weekly report",
    from: '"Acme Reports" <reports@acme.com>',
    text: "Here is your weekly summary.",
  });
  await send({
    subject: "Invoice #123",
    from: '"Acme Billing" <billing@acme.com>',
    text: "Your invoice is attached.",
    attachments: [
      { filename: "invoice.pdf", content: Buffer.alloc(126976, 1), contentType: "application/pdf" },
    ],
  });
  await send({
    subject: "Reset your password",
    from: '"Acme" <no-reply@acme.com>',
    text: "Reset: http://localhost:3000/reset/xyz",
    html: `<p>Someone asked to reset your password.</p><p><a href="http://localhost:3000/reset/xyz">Reset password</a></p>`,
  });
  await send({
    subject: "Welcome to Acme",
    cc: "team@acme.com",
    text: "Hello John, thanks for signing up. Activate: http://localhost:3000/activate/abc",
    html: welcome,
  });
}

const browser = await chromium.launch();
const shot = (page, name) => page.screenshot({ path: `${DOCS}/${name}.png` });

async function open(opts) {
  const ctx = await browser.newContext({
    viewport: { width: 1280, height: 760 },
    deviceScaleFactor: 2,
    ...opts,
  });
  const page = await ctx.newPage();
  await page.goto(BASE);
  await page
    .getByRole("listbox", { name: "Inbox" })
    .getByRole("option", { name: /Welcome to Acme/ })
    .click();
  await page.getByRole("heading", { name: "Welcome to Acme" }).waitFor();
  await page.frameLocator('iframe[title="HTML preview"]').getByText("Activate account").waitFor();
  return page;
}

if (process.env.MAILPEEK_AUTH !== "1") {
  await seed();

  // Main view, light and dark. A message arriving while the page is open shows the "new" dot.
  for (const colorScheme of ["light", "dark"]) {
    const page = await open({ colorScheme });
    if (colorScheme === "light") {
      await send({
        subject: "Your order has shipped",
        from: '"Acme Store" <orders@acme.com>',
        text: "Track your package.",
      });
      await page
        .getByRole("listbox", { name: "Inbox" })
        .getByRole("option", { name: /Your order has shipped/ })
        .waitFor();
      await page.waitForTimeout(300);
    }
    await shot(page, `screenshot-${colorScheme}`);
    await page.context().close();
  }

  // Links tab
  {
    const page = await open({});
    await page.getByRole("tab", { name: /Links/ }).click();
    await page.waitForTimeout(250);
    await shot(page, "screenshot-links");
    // Attachments tab
    await page
      .getByRole("listbox", { name: "Inbox" })
      .getByRole("option", { name: /Invoice #123/ })
      .click();
    await page.getByRole("tab", { name: /Attachments/ }).click();
    await page.waitForTimeout(250);
    await shot(page, "screenshot-attachments");
    await page.context().close();
  }

  // Raw tab with MIME syntax highlighting (light and dark)
  for (const colorScheme of ["light", "dark"]) {
    const page = await open({ colorScheme });
    await page
      .getByRole("listbox", { name: "Inbox" })
      .getByRole("option", { name: /Invoice #123/ })
      .click();
    await page.getByRole("tab", { name: "Raw" }).click();
    await page.locator('[data-token="boundary"]').first().waitFor();
    await page.waitForTimeout(250);
    await shot(page, colorScheme === "light" ? "screenshot-raw" : "screenshot-raw-dark");
    await page.context().close();
  }

  // Collapsed header + tablet width
  {
    const page = await open({});
    await page.getByRole("button", { name: "Email details" }).click();
    await page.getByRole("button", { name: "Tablet width (768px)" }).click();
    await page.waitForTimeout(300);
    await shot(page, "screenshot-tablet");
    await page.context().close();
  }

  // Full screen at mobile width
  {
    const page = await open({});
    await page.getByRole("button", { name: "Full screen preview" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "Mobile width (375px)" }).click();
    await page.waitForTimeout(300);
    await shot(page, "screenshot-fullscreen");
    await page.context().close();
  }

  // Phone layout
  {
    const ctx = await browser.newContext({
      viewport: { width: 390, height: 780 },
      deviceScaleFactor: 2,
      isMobile: true,
      hasTouch: true,
    });
    const page = await ctx.newPage();
    await page.goto(BASE);
    await page.getByRole("listbox", { name: "Inbox" }).getByRole("option").first().waitFor();
    await page.waitForTimeout(300);
    await shot(page, "screenshot-phone");
    await ctx.close();
  }

  // Empty state
  {
    await fetch(`${BASE}/api/v1/messages`, { method: "DELETE" });
    const ctx = await browser.newContext({
      viewport: { width: 1280, height: 760 },
      deviceScaleFactor: 2,
    });
    const page = await ctx.newPage();
    await page.goto(BASE);
    await page.getByRole("heading", { name: "No emails yet" }).waitFor();
    await page.waitForTimeout(300);
    await shot(page, "screenshot-empty");
    await ctx.close();
  }
}

// Login screens: `make screenshots` runs this script a second time, with
// MAILPEEK_AUTH=1, against an instance started with --ui-auth admin:secret.
const AUTH_BASE = process.env.MAILPEEK_AUTH === "1" ? BASE : undefined;
if (AUTH_BASE) {
  const newPage = async (colorScheme = "light") => {
    const ctx = await browser.newContext({
      viewport: { width: 1280, height: 760 },
      deviceScaleFactor: 2,
      colorScheme,
    });
    return ctx.newPage();
  };
  const signIn = async (page, pass) => {
    await page.getByLabel("Username").fill("admin");
    await page.getByLabel("Password").fill(pass);
    await page.getByRole("button", { name: "Sign in" }).click();
  };

  for (const colorScheme of ["light", "dark"]) {
    const page = await newPage(colorScheme);
    await page.goto(AUTH_BASE);
    await page.getByRole("heading", { name: "Sign in" }).waitFor();
    await shot(page, colorScheme === "light" ? "screenshot-login" : "screenshot-login-dark");
    await page.context().close();
  }

  // Wrong password, then signed in.
  const page = await newPage();
  await page.goto(AUTH_BASE);
  await signIn(page, "wrong");
  await page.getByRole("alert").waitFor();
  await shot(page, "screenshot-login-error");
  await signIn(page, "secret");
  await page.getByRole("heading", { name: "No emails yet" }).waitFor();
  await page.waitForTimeout(300);
  await shot(page, "screenshot-login-signed-in");
  await page.context().close();
}

await browser.close();
t.close();
