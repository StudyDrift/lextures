package emailtemplates

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// mdCompiler renders GFM Markdown to HTML. Raw HTML in source is not enabled
// (html.WithUnsafe is intentionally omitted); bluemonday re-sanitizes after.
var mdCompiler = goldmark.New(
	goldmark.WithExtensions(extension.GFM, extension.Table, extension.Strikethrough, extension.Linkify),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
)

// Compile renders Markdown to email-safe HTML: goldmark (GFM + hard breaks +
// autolinks) then bluemonday via SanitizeHTML. Merge tokens like {{link}} are
// preserved for post-compile substitution (including inside link hrefs —
// goldmark would otherwise percent-encode the braces).
func Compile(markdown string) (string, error) {
	md := strings.TrimSpace(markdown)
	if md == "" {
		return "", fmt.Errorf("emailtemplates: empty markdown")
	}

	// Protect {{tokens}} so goldmark does not URL-encode them in hrefs.
	protected, restore := protectMergeTokens(md)

	var buf bytes.Buffer
	if err := mdCompiler.Convert([]byte(protected), &buf); err != nil {
		RecordCompile(false)
		return "", fmt.Errorf("emailtemplates: compile markdown: %w", err)
	}
	out := restore(SanitizeHTML(buf.String()))
	if strings.TrimSpace(out) == "" {
		RecordCompile(false)
		return "", fmt.Errorf("emailtemplates: compile produced empty HTML")
	}
	RecordCompile(true)
	return out, nil
}

// protectMergeTokens replaces {{field}} with stable alphanumeric placeholders
// that survive Markdown link parsing and HTML attribute encoding.
func protectMergeTokens(src string) (string, func(string) string) {
	// Placeholder format: zzmtNzz (unlikely in prose; no special URL chars).
	placeholders := make(map[string]string)
	var i int
	protected := mergeTokenRe.ReplaceAllStringFunc(src, func(token string) string {
		key := fmt.Sprintf("zzmt%dzz", i)
		i++
		placeholders[key] = token
		return key
	})
	restore := func(html string) string {
		out := html
		for key, token := range placeholders {
			out = strings.ReplaceAll(out, key, token)
			// goldmark/bluemonday may leave percent-encoded placeholders.
			out = strings.ReplaceAll(out, url.QueryEscape(key), token)
			out = strings.ReplaceAll(out, url.PathEscape(key), token)
		}
		return out
	}
	return protected, restore
}
