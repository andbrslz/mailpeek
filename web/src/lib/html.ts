import { attachmentUrl, type Message } from "../api";

export function prepareHtml(message: Message, origin = window.location.origin): string {
  const byCid = new Map<string, string>();
  for (const a of message.attachments) {
    if (a.contentId) byCid.set(a.contentId.toLowerCase(), origin + attachmentUrl(message.id, a.id));
  }
  const html = message.html.replace(/cid:([^"'\s)>]+)/gi, (match, cid: string) => {
    let key = cid;
    try {
      key = decodeURIComponent(cid);
    } catch {}
    return byCid.get(key.toLowerCase()) ?? match;
  });
  return `<base target="_blank">${html}`;
}
