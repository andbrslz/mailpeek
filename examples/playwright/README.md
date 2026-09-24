# Playwright example

A tiny demo app (`app/server.js`) that sends real emails over SMTP (registration with an activation link, and password reset), tested with `@mailpeek/playwright`.

```bash
# from the repository root
npm ci
npx playwright install chromium

# Mailpeek is started with the suite: Docker by default, or MAILPEEK_BINARY=../../bin/mailpeek
cd examples/playwright
npx playwright test
```

`playwright.config.ts` starts Mailpeek with `mailpeekWebServer()` (unless one is already running on the ports) and the demo app. Environment knobs: `MAILPEEK_SMTP_PORT`, `MAILPEEK_HTTP_PORT`, `MAILPEEK_BINARY`, `APP_PORT`.

The tests are in [`tests/auth.spec.ts`](tests/auth.spec.ts).
