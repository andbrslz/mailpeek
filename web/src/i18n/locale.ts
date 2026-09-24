import { locales, type Locale } from "./messages";

const storageKey = "mailpeek:locale";
const cookieName = "mailpeek_lang";

const europeanPortuguese = ["pt-pt", "pt-ao", "pt-mz", "pt-cv", "pt-gw", "pt-st", "pt-tl"];
const britishEnglish = ["en-gb", "en-ie", "en-au", "en-nz", "en-za", "en-in"];

export function matchLocale(tag: string): Locale | undefined {
  const lower = tag.trim().toLowerCase();
  const exact = locales.find((l) => l.toLowerCase() === lower);
  if (exact) return exact;
  const lang = lower.split("-")[0];
  if (lang === "pt") return europeanPortuguese.includes(lower) ? "pt-PT" : "pt-BR";
  if (lang === "en") return britishEnglish.includes(lower) ? "en-GB" : "en-US";
  if (lang === "es") return "es-ES";
  if (lang === "fr") return "fr-FR";
  return undefined;
}

export function detectLocale(preferred: readonly string[] = navigator.languages): Locale {
  for (const tag of preferred) {
    const match = matchLocale(tag);
    if (match) return match;
  }
  return "en-US";
}

export function readSavedLocale(): Locale | undefined {
  try {
    const saved = window.localStorage.getItem(storageKey);
    return locales.find((l) => l === saved);
  } catch {
    return undefined;
  }
}

export function saveLocale(locale: Locale | undefined) {
  try {
    if (locale) window.localStorage.setItem(storageKey, locale);
    else window.localStorage.removeItem(storageKey);
  } catch {}
  document.cookie = locale
    ? `${cookieName}=${locale}; Path=/; Max-Age=31536000; SameSite=Lax`
    : `${cookieName}=; Path=/; Max-Age=0; SameSite=Lax`;
}

export function toLocale(value: string): Locale | undefined {
  return locales.find((l) => l === value);
}
