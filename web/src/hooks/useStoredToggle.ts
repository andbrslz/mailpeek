import { useCallback, useEffect, useState } from "react";

function read(key: string, fallback: boolean): boolean {
  try {
    const v = window.localStorage.getItem(key);
    return v === null ? fallback : v === "on";
  } catch {
    return fallback;
  }
}

export function useStoredToggle(key: string, fallback: boolean) {
  const [value, setValue] = useState(() => read(key, fallback));

  useEffect(() => {
    try {
      window.localStorage.setItem(key, value ? "on" : "off");
    } catch {}
  }, [key, value]);

  const toggle = useCallback(() => setValue((v) => !v), []);
  return [value, toggle] as const;
}
