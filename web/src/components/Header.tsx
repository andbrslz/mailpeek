import { Bell, BellOff, LogOut, PanelLeftClose, PanelLeftOpen, Search, X } from "lucide-react";
import type { RefObject } from "react";
import { useI18n } from "../i18n/context";
import { ClearButton } from "./header/ClearButton";
import { LanguagePicker } from "./header/LanguagePicker";
import { ThemeToggle } from "./header/ThemeToggle";
import { IconButton, Kbd } from "./ui";

export function Logo() {
  const { t } = useI18n();
  return (
    <div className="flex shrink-0 items-center gap-2.5 text-lg font-semibold tracking-tight">
      <img src="/favicon.svg" alt="" className="size-8 shrink-0" />
      <span className="max-sm:sr-only">Mailpeek</span>
      <span className="hidden rounded-full border border-blue-200 bg-blue-50 px-2 py-0.5 text-[10px] font-semibold tracking-wider text-blue-700 uppercase lg:inline dark:border-blue-900 dark:bg-blue-950 dark:text-blue-300">
        {t("header.badge")}
      </span>
    </div>
  );
}

export function Header({
  query,
  onQueryChange,
  onClear,
  canClear,
  searchRef,
  soundEnabled,
  onToggleSound,
  listHidden,
  onToggleList,
  onSignOut,
}: {
  query: string;
  onQueryChange: (q: string) => void;
  onClear: () => unknown;
  canClear: boolean;
  searchRef: RefObject<HTMLInputElement | null>;
  soundEnabled: boolean;
  onToggleSound: () => void;
  listHidden: boolean;
  onToggleList: () => void;
  onSignOut?: () => unknown;
}) {
  const { t } = useI18n();
  return (
    <header className="flex shrink-0 flex-wrap items-center gap-x-3 gap-y-2 border-b border-zinc-200 px-3 py-2 sm:h-16 sm:flex-nowrap sm:py-0 sm:px-6 dark:border-zinc-800">
      <Logo />
      <IconButton
        label={t("header.list")}
        title={listHidden ? t("header.showList") : t("header.hideList")}
        aria-expanded={!listHidden}
        aria-controls="email-list"
        onClick={onToggleList}
        className="shrink-0 max-md:hidden"
      >
        {listHidden ? <PanelLeftOpen className="size-4" /> : <PanelLeftClose className="size-4" />}
      </IconButton>
      <div className="hidden flex-1 sm:block" />
      <label className="relative order-last flex w-full min-w-0 items-center sm:order-none sm:max-w-96">
        <Search className="pointer-events-none absolute left-2.5 size-3.5 text-zinc-400" />
        <input
          ref={searchRef}
          type="search"
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Escape") {
              onQueryChange("");
              e.currentTarget.blur();
            }
          }}
          placeholder={t("header.search")}
          aria-label={t("header.searchLabel")}
          className="h-11 w-full rounded-lg sm:h-9 border border-zinc-200 bg-zinc-50 pr-8 pl-8 text-base sm:text-sm outline-none placeholder:text-zinc-400 focus:border-blue-500 focus:bg-white focus:ring-[3px] focus:ring-blue-500/15 dark:border-zinc-800 dark:bg-zinc-900 dark:focus:bg-zinc-950 [&::-webkit-search-cancel-button]:hidden"
        />
        <span className="absolute right-2">
          {query ? (
            <button
              type="button"
              aria-label={t("header.clearSearch")}
              onClick={() => onQueryChange("")}
              className="rounded p-0.5 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200"
            >
              <X className="size-3.5" />
            </button>
          ) : (
            <span className="hidden sm:inline">
              <Kbd>/</Kbd>
            </span>
          )}
        </span>
      </label>
      <div className="ml-auto flex shrink-0 items-center gap-1 sm:ml-0 sm:gap-3 max-sm:[&>button]:size-11 max-sm:[&>button]:justify-center max-sm:[&>div>button]:size-11">
        <IconButton
          label={t("header.sound")}
          title={soundEnabled ? t("header.soundOn") : t("header.soundOff")}
          aria-pressed={soundEnabled}
          onClick={onToggleSound}
          className="shrink-0"
        >
          {soundEnabled ? <Bell className="size-4" /> : <BellOff className="size-4" />}
        </IconButton>
        <ThemeToggle />
        <LanguagePicker />
        <ClearButton onConfirm={onClear} disabled={!canClear} />
        {onSignOut && (
          <IconButton
            label={t("header.signOut")}
            onClick={() => void onSignOut()}
            className="shrink-0"
          >
            <LogOut className="size-4" />
          </IconButton>
        )}
      </div>
    </header>
  );
}
