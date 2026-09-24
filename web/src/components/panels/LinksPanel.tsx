import { ExternalLink } from "lucide-react";
import type { Message } from "../../api";
import { useI18n } from "../../i18n/context";
import { cn } from "../../lib/cn";
import { CopyButton } from "../ui";
import { Empty } from "./Empty";

export function LinksPanel({ message }: { message: Message }) {
  const { t } = useI18n();
  if (message.links.length === 0) return <Empty>{t("links.none")}</Empty>;
  return (
    <ul className="divide-y divide-zinc-100 dark:divide-zinc-900">
      {message.links.map((l) => (
        <li key={`${l.text}|${l.href}`} className="group flex items-center gap-3 px-5 py-3">
          <div className="min-w-0 flex-1">
            <div className={cn("truncate text-sm font-medium", !l.text && "text-zinc-400 italic")}>
              {l.text || t("links.noText")}
            </div>
            <div className="mt-0.5 truncate font-mono text-xs text-zinc-500" title={l.href}>
              {l.href}
            </div>
          </div>
          <CopyButton value={l.href} label={t("links.copy")} showLabel />
          <a
            href={l.href}
            target="_blank"
            rel="noopener noreferrer"
            title={t("links.openTitle")}
            aria-label={t("links.open", { target: l.text || l.href })}
            className="rounded p-1 text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
          >
            <ExternalLink className="size-3.5" />
          </a>
        </li>
      ))}
    </ul>
  );
}
