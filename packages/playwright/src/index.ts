import { test as base, expect, type TestInfo } from "@playwright/test";
import {
  Mailpeek,
  type Email,
  type Inbox,
  type InboxOptions,
  type MessageFilter,
  type MessageSummary,
  type WaitOptions,
} from "@mailpeek/client";

/** The `mail` fixture available in every test. */
export interface MailFixture {
  /** The underlying client, for anything not covered below. */
  readonly client: Mailpeek;
  messages(filter?: MessageFilter): Promise<MessageSummary[]>;
  latest(filter?: MessageFilter): Promise<Email | undefined>;
  waitFor(options?: WaitOptions): Promise<Email>;
  /** Waits for `count` matching emails and returns them in arrival order. */
  waitForEmails(count: number, options?: WaitOptions): Promise<Email[]>;
  get(id: string): Promise<Email>;
  delete(id: string): Promise<void>;
  clear(filter?: MessageFilter): Promise<number>;
  /**
   * A fresh, unique inbox for this test: `test-k7x92ab1@mailpeek.local`.
   * Cleared when the test ends (see `mailpeekCleanup`); on failure its
   * emails are attached to the report.
   */
  createInbox(options?: InboxOptions): Inbox;
  /**
   * An inbox shared by all tests of the current worker
   * (`worker-3-ab12cd34@mailpeek.local`), cleared after each passing test
   * that uses it. A failed test's worker is replaced, so the next one gets
   * a new address.
   */
  workerInbox(): Inbox;
}

/** When a test's inboxes are cleared once it ends. */
export type MailpeekCleanup = "passed" | "always" | "never";

export interface MailpeekTestOptions {
  /** Mailpeek HTTP address. Defaults to MAILPEEK_URL or http://localhost:8026. */
  mailpeekUrl: string;
  /**
   * "passed" (default): clear the test's inboxes when it passes and keep
   * them for inspection when it fails. "always" or "never" otherwise.
   */
  mailpeekCleanup: MailpeekCleanup;
  /** Attach the test's emails (.eml, HTML, summary) to the report when it fails. Default true. */
  mailpeekAttachOnFailure: boolean;
}

interface WorkerFixtures {
  mailpeekUrl: string;
  mailpeek: Mailpeek;
  mailpeekWorkerInbox: Inbox;
}

interface TestFixtures {
  mailpeekCleanup: MailpeekCleanup;
  mailpeekAttachOnFailure: boolean;
  mail: MailFixture;
}

export const test = base.extend<TestFixtures, WorkerFixtures>({
  mailpeekUrl: [
    process.env.MAILPEEK_URL || "http://localhost:8026",
    { scope: "worker", option: true },
  ],
  mailpeekCleanup: ["passed", { option: true }],
  mailpeekAttachOnFailure: [true, { option: true }],

  mailpeek: [
    async ({ mailpeekUrl }, use) => {
      await use(new Mailpeek({ baseUrl: mailpeekUrl }));
    },
    { scope: "worker" },
  ],

  mailpeekWorkerInbox: [
    async ({ mailpeek }, use, workerInfo) => {
      await use(mailpeek.createInbox({ prefix: `worker-${workerInfo.workerIndex}` }));
    },
    { scope: "worker" },
  ],

  mail: async (
    { mailpeek, mailpeekWorkerInbox, mailpeekCleanup, mailpeekAttachOnFailure },
    use,
    testInfo,
  ) => {
    const inboxes = new Set<Inbox>();
    await use({
      client: mailpeek,
      messages: (filter) => mailpeek.messages(filter),
      latest: (filter) => mailpeek.latest(filter),
      waitFor: (options) => mailpeek.waitFor(options),
      waitForEmails: (count, options) => mailpeek.waitForEmails(count, options),
      get: (id) => mailpeek.get(id),
      delete: (id) => mailpeek.delete(id),
      clear: (filter) => mailpeek.clear(filter),
      createInbox: (options) => {
        const inbox = mailpeek.createInbox(options);
        inboxes.add(inbox);
        return inbox;
      },
      workerInbox: () => {
        inboxes.add(mailpeekWorkerInbox);
        return mailpeekWorkerInbox;
      },
    });

    const failed = testInfo.status !== testInfo.expectedStatus;
    if (failed && mailpeekAttachOnFailure) {
      await attachEmails(testInfo, mailpeek, [...inboxes]).catch(() => undefined);
    }
    if (mailpeekCleanup === "always" || (mailpeekCleanup === "passed" && !failed)) {
      await Promise.all([...inboxes].map((inbox) => inbox.clear().catch(() => undefined)));
    }
  },
});

const MAX_ATTACHED = 5;

/** Attaches each inbox's newest emails to the Playwright report. */
async function attachEmails(testInfo: TestInfo, mailpeek: Mailpeek, inboxes: Inbox[]) {
  for (const inbox of inboxes) {
    const messages = await inbox.messages();
    const link = mailpeek.inspectUrl({ address: inbox.address });
    const summary = messages.map((m) => ({
      id: m.id,
      subject: m.subject,
      from: m.from.address,
      to: m.to.map((a) => a.address),
      receivedAt: m.createdAt,
    }));
    await testInfo.attach(`mailpeek ${inbox.address}`, {
      contentType: "application/json",
      body: JSON.stringify({ address: inbox.address, inspect: link, messages: summary }, null, 2),
    });
    for (const [i, m] of messages.slice(0, MAX_ATTACHED).entries()) {
      const name = `email ${i + 1} - ${m.subject || "(no subject)"}`.slice(0, 80);
      await testInfo.attach(`${name}.eml`, {
        contentType: "message/rfc822",
        body: await mailpeek.raw(m.id),
      });
      const email = await mailpeek.get(m.id);
      if (email.html) {
        await testInfo.attach(`${name}.html`, { contentType: "text/html", body: email.html });
      }
    }
  }
}

export { mailpeekWebServer, type MailpeekWebServerOptions } from "./server.js";
export { expect };
export * from "@mailpeek/client";
