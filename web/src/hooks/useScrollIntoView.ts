import { useEffect, useRef } from "react";

export function useScrollIntoView<T extends HTMLElement>(key: unknown) {
  const ref = useRef<T>(null);
  useEffect(() => {
    ref.current?.scrollIntoView({ block: "nearest" });
  }, [key]);
  return ref;
}
