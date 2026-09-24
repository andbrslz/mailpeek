import { Maximize2 } from "lucide-react";
import type { Message } from "../../api";
import { usePreview } from "../../hooks/usePreview";
import { useI18n } from "../../i18n/context";
import { Empty } from "./Empty";
import { FullscreenPreview } from "./FullscreenPreview";
import { PreviewFrame, WidthToggle } from "./PreviewFrame";

export function HtmlPanel({ message }: { message: Message }) {
  const { t } = useI18n();
  const { stage, width, setWidth, fullscreen, open, close, closed, srcDoc } = usePreview(message);
  if (!message.html) return <Empty>{t("html.none")}</Empty>;

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-1 border-b border-zinc-200 bg-zinc-50/70 px-5 py-2 dark:border-zinc-800 dark:bg-zinc-900/40">
        <span className="mr-auto text-[11px] font-medium tracking-wide text-zinc-500 uppercase">
          {t("html.preview")}
        </span>
        <WidthToggle width={width} onChange={setWidth} />
        <span className="mx-1 h-4 w-px bg-zinc-200 dark:bg-zinc-800" />
        <button
          type="button"
          title={t("html.fullscreenTitle")}
          aria-label={t("html.fullscreen")}
          onClick={open}
          className="rounded-md p-2 text-zinc-400 transition-colors hover:text-zinc-700 dark:hover:text-zinc-200"
        >
          <Maximize2 className="size-3.5" />
        </button>
      </div>
      <div
        ref={stage}
        className="min-h-0 flex-1 overflow-hidden bg-zinc-100/80 p-3 sm:p-5 dark:bg-zinc-950"
      >
        {fullscreen !== "open" && <PreviewFrame srcDoc={srcDoc} width={width} />}
      </div>
      {fullscreen !== "closed" && (
        <FullscreenPreview
          subject={message.subject}
          srcDoc={srcDoc}
          width={width}
          onWidthChange={setWidth}
          origin={stage}
          closing={fullscreen === "closing"}
          onClose={close}
          onClosed={closed}
        />
      )}
    </div>
  );
}
