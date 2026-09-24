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
  FailureOptions,
  MailpeekOptions,
  MessageData,
  MessageFilter,
  MessageSummary,
  RequestOptions,
  ServerInfo,
  SmtpFailure,
  WaitOptions,
} from "./types.js";
