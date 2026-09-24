import { Check, Copy } from "lucide-react";
import type { ButtonHTMLAttributes, ReactNode } from "react";
import { useCopyToClipboard } from "../hooks/useCopyToClipboard";
import { useI18n } from "../i18n/context";
import { cn } from "../lib/cn";

export function CopyButton({
  value,
  label,
  className,
  showLabel = false,
}: {
  value: string;
  label?: string;
  className?: string;
  showLabel?: boolean;
}) {
  const { copied, copy } = useCopyToClipboard();
  const { t } = useI18n();
  const text = label ?? t("copy.copy");
  const Icon = copied ? Check : Copy;
  return (
    <button
      type="button"
      title={copied ? t("copy.copied") : text}
      aria-label={text}
      onClick={() => void copy(value)}
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md text-zinc-500 transition-colors hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100",
        showLabel ? "px-2 py-1 text-xs font-medium" : "p-1",
        className,
      )}
    >
      <Icon className={cn("size-3.5", copied && "text-emerald-600 dark:text-emerald-400")} />
      {showLabel && (copied ? t("copy.copied") : text)}
    </button>
  );
}

type IconButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  label: string;
  children: ReactNode;
  tone?: "default" | "danger";
};

export function IconButton({
  label,
  children,
  tone = "default",
  className,
  ...rest
}: IconButtonProps) {
  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      className={cn(
        "inline-flex size-8 items-center justify-center rounded-md text-zinc-500 transition-colors hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800",
        tone === "danger"
          ? "hover:text-red-600 dark:hover:text-red-400"
          : "hover:text-zinc-900 dark:hover:text-zinc-100",
        className,
      )}
      {...rest}
    >
      {children}
    </button>
  );
}

export function Kbd({ children }: { children: ReactNode }) {
  return (
    <kbd className="rounded border border-zinc-200 px-1 font-sans text-[10px] font-medium text-zinc-400 dark:border-zinc-700 dark:text-zinc-500">
      {children}
    </kbd>
  );
}
