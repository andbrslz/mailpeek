import { ArrowLeft, ChevronUp, Download, Trash2 } from "lucide-react";
import { rawUrl, type Message } from "../../api";
import { useI18n } from "../../i18n/context";
import { cn } from "../../lib/cn";
import { addressLabel, formatListTime } from "../../lib/format";
import { IconButton } from "../ui";
import { MessageDetails } from "./MessageDetails";

export function MessageHeader({
  message,
  collapsed,
  onToggleCollapsed,
  onDelete,
  onBack,
}: {
  message: Message;
  collapsed: boolean;
  onToggleCollapsed: () => void;
  onDelete: () => void;
  onBack: () => void;
}) {
  const { t, locale } = useI18n();
  return (
    <div
      className={cn(
        "border-b border-zinc-200 px-5 transition-[padding] duration-200 ease-out sm:px-7 dark:border-zinc-800",
        collapsed ? "py-2.5" : "pt-6 pb-5",
      )}
    >
      <div className={cn("flex gap-2", collapsed ? "items-center" : "items-start")}>
        <IconButton label={t("view.back")} onClick={onBack} className="-ml-2 md:hidden">
          <ArrowLeft className="size-4" />
        </IconButton>
        <h1
          className={cn(
            "min-w-0 flex-1 leading-snug font-semibold tracking-tight transition-[font-size] duration-200 ease-out",
            collapsed ? "truncate text-base" : "text-xl break-words",
            !message.subject && "text-zinc-400 italic",
          )}
          title={collapsed ? message.subject : undefined}
        >
          {message.subject || t("list.noSubject")}
        </h1>
        {collapsed && (
          <span className="hidden max-w-[40%] shrink-0 animate-fade-in truncate text-[13px] text-zinc-500 lg:inline">
            {addressLabel(message.from, t("address.unknown"))} ·{" "}
            {formatListTime(message.createdAt, locale)}
          </span>
        )}
        <IconButton
          label={t("view.details")}
          title={collapsed ? t("view.showDetails") : t("view.hideDetails")}
          aria-expanded={!collapsed}
          aria-controls="email-details"
          onClick={onToggleCollapsed}
        >
          <ChevronUp
            className={cn("size-4 transition-transform duration-200", collapsed && "rotate-180")}
          />
        </IconButton>
        <a
          href={rawUrl(message.id, true)}
          title={t("view.download")}
          aria-label={t("view.download")}
          className="inline-flex size-8 items-center justify-center rounded-md text-zinc-500 hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
        >
          <Download className="size-4" />
        </a>
        <IconButton label={t("view.delete")} tone="danger" onClick={onDelete}>
          <Trash2 className="size-4" />
        </IconButton>
      </div>
      <MessageDetails message={message} collapsed={collapsed} />
    </div>
  );
}
