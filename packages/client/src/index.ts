export {
  Mailpeek,
  MailpeekError,
  MailpeekTimeoutError,
  type TimeoutDiagnostics,
} from "./client.js";
export { explainMismatch } from "./diagnose.js";
export { Email, type LinkMatcher } from "./email.js";
export { Inbox, randomId, type InboxClient, type InboxOptions } from "./inbox.js";
export type {
  Address,
  Attachment,
  Link,
  MailpeekOptions,
  MessageData,
  MessageFilter,
  MessageSummary,
  RequestOptions,
  ServerInfo,
  WaitOptions,
} from "./types.js";
