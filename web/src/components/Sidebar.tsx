import { Inbox, Mail } from "lucide-react";
import type { CSSProperties, ReactNode } from "react";
import { useI18n } from "../i18n/context";
import { cn } from "../lib/cn";

export function Sidebar({
  count,
  width,
  hidden,
  hiddenOnPhone,
  animate,
  footer,
  children,
}: {
  count: number;
  width: number;
  hidden: boolean;
  hiddenOnPhone: boolean;
  animate: boolean;
  footer: ReactNode;
  children: ReactNode;
}) {
  const { t } = useI18n();
  return (
    <aside
      id="email-list"
      style={{ "--sidebar-width": `${width}px` } as CSSProperties}
      className={cn(
        "min-h-0 w-full shrink-0 flex-col overflow-hidden border-zinc-200 bg-zinc-50/70 md:flex md:border-r dark:border-zinc-800 dark:bg-zinc-900/30",
        hiddenOnPhone ? "hidden" : "flex",
        hidden ? "md:invisible md:w-0 md:border-r-0" : "md:visible md:w-[var(--sidebar-width)]",
        animate && "md:transition-[width,visibility] md:duration-200 md:ease-out",
      )}
    >
      <div className="flex min-h-0 flex-1 flex-col md:h-full md:w-[var(--sidebar-width)] md:flex-none">
        <div className="flex h-16 shrink-0 items-center justify-between px-5 text-sm font-semibold">
          <span className="flex items-center gap-2">
            <Inbox className="size-4 text-blue-500" /> {t("app.inbox")}
          </span>
          <span className="rounded-md border border-zinc-200 bg-white px-2 py-0.5 text-xs text-zinc-500 tabular-nums dark:border-zinc-700 dark:bg-zinc-900">
            {count}
          </span>
        </div>
        {children}
        {footer}
      </div>
    </aside>
  );
}

export function SidebarPlaceholder() {
  const { t } = useI18n();
  return (
    <div className="flex flex-1 flex-col items-center justify-center px-8 pb-16 text-center">
      <div className="mb-4 flex size-11 items-center justify-center rounded-xl border border-dashed border-zinc-300 text-zinc-400 dark:border-zinc-700 dark:text-zinc-500">
        <Mail className="size-5" strokeWidth={1.5} aria-hidden="true" />
      </div>
      <p className="max-w-48 text-sm leading-6 text-zinc-500 dark:text-zinc-400">
        {t("app.emptyList")}
      </p>
    </div>
  );
}
