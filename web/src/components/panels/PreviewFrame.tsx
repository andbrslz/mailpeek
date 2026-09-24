import { Monitor, Smartphone, Tablet } from "lucide-react";
import type { PreviewWidth } from "../../hooks/usePreview";
import { useI18n } from "../../i18n/context";
import type { MessageKey } from "../../i18n/messages";
import { cn } from "../../lib/cn";

const widthOptions: { value: PreviewWidth; label: MessageKey; Icon: typeof Monitor }[] = [
  { value: "desktop", label: "width.desktop", Icon: Monitor },
  { value: "tablet", label: "width.tablet", Icon: Tablet },
  { value: "mobile", label: "width.mobile", Icon: Smartphone },
];

const frameWidth: Record<PreviewWidth, string> = {
  desktop: "w-full",
  tablet: "mx-auto w-[768px]",
  mobile: "mx-auto w-[375px]",
};

export function WidthToggle({
  width,
  onChange,
}: {
  width: PreviewWidth;
  onChange: (w: PreviewWidth) => void;
}) {
  const { t } = useI18n();
  return (
    <>
      {widthOptions.map(({ value, label, Icon }) => (
        <button
          key={value}
          type="button"
          title={t(label)}
          aria-label={t(label)}
          aria-pressed={width === value}
          onClick={() => onChange(value)}
          className={cn(
            "rounded-md p-2 transition-colors",
            width === value
              ? "bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-100"
              : "text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200",
          )}
        >
          <Icon className="size-3.5" />
        </button>
      ))}
    </>
  );
}

export function PreviewFrame({ srcDoc, width }: { srcDoc: string; width: PreviewWidth }) {
  const { t } = useI18n();
  return (
    <iframe
      title={t("html.frame")}
      sandbox="allow-popups allow-popups-to-escape-sandbox"
      referrerPolicy="no-referrer"
      srcDoc={srcDoc}
      className={cn(
        "h-full max-w-full rounded-lg border border-zinc-200 bg-white shadow-sm dark:border-zinc-700",
        frameWidth[width],
      )}
    />
  );
}
