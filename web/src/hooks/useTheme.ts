import { useCallback, useEffect, useState, useSyncExternalStore } from "react";
import {
  applyTheme,
  saveTheme,
  savedTheme,
  subscribeSystemTheme,
  systemTheme,
  type Theme,
} from "../lib/theme";

export function useTheme() {
  const [choice, setChoice] = useState<Theme | undefined>(savedTheme);
  const system = useSyncExternalStore(subscribeSystemTheme, systemTheme, () => "light" as const);
  const theme = choice ?? system;

  useEffect(() => applyTheme(theme), [theme]);

  const toggle = useCallback(() => {
    const next = theme === "dark" ? "light" : "dark";
    saveTheme(next);
    setChoice(next);
  }, [theme]);

  return { theme, toggle };
}
