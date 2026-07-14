package emailtemplates

import (
	"github.com/microcosm-cc/bluemonday"
)

var emailSanitizePolicy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("src", "alt", "width", "style").OnElements("img")
	// Email-safe layout tables (COPPA notice and similar).
	p.AllowElements("table", "thead", "tbody", "tr", "th", "td")
	p.AllowAttrs("style", "colspan", "rowspan", "align", "valign", "width", "border", "cellpadding", "cellspacing").OnElements("table", "tr", "th", "td", "thead", "tbody")
	p.AllowElements("strong", "em", "b", "i", "u", "p", "br", "ul", "ol", "li", "h1", "h2", "h3", "h4", "a", "img", "span", "div")
	p.AllowAttrs("style").OnElements("p", "div", "span", "h1", "h2", "h3", "h4", "a", "ul", "ol", "li", "strong", "em")
	return p
}()

// SanitizeHTML strips unsafe markup from stored email template HTML.
func SanitizeHTML(html string) string {
	return emailSanitizePolicy.Sanitize(html)
}
