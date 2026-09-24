# @mailpeek/client

TypeScript client for [Mailpeek](https://github.com/OWNER/mailpeek): wait for, inspect and assert on emails in Vitest, Jest, Playwright or plain scripts.

```bash
npm install -D @mailpeek/client
```

```ts
import { Mailpeek } from "@mailpeek/client";

const mailpeek = new Mailpeek(); // MAILPEEK_URL or http://localhost:8026

const inbox = mailpeek.createInbox(); // test-k7x92ab1@mailpeek.local
await signUp({ email: inbox.address });

const email = await inbox.waitForEmail({ subject: "Welcome" });
expect(email.text).toContain("Hello John");
const link = email.findLink("Activate account"); // { text, href }
```

## API

| Method | Returns |
| --- | --- |
| `messages(filter?)` | `MessageSummary[]`, newest first |
| `get(id)` | `Email` |
| `latest(filter?)` | `Email \| undefined` |
| `waitFor({ ...filter, timeout?, signal? })` | `Email`; throws `MailpeekTimeoutError` (default timeout 10 s) that lists what arrived and why it didn't match |
| `waitForEmails(count, { ...filter, timeout?, signal? })` | `Email[]`, the first `count` matches in arrival order |
| `delete(id)` | `void` |
| `clear(filter?)` | number deleted (all messages when no filter) |
| `raw(id)` | raw MIME `string` |
| `attachment(messageId, attachmentId)` | `Uint8Array` |
| `health()` | `boolean` |
| `info()` | version, ports, limits and `store` usage (`messages`, `bytes`, `evicted`) |
| `createInbox({ prefix?, domain? })` | `Inbox` with a unique address, matched exactly |
| `inbox(address)` | `Inbox` for an existing address (exact match) |

Every method also takes `{ signal }` (the last argument; inside the options for `waitFor`) to cancel it.

Filters (combined with AND): `address` (exact recipient address: To, Cc or envelope), and the case-insensitive substrings `to`, `from`, `subject`, `q` (any of the three); plus `since` (`Date`, ISO string or Unix ms).

`Email` has plain fields (`id`, `from`, `to`, `cc`, `replyTo`, `subject`, `text`, `html`, `headers`, `attachments`, `links`, `envelope`, `date`, `createdAt`) and the helpers `findLink()`, `hasLink()`, `findCode()`, `getHeader()`, `getHeaders()` and `hasAttachment()`. `findLink()` and `findCode()` throw a descriptive error when nothing matches.

```ts
const [welcome, verify] = await inbox.waitForEmails(2);
const code = verify.findCode(); // "482913": the code after "code", "OTP", "PIN"…, else the first 6-digit number
```

Options: `new Mailpeek({ baseUrl, auth, timeout, requestTimeout, inboxDomain, fetch })`. `timeout` (10 s) is how long `waitFor` waits for an email; `requestTimeout` (10 s) bounds each HTTP request, so a stuck connection can't hang a test.

When Mailpeek runs with `--ui-auth`, pass the credentials in the URL (`MAILPEEK_URL=http://admin:secret@localhost:8026`) or as `auth: { username, password }`. A `401` produces an error that says how to fix it.

Requires Node 20+. Ships ESM and CommonJS builds.
