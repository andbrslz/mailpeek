import { Trash2 } from "lucide-react";
import { useConfirmAction } from "../../hooks/useConfirmAction";
import { useI18n } from "../../i18n/context";
import { cn } from "../../lib/cn";

export function ClearButton({ onConfirm, disabled }: { onConfirm: () => void; disabled: boolean }) {
  const { t } = useI18n();
  const { armed, trigger } = useConfirmAction(onConfirm);
  return (
    <button
      type="button"
      disabled={disabled}
      aria-label={armed ? t("header.confirmClear") : t("header.clearInbox")}
      onClick={trigger}
      className={cn(
        "inline-flex h-8 shrink-0 items-center gap-1.5 rounded-md border px-2.5 text-sm font-medium transition-colors disabled:pointer-events-none disabled:opacity-40",
        armed
          ? "border-red-600 bg-red-600 text-white hover:bg-red-700"
          : "border-zinc-200 text-zinc-700 hover:bg-zinc-100 dark:border-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-900",
      )}
    >
      <Trash2 className="size-3.5" />
      <span className="hidden sm:inline">
        {armed ? t("header.confirmClear") : t("header.clear")}
      </span>
    </button>
  );
}
