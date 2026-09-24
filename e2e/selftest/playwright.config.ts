import type { MailpeekTestOptions } from "@mailpeek/playwright";
import { defineConfig } from "@playwright/test";

// Used by attachments.spec.ts: runs a deliberately failing test to check the
// fixture attaches emails to the report. Not part of the normal suite.
export default defineConfig<MailpeekTestOptions>({
  testDir: ".",
  testMatch: /failing\.selftest\.ts/,
  // Never share test-results/ with the main run: Playwright empties it on start.
  outputDir: "../selftest-results",
  reporter: [["json"]],
  use: { mailpeekUrl: process.env.MAILPEEK_URL },
});
