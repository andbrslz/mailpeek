# Mailpeek

[![CI](https://github.com/andbrslz/mailpeek/actions/workflows/ci.yml/badge.svg)](https://github.com/andbrslz/mailpeek/actions/workflows/ci.yml)
[![npm](https://img.shields.io/npm/v/@mailpeek-dev/playwright?label=%40mailpeek-dev%2Fplaywright)](https://www.npmjs.com/package/@mailpeek-dev/playwright)
[![Docker pulls](https://img.shields.io/docker/pulls/4ndbrslz/mailpeek)](https://hub.docker.com/r/4ndbrslz/mailpeek)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

**Email testing for developers.**

Catch emails locally.
Inspect them in a clean Web UI.
Test them directly from Playwright.

One binary · zero config · ~8 MB · no database · CI friendly

![Mailpeek Web UI](docs/screenshot-light.png)

<table>
<tr>
<th>Development</th>
<th>Testing</th>
</tr>
<tr>
<td>

```bash
docker run --rm \
  -p 1026:1026 -p 8026:8026 \
  4ndbrslz/mailpeek

# point your app at localhost:1026
open http://localhost:8026
```

</td>
<td>

```ts
const inbox = mail.createInbox();

// ...sign up with inbox.address

const email = await inbox.waitForEmail({
  subject: "Welcome",
});
await page.goto(email.findLink("Activate account").href);
```

</td>
</tr>
</table>

Setting it up with an AI coding agent? Point it at [`llms.txt`](llms.txt): install, app settings for common frameworks, verification and troubleshooting on one page.

Both modes use the same server: SMTP → MIME parser → in-memory store (optionally mirrored to a directory) → event broker, consumed by the Web UI (REST + SSE) and by the SDK (REST + wait API).

**Status:** 0.x. Mailpeek is young and developed by one maintainer. The REST API, the SDK and the CLI flags may still change between minor versions; every change is listed in the [changelog](CHANGELOG.md). Bug reports and feedback are very welcome in [issues](https://github.com/andbrslz/mailpeek/issues).

---

## Scope and alternatives

Mailpeek does one thing: catch emails so you can look at them and assert on them in tests. It never sends mail anywhere.

| | Mailpeek | [Mailpit](https://github.com/axllent/mailpit) | [MailHog](https://github.com/mailhog/MailHog) |
| --- | --- | --- | --- |
| Actively maintained | yes | yes | no (last release in 2020) |
| First-party Playwright fixture and TypeScript client (`waitForEmail`, isolated inbox per test) | yes | no (REST API) | no (REST API) |
| Simulate SMTP failures to test retries | yes | yes | yes |
| Keep messages across restarts | optional (`--data-dir`) | yes (SQLite) | optional (MongoDB, maildir) |
| Relay / release to a real SMTP server | no, on purpose | yes | yes |
| POP3 or IMAP access | no, on purpose | POP3 | no |

Choose **Mailpit** if you need to relay emails, read them with a mail client over POP3, or run spam and HTML compatibility checks. Choose **Mailpeek** when the main job is automated tests of the emails your application sends.

---

## Quick start

### Docker

```bash
docker run --rm -p 1026:1026 -p 8026:8026 4ndbrslz/mailpeek
```

Configure your application:

```text
SMTP_HOST=localhost
SMTP_PORT=1026
```

Open **http://localhost:8026**. New emails appear instantly.

No TLS and no credentials are needed. If your mailer insists on authenticating, any username and password is accepted (`AUTH PLAIN`/`LOGIN`), unless you [set credentials](#credentials-optional); if it insists on TLS, start Mailpeek with [`--smtp-tls`](#tls-optional). Mailpeek never delivers or relays mail anywhere.

### Other ports

Mailpeek's defaults (1026 for SMTP, 8026 for the Web UI) are one above Mailpit's and MailHog's 1025/8025, so both can run side by side without configuration. To pick other ports, use the **same number inside and outside the container**, so the Web UI shows the address your application should really use:

```bash
docker run --rm \
  -e MAILPEEK_SMTP_PORT=1027 -e MAILPEEK_HTTP_PORT=8027 \
  -p 1027:1027 -p 8027:8027 \
  4ndbrslz/mailpeek
```

With the `docker-compose.yml` in this repository:

```bash
MAILPEEK_SMTP_PORT=1027 MAILPEEK_HTTP_PORT=8027 docker compose up -d --build
```

Then use `SMTP_PORT=1027`, open **http://localhost:8027** and set `MAILPEEK_URL=http://localhost:8027` for the SDK and the Playwright fixture.

### Binary

Linux and macOS: the install script picks the binary for your system, verifies its SHA-256 checksum and installs it to `/usr/local/bin` (or `~/.local/bin` when that is not writable):

```bash
curl -fsSL https://github.com/andbrslz/mailpeek/releases/latest/download/install.sh | sh
mailpeek
```

`MAILPEEK_VERSION=v1.2.3` installs a specific release and `MAILPEEK_INSTALL_DIR` chooses the directory.

Windows (PowerShell):

```powershell
Invoke-WebRequest https://github.com/andbrslz/mailpeek/releases/latest/download/mailpeek-windows-amd64.exe -OutFile mailpeek.exe
.\mailpeek.exe
```

Each release has `mailpeek-linux-amd64`, `mailpeek-linux-arm64`, `mailpeek-darwin-amd64`, `mailpeek-darwin-arm64` and `mailpeek-windows-amd64.exe`, with `checksums.txt`.

```text
Mailpeek

SMTP  smtp://localhost:1026
Web   http://localhost:8026

Ready in 2ms
```

### From source

```bash
make build        # builds the Web UI, then bin/mailpeek with the UI embedded
./bin/mailpeek
```

Requires Go 1.25+ and Node 20.19+.

`go install` is not a supported way to install Mailpeek: the Web UI is built by Vite and embedded at build time, so a `go install` binary serves the API only (its banner says so). Use a release binary, the Docker image or `make build`.

---

## Playwright quick start

```bash
npm install -D @mailpeek-dev/playwright
```

```ts
import { test, expect } from "@mailpeek-dev/playwright";

test("welcome email", async ({ page, mail }) => {
  const inbox = mail.createInbox();

  await page.goto("/register");
  await page.getByLabel("Email").fill(inbox.address);
  await page.getByRole("button", { name: "Register" }).click();

  const email = await inbox.waitForEmail();

  expect(email.subject).toBe("Welcome");
});
```

The `mail` fixture talks to `MAILPEEK_URL` (default `http://localhost:8026`). You can also set it in `playwright.config.ts`:

```ts
import type { MailpeekTestOptions } from "@mailpeek-dev/playwright";
import { defineConfig } from "@playwright/test";

export default defineConfig<MailpeekTestOptions>({
  use: { mailpeekUrl: "http://localhost:8026" },
});
```

### Password reset, end to end

```ts
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
```

A runnable version, with a small demo app that sends real email, is in [`examples/playwright`](examples/playwright).

### Test isolation

Parallel tests must never read each other's email. The recommended way is one **inbox per test**:

```ts
const inbox = mail.createInbox(); // test-k7x92ab1@mailpeek.local
```

Nothing is created on the server. An inbox is a unique address matched **exactly** (To, Cc or the SMTP envelope), so `inbox.waitForEmail()` is the same as `mail.waitFor({ address: inbox.address })`, and `ana@example.com` never picks up mail for `joana@example.com`. It runs safely with `playwright test --workers=8`.

| API                  | Address                              | Scope                                  |
| -------------------- | ------------------------------------ | -------------------------------------- |
| `mail.createInbox()` | `test-<random>@mailpeek.local`       | one test (recommended)                 |
| `mail.workerInbox()` | `worker-<n>-<random>@mailpeek.local` | one Playwright worker                  |

Inbox methods: `address`, `messages()`, `latest()`, `waitForEmail(options?)`, `waitForEmails(count, options?)`, `failNext(options?)`, `clear()`. `mail.client.inbox("ana@example.com")` gives the same view for an address you already have.

`createInbox({ prefix, domain })` changes the address, for example when your app rejects `.local` domains:

```ts
mail.createInbox({ prefix: "signup", domain: "example.test" });
```

`mail` also exposes `messages()`, `latest()`, `waitFor()`, `waitForEmails()`, `get()`, `delete()`, `clear()` and `client` (the underlying `Mailpeek` instance).

### Several emails and verification codes

```ts
const [welcome, verify] = await inbox.waitForEmails(2); // in the order they arrived
const code = verify.findCode(); // "482913"
await page.getByLabel("Code").fill(code);
```

`findCode()` prefers a code right after words like "code", "código", "OTP", "PIN" or "verification", then falls back to the first 6-digit number; `findCode({ pattern: /token=(\w+)/ })` handles other formats.

### Delivery failures and retries

`inbox.failNext()` makes the next delivery to that inbox fail, so a test can check that the application retries, queues or reports the error. Other inboxes, and parallel tests, are not affected:

```ts
test("the welcome email is retried after a temporary failure", async ({ page, mail }) => {
  const inbox = mail.createInbox();
  await inbox.failNext({ code: 451 }); // the next delivery gets "451 4.3.0 Temporary failure"

  await signUp(page, inbox.address);

  const email = await inbox.waitForEmail({ subject: "Welcome" }); // the retry arrived
  expect(email.subject).toBe("Welcome");
});
```

Options: `code` (400 to 599, default `451`), `stage` (`"data"`, the default, refuses the message after it was sent; `"rcpt"` refuses the recipient, like an unknown user with `550`), `message` and `count` (default 1). Failed deliveries are not stored. `mail.client.failNext({ address?, ... })` does the same for any recipient, `mail.client.smtpFailures()` lists pending rules and `mail.client.clearFailures()` removes them; `inbox.clear()` also removes the inbox's rules.

### When a test fails

- **Emails are attached to the Playwright report**: a summary of each inbox the test used, plus `.eml` and HTML files of its newest emails, so you can see what the app really sent.
- **They are kept in Mailpeek** for inspection, while inboxes of passing tests are cleared when the test ends.
- **A timeout explains itself.** Instead of just "timed out", `MailpeekTimeoutError` lists what arrived and why it didn't match:

```text
No email matching { address="test-k7x92ab1@mailpeek.local", subject="Welcome" } arrived within 10000ms.
Mailpeek holds 3 messages. Most recent:
  - "Reset your password" to test-k7x92ab1@mailpeek.local at 14:32:07.412: subject does not contain "Welcome"
  - "Welcome" to test-p0q1r2s3@mailpeek.local at 14:32:06.980: not sent to "test-k7x92ab1@mailpeek.local"
Inspect: http://localhost:8026/?q=test-k7x92ab1%40mailpeek.local
```

Configure it per project or per test:

```ts
export default defineConfig<MailpeekTestOptions>({
  use: {
    mailpeekCleanup: "passed", // "passed" (default) | "always" | "never"
    mailpeekAttachOnFailure: true, // default
  },
});
```

### Start Mailpeek with the suite

```ts
import { mailpeekWebServer } from "@mailpeek-dev/playwright";

export default defineConfig({
  webServer: [
    mailpeekWebServer(), // docker run … 4ndbrslz/mailpeek, reused if already running
    { command: "npm run dev", url: "http://localhost:3000" },
  ],
});
```

Options: `smtpPort`, `httpPort`, `binary` (use a local `mailpeek` instead of Docker), `image`, `args` (e.g. `["--max-messages", "1000"]`), `reuseExistingServer`, `timeout`.

---

## TypeScript client (Vitest, Jest, scripts)

```bash
npm install -D @mailpeek-dev/client
```

```ts
import { Mailpeek } from "@mailpeek-dev/client";

const mailpeek = new Mailpeek({ baseUrl: "http://localhost:8026" });

const messages = await mailpeek.messages();             // summaries, newest first
const latest = await mailpeek.latest({ to: "john@example.com" }); // Email | undefined
const email = await mailpeek.waitFor({ to: "john@example.com", subject: "Welcome" });

await mailpeek.delete(email.id);
await mailpeek.clear();
```

Works in Node 20+ (ESM and CommonJS).

### The `Email` object

Fields are plain properties, so `expect` works as usual in Playwright, Vitest and Jest:

```ts
expect(email.subject).toBe("Welcome");
expect(email.text).toContain("Hello");
expect(email.links).toContainEqual(expect.objectContaining({ text: "Activate account" }));
```

| Property                          | Type                       |
| --------------------------------- | -------------------------- |
| `id`, `messageId`                 | `string`                   |
| `from`                            | `{ name?, address }`       |
| `to`, `cc`, `replyTo`             | `{ name?, address }[]`     |
| `subject`, `text`, `html`         | `string`                   |
| `headers`                         | `Record<string, string[]>` |
| `attachments`                     | `{ id, filename, contentType, contentId?, inline, size }[]` |
| `links`                           | `{ text, href }[]`         |
| `envelope`                        | `{ from, to[] }` (SMTP envelope; includes Bcc recipients) |
| `date`, `createdAt`               | `Date`                     |

| Helper                                  | Behavior |
| --------------------------------------- | -------- |
| `findLink("Activate account")`          | Exact link text (case/whitespace-insensitive), then text substring, then href substring. Also accepts a `RegExp` or a predicate. **Throws** with the list of available links when nothing matches. |
| `hasLink(matcher)`                      | Boolean version of `findLink`. |
| `getHeader("x-campaign")`               | First value, case-insensitive. `getHeaders()` returns all values. |
| `hasAttachment("invoice.pdf")`          | Filename match (string or `RegExp`); no argument means "any attachment". |

### Filters and waiting

Filters are combined with AND. Text filters are **case-insensitive substrings**, except `address`:

| Filter    | Matches                                                            |
| --------- | ------------------------------------------------------------------ |
| `address` | a recipient address **exactly** (To, Cc or envelope, so Bcc works); what inboxes use |
| `to`      | To, Cc or envelope recipient, as a substring of the address or name |
| `from`    | From header or envelope sender                                     |
| `subject` | Subject                                                            |
| `body`    | text or HTML body                                                  |
| `q`       | subject **or** from **or** to **or** body                          |
| `since`   | received at or after (`Date`, ISO string, or Unix ms)              |

`waitFor()` resolves with the **newest matching message that already exists**, or waits for one to arrive. Use `since` when a test triggers the same email twice:

```ts
const since = new Date();
await page.getByRole("button", { name: "Resend" }).click();
const email = await inbox.waitForEmail({ since });
```

Waiting happens on the server without polling. Timeouts longer than the 30 s server limit are split into several requests automatically.

### Timeouts and cancellation

| Option | Default | Meaning |
| --- | --- | --- |
| `timeout` (`new Mailpeek({ timeout })` or per `waitFor`) | 10 s | How long to wait **for an email**. |
| `requestTimeout` (`new Mailpeek({ requestTimeout })`) | 10 s | Transport deadline for **each HTTP request**, so a stuck connection can't hang a test or its cleanup. |
| `signal` (last argument of every method) | none | Cancels the operation, e.g. `mailpeek.messages({}, { signal })`. |

`mailpeek.info()` returns the server's version, ports, limits and current usage (messages, bytes, evictions).

---

## CI

### GitHub Actions

```yaml
jobs:
  e2e:
    runs-on: ubuntu-latest
    services:
      mailpeek:
        image: 4ndbrslz/mailpeek
        ports:
          - 1026:1026
          - 8026:8026
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-node@v7
        with: { node-version: 22 }
      - run: npm ci
      - run: npx playwright install --with-deps chromium
      - run: npm run test:e2e
        env:
          SMTP_HOST: localhost
          SMTP_PORT: 1026
          MAILPEEK_URL: http://localhost:8026
```

### Docker Compose

```yaml
services:
  mailpeek:
    image: 4ndbrslz/mailpeek
    ports:
      - "1026:1026"
      - "8026:8026"

  app:
    build: .
    environment:
      SMTP_HOST: mailpeek
      SMTP_PORT: 1026
```

Inside the Compose network your app reaches Mailpeek at `mailpeek:1026`. If you change the port with `MAILPEEK_SMTP_PORT` (see [Other ports](#other-ports)), Mailpeek listens on that port inside the container too, so the app uses `mailpeek:<that port>`.

See [`examples/docker-compose`](examples/docker-compose) for a runnable version. The image has a built-in healthcheck, so `depends_on: { mailpeek: { condition: service_healthy } }` works.

### Capacity for large suites

Mailpeek keeps at most `MAILPEEK_MAX_MESSAGES` messages (1000) and `MAILPEEK_MAX_STORE_SIZE` bytes (256 MB) and removes the **oldest** first. A busy parallel suite can push out an email that a slower test is still waiting for. In CI:

- Raise the limits to cover a full run, e.g. `MAILPEEK_MAX_MESSAGES=5000`. Typical emails are a few KB, so 5000 fit easily in the default 256 MB.
- Keep the default `mailpeekCleanup: "passed"`: passing tests clear their inboxes as they finish, so the store holds only what is still needed.
- If it still happens, the timeout error says so ("N message(s) were removed because Mailpeek reached its limits"), and `mailpeek.info()` reports `store.evicted`.

---

## Configuration

Zero config by default. CLI flags take precedence over environment variables.

| Flag                 | Environment                 | Default   |
| -------------------- | --------------------------- | --------- |
| `--smtp-port`        | `MAILPEEK_SMTP_PORT`        | `1026`    |
| `--http-port`        | `MAILPEEK_HTTP_PORT`        | `8026`    |
| `--max-messages`     | `MAILPEEK_MAX_MESSAGES`     | `1000` (oldest removed first) |
| `--max-message-size` | `MAILPEEK_MAX_MESSAGE_SIZE` | `10MB`    |
| `--max-store-size`   | `MAILPEEK_MAX_STORE_SIZE`   | `256MB` (memory for all messages; oldest removed first) |
| `--host`             | `MAILPEEK_HOST`             | `localhost` (this machine only; the Docker image uses `0.0.0.0`) |
| `--smtp-auth`        | `MAILPEEK_SMTP_AUTH`        | off (any login accepted) |
| `--ui-auth`          | `MAILPEEK_UI_AUTH`          | off (open)                |
| `--smtp-tls`         | `MAILPEEK_SMTP_TLS`         | off; `true` offers STARTTLS with a self-signed certificate |
| `--smtp-tls-cert`, `--smtp-tls-key` | `MAILPEEK_SMTP_TLS_CERT`, `MAILPEEK_SMTP_TLS_KEY` | off; PEM files for STARTTLS instead of the self-signed certificate |
| `--data-dir`         | `MAILPEEK_DATA_DIR`         | off (memory only); see [Keeping emails across restarts](#keeping-emails-across-restarts) |

```bash
mailpeek --smtp-port 1026 --http-port 8026 --max-messages 1000
```

The binary only accepts connections from this machine (`127.0.0.1` and `::1`), so captured emails stay private on shared networks. Use `--host 0.0.0.0` when other machines, or an application in a container, must reach a Mailpeek running outside Docker. The Docker image already listens on all interfaces, which the container needs.

In Docker, the published port and the container port should be the same (`-e MAILPEEK_SMTP_PORT=1027 -p 1027:1027`); see [Other ports](#other-ports).

### Credentials (optional)

Both are off by default and independent of each other. Values look like `user:password` (the password may contain `:`).

| Setting | Effect |
| --- | --- |
| `MAILPEEK_SMTP_AUTH=app:secret` | SMTP clients must log in with `AUTH PLAIN` or `LOGIN` before sending (`530` without a login, `535` with wrong credentials). Configure your app with `SMTP_USER=app` / `SMTP_PASS=secret`. |
| `MAILPEEK_UI_AUTH=admin:secret` | The Web UI shows a sign-in screen; the API accepts that session or HTTP Basic credentials (SDK, `curl -u`). `/api/v1/health` stays open for health checks. |

```bash
docker run --rm -p 1026:1026 -p 8026:8026 \
  -e MAILPEEK_SMTP_AUTH=app:secret \
  -e MAILPEEK_UI_AUTH=admin:secret \
  4ndbrslz/mailpeek
```

The SDK and the Playwright fixture read the Web UI credentials from the URL, so tests only need `MAILPEEK_URL=http://admin:secret@localhost:8026` (or `new Mailpeek({ auth: { username, password } })`).

<p>
  <img src="docs/screenshot-login.png" alt="Sign-in screen (light)" width="49%">
  <img src="docs/screenshot-login-dark.png" alt="Sign-in screen (dark)" width="49%">
</p>

Any page opened without a session goes to `/login` and returns there after signing in. A wrong password shows an error on the same screen (with a short delay to slow down guessing). The session is an `HttpOnly`, `SameSite=Lax` cookie valid for 7 days (until Mailpeek restarts); **Sign out** is in the header. Unauthenticated API calls get `401 {"error":"authentication required"}`, without the header that would make browsers pop up their own dialog.

Both logins travel in plain text (SMTP without TLS, the sign-in form and HTTP Basic over `http://`). They keep other people on a shared network or CI host out of your inbox; they are not a substitute for not exposing Mailpeek to the internet. The banner shows `(login required)` next to each protected address, and the Web UI shows when SMTP needs a login.

### TLS (optional)

Some mailers refuse to send without TLS. `--smtp-tls` (or `MAILPEEK_SMTP_TLS=true`) makes SMTP offer `STARTTLS` with a certificate generated at startup for `localhost`, `127.0.0.1`, `::1`, `mailpeek` and the machine name. It is self-signed, so tell the application not to verify it (Nodemailer: `tls: { rejectUnauthorized: false }`), or pass a certificate it trusts with `--smtp-tls-cert cert.pem --smtp-tls-key key.pem`. Plain connections keep working: clients choose whether to upgrade.

### Activity log

Every received email is logged on standard output, and so is every rejection, so CI logs answer "did the email arrive?":

```text
2026/09/24 10:15:02 received 3f9c1a2b4d5e6f70 from app@acme.test to ana@example.com "Welcome" (3.2 KB)
2026/09/24 10:15:09 rejected message from 172.18.0.3:51234 to bia@example.com: over --max-message-size (10MB)
2026/09/24 10:15:11 SMTP login failed for user "app" from 172.18.0.3:51240: wrong username or password
```

Other commands:

- `mailpeek send --to ana@example.com` sends a test email to a running Mailpeek and prints its id, to check that everything is wired up. Flags: `--subject`, `--text`, `--html`, `--from`, `--host`, `--port` (default `MAILPEEK_SMTP_PORT` or 1026) and `--auth user:password` (default `MAILPEEK_SMTP_AUTH`). In Docker: `docker exec <container> /mailpeek send --to ana@example.com`.
- `mailpeek healthcheck` exits 0 when the HTTP server is healthy (used by the Docker `HEALTHCHECK`).
- `mailpeek version`.

### Keeping emails across restarts

By default messages live in memory only and are gone when Mailpeek stops, which is what a test run wants. For day-to-day development, `--data-dir` (or `MAILPEEK_DATA_DIR`) keeps them in a directory and loads them again at startup:

```bash
mailpeek --data-dir ./emails

docker run --rm -p 1026:1026 -p 8026:8026 \
  -e MAILPEEK_DATA_DIR=/data -v mailpeek-data:/data \
  4ndbrslz/mailpeek
```

Each message is stored as `<id>.eml`, the original email that any mail client opens, next to `<id>.json` with its envelope and arrival time. Deleting, clearing and the limits (`--max-messages`, `--max-store-size`) apply to the directory too, and a restart with smaller limits keeps only the newest messages. The banner shows the directory and how many messages were loaded. With a bind mount (`-v ./emails:/data`) the directory must be writable by the container user (uid 65534).

---

## Web UI

- Inbox, newest first, with live updates over server-sent events (automatic reconnection, no polling). The list loads 50 emails at a time as you scroll, so large inboxes stay fast
- New emails are marked until opened; the unread count appears as a badge on the browser tab icon and in the title, e.g. `(2) Mailpeek`
- A short synthesized chime plays when an email arrives (after your first click on the page, as browsers require). Turn it off with the bell button in the header; the choice is remembered in the browser
- Reading a message is never interrupted: new arrivals do not change the selection
- Search across subject, from, to and the body (uses the API filters)
- Tabs: **HTML** (sandboxed), **Text**, **Headers**, **Raw** (MIME syntax highlighting), **Attachments** (image thumbnails, open or download), **Links** (copy or open each URL)
- HTML preview at desktop, tablet (768 px) or mobile (375 px) width, and in **full screen** (`Esc` to close)
- Collapse the email details (From, To, date) to give the preview more room; the choice is remembered
- Inline `cid:` images are shown in the HTML preview
- Download the original `.eml`, delete a message, clear the inbox
- Resize the email list by dragging its edge (arrow keys work too; double-click resets), or hide it completely for a full-width email
- Keyboard: `j`/`k` or arrows to move, `/` to search, `[` to show/hide the list, `Delete` to delete
- The search is part of the URL (`/?q=…`), so links from failing tests open the UI already filtered
- Light and dark themes: follows the system until you switch with the theme button in the header (the choice is remembered), and a layout for phones
- In English (US/UK), Portuguese (Brazil/Portugal), Spanish and French. The browser language is detected; the language picker in the header (it shows the current language's round flag; its list shows each flag next to the language's name, mouse or keyboard) overrides it and the choice is remembered (the sign-in screen follows it too). Dates and sizes use the chosen locale

### Screenshots

| | |
| --- | --- |
| ![Inbox and HTML preview, light theme](docs/screenshot-light.png) **Inbox and preview.** New emails are marked until opened. | ![Dark theme](docs/screenshot-dark.png) **Dark theme.** Follows your system setting, or the theme button. |
| ![Preview at tablet width with the header collapsed](docs/screenshot-tablet.png) **Tablet width, header collapsed.** More room for the email. | ![Full screen preview at mobile width](docs/screenshot-fullscreen.png) **Full screen.** Desktop, tablet or mobile width; `Esc` closes. |
| ![Links tab](docs/screenshot-links.png) **Links.** Every link in the email, ready to copy or open. | ![Attachments tab](docs/screenshot-attachments.png) **Attachments.** Type, size and download. |
| ![Raw tab with syntax highlighting](docs/screenshot-raw.png) **Raw.** Headers, encoded words and multipart boundaries highlighted; base64 data dimmed. | ![Raw tab, dark theme](docs/screenshot-raw-dark.png) **Raw, dark theme.** |
| ![Empty state](docs/screenshot-empty.png) **Empty state.** Shows the SMTP address to use. | ![Phone layout](docs/screenshot-phone.png) **Phone.** Inbox first, email on tap. |
| ![Sign-in screen](docs/screenshot-login.png) **Sign in.** Shown when `--ui-auth` is set. | ![Wrong password](docs/screenshot-login-error.png) **Wrong password.** The username is kept. |
| ![Signed in with SMTP login required](docs/screenshot-login-signed-in.png) **Signed in.** The padlock notes that SMTP needs a login too; **Sign out** is in the header. | |

Regenerate them with `make screenshots` (needs ports 1026/8026 free).

---

## REST API

Base path: `/api/v1`. All responses are JSON unless noted.

| Method   | Path                                   | Description |
| -------- | -------------------------------------- | ----------- |
| `GET`    | `/messages`                            | `{ messages: Summary[], count, nextCursor? }`, newest first. Filters: `address` (exact), `to`, `from`, `subject`, `body`, `q`, `since`. Optional paging: `limit`, then `cursor=<nextCursor>` for the next page (absent on the last one). Without `limit`, every match is returned. |
| `GET`    | `/messages/count`                      | `{ count }` of messages matching the same filters. |
| `DELETE` | `/messages`                            | Delete matching messages (all when no filter). `{ deleted }` |
| `GET`    | `/messages/latest`                     | Newest matching message, or `404`. |
| `GET`    | `/messages/wait`                       | Newest matching message; waits for one if none exists. `timeout` in ms (default 10000, max 30000). **`200`** + message, or **`204`** on timeout. |
| `GET`    | `/messages/{id}`                       | Full message, including `text`, `html`, `headers`, `attachments`, `links`. |
| `DELETE` | `/messages/{id}`                       | `204`, or `404`. |
| `GET`    | `/messages/{id}/raw`                   | Raw MIME source as `text/plain`. Add `?download=1` for `{id}.eml`. |
| `GET`    | `/messages/{id}/attachments/{attachmentId}` | Attachment download. |
| `GET`    | `/events`                              | Server-sent events: `message.created`, `message.deleted` (`data: {"id":"…"}`), `messages.cleared`. |
| `GET`    | `/health`                              | `{"status":"ok"}` |
| `GET`    | `/openapi.json`                        | OpenAPI 3.1 description of this API. |
| `POST`   | `/smtp/failures`                       | Make upcoming deliveries fail: JSON `{ stage?: "rcpt" \| "data", code?: 451, message?, address?, count?: 1 }`. `201` + rule. |
| `GET`    | `/smtp/failures`                       | Pending simulated failures. |
| `DELETE` | `/smtp/failures`                       | Remove them (only one address's with `?address=`). `{ deleted }` |
| `DELETE` | `/smtp/failures/{id}`                  | Remove one rule. `204`, or `404`. |
| `GET`    | `/info`                                | Version, ports, limits and `store` usage: `messages`, `bytes`, `evicted`. |

```bash
curl "localhost:8026/api/v1/messages/wait?to=john@example.com&subject=Welcome&timeout=5000"
```

---

## Security

Mailpeek is a development tool: do not expose it to the internet. Authentication is off by default ([optional credentials](#credentials-optional) exist for shared machines). Even so:

- **No relay.** Every recipient is accepted, and nothing is ever delivered or forwarded.
- **HTML never runs.** Previews render in an `<iframe sandbox>` without `allow-scripts` or `allow-same-origin`, under a Content-Security-Policy that only allows Mailpeek's own scripts. Links found in emails are never fetched.
- **Local by default.** The binary listens on `127.0.0.1` and `::1` only, so other machines on the network cannot read captured emails unless you pass `--host 0.0.0.0` (the Docker image does, inside the container).
- **Safe attachments.** Downloads use `Content-Disposition: attachment` with a sanitized filename, `X-Content-Type-Options: nosniff` and `Content-Security-Policy: sandbox`. Only PNG, JPEG, GIF, WebP, AVIF and BMP images can be opened in the browser (never SVG or HTML). Attachments are looked up by ID in memory, never by a path taken from the request, so there is no path traversal.
- **Limits.** Maximum message size (`SIZE` is advertised and enforced), a total memory budget for stored messages (`--max-store-size`), at most 100 simultaneous SMTP connections (more get `421`), bounded concurrent MIME parsing, 100 recipients per message, bounded SMTP line length, idle/data/write timeouts on SMTP, header/read/write/idle timeouts and a 64 KB header limit on HTTP, and disconnection after repeated protocol errors.
- **Graceful shutdown** on `SIGINT`/`SIGTERM`: SMTP stops accepting and finishes in-flight messages, SSE streams and pending wait requests end, and HTTP drains.

---

## Footprint

Measured on an Apple M-series laptop (see [docs/performance.md](docs/performance.md) for the methodology):

| Metric                              | Target   | Measured |
| ----------------------------------- | -------- | -------- |
| Binary (linux/amd64, UI embedded)   | < 15 MB  | 7.9 MB   |
| Docker image, compressed            | < 15 MB  | 3.2 MB   |
| Idle memory (container)             | < 20 MB  | ~5 MB    |
| Startup (exec → healthy)            | < 100 ms | ~5 ms    |

---

## Development

```text
cmd/mailpeek        entry point, CLI, banner, healthcheck command
internal/config     defaults < environment < flags
internal/smtp       receive-only SMTP server
internal/mail       Message model, MIME parser, link extraction
internal/store      bounded in-memory store, filters and the optional data directory
internal/events     in-memory pub/sub broker
internal/api        REST, wait API, SSE, embedded UI
internal/app        wiring and graceful shutdown
web/                React + Vite + Tailwind UI (embedded with go:embed)
packages/client     @mailpeek-dev/client
packages/playwright @mailpeek-dev/playwright
e2e/                Playwright suite for Mailpeek itself
examples/           Playwright project with a demo app; Docker Compose
scripts/            maintenance scripts (documentation screenshots)
docs/               performance methodology and screenshots
```

```bash
npm ci
make build           # UI + binary
make check           # every quality gate (below)
make e2e             # Mailpeek E2E (binary + Playwright)
make example         # the documented Playwright example
make bench           # Go benchmarks
make docker          # Docker image
make screenshots     # regenerate docs/*.png
```

UI development with hot reload: run `go run ./cmd/mailpeek` in one terminal and `npm run dev -w @mailpeek/web` in another (Vite proxies `/api` to `:8026`).

### Quality gates

`make check` runs them all, and CI runs the same steps on every push and pull request:

| Gate | Command | Passes when |
| --- | --- | --- |
| Lint | `make lint` | `gofmt`, `go vet`, `golangci-lint`, ESLint and Prettier report nothing |
| UI size | `npm run ui:size` | no file in `web/src` is longer than 300 lines |
| UI i18n | `npm run ui:i18n` | no untranslated text in the UI components (JSX text or labels) |
| Typecheck | `make typecheck` | TypeScript strict passes in every workspace |
| Build | `make build` | UI, SDK packages and the binary build |
| Tests | `make test-race`, `npm test`, `make e2e` | Go (with `-race`), SDK unit tests and Playwright E2E pass |
| Audit | `make audit` | `npm audit` and `govulncheck` find no known vulnerabilities |
| React Doctor | `make doctor` | [React Doctor](https://react.doctor) reports no issues (score 100) |

The Web UI's own conventions (presentational components, logic in hooks and contexts, translations) are in [`web/README.md`](web/README.md).

### Not included (on purpose)

No database, Redis, user accounts, relay/forwarding, IMAP/POP3, WebSockets, GraphQL or plugins. Messages are kept in memory, optionally mirrored to plain files with `--data-dir`. See [Scope and alternatives](#scope-and-alternatives).

### Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security issues: [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE). The language picker's round flags come from [circle-flags](https://github.com/HatScripts/circle-flags) (MIT).
