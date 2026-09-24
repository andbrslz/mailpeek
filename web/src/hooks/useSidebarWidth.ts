import { useCallback, useEffect, useState, useSyncExternalStore } from "react";

const storageKey = "mailpeek:list-width";
export const sidebarMin = 240;
export const sidebarMax = 640;
export const sidebarDefault = 360;

const bound = (width: number) => Math.round(Math.min(Math.max(width, sidebarMin), sidebarMax));

function fit(width: number, viewport: number): number {
  return Math.min(width, Math.max(sidebarMin, viewport - 420));
}

function read(): number {
  try {
    const n = Number(window.localStorage.getItem(storageKey));
    return n ? bound(n) : sidebarDefault;
  } catch {
    return sidebarDefault;
  }
}

const subscribe = (onChange: () => void) => {
  window.addEventListener("resize", onChange);
  return () => window.removeEventListener("resize", onChange);
};
const viewportWidth = () => window.innerWidth;

export function useSidebarWidth() {
  const [preferred, setPreferred] = useState(read);
  const viewport = useSyncExternalStore(subscribe, viewportWidth, () => 1280);
  const setWidth = useCallback((w: number) => setPreferred(fit(bound(w), window.innerWidth)), []);

  useEffect(() => {
    const t = window.setTimeout(() => {
      try {
        window.localStorage.setItem(storageKey, String(preferred));
      } catch {}
    }, 200);
    return () => window.clearTimeout(t);
  }, [preferred]);

  return { width: fit(preferred, viewport), setWidth, reset: () => setWidth(sidebarDefault) };
}
