package context

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

var (
	mdLinkRe   = regexp.MustCompile(`\[[^\]]*\]\((https?://[^)\s]+)\)`)
	bareURLRe  = regexp.MustCompile(`(?i)\bhttps?://[^\s<>"'` + "`" + `]+`)
	angleURLRe = regexp.MustCompile(`<(https?://[^>]+)>`)
)

// DiscoverLinks finds external http(s) URLs in Markdown plus explicit config links (FR-3).
func DiscoverLinks(markdown string, configJSON json.RawMessage, extra []string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(raw string) {
		raw = strings.TrimSpace(raw)
		raw = strings.TrimRight(raw, ".,);]}")
		if raw == "" {
			return
		}
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return
		}
		norm, err := NormalizeURL(raw)
		if err != nil {
			norm = raw
		}
		if _, ok := seen[norm]; ok {
			return
		}
		seen[norm] = struct{}{}
		out = append(out, norm)
	}
	for _, m := range mdLinkRe.FindAllStringSubmatch(markdown, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range angleURLRe.FindAllStringSubmatch(markdown, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range bareURLRe.FindAllString(markdown, -1) {
		add(m)
	}
	for _, u := range extra {
		add(u)
	}
	for _, u := range linksFromConfig(configJSON) {
		add(u)
	}
	return out
}

func linksFromConfig(configJSON json.RawMessage) []string {
	if len(configJSON) == 0 || string(configJSON) == "null" {
		return nil
	}
	var generic map[string]any
	if err := json.Unmarshal(configJSON, &generic); err != nil {
		return nil
	}
	var out []string
	collect := func(v any) {
		switch t := v.(type) {
		case string:
			if strings.HasPrefix(strings.ToLower(strings.TrimSpace(t)), "http") {
				out = append(out, t)
			}
		case []any:
			for _, item := range t {
				if s, ok := item.(string); ok {
					out = append(out, s)
				}
			}
		}
	}
	for _, key := range []string{"links", "urls", "sourceLinks", "referenceLinks"} {
		if v, ok := generic[key]; ok {
			collect(v)
		}
	}
	return out
}

// SectionBodyForInstance extracts the Markdown section containing the lex-tool fence
// for instanceID; falls back to full markdown.
func SectionBodyForInstance(markdown, instanceID string) string {
	if instanceID == "" || markdown == "" {
		return markdown
	}
	needle := `"id": "` + instanceID + `"`
	needle2 := `"id":"` + instanceID + `"`
	idx := strings.Index(markdown, needle)
	if idx < 0 {
		idx = strings.Index(markdown, needle2)
	}
	if idx < 0 {
		return markdown
	}
	// Walk back to previous heading or start.
	start := strings.LastIndex(markdown[:idx], "\n#")
	if start < 0 {
		start = 0
	} else {
		start++ // skip newline
	}
	rest := markdown[idx:]
	endRel := strings.Index(rest, "\n#")
	end := len(markdown)
	if endRel >= 0 {
		end = idx + endRel
	}
	return strings.TrimSpace(markdown[start:end])
}
