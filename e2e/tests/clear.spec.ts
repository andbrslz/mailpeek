import { Mailpeek } from "@mailpeek-dev/client";
import { expect, test } from "@playwright/test";
import { liveEvents } from "./live";
import { httpPort, smtpPort } from "./ports";
import { sendMail } from "./smtp";

// These run after every other test (see the "clear" project) and one at a
// time: clearing and unread counts affect the whole server.
test.describe.configure({ mode: "serial" });

const mailpeek = new Mailpeek({ baseUrl: `http://127.0.0.1:${httpPort}` });

test("clear the inbox and show the empty state", async ({ page, context }) => {
  await sendMail({ to: "someone@acme.test", subject: "Before clear", text: "x" });
  await mailpeek.waitFor({ subject: "Before clear" });

  await page.goto("/");
  await expect(
    page.getByRole("listbox", { name: "Inbox" }).getByRole("option").first(),
  ).toBeVisible();

  const clear = page.getByRole("button", { name: "Clear" });
  await clear.click();
  await page.getByRole("button", { name: "Confirm clear" }).click();

  await expect(page.getByRole("heading", { name: "No emails yet" })).toBeVisible();
  await expect(page.getByText(`127.0.0.1:${smtpPort}`).first()).toBeVisible();
  expect(await mailpeek.messages()).toHaveLength(0);

  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  // Setup is actionable directly from the empty panel on desktop too.
  await expect(
    page.getByRole("main").getByRole("button", { name: "Copy SMTP address" }),
  ).toBeVisible();
  await page.getByRole("main").getByRole("button", { name: "Copy SMTP address" }).click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(`127.0.0.1:${smtpPort}`);

  // A new email replaces the empty state in real time.
  await sendMail({ to: "someone@acme.test", subject: "After clear", text: "y" });
  await expect(page.getByRole("heading", { name: "After clear" })).toBeVisible();
});

test("unread badge on the favicon and title, with a sound that can be muted", async ({ page }) => {
  await mailpeek.clear();
  // Count synthesized notes instead of listening for audio.
  await page.addInitScript(() => {
    const w = window as unknown as { chimeNotes: number };
    w.chimeNotes = 0;
    const original = AudioContext.prototype.createOscillator;
    AudioContext.prototype.createOscillator = function (this: AudioContext) {
      w.chimeNotes++;
      return original.call(this);
    };
  });
  const chimeNotes = () =>
    page.evaluate(() => (window as unknown as { chimeNotes: number }).chimeNotes);
  const favicon = page.locator('link[rel="icon"]');

  const live = liveEvents(page);
  await page.goto("/");
  await live;
  await page.getByRole("heading", { name: "No emails yet" }).click(); // user gesture unlocks audio
  await expect(page).toHaveTitle("Mailpeek");

  // The first email is selected automatically, so it is not unread.
  await sendMail({ to: "a@acme.test", subject: "First", text: "1" });
  await expect(page.getByRole("heading", { name: "First" })).toBeVisible();
  await expect(page).toHaveTitle("Mailpeek");
  await expect.poll(chimeNotes).toBe(2); // two-note chime

  await sendMail({ to: "a@acme.test", subject: "Second", text: "2" });
  await expect(page).toHaveTitle("(1) Mailpeek");
  await expect(favicon).toHaveAttribute("href", /^data:image\/png/);
  // The unread dot is announced as part of the item: "New Second …".
  await expect(
    page.getByRole("listbox", { name: "Inbox" }).getByRole("option", { name: /Second/ }),
  ).toHaveAccessibleName(/^New Second/);
  await expect(
    page.getByRole("listbox", { name: "Inbox" }).getByRole("option", { name: /First/ }),
  ).toHaveAccessibleName(/^First/);

  // Opening it clears the badge.
  await page
    .getByRole("listbox", { name: "Inbox" })
    .getByRole("option", { name: /Second/ })
    .click();
  await expect(page).toHaveTitle("Mailpeek");
  await expect(favicon).toHaveAttribute("href", "/favicon.svg");

  // Muted: no sound, and the choice survives a reload.
  const toggle = page.getByRole("button", { name: "New email sound" });
  await expect(toggle).toHaveAttribute("aria-pressed", "true");
  await toggle.click();
  await expect(toggle).toHaveAttribute("aria-pressed", "false");
  const before = await chimeNotes();
  await page.waitForTimeout(1100); // past the one-chime-per-second throttle
  await sendMail({ to: "a@acme.test", subject: "Third", text: "3" });
  await expect(page).toHaveTitle("(1) Mailpeek");
  expect(await chimeNotes()).toBe(before);

  await page.reload();
  await expect(page.getByRole("button", { name: "New email sound" })).toHaveAttribute(
    "aria-pressed",
    "false",
  );
});
