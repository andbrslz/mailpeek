import { createContext, useContext } from "react";
import { messages, type Locale, type MessageKey } from "./messages";

export interface I18n {
  locale: Locale;
  choice: Locale | undefined;
  setChoice: (locale: Locale | undefined) => void;
  t: (key: MessageKey, vars?: Record<string, string | number>) => string;
}

export function translate(
  locale: Locale,
  key: MessageKey,
  vars: Record<string, string | number> = {},
): string {
  return messages[locale][key].replace(/\{(\w+)\}/g, (m, name: string) =>
    name in vars ? String(vars[name]) : m,
  );
}

export const I18nContext = createContext<I18n>({
  locale: "en-US",
  choice: undefined,
  setChoice: () => undefined,
  t: (key, vars) => translate("en-US", key, vars),
});

export const useI18n = () => useContext(I18nContext);
