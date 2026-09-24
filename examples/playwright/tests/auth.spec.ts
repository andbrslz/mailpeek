import { expect, test } from "@mailpeek/playwright";

test("user receives welcome email", async ({ page, mail }) => {
  const inbox = mail.createInbox();

  await page.goto("/register");
  await page.getByLabel("Name").fill("John");
  await page.getByLabel("Email").fill(inbox.address);
  await page.getByRole("button", { name: "Register" }).click();

  const email = await inbox.waitForEmail({ subject: "Welcome" });

  expect(email.subject).toBe("Welcome");
  expect(email.text).toContain("Hello John");
  expect(email.links).toContainEqual(expect.objectContaining({ text: "Activate account" }));

  const link = email.findLink("Activate account");
  await page.goto(link.href);
  await expect(page.getByText("Account activated")).toBeVisible();
});

test("password reset", async ({ page, mail }) => {
  const inbox = mail.createInbox();

  await page.goto("/forgot-password");
  await page.getByLabel("Email").fill(inbox.address);
  await page.getByRole("button", { name: "Reset password" }).click();

  const email = await inbox.waitForEmail({ subject: "Reset your password" });
  const link = email.findLink("Reset password");
  expect(link).toBeDefined();

  await page.goto(link.href);
  await expect(page.getByText("Choose a new password")).toBeVisible();
});

test("one inbox per worker", async ({ page, mail }) => {
  const inbox = mail.workerInbox();

  await page.goto("/forgot-password");
  await page.getByLabel("Email").fill(inbox.address);
  await page.getByRole("button", { name: "Reset password" }).click();

  const email = await inbox.waitForEmail();
  expect(email.to[0]?.address).toBe(inbox.address);
});

test("signup sends a welcome email and a verification code", async ({ page, mail }) => {
  const inbox = mail.createInbox();

  await page.goto("/register");
  await page.getByLabel("Name").fill("Ana");
  await page.getByLabel("Email").fill(inbox.address);
  await page.getByRole("button", { name: "Register" }).click();

  const [welcome, code] = await inbox.waitForEmails(2);
  expect(welcome?.subject).toBe("Welcome");
  expect(code?.subject).toBe("Your verification code");
  expect(code?.findCode()).toMatch(/^\d{6}$/);
});
