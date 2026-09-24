import nodemailer from "nodemailer";
import type Mail from "nodemailer/lib/mailer";
import { smtpPort } from "./ports";

const transport = nodemailer.createTransport({ host: "127.0.0.1", port: smtpPort, secure: false });

/** Sends an email to the Mailpeek under test, the way an application would. */
export async function sendMail(options: Mail.Options): Promise<void> {
  await transport.sendMail({ from: '"Acme" <hello@acme.test>', ...options });
}

export const welcomeHtml = (name: string, activateUrl: string) => `
<!doctype html>
<html><body style="font-family: sans-serif">
  <h1>Welcome to Acme</h1>
  <p>Hello ${name}, thanks for signing up.</p>
  <p><a href="${activateUrl}" style="background:#2563eb;color:#fff;padding:8px 12px">Activate account</a></p>
  <p><a href="https://acme.test/unsubscribe/xyz">Unsubscribe</a></p>
  <img src="cid:logo@acme" alt="Acme logo">
</body></html>`;

// 1x1 transparent PNG.
export const logoPng = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=",
  "base64",
);
