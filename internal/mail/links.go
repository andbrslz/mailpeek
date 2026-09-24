package mail

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var urlPattern = regexp.MustCompile(`https?://[^\s<>"'` + "`" + `]+`)

func ExtractLinks(htmlBody, textBody string) []Link {
	if strings.TrimSpace(htmlBody) != "" {
		return htmlLinks(htmlBody)
	}
	return textLinks(textBody)
}

func htmlLinks(body string) []Link {
	links := []Link{}
	seen := map[Link]bool{}
	z := html.NewTokenizer(strings.NewReader(body))

	var (
		inLink   bool
		href     string
		text     strings.Builder
		fallback string
	)
	flush := func() {
		if !inLink {
			return
		}
		inLink = false
		l := Link{Text: collapseSpace(text.String()), Href: strings.TrimSpace(href)}
		if l.Text == "" {
			l.Text = collapseSpace(fallback)
		}
		if l.Href != "" && !seen[l] {
			seen[l] = true
			links = append(links, l)
		}
	}

	for {
		switch z.Next() {
		case html.ErrorToken:
			flush()
			return links
		case html.StartTagToken, html.SelfClosingTagToken:
			name, hasAttr := z.TagName()
			attrs := map[string]string{}
			for hasAttr {
				var k, v []byte
				k, v, hasAttr = z.TagAttr()
				attrs[string(k)] = string(v)
			}
			switch string(name) {
			case "a":
				flush()
				inLink, href = true, attrs["href"]
				text.Reset()
				fallback = firstNonEmpty(attrs["aria-label"], attrs["title"])
			case "img":
				if inLink && fallback == "" {
					fallback = attrs["alt"]
				}
			case "br":
				text.WriteByte(' ')
			}
		case html.EndTagToken:
			if name, _ := z.TagName(); string(name) == "a" {
				flush()
			}
		case html.TextToken:
			if inLink {
				text.Write(z.Text())
			}
		}
	}
}

func textLinks(body string) []Link {
	links := []Link{}
	seen := map[string]bool{}
	for _, u := range urlPattern.FindAllString(body, -1) {
		u = strings.TrimRight(u, ".,;:!?)]}>")
		if !seen[u] {
			seen[u] = true
			links = append(links, Link{Href: u})
		}
	}
	return links
}

func collapseSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
