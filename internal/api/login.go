package api

import (
	"html/template"
	"net/http"
)

type loginView struct {
	Next     string
	Username string
	Failed   bool
	T        loginText
}

func renderLogin(w http.ResponseWriter, r *http.Request, status int, v loginView) {
	v.T = loginLocale(r)
	h := w.Header()
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Security-Policy",
		"default-src 'none'; style-src 'unsafe-inline'; img-src 'self'; form-action 'self'; frame-ancestors 'none'")
	h.Set("X-Frame-Options", "DENY")
	w.WriteHeader(status)
	_ = loginTemplate.Execute(w, v)
}

var loginTemplate = template.Must(template.New("login").Parse(`<!doctype html>
<html lang="{{.T.Lang}}">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light dark">
<link rel="icon" type="image/svg+xml" href="/favicon.svg">
<title>{{.T.Title}} · Mailpeek</title>
<style>
  :root { --bg:#f8fafc; --card:#fff; --fg:#18181b; --muted:#64748b; --border:#e2e8f0;
          --input:#fff; --accent:#2563eb; --ring:rgb(37 99 235 / .14); --error-bg:#fef2f2; --error:#b91c1c;
          --glow:rgb(59 130 246 / .08); --shadow:rgb(15 23 42 / .06); }
  @media (prefers-color-scheme: dark) {
    :root { --bg:#09090b; --card:#141416; --fg:#fafafa; --muted:#a1a1aa; --border:#2b2b30;
            --input:#18181b; --accent:#3b82f6; --ring:rgb(59 130 246 / .22); --error-bg:#350f16; --error:#fca5a5;
            --glow:rgb(59 130 246 / .09); --shadow:rgb(0 0 0 / .2); }
  }
  * { box-sizing:border-box; }
  body { margin:0; min-height:100vh; min-height:100svh; display:grid; place-items:center; padding:40px 20px;
         background:radial-gradient(ellipse at 50% 35%, var(--glow), transparent 60%), var(--bg); color:var(--fg);
         font:14px/1.6 Inter, system-ui, -apple-system, "Segoe UI", sans-serif; -webkit-font-smoothing:antialiased; }
  .layout { width:100%; max-width:420px; }
  .brand { display:flex; justify-content:center; align-items:center; gap:11px; margin-bottom:30px;
           font-weight:650; font-size:21px; letter-spacing:-.6px; }
  .brand svg { border-radius:10px; box-shadow:0 2px 5px var(--shadow); }
  main { background:var(--card); border:1px solid var(--border); border-radius:20px; padding:36px;
         box-shadow:0 2px 4px var(--shadow), 0 20px 60px -20px var(--shadow); }
  .intro { margin-bottom:28px; }
  .eyebrow { display:flex; align-items:center; gap:6px; color:var(--muted); font-size:11px;
             font-weight:600; letter-spacing:.08em; text-transform:uppercase; margin:0 0 14px; }
  h1 { margin:0 0 8px; font-size:28px; font-weight:650; letter-spacing:-1px; line-height:1.25; }
  .lead { margin:0; color:var(--muted); line-height:1.7; }
  .field { margin-bottom:20px; }
  label { display:block; font-size:13px; font-weight:600; margin-bottom:8px; }
  input { width:100%; min-height:46px; padding:11px 13px; border:1px solid var(--border); border-radius:9px;
          background:var(--input); color:var(--fg); font:inherit; outline:none; transition:border-color .15s, box-shadow .15s; }
  input::placeholder { color:var(--muted); opacity:.8; }
  input:focus { border-color:var(--accent); box-shadow:0 0 0 3px var(--ring); }
  button { display:flex; align-items:center; justify-content:center; gap:9px; width:100%; min-height:46px;
           margin-top:6px; padding:11px 16px; border:1px solid transparent; border-radius:9px; background:var(--accent);
           color:#fff; font:inherit; font-weight:600; cursor:pointer; box-shadow:0 2px 3px var(--shadow); }
  button:hover { filter:brightness(1.08); }
  button:active { transform:translateY(1px); }
  button:focus-visible, summary:focus-visible { outline:2px solid var(--accent); outline-offset:4px; }
  .error { display:flex; gap:9px; align-items:flex-start; margin:0 0 22px; padding:12px; border-radius:9px;
           background:var(--error-bg); color:var(--error); font-size:13px; }
  .error svg { flex:none; margin-top:2px; }
  details { margin-top:28px; padding-top:20px; border-top:1px solid var(--border); color:var(--muted); font-size:13px; }
  summary { width:fit-content; cursor:pointer; border-radius:3px; }
  summary:hover { color:var(--fg); }
  details p { margin:12px 0 0; line-height:1.7; }
  .footer { margin:24px 0 0; text-align:center; color:var(--muted); font-size:12px; }
  @media (max-width:440px) {
    body { padding:28px 16px; }
    main { padding:28px 24px; border-radius:16px; }
    input { font-size:16px; }
    .brand { margin-bottom:24px; }
  }
  @media (prefers-reduced-motion:reduce) { input { transition:none; } }
</style>
</head>
<body>
<div class="layout">
  <div class="brand">
    <svg width="36" height="36" viewBox="0 0 32 32" aria-hidden="true"><rect width="32" height="32" rx="7" fill="#18181b"/><path d="M8 11.5A1.5 1.5 0 0 1 9.5 10h13a1.5 1.5 0 0 1 1.5 1.5v9a1.5 1.5 0 0 1-1.5 1.5h-13A1.5 1.5 0 0 1 8 20.5z" fill="none" stroke="#fff" stroke-width="1.8"/><path d="m8.6 11 7.4 5.5 7.4-5.5" fill="none" stroke="#3b82f6" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"/></svg>
    Mailpeek
  </div>
  <main aria-labelledby="login-title">
  <header class="intro">
    <p class="eyebrow"><svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><rect x="5" y="10" width="14" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></svg>{{.T.Eyebrow}}</p>
    <h1 id="login-title">{{.T.Heading}}</h1>
    <p class="lead">{{.T.Lead1}}<br>{{.T.Lead2}}</p>
  </header>
  {{if .Failed}}
  <p class="error" id="login-error" role="alert">
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="10"/><path d="M12 8v4M12 16h.01"/></svg>
    <span>{{.T.Failed}}</span>
  </p>
  {{end}}
  <form method="post" action="/login">
    <input type="hidden" name="next" value="{{.Next}}">
    <div class="field">
      <label for="username">{{.T.Username}}</label>
      <input id="username" name="username" autocomplete="username" autocapitalize="none" spellcheck="false" placeholder="{{.T.UsernameHint}}" required value="{{.Username}}"{{if .Failed}} aria-invalid="true" aria-describedby="login-error"{{end}}{{if not .Username}} autofocus{{end}}>
    </div>
    <div class="field">
      <label for="password">{{.T.Password}}</label>
      <input id="password" name="password" type="password" autocomplete="current-password" placeholder="{{.T.PasswordHint}}" required{{if .Failed}} aria-invalid="true" aria-describedby="login-error"{{end}}{{if .Username}} autofocus{{end}}>
    </div>
    <button type="submit">{{.T.Button}} <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h14m-6-6 6 6-6 6"/></svg></button>
  </form>
  <details>
    <summary>{{.T.HelpSummary}}</summary>
    <p>{{.T.Help}}</p>
  </details>
  </main>
  <p class="footer">Mailpeek &middot; {{.T.Tagline}}</p>
</div>
</body>
</html>
`))
