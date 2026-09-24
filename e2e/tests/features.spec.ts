import { expect, test } from "@mailpeek/playwright";
import { logoPng, sendMail } from "./smtp";

test("the search finds text in the email body", async ({ page, mail }) => {
  const inbox = mail.createInbox();
  const code = `code-${Date.now()}`;
  await sendMail({
    to: inbox.address,
    subject: "Your sign-in code",
    text: `Use ${code} to sign in.`,
  });
  await inbox.waitForEmail({ body: code });

  await page.goto("/");
  await page.getByRole("searchbox", { name: "Search emails" }).fill(code);
  const options = page.getByRole("listbox", { name: "Inbox" }).getByRole("option");
  await expect(options).toHaveCount(1);
  await expect(options.first()).toContainText("Your sign-in code");
});

test("image attachments have a preview and open in the browser", async ({ page, mail }) => {
  const inbox = mail.createInbox();
  await sendMail({
    to: inbox.address,
    subject: "Screenshot attached",
    text: "See the screenshot.",
    attachments: [
      { filename: "screenshot.png", content: logoPng, contentType: "image/png" },
      { filename: "notes.txt", content: "plain notes", contentType: "text/plain" },
    ],
  });
  await inbox.waitForEmail();

  await page.goto("/");
  await page.getByRole("searchbox", { name: "Search emails" }).fill(inbox.address);
  await page.getByRole("option", { name: /Screenshot attached/ }).click();
  await page.getByRole("tab", { name: /Attachments/ }).click();

  const preview = page.getByRole("img", { name: "Preview of screenshot.png" });
  await expect(preview).toBeVisible();
  await expect.poll(() => preview.evaluate((img: HTMLImageElement) => img.naturalWidth)).toBe(1);
  await expect(page.getByRole("link", { name: "Open" })).toHaveCount(1);

  const [popup] = await Promise.all([
    page.waitForEvent("popup"),
    page.getByRole("link", { name: "Open" }).click(),
  ]);
  const response = await popup.waitForEvent("load").then(() => popup.request.get(popup.url()));
  expect(response.headers()["content-disposition"]).toMatch(/^inline;/);
  expect(response.headers()["content-type"]).toBe("image/png");
});

test("failNext makes a delivery fail so the application can retry", async ({ mail }) => {
  const inbox = mail.createInbox();
  const other = mail.createInbox();
  await inbox.failNext({ code: 451 });

  await sendMail({ to: other.address, subject: "Unaffected", text: "x" });
  await expect(sendMail({ to: inbox.address, subject: "Try 1", text: "x" })).rejects.toThrow(/451/);
  await sendMail({ to: inbox.address, subject: "Try 2", text: "x" });

  const email = await inbox.waitForEmail();
  expect(email.subject).toBe("Try 2");
  expect(await inbox.messages()).toHaveLength(1);
  expect(await other.messages()).toHaveLength(1);
  expect(await mail.client.smtpFailures()).not.toContainEqual(
    expect.objectContaining({ address: inbox.address }),
  );
});
