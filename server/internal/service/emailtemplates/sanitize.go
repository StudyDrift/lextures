package emailtemplates

import (
	"github.com/microcosm-cc/bluemonday"
)

var emailSanitizePolicy = func() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("src", "alt", "width", "style").OnElements("img")
	p.AllowElements("strong", "em", "b", "i", "u", "p", "br", "ul", "ol", "li", "h1", "h2", "h3", "h4", "a", "img", "span", "div")
	return p
}()

// SanitizeHTML strips unsafe markup from stored email template HTML.
func SanitizeHTML(html string) string {
	return emailSanitizePolicy.Sanitize(html)
}
