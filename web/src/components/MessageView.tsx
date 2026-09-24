import type { ReactNode } from "react";
import type { Message } from "../api";
import { useI18n } from "../i18n/context";
import { cn } from "../lib/cn";
import { effectiveTab, messageTabs, type Tab } from "../lib/tabs";
import { MessageHeader } from "./message/MessageHeader";
import { MessageTabs } from "./message/MessageTabs";
import {
  AttachmentsPanel,
  HeadersPanel,
  HtmlPanel,
  LinksPanel,
  RawPanel,
  TextPanel,
} from "./panels";

const panels: Record<Tab, (p: { message: Message }) => ReactNode> = {
  html: HtmlPanel,
  text: TextPanel,
  headers: HeadersPanel,
  raw: RawPanel,
  attachments: AttachmentsPanel,
  links: LinksPanel,
};

export function MessageView({
  message,
  tab,
  onTabChange,
  collapsed,
  onToggleCollapsed,
  onDelete,
  onBack,
}: {
  message: Message;
  tab: Tab;
  onTabChange: (t: Tab) => void;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  onDelete: () => void;
  onBack: () => void;
}) {
  const { t } = useI18n();
  const active = effectiveTab(tab, message);
  const Panel = panels[active];
  return (
    <article className="flex min-h-0 flex-1 flex-col" aria-label={t("view.email")}>
      <MessageHeader
        message={message}
        collapsed={collapsed}
        onToggleCollapsed={onToggleCollapsed}
        onDelete={onDelete}
        onBack={onBack}
      />
      <MessageTabs tabs={messageTabs(message)} active={active} onChange={onTabChange} />
      <div
        role="tabpanel"
        className={cn("min-h-0 flex-1", active === "html" ? "overflow-hidden" : "overflow-y-auto")}
      >
        <Panel message={message} />
      </div>
    </article>
  );
}
