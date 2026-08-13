package marketingcontentimport

import (
	"bufio"
	"fmt"
	"strings"
)

var allowedFrontmatter = map[string]bool{
	"title": true, "description": true, "date": true, "updated": true,
	"author": true, "reviewedBy": true, "reviewedAt": true, "reviewDue": true,
	"pillar": true, "briefRef": true, "citations": true, "category": true,
	"roles": true, "segments": true, "verifiedAgainst": true, "relatedTo": true,
	"primaryQuestion": true, "cluster": true, "keywords": true,
	// Existing editorial-lint metadata intentionally retained in extra.import.metadata.
	"contentContract": true, "supportTicketThemes": true,
}

type frontmatter struct {
	Values map[string]string
	Lists  map[string][]string
	Body   string
}

func parseFrontmatter(path, raw string) (frontmatter, error) {
	out := frontmatter{Values: map[string]string{}, Lists: map[string][]string{}, Body: raw}
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return out, nil
	}
	end := strings.Index(normalized[4:], "\n---\n")
	if end < 0 {
		return out, fmt.Errorf("%s: unterminated front matter", path)
	}
	header := normalized[4 : 4+end]
	out.Body = strings.TrimSpace(normalized[4+end+5:])
	s := bufio.NewScanner(strings.NewReader(header))
	line := 1
	for s.Scan() {
		line++
		text := strings.TrimSpace(s.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		colon := strings.IndexByte(text, ':')
		if colon < 1 {
			return out, fmt.Errorf("%s:%d: invalid front matter", path, line)
		}
		key := strings.TrimSpace(text[:colon])
		if !allowedFrontmatter[key] {
			return out, fmt.Errorf("%s:%d: unknown front-matter key %q", path, line, key)
		}
		value := unquote(strings.TrimSpace(text[colon+1:]))
		out.Values[key] = value
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			out.Lists[key] = splitList(value[1:len(value)-1], ",")
		} else if key == "citations" {
			out.Lists[key] = splitList(value, "|")
		}
	}
	return out, s.Err()
}

func unquote(v string) string {
	if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
		return v[1 : len(v)-1]
	}
	return v
}

func splitList(v, sep string) []string {
	if strings.TrimSpace(v) == "" {
		return []string{}
	}
	parts := strings.Split(v, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(unquote(strings.TrimSpace(p))); p != "" {
			out = append(out, p)
		}
	}
	return out
}
