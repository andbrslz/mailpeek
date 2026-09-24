// A tiny demo application that sends real emails over SMTP: registration
// (welcome + activation link) and password reset. It exists only so the
// Playwright example has something realistic to test.
import { randomBytes, randomInt } from "node:crypto";
import { createServer } from "node:http";
import nodemailer from "nodemailer";

const port = Number(process.env.PORT ?? 3000);
const baseUrl = process.env.APP_URL ?? `http://localhost:${port}`;
const transport = nodemailer.createTransport({
  host: process.env.SMTP_HOST ?? "localhost",
  port: Number(process.env.SMTP_PORT ?? 1026),
  secure: false,
});

const escape = (s) =>
  String(s).replace(
    /[&<>"']/g,
    (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c],
  );

const page = (title, body) => `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>${escape(title)}</title>
<style>body{font-family:system-ui,sans-serif;max-width:28rem;margin:4rem auto;padding:0 1rem}
label,input,button{display:block;width:100%;margin-top:.5rem}input,button{padding:.5rem;font:inherit}</style>
</head><body>${body}</body></html>`;

const form = (title, action, button, extra = "") =>
  page(
    title,
    `<h1>${title}</h1><form method="post" action="${action}">${extra}
     <label for="email">Email</label><input id="email" name="email" type="email" required>
     <button type="submit">${button}</button></form>`,
  );

async function readForm(req) {
  let body = "";
  for await (const chunk of req) body += chunk;
  return new URLSearchParams(body);
}

async function register(req) {
  const data = await readForm(req);
  const name = data.get("name") || "there";
  const email = data.get("email");
  const link = `${baseUrl}/activate/${randomBytes(8).toString("hex")}`;
  await transport.sendMail({
    from: '"Acme" <hello@acme.test>',
    to: email,
    subject: "Welcome",
    text: `Hello ${name}, welcome to Acme!\n\nActivate your account: ${link}`,
    html: `<h1>Welcome to Acme</h1><p>Hello ${escape(name)}, welcome to Acme!</p>
           <p><a href="${link}">Activate account</a></p>`,
  });
  const code = String(randomInt(100000, 1000000));
  await transport.sendMail({
    from: '"Acme" <hello@acme.test>',
    to: email,
    subject: "Your verification code",
    text: `Your verification code is ${code}. It expires in 10 minutes.`,
  });
  return page(
    "Check your inbox",
    `<h1>Check your inbox</h1><p>We sent a welcome email to ${escape(email)}.</p>`,
  );
}

async function forgotPassword(req) {
  const email = (await readForm(req)).get("email");
  const link = `${baseUrl}/reset/${randomBytes(8).toString("hex")}`;
  await transport.sendMail({
    from: '"Acme" <no-reply@acme.test>',
    to: email,
    subject: "Reset your password",
    text: `Reset your password: ${link}`,
    html: `<p>Someone asked to reset your password.</p><p><a href="${link}">Reset password</a></p>`,
  });
  return page(
    "Check your inbox",
    `<h1>Check your inbox</h1><p>We sent a reset link to ${escape(email)}.</p>`,
  );
}

const routes = {
  "GET /": () =>
    page(
      "Acme",
      `<h1>Acme</h1><a href="/register">Register</a> · <a href="/forgot-password">Forgot password</a>`,
    ),
  "GET /register": () =>
    form(
      "Register",
      "/register",
      "Register",
      `<label for="name">Name</label><input id="name" name="name">`,
    ),
  "POST /register": register,
  "GET /forgot-password": () => form("Forgot password", "/forgot-password", "Reset password"),
  "POST /forgot-password": forgotPassword,
};

createServer(async (req, res) => {
  const path = new URL(req.url, baseUrl).pathname;
  let handler = routes[`${req.method} ${path}`];
  if (!handler && req.method === "GET" && path.startsWith("/activate/")) {
    handler = () => page("Activated", "<h1>Account activated</h1>");
  }
  if (!handler && req.method === "GET" && path.startsWith("/reset/")) {
    handler = () =>
      page(
        "Reset password",
        `<h1>Choose a new password</h1><label for="pw">New password</label><input id="pw" type="password">`,
      );
  }
  if (!handler) {
    res.writeHead(404).end("Not found");
    return;
  }
  try {
    const html = await handler(req);
    res.writeHead(200, { "Content-Type": "text/html; charset=utf-8" }).end(html);
  } catch (err) {
    console.error(err);
    res.writeHead(500).end(`Could not send email: ${err.message}`);
  }
}).listen(port, () => console.log(`Demo app on ${baseUrl}`));
