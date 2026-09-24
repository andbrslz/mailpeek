import { expect, test } from "@mailpeek-dev/playwright";
import nodemailer from "nodemailer";

test("fails after receiving an email", async ({ mail }) => {
  const inbox = mail.createInbox();
  process.stdout.write(`INBOX=${inbox.address}\n`);
  await nodemailer
    .createTransport({ host: "127.0.0.1", port: Number(process.env.SMTP_PORT), secure: false })
    .sendMail({
      from: "app@acme.test",
      to: inbox.address,
      subject: "Kept for debugging",
      html: "<p>debug me</p>",
    });
  await inbox.waitForEmail();
  expect("real").toBe("expected"); // fail on purpose
});
