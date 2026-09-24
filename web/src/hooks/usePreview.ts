import { useMemo, useRef, useState } from "react";
import type { Message } from "../api";
import { prepareHtml } from "../lib/html";

export type PreviewWidth = "desktop" | "tablet" | "mobile";
export type FullscreenState = "closed" | "open" | "closing";

export function usePreview(message: Message) {
  const [width, setWidth] = useState<PreviewWidth>("desktop");
  const [fullscreen, setFullscreen] = useState<FullscreenState>("closed");
  const stage = useRef<HTMLDivElement>(null);
  const srcDoc = useMemo(() => prepareHtml(message), [message]);
  return {
    width,
    setWidth,
    fullscreen,
    open: () => setFullscreen("open"),
    close: () => setFullscreen("closing"),
    closed: () => setFullscreen("closed"),
    stage,
    srcDoc,
  };
}
