import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import { I18nContext, translate, type I18n } from "./context";
import { detectLocale, readSavedLocale, saveLocale } from "./locale";
import type { Locale } from "./messages";

const subscribe = (onChange: () => void) => {
  window.addEventListener("languagechange", onChange);
  return () => window.removeEventListener("languagechange", onChange);
};
const browserLocale = () => detectLocale();

export function I18nProvider({ children }: { children: ReactNode }) {
  const [choice, setChoiceState] = useState<Locale | undefined>(readSavedLocale);
  const detected = useSyncExternalStore(subscribe, browserLocale, () => "en-US" as const);
  const locale = choice ?? detected;

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const setChoice = useCallback((next: Locale | undefined) => {
    saveLocale(next);
    setChoiceState(next);
  }, []);

  const value = useMemo<I18n>(
    () => ({ locale, choice, setChoice, t: (key, vars) => translate(locale, key, vars) }),
    [locale, choice, setChoice],
  );
  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}
