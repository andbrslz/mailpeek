import { X } from "lucide-react";
import type { RefObject } from "react";
import { useFullscreenDialog } from "../../hooks/useFullscreenDialog";
import type { PreviewWidth } from "../../hooks/usePreview";
import { useI18n } from "../../i18n/context";
import { cn } from "../../lib/cn";
import { PreviewFrame, WidthToggle } from "./PreviewFrame";

export function FullscreenPreview({
  subject,
  srcDoc,
  width,
  onWidthChange,
  origin,
  closing,
  onClose,
  onClosed,
}: {
  subject: string;
  srcDoc: string;
  width: PreviewWidth;
  onWidthChange: (w: PreviewWidth) => void;
  origin: RefObject<HTMLElement | null>;
  closing: boolean;
  onClose: () => void;
  onClosed: () => void;
}) {
  const { t } = useI18n();
  const { dialog, closeButton } = useFullscreenDialog({ origin, closing, onClose, onClosed });
  return (
    <dialog
      ref={dialog}
      aria-label={t("html.fullscreen")}
      className="m-0 hidden h-dvh max-h-none w-dvw max-w-none flex-col border-0 bg-zinc-100 p-0 text-inherit open:flex backdrop:bg-transparent dark:bg-zinc-950"
    >
      <div className="flex h-12 shrink-0 items-center gap-1 border-b border-zinc-200 bg-white px-4 dark:border-zinc-800 dark:bg-zinc-900">
        <span
          className={cn("mr-auto truncate text-sm font-medium", !subject && "text-zinc-400 italic")}
        >
          {subject || t("list.noSubject")}
        </span>
        <WidthToggle width={width} onChange={onWidthChange} />
        <span className="mx-1 h-4 w-px bg-zinc-200 dark:bg-zinc-800" />
        <button
          ref={closeButton}
          type="button"
          title={t("html.closeTitle")}
          aria-label={t("html.close")}
          onClick={onClose}
          className="rounded-md p-2 text-zinc-500 transition-colors hover:bg-zinc-100 hover:text-zinc-900 dark:text-zinc-400 dark:hover:bg-zinc-800 dark:hover:text-zinc-100"
        >
          <X className="size-4" />
        </button>
      </div>
      <div className="min-h-0 flex-1 p-3 sm:p-6">
        <PreviewFrame srcDoc={srcDoc} width={width} />
      </div>
    </dialog>
  );
}
