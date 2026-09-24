import { Download } from "lucide-react";
import { rawUrl, type Message } from "../../api";
import { useRawSource } from "../../hooks/useRawSource";
import { useI18n } from "../../i18n/context";
import { CopyButton } from "../ui";
import { Empty } from "./Empty";
import { HighlightedMime } from "./HighlightedMime";

export function RawPanel({ message }: { message: Message }) {
  const { t } = useI18n();
  const raw = useRawSource(message.id);
  if (raw.error)
    return <Empty>{t("raw.error", { error: t(raw.error.key, raw.error.vars) })}</Empty>;
  if (raw.text === undefined) return <Empty>{t("raw.loading")}</Empty>;
  return (
    <div className="relative">
      <div className="absolute top-2 right-3 flex items-center gap-1">
        <CopyButton value={raw.text} showLabel />
        <a
          href={rawUrl(message.id, true)}
          className="inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-zinc-500 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
        >
          <Download className="size-3.5" /> .eml
        </a>
      </div>
      <HighlightedMime raw={raw.text} />
    </div>
  );
}
