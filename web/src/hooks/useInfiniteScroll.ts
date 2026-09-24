import { useEffect, useRef } from "react";

export function useInfiniteScroll<L extends HTMLElement, S extends HTMLElement>(
  onEnd: () => void,
  key: unknown,
  margin = 400,
) {
  const listRef = useRef<L>(null);
  const sentinelRef = useRef<S>(null);
  const callback = useRef(onEnd);
  useEffect(() => {
    callback.current = onEnd;
  }, [onEnd]);

  useEffect(() => {
    const root = listRef.current;
    const sentinel = sentinelRef.current;
    if (!root || !sentinel) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) callback.current();
      },
      { root, rootMargin: `0px 0px ${margin}px 0px` },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [key, margin]);

  return { listRef, sentinelRef };
}
