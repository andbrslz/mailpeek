export interface MailpeekWebServerOptions {
  /** SMTP port (default 1026). */
  smtpPort?: number;
  /** Web UI / API port (default 8026). */
  httpPort?: number;
  /** Docker image (default 4ndbrslz/mailpeek). Ignored when `binary` is set. */
  image?: string;
  /** Run a local binary instead of Docker, e.g. "./bin/mailpeek" or "mailpeek". */
  binary?: string;
  /** Extra flags, e.g. ["--max-messages", "1000"]. */
  args?: string[];
  /** Reuse a Mailpeek that is already running (default: true, except on CI). */
  reuseExistingServer?: boolean;
  /** Startup timeout in ms (default 60000, to allow for an image pull). */
  timeout?: number;
}

/**
 * A `webServer` entry for playwright.config.ts that starts Mailpeek before
 * the tests and stops it afterwards:
 *
 * ```ts
 * webServer: [mailpeekWebServer(), { command: "npm run dev", url: "http://localhost:3000" }]
 * ```
 */
export function mailpeekWebServer(options: MailpeekWebServerOptions = {}) {
  const smtpPort = options.smtpPort ?? 1026;
  const httpPort = options.httpPort ?? 8026;
  const flags = [
    `--smtp-port ${smtpPort}`,
    `--http-port ${httpPort}`,
    ...(options.args ?? []),
  ].join(" ");
  const command = options.binary
    ? `${options.binary} ${flags}`
    : `docker run --rm -p ${smtpPort}:${smtpPort} -p ${httpPort}:${httpPort} ` +
      `${options.image ?? "4ndbrslz/mailpeek"} ${flags}`;
  return {
    command,
    url: `http://localhost:${httpPort}/api/v1/health`,
    reuseExistingServer: options.reuseExistingServer ?? !process.env.CI,
    timeout: options.timeout ?? 60_000,
    stdout: "ignore" as const,
    stderr: "pipe" as const,
  };
}
