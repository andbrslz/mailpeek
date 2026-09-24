# @mailpeek-dev/playwright

Playwright fixture for [Mailpeek](https://github.com/andbrslz/mailpeek): an isolated inbox per test and `waitFor()` for emails.

```bash
npm install -D @mailpeek-dev/playwright
```

```ts
import { test, expect } from "@mailpeek-dev/playwright";

test("password reset", async ({ page, mail }) => {
  const inbox = mail.createInbox();

  await page.goto("/forgot-password");
  await page.getByLabel("Email").fill(inbox.address);
  await page.getByRole("button", { name: "Reset password" }).click();

  const email = await inbox.waitForEmail({ subject: "Reset your password" });
  const link = email.findLink("Reset password");

  await page.goto(link.href);
  await expect(page.getByText("Choose a new password")).toBeVisible();
});
```

## The `mail` fixture

| Member | Description |
| --- | --- |
| `createInbox(options?)` | Unique inbox for this test (`test-<random>@mailpeek.local`), matched exactly. Recommended for parallel runs. |
| `workerInbox()` | Inbox shared by the current worker (`worker-<index>-<random>@mailpeek.local`). |
| `messages()`, `latest()`, `waitFor()`, `waitForEmails()`, `get()`, `delete()`, `clear()` | Same as `@mailpeek-dev/client`. |
| `client` | The underlying `Mailpeek` instance. |

Inbox: `address`, `messages()`, `latest()`, `waitForEmail(options?)`, `waitForEmails(count, options?)`, `failNext(options?)`, `clear()`.

`inbox.failNext({ code: 451 })` makes the next delivery to that inbox fail, to test that the application retries or reports it; other tests are not affected.

## When a test fails

The fixture attaches, for each inbox the test used, a JSON summary and the newest emails as `.eml` and HTML to the Playwright report, and keeps them in Mailpeek. Inboxes of passing tests are cleared when the test ends. A `waitFor` timeout lists what arrived and why it didn't match, with a link to the Web UI filtered to that inbox.

| Option | Default | |
| --- | --- | --- |
| `mailpeekCleanup` | `"passed"` | Clear the test's inboxes when it passes (`"always"`, `"never"`). |
| `mailpeekAttachOnFailure` | `true` | Attach emails to the report on failure. |

## Start Mailpeek with the suite

```ts
import { mailpeekWebServer } from "@mailpeek-dev/playwright";

export default defineConfig({
  webServer: [mailpeekWebServer(), { command: "npm run dev", url: "http://localhost:3000" }],
});
```

Options: `smtpPort` (1026), `httpPort` (8026), `binary` (a local `mailpeek` instead of Docker), `image`, `args`, `reuseExistingServer`, `timeout`.

## Configuration

The Mailpeek URL comes from `MAILPEEK_URL` (default `http://localhost:8026`), or from the `mailpeekUrl` option. If Mailpeek runs with `--ui-auth`, include the credentials: `http://admin:secret@localhost:8026`.

```ts
import type { MailpeekTestOptions } from "@mailpeek-dev/playwright";
import { defineConfig } from "@playwright/test";

export default defineConfig<MailpeekTestOptions>({
  use: { mailpeekUrl: "http://localhost:8026" },
});
```

`test` and `expect` are Playwright's, extended with the fixture. Everything from `@mailpeek-dev/client` is re-exported.
