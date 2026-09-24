# Mailpeek Web UI

React + Vite + TypeScript + Tailwind CSS. It is compiled into `web/dist` and embedded in the Go binary.

```bash
npm run dev -w @mailpeek/web   # Vite on :5173, /api proxied to MAILPEEK_URL (default :8026)
npm run build -w @mailpeek/web # what the binary embeds
```

## Structure

```text
src/
  main.tsx            providers + the page
  pages/              composition only
  components/         presentational components (panels/, message/, header/)
  context/            InboxContext (data, live updates, actions), LayoutContext (UI layout)
  hooks/              all behaviour: state, effects, subscriptions, browser APIs
  i18n/               locale detection, provider, dictionaries (locales/*.ts)
  lib/                pure helpers (formatting, MIME highlighting, SSE connection, …)
  api.ts              REST calls and types
```

## Rules

- **Components and pages are presentational.** They render props and call handlers. State, effects, timers, subscriptions and browser APIs live in hooks (`src/hooks`) or contexts (`src/context`). A component may read a context or a small UI hook (for example `useCopyToClipboard`), but the logic itself is in the hook.
- **Pages only compose.** Decisions such as "what does the list show" are computed in the state hooks (`useInboxState`, `useLayoutState`) and passed down.
- **At most 300 lines per file** in `src/` (`npm run ui:size`). A longer file is a sign it needs splitting.
- **React Doctor must report no issues** (`npm run doctor`, score 100). Fix the cause; don't disable rules.
- **No user-visible text in components.** Use `const { t } = useI18n()` and a key from `i18n/locales/en-US.ts`. Error messages too: hooks keep a `Problem` (a key, see `lib/errors.ts`) and the component translates it. `npm run ui:i18n` fails on literal JSX text or `title`/`aria-label`/`placeholder`/`alt` strings.
- **Accessibility:** every control has an accessible name; the E2E suite selects by role and name.

## Translations

Supported: `en-US`, `en-GB`, `pt-BR`, `pt-PT`, `es-ES`, `fr-FR`.

1. Add the key and English text to `src/i18n/locales/en-US.ts`.
2. Add it to `pt-BR.ts`, `es-ES.ts` and `fr-FR.ts`. TypeScript fails until every locale has it. `pt-PT` inherits from `pt-BR`, so only override real differences (e.g. "ecrã", "transferir", "eliminar"); `en-GB` reuses the English text with British dates.
3. Add its round flag to `src/i18n/flags.ts` (from [circle-flags](https://github.com/HatScripts/circle-flags), MIT) and its own name to `localeNames` in `src/i18n/messages.ts`.
4. The sign-in screen is rendered by the Go server: its strings are in `internal/api/login_i18n.go`.

The language is detected from the browser (`pt` → pt-BR, `pt-PT`/`pt-AO`… → pt-PT, `en-AU`/`en-IE`… → en-GB). A choice made in the header is saved in `localStorage` (`mailpeek:locale`) and in the `mailpeek_lang` cookie, so the server-rendered sign-in screen uses it too.
