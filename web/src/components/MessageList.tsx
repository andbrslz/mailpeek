import { Paperclip } from "lucide-react";
import { useInfiniteScroll } from "../hooks/useInfiniteScroll";
import { useScrollIntoView } from "../hooks/useScrollIntoView";
import type { MessageSummary } from "../api";
import { useI18n } from "../i18n/context";
import { addressLabel, formatDateTime, formatListTime } from "../lib/format";
import { cn } from "../lib/cn";

export function MessageList({
  messages,
  hasMore,
  onLoadMore,
  selectedId,
  unread,
  onSelect,
}: {
  messages: MessageSummary[];
  hasMore: boolean;
  onLoadMore: () => void;
  selectedId: string | null;
  unread: (m: MessageSummary) => boolean;
  onSelect: (id: string) => void;
}) {
  const { t, locale } = useI18n();
  const selectedRef = useScrollIntoView<HTMLLIElement>(selectedId);
  const { listRef, sentinelRef } = useInfiniteScroll<HTMLDivElement, HTMLDivElement>(
    onLoadMore,
    messages,
  );

  return (
    <div ref={listRef} className="flex-1 overflow-y-auto px-2.5 pb-3">
      <ul role="listbox" aria-label={t("app.inbox")} className="space-y-1.5">
        {messages.map((m) => {
          const selected = m.id === selectedId;
          const isUnread = !selected && unread(m);
          return (
            <li
              key={m.id}
              ref={selected ? selectedRef : undefined}
              role="option"
              aria-selected={selected}
              tabIndex={0}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  onSelect(m.id);
                }
              }}
              onClick={() => onSelect(m.id)}
              className={cn(
                "relative cursor-pointer rounded-xl border px-3.5 py-4 transition-colors",
                selected
                  ? "border-blue-200 bg-blue-50 shadow-sm dark:border-blue-500/30 dark:bg-blue-500/10"
                  : "border-transparent hover:border-zinc-200 hover:bg-white dark:hover:border-zinc-700 dark:hover:bg-zinc-900/60",
              )}
            >
              {selected && (
                <span className="absolute inset-y-4 left-0 w-0.5 rounded-full bg-blue-600" />
              )}
              <div className="flex items-baseline gap-2">
                {isUnread && (
                  <span
                    role="img"
                    aria-label={t("list.new")}
                    className="size-1.5 shrink-0 self-center rounded-full bg-blue-600"
                  />
                )}
                <span
                  className={cn(
                    "flex-1 truncate text-sm",
                    isUnread ? "font-semibold" : "font-medium",
                    !m.subject && "text-zinc-400 italic",
                  )}
                >
                  {m.subject || t("list.noSubject")}
                </span>
                <time
                  dateTime={m.createdAt}
                  title={formatDateTime(m.createdAt, locale)}
                  className="shrink-0 text-xs text-zinc-500 tabular-nums"
                >
                  {formatListTime(m.createdAt, locale)}
                </time>
              </div>
              <div className="mt-2 truncate text-[13px] text-zinc-600 dark:text-zinc-400">
                {addressLabel(m.from, t("address.unknown"))}
              </div>
              <div className="mt-1.5 flex items-center gap-2 text-xs text-zinc-500">
                <span className="flex-1 truncate">
                  {t("list.to", { list: m.to.map((a) => a.address).join(", ") || t("list.none") })}
                </span>
                {m.attachments > 0 && (
                  <span
                    className="flex shrink-0 items-center gap-0.5"
                    title={t("list.attachments", { count: m.attachments })}
                  >
                    <Paperclip className="size-3" />
                    {m.attachments}
                  </span>
                )}
              </div>
            </li>
          );
        })}
      </ul>
      {hasMore && (
        <div
          ref={sentinelRef}
          aria-hidden="true"
          className="py-3 text-center text-xs text-zinc-500"
        >
          {t("list.loadingMore")}
        </div>
      )}
    </div>
  );
}
