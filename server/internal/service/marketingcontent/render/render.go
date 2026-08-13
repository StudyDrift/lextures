// Package render implements the safe marketing-content Markdown dialect.
package render

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"
	"unicode"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldhtml "github.com/yuin/goldmark/renderer/html"
)

const MaxInputBytes = 1 << 20

type StatsResult struct {
	WordCount      int   `json:"wordCount"`
	PassageLengths []int `json:"passageLengths"`
	HeadingCount   int   `json:"headingCount"`
	FAQCount       int   `json:"faqCount"`
}

var (
	directiveRE = regexp.MustCompile(`(?ms)^:::[ \t]*(key-takeaways|answer|definition|comparison-table|steps|faq|callout|stat|sources)(?:[ \t]+([^\n]*))?\r?\n(.*?)^:::[ \t]*$`)
	headingRE   = regexp.MustCompile(`(?m)^#{2,4}\s+(.+)$`)
	faqRE       = regexp.MustCompile(`(?m)^###\s+.+\?\s*$`)
	markupRE    = regexp.MustCompile(`<[^>]*>|[#>*_` + "`" + `\[\](){}|:-]`)
	paragraphRE = regexp.MustCompile(`\r?\n\s*\r?\n`)
)

func markdown() goldmark.Markdown {
	return goldmark.New(goldmark.WithExtensions(extension.GFM, extension.Linkify, extension.Typographer), goldmark.WithRendererOptions(goldhtml.WithUnsafe()))
}

func renderFragment(source string) string {
	var out bytes.Buffer
	_ = markdown().Convert([]byte(source), &out)
	return out.String()
}

func renderDirectives(source string) (string, map[string]string) {
	replacements := map[string]string{}
	sequence := 0
	converted := directiveRE.ReplaceAllStringFunc(source, func(block string) string {
		m := directiveRE.FindStringSubmatch(block)
		name, args, content := strings.ToLower(m[1]), strings.TrimSpace(m[2]), strings.TrimSpace(m[3])
		token := fmt.Sprintf("CONTENTDIRECTIVE%dTOKEN", sequence)
		id := sequence
		sequence++
		body := renderFragment(content)
		var value string
		switch name {
		case "key-takeaways":
			value = fmt.Sprintf(`<aside class="content-card content-takeaways" aria-labelledby="key-takeaways-%d"><h2 id="key-takeaways-%d">Key takeaways</h2>%s</aside>`, id, id, body)
		case "answer":
			value = `<section class="content-card content-answer" aria-label="Direct answer">` + body + `</section>`
		case "definition":
			term := directiveArg(args, "term")
			value = `<dfn class="content-definition" data-definition-term="` + html.EscapeString(term) + `"><strong>` + html.EscapeString(term) + `</strong>` + body + `</dfn>`
		case "comparison-table":
			summary := directiveArg(args, "summary")
			caption := ""
			if summary != "" {
				caption = `<p class="content-table-summary">` + html.EscapeString(summary) + `</p>`
			}
			value = `<section class="content-comparison">` + caption + `<div class="content-table-scroll">` + body + `</div></section>`
		case "steps":
			value = `<section class="content-steps" data-how-to>` + body + `</section>`
		case "faq":
			value = renderFAQ(content, id)
		case "callout":
			kind := args
			if kind != "warning" && kind != "tip" {
				kind = "note"
			}
			label := map[string]string{"note": "Note", "warning": "Warning", "tip": "Tip"}[kind]
			value = `<aside class="content-callout content-callout-` + kind + `"><strong>` + label + `</strong>` + body + `</aside>`
		case "stat":
			inline := strings.TrimSuffix(renderFragment(content), "\n")
			inline = strings.TrimSuffix(strings.TrimPrefix(inline, "<p>"), "</p>")
			value = `<figure class="content-stat"><blockquote>` + inline + `</blockquote>` + optionalCaption(args) + `</figure>`
		case "sources":
			if content != "" {
				value = `<section class="content-sources"><h2>Sources</h2>` + body + `</section>`
			}
		}
		replacements[token] = value
		return token
	})
	return converted, replacements
}

func directiveArg(value, key string) string {
	v := strings.TrimSpace(value)
	v = strings.TrimPrefix(v, key)
	v = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v), "="))
	return strings.Trim(v, `"'`)
}
func optionalCaption(v string) string {
	if strings.TrimSpace(v) == "" {
		return ""
	}
	return `<figcaption>` + html.EscapeString(strings.TrimSpace(v)) + `</figcaption>`
}

func renderFAQ(content string, sequence int) string {
	var b strings.Builder
	b.WriteString(`<section class="content-faq" data-faq><h2>Frequently asked questions</h2>`)
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	type entry struct{ question, answer string }
	entries := []entry{}
	var current *entry
	for _, line := range lines {
		if strings.HasPrefix(line, "### ") && strings.HasSuffix(strings.TrimSpace(line), "?") {
			entries = append(entries, entry{question: strings.TrimSpace(strings.TrimPrefix(line, "### "))})
			current = &entries[len(entries)-1]
			continue
		}
		if current != nil {
			current.answer += line + "\n"
		}
	}
	for i, item := range entries {
		id := fmt.Sprintf("faq-q-%d-%d", sequence, i)
		b.WriteString(`<details class="content-faq-item" open><summary id="` + id + `">` + html.EscapeString(item.question) + `</summary><div aria-labelledby="` + id + `">` + renderFragment(strings.TrimSpace(item.answer)) + `</div></details>`)
	}
	b.WriteString(`</section>`)
	return b.String()
}

func policy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("p", "br", "hr", "em", "strong", "del", "blockquote", "pre", "code", "ul", "ol", "li", "table", "thead", "tbody", "tr", "th", "td", "h1", "h2", "h3", "h4", "h5", "h6", "a", "img", "picture", "source", "aside", "section", "div", "dfn", "figure", "figcaption", "details", "summary", "sup", "span")
	p.AllowAttrs("class", "id", "tabindex", "aria-label", "aria-labelledby", "aria-hidden", "open", "data-faq", "data-how-to", "data-definition-term").Globally()
	p.AllowAttrs("href", "target", "rel").OnElements("a")
	p.AllowAttrs("src", "srcset", "type", "alt", "width", "height", "loading", "decoding").OnElements("img", "source")
	p.AllowURLSchemes("http", "https", "mailto")
	p.RequireNoFollowOnLinks(false)
	return p
}

func HTML(source string) (string, error) {
	if len(source) > MaxInputBytes {
		return "", errors.New("marketing content exceeds 1 MB")
	}
	prepared, replacements := renderDirectives(source)
	output := renderFragment(prepared)
	for token, replacement := range replacements {
		output = strings.ReplaceAll(output, "<p>"+token+"</p>\n", replacement)
		output = strings.ReplaceAll(output, token, replacement)
	}
	output = regexp.MustCompile(`\[\^(\d+)\]`).ReplaceAllString(output, `<sup class="content-citation"><a href="#source-$1" aria-label="Source $1">$1</a></sup>`)
	output = addHeadingIDs(output)
	output = regexp.MustCompile(`<a href="(https?://[^"]+)"`).ReplaceAllString(output, `<a href="$1" target="_blank" rel="noopener noreferrer"`)
	return policy().Sanitize(output), nil
}

func addHeadingIDs(value string) string {
	re := regexp.MustCompile(`(?is)<h([2-4])>(.*?)</h[2-4]>`)
	return re.ReplaceAllStringFunc(value, func(v string) string {
		m := re.FindStringSubmatch(v)
		text := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(m[2], "")
		return `<h` + m[1] + ` id="` + slugify(text) + `" tabindex="-1">` + m[2] + `</h` + m[1] + `>`
	})
}
func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
			dash = false
		} else if unicode.IsSpace(r) || r == '-' {
			if b.Len() > 0 && !dash {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	return strings.TrimSuffix(b.String(), "-")
}

func PlainText(source string) string {
	rendered, err := HTML(source)
	if err != nil {
		return ""
	}
	text := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(rendered, " ")
	return strings.Join(strings.Fields(html.UnescapeString(text)), " ")
}
func Stats(source string) StatsResult {
	plain := func(s string) int { return len(strings.Fields(markupRE.ReplaceAllString(s, " "))) }
	// Match www/scripts/content-lint/core.mjs: strip directive blocks and headings before
	// measuring self-contained passages so answer/takeaway/faq blocks do not skew the score.
	passageSource := directiveRE.ReplaceAllString(source, "")
	passageSource = regexp.MustCompile(`(?m)^#{1,6}.+$`).ReplaceAllString(passageSource, "")
	lengths := []int{}
	for _, p := range paragraphRE.Split(passageSource, -1) {
		if n := plain(p); n >= 20 {
			lengths = append(lengths, n)
		}
	}
	return StatsResult{WordCount: plain(source), PassageLengths: lengths, HeadingCount: len(headingRE.FindAllString(source, -1)), FAQCount: len(faqRE.FindAllString(source, -1))}
}
