import { mailpeekWebServer, type MailpeekTestOptions } from "@mailpeek/playwright";
import { defineConfig, devices } from "@playwright/test";

// Mailpeek is started with the suite (Docker by default; set MAILPEEK_BINARY to
// use a local binary) unless one already answers on MAILPEEK_SMTP_PORT /
// MAILPEEK_HTTP_PORT. The demo app listens on APP_PORT.
const smtpPort = Number(process.env.MAILPEEK_SMTP_PORT ?? 1026);
const httpPort = Number(process.env.MAILPEEK_HTTP_PORT ?? 8026);
const appPort = process.env.APP_PORT ?? "3000";

export default defineConfig<MailpeekTestOptions>({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  reporter: "list",
  use: {
    baseURL: `http://localhost:${appPort}`,
    mailpeekUrl: `http://localhost:${httpPort}`,
  },
  webServer: [
    mailpeekWebServer({ smtpPort, httpPort, binary: process.env.MAILPEEK_BINARY }),
    {
      command: "node app/server.js",
      url: `http://localhost:${appPort}`,
      env: { PORT: appPort, SMTP_HOST: "localhost", SMTP_PORT: String(smtpPort) },
      reuseExistingServer: !process.env.CI,
    },
  ],
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
