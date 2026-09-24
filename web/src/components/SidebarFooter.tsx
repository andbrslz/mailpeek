import { Lock } from "lucide-react";
import type { ConnectionStatus } from "../hooks/useEvents";
import { useI18n } from "../i18n/context";
import type { MessageKey } from "../i18n/messages";
import { cn } from "../lib/cn";
import { CopyButton } from "./ui";

const statusLabel: Record<ConnectionStatus, MessageKey> = {
  live: "status.live",
  connecting: "status.connecting",
  offline: "status.offline",
};

const statusColor: Record<ConnectionStatus, string> = {
  live: "bg-emerald-500",
  connecting: "bg-amber-500",
  offline: "bg-red-500",
};

export function SidebarFooter({
  smtpAddress,
  smtpAuth,
  status,
}: {
  smtpAddress: string;
  smtpAuth: boolean;
  status: ConnectionStatus;
}) {
  const { t } = useI18n();
  return (
    <div className="flex min-h-14 shrink-0 items-center gap-2 border-t border-zinc-200 px-4 text-xs text-zinc-500 dark:border-zinc-800">
      <span className="font-medium tracking-wide uppercase">SMTP</span>
      <code
        title={smtpAddress}
        className="min-w-0 truncate font-mono text-zinc-700 dark:text-zinc-300"
      >
        {smtpAddress}
      </code>
      <CopyButton value={smtpAddress} label={t("smtp.copy")} />
      {smtpAuth && (
        <span
          role="img"
          aria-label={t("smtp.loginRequired")}
          title={t("smtp.loginRequiredTitle")}
          className="flex items-center"
        >
          <Lock className="size-3" aria-hidden="true" />
        </span>
      )}
      <span className="flex-1" />
      <span
        className="flex shrink-0 items-center gap-1.5 whitespace-nowrap"
        role="status"
        title={t("status.title")}
      >
        <span className={cn("size-1.5 rounded-full", statusColor[status])} />
        {t(statusLabel[status])}
      </span>
    </div>
  );
}
