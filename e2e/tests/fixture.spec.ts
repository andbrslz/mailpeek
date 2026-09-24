import { expect, MailpeekTimeoutError, test } from "@mailpeek-dev/playwright";
import { sendMail, welcomeHtml } from "./smtp";

// These run in parallel across 4 workers: each test must only ever see its
// own email.
for (let i = 1; i <= 8; i++) {
  test(`createInbox isolates parallel tests #${i}`, async ({ mail }) => {
    const inbox = mail.createInbox();
    expect(inbox.address).toMatch(/^test-[a-z0-9]{8}@mailpeek\.local$/);

    await sendMail({ to: inbox.address, subject: `Welcome #${i}`, text: `Hello user ${i}` });
    const email = await inbox.waitForEmail({ subject: "Welcome" });

    expect(email.subject).toBe(`Welcome #${i}`);
    expect(email.text).toContain(`Hello user ${i}`);
    expect(await inbox.messages()).toHaveLength(1);
  });
}

test("waitFor resolves when the email arrives later", async ({ mail }) => {
  const inbox = mail.createInbox();
  const pending = mail.waitFor({ to: inbox.address, subject: "Delayed", timeout: 5000 });
  setTimeout(() => void sendMail({ to: inbox.address, subject: "Delayed", text: "late" }), 300);
  const email = await pending;
  expect(email.text).toContain("late");
});

test("waitFor fails with a clear timeout error", async ({ mail }) => {
  const started = Date.now();
  const error = await mail
    .waitFor({ to: "nobody@mailpeek.local", subject: "Never", timeout: 300 })
    .catch((e: unknown) => e);
  expect(error).toBeInstanceOf(MailpeekTimeoutError);
  expect(String(error)).toContain('to="nobody@mailpeek.local"');
  expect(Date.now() - started).toBeGreaterThanOrEqual(300);
});

test("workerInbox is unique per worker and cleaned between tests", async ({ mail }, testInfo) => {
  const inbox = mail.workerInbox();
  expect(inbox.address).toMatch(
    new RegExp(`^worker-${testInfo.workerIndex}-[a-z0-9]{8}@mailpeek\\.local$`),
  );
  expect(await inbox.messages()).toHaveLength(0);
  await sendMail({ to: inbox.address, subject: "Worker mail", text: "w" });
  await expect((await inbox.waitForEmail()).subject).toBe("Worker mail");
});

test("findLink + page.goto follows the activation link", async ({ page, mail }) => {
  await page.route("http://app.test/**", (route) =>
    route.fulfill({ contentType: "text/html", body: "<h1>Account activated</h1>" }),
  );
  const inbox = mail.createInbox();
  await sendMail({
    to: inbox.address,
    subject: "Welcome",
    html: welcomeHtml("John", "http://app.test/activate/abc"),
  });

  const email = await inbox.waitForEmail({ subject: "Welcome" });
  const link = email.findLink("Activate account");
  expect(link).toEqual({ text: "Activate account", href: "http://app.test/activate/abc" });

  await page.goto(link.href);
  await expect(page.getByRole("heading", { name: "Account activated" })).toBeVisible();
});

test("inboxes match the address exactly", async ({ mail }) => {
  const token = Math.random().toString(36).slice(2, 8);
  const ana = mail.client.inbox(`ana-${token}@mailpeek.local`);
  await sendMail({ to: `joana-${token}@mailpeek.local`, subject: "For Joana", text: "j" });
  await mail.waitFor({ to: `joana-${token}@`, subject: "For Joana" });

  const error = await ana.waitForEmail({ timeout: 300 }).catch((e: unknown) => e);
  expect(error).toBeInstanceOf(MailpeekTimeoutError);
  // The diagnostics explain what arrived and why it did not match.
  expect(String(error)).toContain("Mailpeek holds");
  expect(String(error)).toContain(`Inspect: `);
});

test("waitForEmails and findCode for multi-email flows", async ({ mail }) => {
  const inbox = mail.createInbox();
  await sendMail({ to: inbox.address, subject: "Welcome", text: "Hello!" });
  await sendMail({
    to: inbox.address,
    subject: "Confirm your email",
    html: "<p>Your verification code:</p><p><b>482913</b></p>",
  });

  const [welcome, confirm] = await inbox.waitForEmails(2);
  expect(welcome?.subject).toBe("Welcome");
  expect(confirm?.findCode()).toBe("482913");
});

test.describe("cleanup", () => {
  test.describe.configure({ mode: "serial" });
  let address = "";

  test("inboxes of a passing test are cleared afterwards", async ({ mail }) => {
    const inbox = mail.createInbox();
    address = inbox.address;
    await sendMail({ to: address, subject: "Temporary", text: "t" });
    await inbox.waitForEmail();
  });

  test("…so they are empty in the next test", async ({ mail }) => {
    expect(await mail.client.inbox(address).messages()).toHaveLength(0);
  });
});
