import { enUS } from "./locales/en-US";
import { esES } from "./locales/es-ES";
import { frFR } from "./locales/fr-FR";
import { ptBR } from "./locales/pt-BR";
import { ptPT } from "./locales/pt-PT";

export type MessageKey = keyof typeof enUS;
export type Messages = Record<MessageKey, string>;

export const locales = ["en-US", "en-GB", "pt-BR", "pt-PT", "es-ES", "fr-FR"] as const;
export type Locale = (typeof locales)[number];

export const localeNames: Record<Locale, string> = {
  "en-US": "English (US)",
  "en-GB": "English (UK)",
  "pt-BR": "Português (Brasil)",
  "pt-PT": "Português (Portugal)",
  "es-ES": "Español (España)",
  "fr-FR": "Français (France)",
};

export const messages: Record<Locale, Messages> = {
  "en-US": enUS,
  "en-GB": enUS,
  "pt-BR": ptBR,
  "pt-PT": ptPT,
  "es-ES": esES,
  "fr-FR": frFR,
};
