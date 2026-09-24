import { Mailpeek } from "@mailpeek/client";
import { expect, test } from "@playwright/test";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { httpPort, smtpPort } from "./ports";

// Runs a separate Playwright process with a deliberately failing test and
// checks what the fixture did: emails attached to the report, inbox kept.
test("a failing test gets its emails attached and kept", async () => {
  test.setTimeout(90_000);
  const cwd = fileURLToPath(new URL("..", import.meta.url));
  const run = spawnSync("npx", ["playwright", "test", "-c", "selftest/playwright.config.ts"], {
    cwd,
    encoding: "utf8",
    env: {
      ...process.env,
      MAILPEEK_URL: `http://127.0.0.1:${httpPort}`,
      SMTP_PORT: String(smtpPort),
      CI: "",
    },
    timeout: 80_000,
  });
  const report = JSON.parse(run.stdout.slice(run.stdout.indexOf("{"))) as {
    suites: {
      specs: {
        tests: {
          results: {
            status: string;
            stdout: { text?: string }[];
            attachments: { name: string; contentType: string }[];
          }[];
        }[];
      }[];
    }[];
  };
  const result = report.suites[0]!.specs[0]!.tests[0]!.results[0]!;
  expect(result.status).toBe("failed");

  const names = result.attachments.map((a) => `${a.contentType} ${a.name}`);
  expect(names).toContainEqual(expect.stringMatching(/^application\/json mailpeek test-/));
  expect(names).toContain("message/rfc822 email 1 - Kept for debugging.eml");
  expect(names).toContain("text/html email 1 - Kept for debugging.html");

  // Failed tests keep their emails for inspection.
  const address = result.stdout
    .map((s) => s.text ?? "")
    .join("")
    .match(/INBOX=(\S+)/)![1]!;
  const mailpeek = new Mailpeek({ baseUrl: `http://127.0.0.1:${httpPort}` });
  expect(await mailpeek.inbox(address).messages()).toHaveLength(1);
});
