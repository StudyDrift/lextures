package emailtemplates

import (
	"regexp"
	"strings"
)

var mergeTokenRe = regexp.MustCompile(`\{\{([a-zA-Z0-9_.]+)\}\}`)

// Merge replaces {{field}} tokens with values from data. Missing keys become empty strings.
func Merge(template string, data map[string]string) string {
	if template == "" {
		return ""
	}
	return mergeTokenRe.ReplaceAllStringFunc(template, func(token string) string {
		key := strings.TrimSuffix(strings.TrimPrefix(token, "{{"), "}}")
		if v, ok := data[key]; ok {
			return v
		}
		return ""
	})
}

// FindUnknownTokens returns merge tokens not present in allowed keys.
func FindUnknownTokens(template string, allowed map[string]string) []string {
	if allowed == nil {
		allowed = map[string]string{}
	}
	seen := map[string]struct{}{}
	var unknown []string
	for _, m := range mergeTokenRe.FindAllStringSubmatch(template, -1) {
		if len(m) < 2 {
			continue
		}
		key := m[1]
		if _, ok := allowed[key]; ok {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		unknown = append(unknown, key)
	}
	return unknown
}

// StripHTMLTags produces a plain-text fallback from HTML.
func StripHTMLTags(html string) string {
	s := html
	s = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(s, "")
	s = regexp.MustCompile(`<br\s*/?>`).ReplaceAllString(s, "\n")
	s = regexp.MustCompile(`</p>`).ReplaceAllString(s, "\n\n")
	s = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.TrimSpace(regexp.MustCompile(`\n{3,}`).ReplaceAllString(s, "\n\n"))
	return s
}

// MapJobVars converts legacy email job template vars to merge-field keys.
func MapJobVars(vars map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range vars {
		out[k] = v
	}
	if v, ok := vars["courseName"]; ok {
		out["course.title"] = v
	}
	if v, ok := vars["assignmentName"]; ok {
		out["assignment.title"] = v
	}
	if v, ok := vars["dueAt"]; ok {
		out["assignment.due_at"] = v
	}
	if v, ok := vars["discussionTitle"]; ok {
		out["discussion.title"] = v
	}
	if v, ok := vars["unsubscribeUrl"]; ok {
		out["unsubscribe_url"] = v
	}
	if v, ok := vars["expiresAt"]; ok {
		out["expires_at"] = v
	}
	// Bidirectional link ↔ resetUrl for password-reset built-ins vs slot catalog.
	if v, ok := vars["link"]; ok {
		if _, has := out["resetUrl"]; !has {
			out["resetUrl"] = v
		}
	}
	if v, ok := vars["resetUrl"]; ok {
		if _, has := out["link"]; !has {
			out["link"] = v
		}
	}
	if v, ok := vars["studentName"]; ok {
		out["student.name"] = v
	}
	if v, ok := vars["orgName"]; ok {
		out["org.name"] = v
	}
	return out
}
