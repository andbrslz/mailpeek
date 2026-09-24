# Mailpeek

**Email testing for developers.** Catch emails locally, inspect them in a clean Web UI and test them directly from Playwright.

One binary · zero config · ~8 MB · no database · CI friendly

![Mailpeek Web UI](https://raw.githubusercontent.com/andbrslz/mailpeek/master/docs/screenshot-light.png)

## Quick start

```bash
docker run --rm -p 1026:1026 -p 8026:8026 4ndbrslz/mailpeek
```

Point your application at the SMTP server:

```text
SMTP_HOST=localhost
SMTP_PORT=1026
```

Open **http://localhost:8026**. New emails appear instantly.

No TLS and no credentials are needed. If your mailer insists on authenticating, any username and password is accepted. Mailpeek never delivers or relays mail anywhere, and messages live in memory only.

## Tags

| Tag | Meaning |
| --- | --- |
| `latest` | Latest release |
| `0.1` | Latest `0.1.x` release |
| `0.1.1` | Exact version |

Images are published for `linux/amd64` and `linux/arm64`. The image is built `FROM scratch`: just the static binary, running as an unprivileged user.

## Ports

| Port | Purpose |
| --- | --- |
| `1026` | SMTP |
| `8026` | Web UI and REST API |

The defaults are one above Mailpit's and MailHog's 1025/8025, so they can run side by side. To use other ports, keep the **same number inside and outside the container**, so the Web UI shows the address your application should really use:

```bash
docker run --rm \
  -e MAILPEEK_SMTP_PORT=1027 -e MAILPEEK_HTTP_PORT=8027 \
  -p 1027:1027 -p 8027:8027 \
  4ndbrslz/mailpeek
```

## Docker Compose

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
    depends_on:
      mailpeek:
        condition: service_healthy
```

The image has a built-in `HEALTHCHECK`, so `condition: service_healthy` works.

## GitHub Actions

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
      - run: npm ci && npm run test:e2e
        env:
          SMTP_HOST: localhost
          SMTP_PORT: 1026
          MAILPEEK_URL: http://localhost:8026
```

## Testing from code

```bash
npm install -D @mailpeek-dev/playwright   # Playwright fixture with an isolated inbox per test
npm install -D @mailpeek-dev/client       # TypeScript client for Vitest, Jest and scripts
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

## Configuration

Zero config by default. Every setting is an environment variable:

| Environment | Default |
| --- | --- |
| `MAILPEEK_SMTP_PORT` | `1026` |
| `MAILPEEK_HTTP_PORT` | `8026` |
| `MAILPEEK_MAX_MESSAGES` | `100` (oldest removed first) |
| `MAILPEEK_MAX_MESSAGE_SIZE` | `10MB` |
| `MAILPEEK_MAX_STORE_SIZE` | `256MB` (oldest removed first) |
| `MAILPEEK_SMTP_AUTH` | off; `user:password` requires an SMTP login |
| `MAILPEEK_UI_AUTH` | off; `user:password` protects the Web UI and API |
| `MAILPEEK_SMTP_TLS` | off; `true` offers STARTTLS with a self-signed certificate |

For large parallel suites in CI, raise `MAILPEEK_MAX_MESSAGES` (for example `5000`) so no email is removed before a test reads it.

Every received email and every rejection is logged on standard output. To check that everything is wired up:

```bash
docker exec <container> /mailpeek send --to ana@example.com
```

## Links

- Source, full documentation and Web UI tour: https://github.com/andbrslz/mailpeek
- Binaries for Linux, macOS and Windows: https://github.com/andbrslz/mailpeek/releases
- License: MIT
