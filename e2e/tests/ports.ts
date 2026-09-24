// Non-default ports so the suite never collides with a Mailpeek you already run.
export const smtpPort = Number(process.env.MAILPEEK_TEST_SMTP_PORT ?? 2525);
export const httpPort = Number(process.env.MAILPEEK_TEST_HTTP_PORT ?? 8125);

// A second instance with --smtp-auth and --ui-auth (see auth.spec.ts).
export const authSmtpPort = 2526;
export const authHttpPort = 8126;
export const smtpCredentials = { user: "app", pass: "smtp-secret" };
export const uiCredentials = { username: "admin", password: "ui-secret" };
