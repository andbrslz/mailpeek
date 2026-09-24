import { Mailpeek } from "@mailpeek/client";
import { expect, test } from "@playwright/test";
import { httpPort } from "./ports";
import { sendMail } from "./smtp";

const mailpeek = new Mailpeek({ baseUrl: `http://127.0.0.1:${httpPort}` });

test("the list loads more emails as it scrolls", async ({ page }) => {
  const inbox = mailpeek.createInbox();
  for (let i = 1; i <= 60; i++) {
    await sendMail({ to: inbox.address, subject: `Bulk ${i}`, text: "x" });
  }
  await inbox.waitForEmails(60);

  await page.goto("/");
  await page.getByRole("searchbox", { name: "Search emails" }).fill(inbox.address);
  const options = page.getByRole("listbox", { name: "Inbox" }).getByRole("option");
  // One page first; the counter already shows every match.
  await expect(options).toHaveCount(50);
  await expect(page.locator("#email-list").getByText("60", { exact: true })).toBeVisible();

  await options.last().scrollIntoViewIfNeeded();
  await expect(options).toHaveCount(60);
  await expect(options.last()).toContainText("Bulk 1");

  // A new arrival goes on top and keeps everything already loaded.
  await sendMail({ to: inbox.address, subject: "Bulk 61", text: "x" });
  await expect(options.first()).toContainText("Bulk 61");
  await expect(options).toHaveCount(61);
});
