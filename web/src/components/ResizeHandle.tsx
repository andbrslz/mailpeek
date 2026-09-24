import { useResizeHandle } from "../hooks/useResizeHandle";
import { sidebarMax, sidebarMin } from "../hooks/useSidebarWidth";
import { useI18n } from "../i18n/context";
import { cn } from "../lib/cn";

export function ResizeHandle({
  width,
  onResize,
  onReset,
  onDraggingChange,
}: {
  width: number;
  onResize: (width: number) => void;
  onReset: () => void;
  onDraggingChange: (dragging: boolean) => void;
}) {
  const { t } = useI18n();
  const { dragging, handlers } = useResizeHandle({ width, onResize, onReset, onDraggingChange });

  return (
    <div
      role="separator"
      aria-label={t("resize.label")}
      aria-orientation="vertical"
      aria-valuemin={sidebarMin}
      aria-valuemax={sidebarMax}
      aria-valuenow={width}
      tabIndex={0}
      title={t("resize.title")}
      {...handlers}
      className="group relative z-10 -mx-1.5 hidden w-3 shrink-0 cursor-col-resize touch-none md:block"
    >
      <span
        className={cn(
          "absolute inset-y-0 left-1/2 w-0.5 -translate-x-1/2 transition-colors",
          dragging
            ? "bg-blue-500"
            : "bg-transparent group-hover:bg-blue-500/50 group-focus-visible:bg-blue-500",
        )}
      />
    </div>
  );
}
