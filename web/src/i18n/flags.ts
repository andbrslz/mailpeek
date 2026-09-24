import br from "circle-flags/flags/br.svg";
import es from "circle-flags/flags/es.svg";
import fr from "circle-flags/flags/fr.svg";
import gb from "circle-flags/flags/gb.svg";
import pt from "circle-flags/flags/pt.svg";
import us from "circle-flags/flags/us.svg";
import type { Locale } from "./messages";

export const localeFlags: Record<Locale, string> = {
  "en-US": us,
  "en-GB": gb,
  "pt-BR": br,
  "pt-PT": pt,
  "es-ES": es,
  "fr-FR": fr,
};
