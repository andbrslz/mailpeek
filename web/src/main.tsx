import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { InboxProvider } from "./context/InboxProvider";
import { LayoutProvider } from "./context/LayoutProvider";
import { I18nProvider } from "./i18n/I18nProvider";
import "./index.css";
import { applyTheme, savedTheme, systemTheme } from "./lib/theme";
import { InboxPage } from "./pages/InboxPage";

applyTheme(savedTheme() ?? systemTheme());

const root = document.getElementById("root");
if (!root) throw new Error("#root element missing");

createRoot(root).render(
  <StrictMode>
    <I18nProvider>
      <InboxProvider>
        <LayoutProvider>
          <InboxPage />
        </LayoutProvider>
      </InboxProvider>
    </I18nProvider>
  </StrictMode>,
);
