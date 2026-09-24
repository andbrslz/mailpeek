import type { Message } from "../api";
import type { MessageKey } from "../i18n/messages";

export type Tab = "html" | "text" | "headers" | "raw" | "attachments" | "links";

export interface TabInfo {
  id: Tab;
  label: MessageKey;
  count?: number;
}

export function messageTabs(m: Message): TabInfo[] {
  return [
    { id: "html", label: "tab.html" },
    { id: "text", label: "tab.text" },
    { id: "headers", label: "tab.headers" },
    { id: "raw", label: "tab.raw" },
    { id: "attachments", label: "tab.attachments", count: m.attachments.length },
    { id: "links", label: "tab.links", count: m.links.length },
  ];
}

export function effectiveTab(tab: Tab, m: Message): Tab {
  if (tab === "html" && !m.html && m.text) return "text";
  if (tab === "text" && !m.text && m.html) return "html";
  return tab;
}
