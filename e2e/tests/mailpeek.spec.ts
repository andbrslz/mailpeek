import { Mailpeek } from "@mailpeek-dev/client";
import { expect, test, type Page } from "@playwright/test";
import { liveEvents } from "./live";
import { httpPort } from "./ports";
import { logoPng, sendMail, welcomeHtml } from "./smtp";

const mailpeek = new Mailpeek({ baseUrl: `http://127.0.0.1:${httpPort}` });

/** Opens the Web UI filtered to one address, so parallel tests do not interfere. */
async function openInbox(page: Page, address: string) {
  await page.goto("/");
  await page.getByRole("searchbox", { name: "Search emails" }).fill(address);
}

test("SMTP → client → Web UI: the same email in both modes", async ({ page }) => {
  const inbox = mailpeek.createInbox();
  await sendMail({
    to: inbox.address,
    cc: "manager@acme.test",
    subject: "Welcome to Acme",
    text: "Hello John, activate at http://localhost:3000/activate/abc",
    html: welcomeHtml("John", "http://localhost:3000/activate/abc"),
    headers: { "X-Campaign": "onboarding" },
    attachments: [
      {
        filename: "invoice.pdf",
        content: Buffer.from("%PDF-1.4 test"),
        contentType: "application/pdf",
      },
      { filename: "logo.png", content: logoPng, cid: "logo@acme" },
    ],
  });

  // Testing API
  const email = await inbox.waitForEmail({ subject: "Welcome" });
  expect(email.subject).toBe("Welcome to Acme");
  expect(email.text).toContain("Hello John");
  expect(email.to).toEqual([{ address: inbox.address }]);
  expect(email.cc[0]?.address).toBe("manager@acme.test");
  expect(email.getHeader("x-campaign")).toBe("onboarding");
  expect(email.findLink("Activate account").href).toBe("http://localhost:3000/activate/abc");
  expect(email.links).toContainEqual(expect.objectContaining({ text: "Unsubscribe" }));
  expect(email.hasAttachment("invoice.pdf")).toBe(true);
  const pdf = email.attachments.find((a) => a.filename === "invoice.pdf")!;
  expect(new TextDecoder().decode(await mailpeek.attachment(email.id, pdf.id))).toBe(
    "%PDF-1.4 test",
  );

  // Web UI
  await openInbox(page, inbox.address);
  const item = page
    .getByRole("listbox", { name: "Inbox" })
    .getByRole("option", { name: /Welcome to Acme/ });
  await expect(item).toBeVisible();
  await expect(item).toContainText(inbox.address);
  await item.click();

  await expect(page.getByRole("heading", { name: "Welcome to Acme" })).toBeVisible();
  const preview = page.frameLocator('iframe[title="HTML preview"]');
  await expect(preview.getByText("Hello John, thanks for signing up.")).toBeVisible();
  // Inline cid: image is rewritten to the attachment endpoint and loads.
  await expect
    .poll(() => preview.locator("img").evaluate((img: HTMLImageElement) => img.naturalWidth))
    .toBe(1);

  await page.getByRole("tab", { name: "Text" }).click();
  await expect(page.getByRole("tabpanel")).toContainText(
    "activate at http://localhost:3000/activate/abc",
  );

  await page.getByRole("tab", { name: "Headers" }).click();
  await expect(page.getByRole("tabpanel")).toContainText("X-Campaign");

  await page.getByRole("tab", { name: "Raw" }).click();
  await expect(page.getByRole("tabpanel")).toContainText("Content-Type: multipart/");
  // Syntax highlighting: header names, multipart boundaries and base64 data.
  const raw = page.getByRole("tabpanel");
  await expect(raw.locator('[data-token="name"]', { hasText: /^Subject:$/ })).toBeVisible();
  await expect(raw.locator('[data-token="boundary"]').first()).toBeVisible();
  await expect(raw.locator('[data-token="base64"]').first()).toBeAttached();

  await page.getByRole("tab", { name: /Attachments/ }).click();
  const pdfRow = page
    .getByRole("tabpanel")
    .getByRole("listitem")
    .filter({ hasText: "invoice.pdf" });
  await expect(pdfRow).toContainText("PDF");
  await expect(pdfRow).toContainText("13 B");
  const download = page.waitForEvent("download");
  await pdfRow.getByRole("link", { name: "Download" }).click();
  expect((await download).suggestedFilename()).toBe("invoice.pdf");

  await page.getByRole("tab", { name: /Links/ }).click();
  await expect(page.getByRole("tabpanel")).toContainText("Activate account");
  await expect(page.getByRole("tabpanel")).toContainText("http://localhost:3000/activate/abc");
});

test("inbox updates in real time without reloading", async ({ page }) => {
  const inbox = mailpeek.createInbox();
  const live = liveEvents(page);
  await openInbox(page, inbox.address);
  await live;
  await expect(page.getByText(/No emails match/)).toBeVisible();
  // The list's footer (SMTP address, live status) only appears with messages.
  await expect(page.getByRole("status")).toHaveCount(0);

  await sendMail({ to: inbox.address, subject: "First", text: "one" });
  await expect(
    page.getByRole("listbox", { name: "Inbox" }).getByRole("option", { name: /First/ }),
  ).toBeVisible();
  await expect(page.getByRole("status")).toHaveText("Live");
  await sendMail({ to: inbox.address, subject: "Second", text: "two" });
  await expect(page.getByRole("listbox", { name: "Inbox" }).getByRole("option")).toHaveCount(2);
  // Newest first, and the newest message is selected automatically.
  await expect(
    page.getByRole("listbox", { name: "Inbox" }).getByRole("option").first(),
  ).toContainText("Second");
});

test("delete a message from the UI", async ({ page }) => {
  const inbox = mailpeek.createInbox();
  await sendMail({ to: inbox.address, subject: "Keep me", text: "k" });
  await sendMail({ to: inbox.address, subject: "Delete me", text: "d" });
  await inbox.waitForEmail({ subject: "Delete me" });

  await openInbox(page, inbox.address);
  await page
    .getByRole("listbox", { name: "Inbox" })
    .getByRole("option", { name: /Delete me/ })
    .click();
  await page.getByRole("button", { name: "Delete email" }).click();
  await expect(page.getByRole("listbox", { name: "Inbox" }).getByRole("option")).toHaveCount(1);
  await expect(page.getByRole("heading", { name: "Keep me" })).toBeVisible();
  expect(await inbox.messages()).toHaveLength(1);
});

test("email HTML is sandboxed and its scripts never run", async ({ page }) => {
  const inbox = mailpeek.createInbox();
  await sendMail({
    to: inbox.address,
    subject: "Hostile",
    html: `<p id="content">Original content</p>
      <script>document.getElementById("content").textContent = "PWNED"; parent.document.title = "PWNED";</script>
      <img src="x" onerror="document.body.textContent='PWNED'">`,
  });
  await inbox.waitForEmail();

  await openInbox(page, inbox.address);
  await page
    .getByRole("listbox", { name: "Inbox" })
    .getByRole("option", { name: /Hostile/ })
    .click();
  const iframe = page.locator('iframe[title="HTML preview"]');
  await expect(iframe).toHaveAttribute("sandbox", "allow-popups allow-popups-to-escape-sandbox");
  await expect(
    page.frameLocator('iframe[title="HTML preview"]').getByText("Original content"),
  ).toBeVisible();
  await page.waitForTimeout(200);
  await expect(page.frameLocator('iframe[title="HTML preview"]').getByText("PWNED")).toHaveCount(0);
  // Other parallel tests may add an unread count, e.g. "(2) Mailpeek".
  await expect(page).toHaveTitle(/Mailpeek$/);
  await expect(page).not.toHaveTitle(/PWNED/);
});

test("search filters by subject, sender and recipient", async ({ page }) => {
  const inbox = mailpeek.createInbox();
  const token = inbox.address.split("@")[0]!;
  await sendMail({ to: inbox.address, subject: `Invoice ${token}`, text: "x" });
  await sendMail({ to: inbox.address, subject: `Receipt ${token}`, text: "y" });
  await inbox.waitForEmail({ subject: "Receipt" });

  await page.goto("/");
  const search = page.getByRole("searchbox", { name: "Search emails" });
  await search.fill(`Invoice ${token}`);
  await expect(page.getByRole("listbox", { name: "Inbox" }).getByRole("option")).toHaveCount(1);
  await search.fill(token);
  await expect(page.getByRole("listbox", { name: "Inbox" }).getByRole("option")).toHaveCount(2);
});

test("preview at tablet and mobile widths, and in full screen", async ({ page }) => {
  const inbox = mailpeek.createInbox();
  await sendMail({
    to: inbox.address,
    subject: "Wide preview",
    html: welcomeHtml("John", "http://x.test/a"),
  });
  await inbox.waitForEmail();
  await openInbox(page, inbox.address);
  await page
    .getByRole("listbox", { name: "Inbox" })
    .getByRole("option", { name: /Wide preview/ })
    .click();

  const frame = page.locator('iframe[title="HTML preview"]');
  const frameWidth = () => frame.evaluate((el) => Math.round(el.getBoundingClientRect().width));
  await page.getByRole("button", { name: "Tablet width (768px)" }).click();
  await expect.poll(frameWidth).toBe(768);
  await page.getByRole("button", { name: "Mobile width (375px)" }).click();
  await expect.poll(frameWidth).toBe(375);

  await page.getByRole("button", { name: "Full screen preview" }).click();
  const dialog = page.getByRole("dialog", { name: "Full screen preview" });
  await expect(dialog).toBeVisible();
  // It grows out of the inline preview (clip-path animation).
  expect(await dialog.evaluate((el) => el.getAnimations().length)).toBeGreaterThan(0);
  await expect(dialog).toContainText("Wide preview");
  // The chosen width carries over; desktop fills the window.
  await expect(dialog.getByRole("button", { name: "Mobile width (375px)" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  await dialog.getByRole("button", { name: "Desktop width" }).click();
  await expect.poll(frameWidth).toBeGreaterThan(1200);
  await expect(
    page
      .frameLocator('iframe[title="HTML preview"]')
      .getByText("Hello John, thanks for signing up."),
  ).toBeVisible();

  // Shortcuts are inactive behind the dialog; Esc closes it and restores focus.
  await page.keyboard.press("Delete");
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
  await expect(page.getByRole("button", { name: "Full screen preview" })).toBeFocused();
  expect(await inbox.messages()).toHaveLength(1);
});

test("collapse the email header for a larger preview", async ({ page }) => {
  const inbox = mailpeek.createInbox();
  await sendMail({
    to: inbox.address,
    subject: "Tall preview",
    html: welcomeHtml("John", "http://x.test/a"),
  });
  await inbox.waitForEmail();
  await openInbox(page, inbox.address);
  await page
    .getByRole("listbox", { name: "Inbox" })
    .getByRole("option", { name: /Tall preview/ })
    .click();

  const frame = page.locator('iframe[title="HTML preview"]');
  const frameHeight = () => frame.evaluate((el) => Math.round(el.getBoundingClientRect().height));
  const details = page.getByRole("button", { name: "Email details" });
  await expect(details).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByRole("article").getByText("Received")).toBeVisible();
  const before = await frameHeight();

  await expect(page.locator("#email-details")).toHaveCSS(
    "transition-property",
    /grid-template-rows/,
  );
  await expect(details).toHaveCSS("cursor", "pointer");
  await expect(page.getByRole("button", { name: "Email list" })).toHaveCSS("cursor", "pointer");
  await expect(page.getByRole("tab", { name: "Text" })).toHaveCSS("cursor", "pointer");
  await details.click();
  await expect(details).toHaveAttribute("aria-expanded", "false");
  await expect(page.getByRole("article").getByText("Received")).toBeHidden();
  await expect(page.getByRole("heading", { name: "Tall preview" })).toBeVisible();
  await expect.poll(frameHeight).toBeGreaterThan(before + 60);

  // Remembered across reloads (and other messages).
  await openInbox(page, inbox.address);
  await expect(page.getByRole("button", { name: "Email details" })).toHaveAttribute(
    "aria-expanded",
    "false",
  );
  await page.getByRole("button", { name: "Email details" }).click();
  await expect(page.getByRole("article").getByText("Received")).toBeVisible();
});

test("resize and hide the email list", async ({ page }) => {
  const inbox = mailpeek.createInbox();
  await sendMail({ to: inbox.address, subject: "Resizable", text: "r" });
  await inbox.waitForEmail();
  await openInbox(page, inbox.address);
  await page
    .getByRole("listbox", { name: "Inbox" })
    .getByRole("option", { name: /Resizable/ })
    .click();

  const list = page.locator("#email-list");
  const listWidth = () => list.evaluate((el) => Math.round(el.getBoundingClientRect().width));
  const handle = page.getByRole("separator", { name: "Resize email list" });
  await expect(handle).toHaveAttribute("aria-valuenow", "360");
  expect(await listWidth()).toBe(360);

  // Drag
  const box = (await handle.boundingBox())!;
  await page.mouse.move(box.x + box.width / 2, box.y + 200);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width / 2 + 120, box.y + 200, { steps: 5 });
  await page.mouse.up();
  await expect.poll(listWidth).toBe(480);

  // Keyboard, limits and reset
  await handle.focus();
  await page.keyboard.press("ArrowLeft");
  await expect(handle).toHaveAttribute("aria-valuenow", "464");
  await page.keyboard.press("Home");
  await expect.poll(listWidth).toBe(240);
  await handle.dblclick();
  await expect.poll(listWidth).toBe(360);

  // Remembered across reloads
  await page.keyboard.press("End");
  await expect(handle).toHaveAttribute("aria-valuenow", "640");
  await page.waitForTimeout(300); // saved once the width settles
  await openInbox(page, inbox.address);
  await expect.poll(listWidth).toBe(640);

  // Hide and show the list
  const toggle = page.getByRole("button", { name: "Email list" });
  await expect(toggle).toHaveAttribute("aria-expanded", "true");
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-expanded", "false");
  await expect.poll(listWidth).toBe(0);
  await expect(
    page.getByRole("listbox", { name: "Inbox" }).getByRole("option", { name: /Resizable/ }),
  ).toBeHidden();
  await expect(handle).toBeHidden();
  await expect(page.getByRole("heading", { name: "Resizable" })).toBeVisible();

  await page.locator("body").press("[");
  await expect(toggle).toHaveAttribute("aria-expanded", "true");
  await expect.poll(listWidth).toBe(640);
  await expect(
    page.getByRole("listbox", { name: "Inbox" }).getByRole("option", { name: /Resizable/ }),
  ).toBeVisible();
});
