package context

import (
	"bytes"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

var (
	scriptRe = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRe  = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	noscriptRe = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`)
	tagRe    = regexp.MustCompile(`(?s)<[^>]+>`)
	wsRe     = regexp.MustCompile(`\s+`)
)

// ExtractResult is main-content text from a fetched body (FR-6, FR-8).
type ExtractResult struct {
	Title   string
	Text    string
	Lang    string
	Quality string // high | medium | low
}

// ExtractMainContent reduces HTML/plain/PDF-bytes to text.
// PDF support is best-effort stream text extraction (no external deps).
func ExtractMainContent(contentType string, body []byte) (ExtractResult, error) {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch {
	case strings.HasPrefix(ct, "text/html"), strings.HasPrefix(ct, "application/xhtml"):
		return extractHTML(body), nil
	case strings.HasPrefix(ct, "text/plain"), strings.HasPrefix(ct, "text/markdown"), ct == "application/json":
		text := strings.TrimSpace(string(body))
		q := "high"
		if len(text) < 40 {
			q = "low"
		}
		return ExtractResult{Text: text, Quality: q}, nil
	case strings.HasPrefix(ct, "application/pdf"), bytes.HasPrefix(body, []byte("%PDF")):
		text := extractPDFText(body)
		if strings.TrimSpace(text) == "" {
			return ExtractResult{}, ErrUnsupportedType
		}
		return ExtractResult{Text: text, Title: "PDF", Quality: "medium"}, nil
	default:
		// Sniff HTML without content-type.
		trimmed := bytes.TrimSpace(body)
		if bytes.HasPrefix(trimmed, []byte("<!DOCTYPE")) || bytes.HasPrefix(trimmed, []byte("<html")) ||
			bytes.Contains(trimmed[:min(512, len(trimmed))], []byte("<html")) {
			return extractHTML(body), nil
		}
		return ExtractResult{}, ErrUnsupportedType
	}
}

func extractHTML(body []byte) ExtractResult {
	lang := ""
	title := ""
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		// Fallback strip tags.
		text := stripTags(string(body))
		return ExtractResult{Text: text, Quality: qualityOf(text)}
	}
	var mainBuf strings.Builder
	var bodyBuf strings.Builder
	var inMain, inBody, inTitle bool
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch strings.ToLower(n.Data) {
			case "html":
				for _, a := range n.Attr {
					if strings.EqualFold(a.Key, "lang") {
						lang = strings.TrimSpace(a.Val)
					}
				}
			case "script", "style", "noscript", "svg", "iframe":
				return
			case "title":
				inTitle = true
			case "main", "article":
				inMain = true
				defer func() { inMain = false }()
			case "body":
				inBody = true
			case "nav", "footer", "header", "aside":
				if !inMain {
					return
				}
			}
		}
		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t == "" {
				return
			}
			if inTitle && title == "" {
				title = t
			}
			if inMain {
				mainBuf.WriteString(t)
				mainBuf.WriteByte(' ')
			} else if inBody {
				bodyBuf.WriteString(t)
				bodyBuf.WriteByte(' ')
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode && strings.EqualFold(n.Data, "title") {
			inTitle = false
		}
	}
	walk(doc)
	text := strings.TrimSpace(mainBuf.String())
	if text == "" {
		text = strings.TrimSpace(bodyBuf.String())
	}
	if text == "" {
		text = stripTags(string(body))
	}
	text = wsRe.ReplaceAllString(text, " ")
	return ExtractResult{Title: title, Text: text, Lang: lang, Quality: qualityOf(text)}
}

func stripTags(s string) string {
	s = scriptRe.ReplaceAllString(s, " ")
	s = styleRe.ReplaceAllString(s, " ")
	s = noscriptRe.ReplaceAllString(s, " ")
	s = tagRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(wsRe.ReplaceAllString(s, " "))
}

func qualityOf(text string) string {
	n := len([]rune(text))
	switch {
	case n >= 400:
		return "high"
	case n >= 80:
		return "medium"
	default:
		return "low"
	}
}

// extractPDFText pulls printable Latin strings from a PDF byte stream (best-effort).
func extractPDFText(body []byte) string {
	var b strings.Builder
	run := 0
	flush := func() {
		if run >= 4 {
			// already written
		}
		run = 0
	}
	_ = flush
	for i := 0; i < len(body); i++ {
		c := body[i]
		if c >= 32 && c < 127 && (unicode.IsLetter(rune(c)) || unicode.IsDigit(rune(c)) || unicode.IsSpace(rune(c)) || strings.ContainsRune(".,;:!?()-_'\"/", rune(c))) {
			b.WriteByte(c)
			run++
		} else {
			if run > 0 {
				b.WriteByte(' ')
			}
			run = 0
		}
	}
	return wsRe.ReplaceAllString(strings.TrimSpace(b.String()), " ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
