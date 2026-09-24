package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type loginText struct {
	Lang, Title, Eyebrow, Heading, Lead1, Lead2, Failed    string
	Username, UsernameHint, Password, PasswordHint, Button string
	HelpSummary, Help, Tagline                             string
}

var loginTexts = map[string]loginText{
	"en-US": {
		Title: "Sign in", Eyebrow: "Private inbox", Heading: "Sign in",
		Lead1: "Your test emails, all in one place.", Lead2: "Sign in to open your inbox.",
		Failed:   "Wrong username or password. Check your details and try again.",
		Username: "Username", UsernameHint: "Enter your username",
		Password: "Password", PasswordHint: "Enter your password", Button: "Sign in",
		HelpSummary: "Need help signing in?",
		Help:        "Use the credentials configured for this Mailpeek instance. If someone else set it up, ask them for access.",
		Tagline:     "Email testing for developers",
	},
	"pt-BR": {
		Title: "Entrar", Eyebrow: "Caixa de entrada privada", Heading: "Entrar",
		Lead1: "Seus e-mails de teste, todos num só lugar.", Lead2: "Entre para abrir sua caixa de entrada.",
		Failed:   "Usuário ou senha incorretos. Confira os dados e tente novamente.",
		Username: "Usuário", UsernameHint: "Digite seu usuário",
		Password: "Senha", PasswordHint: "Digite sua senha", Button: "Entrar",
		HelpSummary: "Precisa de ajuda para entrar?",
		Help:        "Use as credenciais configuradas para esta instância do Mailpeek. Se foi outra pessoa que a configurou, peça acesso a ela.",
		Tagline:     "Testes de e-mail para desenvolvedores",
	},
	"pt-PT": {
		Title: "Iniciar sessão", Eyebrow: "Caixa de entrada privada", Heading: "Iniciar sessão",
		Lead1: "Os seus e-mails de teste, todos num só lugar.", Lead2: "Inicie sessão para abrir a sua caixa de entrada.",
		Failed:   "Utilizador ou palavra-passe incorretos. Verifique os dados e tente novamente.",
		Username: "Utilizador", UsernameHint: "Introduza o seu utilizador",
		Password: "Palavra-passe", PasswordHint: "Introduza a sua palavra-passe", Button: "Iniciar sessão",
		HelpSummary: "Precisa de ajuda para iniciar sessão?",
		Help:        "Use as credenciais configuradas para esta instância do Mailpeek. Se foi outra pessoa a configurá-la, peça-lhe acesso.",
		Tagline:     "Testes de e-mail para programadores",
	},
	"es-ES": {
		Title: "Iniciar sesión", Eyebrow: "Bandeja privada", Heading: "Iniciar sesión",
		Lead1: "Tus correos de prueba, todos en un solo lugar.", Lead2: "Inicia sesión para abrir tu bandeja de entrada.",
		Failed:   "Usuario o contraseña incorrectos. Revisa los datos e inténtalo de nuevo.",
		Username: "Usuario", UsernameHint: "Introduce tu usuario",
		Password: "Contraseña", PasswordHint: "Introduce tu contraseña", Button: "Iniciar sesión",
		HelpSummary: "¿Necesitas ayuda para iniciar sesión?",
		Help:        "Usa las credenciales configuradas para esta instancia de Mailpeek. Si la configuró otra persona, pídele acceso.",
		Tagline:     "Pruebas de correo para desarrolladores",
	},
	"fr-FR": {
		Title: "Connexion", Eyebrow: "Boîte de réception privée", Heading: "Connexion",
		Lead1: "Tous vos e-mails de test au même endroit.", Lead2: "Connectez-vous pour ouvrir votre boîte de réception.",
		Failed:   "Identifiant ou mot de passe incorrect. Vérifiez vos informations et réessayez.",
		Username: "Identifiant", UsernameHint: "Saisissez votre identifiant",
		Password: "Mot de passe", PasswordHint: "Saisissez votre mot de passe", Button: "Se connecter",
		HelpSummary: "Besoin d’aide pour vous connecter ?",
		Help:        "Utilisez les identifiants configurés pour cette instance de Mailpeek. Si quelqu’un d’autre l’a configurée, demandez-lui l’accès.",
		Tagline:     "Tests d’e-mails pour les développeurs",
	},
}

func init() {
	gb := loginTexts["en-US"]
	loginTexts["en-GB"] = gb
	for code, text := range loginTexts {
		text.Lang = code
		loginTexts[code] = text
	}
}

var (
	europeanPortuguese = map[string]bool{"pt-pt": true, "pt-ao": true, "pt-mz": true, "pt-cv": true, "pt-gw": true, "pt-st": true, "pt-tl": true}
	britishEnglish     = map[string]bool{"en-gb": true, "en-ie": true, "en-au": true, "en-nz": true, "en-za": true, "en-in": true}
)

func matchLocale(tag string) (string, bool) {
	lower := strings.ToLower(strings.TrimSpace(tag))
	for code := range loginTexts {
		if strings.ToLower(code) == lower {
			return code, true
		}
	}
	switch lang, _, _ := strings.Cut(lower, "-"); lang {
	case "pt":
		if europeanPortuguese[lower] {
			return "pt-PT", true
		}
		return "pt-BR", true
	case "en":
		if britishEnglish[lower] {
			return "en-GB", true
		}
		return "en-US", true
	case "es":
		return "es-ES", true
	case "fr":
		return "fr-FR", true
	}
	return "", false
}

func loginLocale(r *http.Request) loginText {
	if c, err := r.Cookie("mailpeek_lang"); err == nil {
		if code, ok := matchLocale(c.Value); ok {
			return loginTexts[code]
		}
	}
	for _, tag := range acceptLanguages(r.Header.Get("Accept-Language")) {
		if code, ok := matchLocale(tag); ok {
			return loginTexts[code]
		}
	}
	return loginTexts["en-US"]
}

func acceptLanguages(header string) []string {
	type tagQ struct {
		tag string
		q   float64
	}
	var tags []tagQ
	for _, part := range strings.Split(header, ",") {
		tag, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if tag == "" || tag == "*" {
			continue
		}
		q := 1.0
		if v, ok := strings.CutPrefix(strings.TrimSpace(params), "q="); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				q = f
			}
		}
		tags = append(tags, tagQ{tag, q})
	}
	sort.SliceStable(tags, func(i, j int) bool { return tags[i].q > tags[j].q })
	out := make([]string, len(tags))
	for i, t := range tags {
		out[i] = t.tag
	}
	return out
}
