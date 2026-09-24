import { ArrowDown, Inbox, Mail, SearchX } from "lucide-react";
import { useI18n } from "../i18n/context";
import { CopyButton } from "./ui";

export function EmptyInbox({ smtpAddress, smtpAuth }: { smtpAddress: string; smtpAuth: boolean }) {
  const { t } = useI18n();
  const [loginBefore, loginAfter] = t("empty.login").split("{env}");
  return (
    <div className="empty-canvas flex min-h-0 flex-1 items-center justify-center overflow-y-auto px-6 py-12">
      <div className="w-full max-w-md text-center">
        <div
          className="relative mx-auto mb-8 flex h-32 w-44 items-end justify-center"
          aria-hidden="true"
        >
          <div className="absolute inset-x-3 top-0 h-28 rounded-full border border-blue-100 dark:border-blue-950" />
          <div className="absolute top-2 flex h-20 w-28 -rotate-12 items-center justify-center rounded-xl border border-zinc-200 bg-white shadow-sm dark:border-zinc-700 dark:bg-zinc-900">
            <Mail className="size-9 text-blue-400" strokeWidth={1.25} />
          </div>
          <div className="relative flex h-16 w-24 items-center justify-center rounded-2xl border border-blue-200 bg-blue-50 shadow-sm dark:border-blue-800 dark:bg-blue-950">
            <Inbox className="size-8 text-blue-500" strokeWidth={1.5} />
          </div>
          <div className="absolute right-2 top-12 rounded-full border border-zinc-200 bg-white p-2 text-blue-500 shadow-sm dark:border-zinc-800 dark:bg-zinc-900">
            <ArrowDown className="size-4" />
          </div>
        </div>
        <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">{t("empty.title")}</h2>
        <p className="mx-auto mt-3 max-w-xs text-sm leading-6 text-zinc-500 dark:text-zinc-400">
          {t("empty.configure")}
        </p>
        <div className="mt-7 overflow-hidden rounded-xl border border-zinc-200 bg-white text-left shadow-sm dark:border-zinc-800 dark:bg-zinc-900/70">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b border-zinc-100 px-4 py-3 dark:border-zinc-800">
            <span className="text-xs font-medium tracking-wider text-zinc-500 uppercase">SMTP</span>
            <CopyButton
              value={smtpAddress}
              label={t("smtp.copy")}
              showLabel
              className="hover:bg-zinc-100 dark:hover:bg-zinc-800"
            />
          </div>
          <div className="px-4 py-4">
            <code className="break-all font-mono text-base font-medium text-zinc-800 dark:text-zinc-200">
              {smtpAddress}
            </code>
          </div>
        </div>
        {smtpAuth && (
          <p className="mt-4 text-xs text-zinc-500">
            {loginBefore}
            <code className="font-mono">MAILPEEK_SMTP_AUTH</code>
            {loginAfter}
          </p>
        )}
        <p className="mt-5 text-xs leading-5 text-zinc-500">{t("empty.footer")}</p>
      </div>
    </div>
  );
}

export function NoResults({ query }: { query: string }) {
  const { t } = useI18n();
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-2 p-6 text-center text-sm text-zinc-500">
      <SearchX className="size-5 text-zinc-400" />
      <p>{t("empty.noResults", { query })}</p>
    </div>
  );
}
