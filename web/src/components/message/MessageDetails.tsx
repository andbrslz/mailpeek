import type { ReactNode } from "react";
import type { Message } from "../../api";
import { useI18n } from "../../i18n/context";
import { cn } from "../../lib/cn";
import { addressList, formatBytes, formatDateTime } from "../../lib/format";

function MetaRow({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-zinc-500">{label}</dt>
      <dd className="min-w-0 break-words">{children}</dd>
    </>
  );
}

export function MessageDetails({ message, collapsed }: { message: Message; collapsed: boolean }) {
  const { t, locale } = useI18n();
  return (
    <div
      id="email-details"
      className={cn(
        "grid transition-[grid-template-rows,opacity,visibility] duration-200 ease-out",
        collapsed ? "invisible grid-rows-[0fr] opacity-0" : "visible grid-rows-[1fr] opacity-100",
      )}
    >
      <div className="min-h-0 overflow-hidden">
        <dl className="mt-3 grid grid-cols-[4.5rem_1fr] gap-x-3 gap-y-1.5 text-[13px]">
          <MetaRow label={t("view.from")}>{addressList([message.from])}</MetaRow>
          <MetaRow label={t("view.to")}>{addressList(message.to) || t("list.none")}</MetaRow>
          {message.cc.length > 0 && (
            <MetaRow label={t("view.cc")}>{addressList(message.cc)}</MetaRow>
          )}
          {message.replyTo.length > 0 && (
            <MetaRow label={t("view.replyTo")}>{addressList(message.replyTo)}</MetaRow>
          )}
          <MetaRow label={t("view.received")}>
            <span className="text-zinc-600 dark:text-zinc-400">
              {formatDateTime(message.createdAt, locale)} · {formatBytes(message.size, locale)}
            </span>
          </MetaRow>
        </dl>
      </div>
    </div>
  );
}
