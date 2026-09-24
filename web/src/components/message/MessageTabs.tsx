import { useI18n } from "../../i18n/context";
import { cn } from "../../lib/cn";
import type { Tab, TabInfo } from "../../lib/tabs";

export function MessageTabs({
  tabs,
  active,
  onChange,
}: {
  tabs: TabInfo[];
  active: Tab;
  onChange: (tab: Tab) => void;
}) {
  const { t } = useI18n();
  return (
    <div
      role="tablist"
      aria-label={t("view.tabs")}
      className="flex gap-1 overflow-x-auto border-b border-zinc-200 px-4 py-2 dark:border-zinc-800"
    >
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={active === tab.id}
          onClick={() => onChange(tab.id)}
          className={cn(
            "flex shrink-0 items-center gap-1.5 rounded-lg px-3 py-2 text-xs font-medium transition-colors",
            active === tab.id
              ? "bg-blue-50 text-blue-700 dark:bg-blue-500/15 dark:text-blue-300"
              : "text-zinc-500 hover:bg-zinc-100 hover:text-zinc-800 dark:hover:bg-zinc-800 dark:hover:text-zinc-200",
          )}
        >
          {t(tab.label)}
          {tab.count !== undefined && tab.count > 0 && (
            <span className="rounded-full bg-zinc-100 px-1.5 text-[11px] text-zinc-600 tabular-nums dark:bg-zinc-800 dark:text-zinc-300">
              {tab.count}
            </span>
          )}
        </button>
      ))}
    </div>
  );
}
