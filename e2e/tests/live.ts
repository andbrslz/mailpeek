import type { Page } from "@playwright/test";

/**
 * Resolves once the page's live-update stream is open, so emails sent next
 * arrive as events. Call it before the navigation that opens the stream.
 */
export const liveEvents = (page: Page) =>
  page.waitForResponse((r) => new URL(r.url()).pathname === "/api/v1/events" && r.ok());
