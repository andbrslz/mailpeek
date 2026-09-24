import { Moon, Sun } from "lucide-react";
import { useTheme } from "../../hooks/useTheme";
import { useI18n } from "../../i18n/context";
import { IconButton } from "../ui";

export function ThemeToggle() {
  const { t } = useI18n();
  const { theme, toggle } = useTheme();
  const dark = theme === "dark";
  return (
    <IconButton
      label={t("header.theme")}
      title={dark ? t("header.themeToLight") : t("header.themeToDark")}
      aria-pressed={dark}
      onClick={toggle}
      className="shrink-0"
    >
      {dark ? <Moon className="size-4" /> : <Sun className="size-4" />}
    </IconButton>
  );
}
