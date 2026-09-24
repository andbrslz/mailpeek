import { useState, type KeyboardEvent, type PointerEvent } from "react";
import { sidebarMax, sidebarMin } from "./useSidebarWidth";

const step = 16;

export function useResizeHandle({
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
  const [drag, setDrag] = useState<{ x: number; width: number } | null>(null);

  const setDragging = (d: { x: number; width: number } | null) => {
    setDrag(d);
    onDraggingChange(d !== null);
    document.documentElement.classList.toggle("resizing", d !== null);
  };
  const endDrag = () => {
    if (drag) setDragging(null);
  };

  const handlers = {
    onPointerDown(e: PointerEvent<HTMLElement>) {
      if (e.button !== 0) return;
      e.preventDefault();
      e.currentTarget.setPointerCapture(e.pointerId);
      setDragging({ x: e.clientX, width });
    },
    onPointerMove(e: PointerEvent<HTMLElement>) {
      if (drag) onResize(drag.width + e.clientX - drag.x);
    },
    onPointerUp: endDrag,
    onPointerCancel: endDrag,
    onDoubleClick: onReset,
    onKeyDown(e: KeyboardEvent<HTMLElement>) {
      const next: Record<string, number> = {
        ArrowLeft: width - step,
        ArrowRight: width + step,
        Home: sidebarMin,
        End: sidebarMax,
      };
      const value = next[e.key];
      if (value !== undefined) {
        e.preventDefault();
        onResize(value);
      }
    },
  };

  return { dragging: drag !== null, handlers };
}
