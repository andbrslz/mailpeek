export type Theme = "light" | "dark";

const storageKey = "mailpeek:theme";
const darkQuery = () => window.matchMedia("(prefers-color-scheme: dark)");

export function savedTheme(): Theme | undefined {
  try {
    const v = window.localStorage.getItem(storageKey);
    return v === "light" || v === "dark" ? v : undefined;
  } catch {
    return undefined;
  }
}

export function saveTheme(theme: Theme) {
  try {
    window.localStorage.setItem(storageKey, theme);
  } catch {}
}

export const systemTheme = (): Theme => (darkQuery().matches ? "dark" : "light");

export function subscribeSystemTheme(onChange: () => void) {
  const query = darkQuery();
  query.addEventListener("change", onChange);
  return () => query.removeEventListener("change", onChange);
}

export function applyTheme(theme: Theme) {
  document.documentElement.classList.toggle("dark", theme === "dark");
}
