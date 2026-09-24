import type { MailpeekTestOptions } from "@mailpeek/playwright";
import { defineConfig, devices } from "@playwright/test";
import {
  authHttpPort,
  authSmtpPort,
  httpPort,
  smtpCredentials,
  smtpPort,
  uiCredentials,
} from "./tests/ports";

// End-to-end tests for Mailpeek itself, against the release binary.
const baseURL = `http://127.0.0.1:${httpPort}`;
const binary = process.env.MAILPEEK_BIN ?? "../bin/mailpeek";

export default defineConfig<MailpeekTestOptions>({
  testDir: "./tests",
  fullyParallel: true,
  workers: 4,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["list"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL,
    mailpeekUrl: baseURL,
    trace: "retain-on-failure",
  },
  webServer: [
    {
      command: `${binary} --smtp-port ${smtpPort} --http-port ${httpPort} --max-messages 1000`,
      url: `${baseURL}/api/v1/health`,
      reuseExistingServer: !process.env.CI,
      stdout: "pipe",
    },
    {
      // Same binary with login required on SMTP and on the Web UI/API.
      command:
        `${binary} --smtp-port ${authSmtpPort} --http-port ${authHttpPort}` +
        ` --smtp-auth ${smtpCredentials.user}:${smtpCredentials.pass}` +
        ` --ui-auth ${uiCredentials.username}:${uiCredentials.password}`,
      url: `http://127.0.0.1:${authHttpPort}/api/v1/health`, // stays open for healthchecks
      reuseExistingServer: !process.env.CI,
      stdout: "pipe",
    },
  ],
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
      testIgnore: /clear\.spec\.ts/,
    },
    {
      // Clearing the inbox affects every test, so it runs after all others.
      name: "clear",
      use: { ...devices["Desktop Chrome"] },
      testMatch: /clear\.spec\.ts/,
      dependencies: ["chromium"],
    },
  ],
});
