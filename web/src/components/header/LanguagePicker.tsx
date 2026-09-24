import { Check, Globe } from "lucide-react";
import { useListboxMenu } from "../../hooks/useListboxMenu";
import { useI18n } from "../../i18n/context";
import { localeFlags } from "../../i18n/flags";
import { toLocale } from "../../i18n/locale";
import { localeNames, locales, type Locale } from "../../i18n/messages";
import { cn } from "../../lib/cn";

type Option = Locale | "auto";
const options: readonly Option[] = ["auto", ...locales];

const round = "size-5 shrink-0 rounded-full ring-1 ring-zinc-900/10 dark:ring-white/15";

function Flag({ locale }: { locale: Option }) {
  if (locale === "auto") {
    return (
      <span className={cn(round, "grid place-items-center bg-zinc-100 dark:bg-zinc-800")}>
        <Globe className="size-3.5 text-zinc-500 dark:text-zinc-400" aria-hidden="true" />
      </span>
    );
  }
  return <img src={localeFlags[locale]} alt="" width={20} height={20} className={round} />;
}

export function LanguagePicker() {
  const { t, locale, choice, setChoice } = useI18n();
  const {
    open,
    active,
    rootRef,
    triggerRef,
    listRef,
    toggle,
    choose,
    onTriggerKeyDown,
    onListKeyDown,
    onBlur,
  } = useListboxMenu<Option>({
    options,
    value: choice ?? "auto",
    onSelect: (v) => setChoice(toLocale(v)),
  });

  return (
    <div ref={rootRef} onBlur={onBlur} className="relative shrink-0">
      <button
        ref={triggerRef}
        type="button"
        aria-label={t("header.language")}
        title={`${t("header.language")} (${localeNames[locale]})`}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls="language-list"
        onClick={toggle}
        onKeyDown={onTriggerKeyDown}
        className={cn(
          "inline-flex size-8 items-center justify-center rounded-full transition-colors hover:bg-zinc-100 focus-visible:ring-2 focus-visible:ring-blue-500/40 focus-visible:outline-none dark:hover:bg-zinc-800",
          open && "bg-zinc-100 dark:bg-zinc-800",
        )}
      >
        <Flag locale={locale} />
      </button>
      {open && (
        <div
          ref={listRef}
          id="language-list"
          role="listbox"
          aria-label={t("header.language")}
          onKeyDown={onListKeyDown}
          className="animate-fade-in absolute top-full right-0 z-50 mt-1.5 w-72 max-w-[calc(100vw-1.5rem)] origin-top-right rounded-lg border border-zinc-200 bg-white p-1 shadow-lg shadow-zinc-900/10 dark:border-zinc-800 dark:bg-zinc-900 dark:shadow-black/40"
        >
          {options.map((option, i) => {
            const selected = option === (choice ?? "auto");
            return (
              <button
                key={option}
                type="button"
                role="option"
                aria-selected={selected}
                tabIndex={i === active ? 0 : -1}
                lang={option === "auto" ? undefined : option}
                onClick={() => choose(option)}
                className={cn(
                  "flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-sm text-zinc-700 transition-colors outline-none hover:bg-zinc-100 focus-visible:bg-zinc-100 dark:text-zinc-200 dark:hover:bg-zinc-800 dark:focus-visible:bg-zinc-800",
                  selected && "font-medium text-zinc-900 dark:text-zinc-50",
                )}
              >
                <Flag locale={option} />
                <span className="flex-1 truncate">
                  {option === "auto"
                    ? `${t("language.auto")}${choice ? "" : ` (${localeNames[locale]})`}`
                    : localeNames[option]}
                </span>
                {selected && (
                  <Check aria-hidden="true" className="size-4 text-blue-600 dark:text-blue-400" />
                )}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
