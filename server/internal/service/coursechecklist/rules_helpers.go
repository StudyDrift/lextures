package coursechecklist

import (
	"strings"
	"time"
	"unicode"

	"github.com/lextures/lextures/server/internal/l10n"
)

func isStaffRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "teacher", "instructor", "ta", "designer", "admin":
		return true
	default:
		return false
	}
}

func isStudentRole(role string) bool {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "student", "learner":
		return true
	default:
		return false
	}
}

func isGradableKind(kind string) bool {
	switch kind {
	case "assignment", "quiz":
		return true
	default:
		return false
	}
}

func isPlaceholderTitle(title string) bool {
	t := strings.TrimSpace(strings.ToLower(title))
	if t == "" {
		return true
	}
	placeholders := []string{
		"untitled", "new course", "untitled course", "course title", "my course",
	}
	for _, p := range placeholders {
		if t == p {
			return true
		}
	}
	return false
}

func validIANATimezone(tz *string) (string, bool) {
	if tz == nil {
		return "", false
	}
	norm, err := l10n.NormalizeTimezone(*tz)
	if err != nil {
		return "", false
	}
	return norm, true
}

func daysUntil(from, to time.Time) int {
	d := int(to.Sub(from).Hours() / 24)
	return d
}

func snapLocale(snap CourseSnapshot) string {
	if snap.CatalogLanguage != "" {
		if n, err := l10n.NormalizeLocale(snap.CatalogLanguage); err == nil {
			return n
		}
		return snap.CatalogLanguage
	}
	return "en"
}

func lexiconForSnap(snap CourseSnapshot) *Lexicon {
	return LexiconFor(snapLocale(snap))
}

func syllabusUnknownFinding(item ItemID) Finding {
	return Finding{
		Status:        StatusUnknown,
		DetailKey:     "coursechecklist.item." + string(item) + ".detail.unknown",
		DetailDefault: "Syllabus content could not be read.",
	}
}

func truncatedDetail(base string, truncated bool) string {
	if !truncated {
		return base
	}
	if base == "" {
		return "Checked first 512 KB."
	}
	return base + " Checked first 512 KB."
}

// languageHeuristicMatch returns true when text roughly matches locale (script/stopwords).
// When text is empty, returns true (locale set is enough).
func languageHeuristicMatch(locale, text string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return true
	}
	primary := strings.ToLower(locale)
	if i := strings.IndexByte(primary, '-'); i > 0 {
		primary = primary[:i]
	}
	switch primary {
	case "ar":
		return scriptRatio(text, unicode.Arabic) >= 0.5
	case "es", "fr", "en":
		// Latin script + stop-word overlap heuristic.
		if scriptRatio(text, unicode.Latin) < 0.5 {
			return false
		}
		lower := strings.ToLower(text)
		stops := map[string][]string{
			"en": {"the", "and", "of", "to", "in", "for", "you", "this", "course"},
			"es": {"de", "la", "el", "en", "y", "los", "las", "del", "que", "para"},
			"fr": {"de", "la", "le", "et", "les", "des", "du", "en", "pour", "vous"},
		}
		list := stops[primary]
		if len(list) == 0 {
			return true
		}
		hits := 0
		for _, w := range list {
			if strings.Contains(lower, " "+w+" ") || strings.HasPrefix(lower, w+" ") {
				hits++
			}
		}
		return hits >= 2 || float64(hits)/float64(len(list)) >= 0.2
	default:
		return true
	}
}

func scriptRatio(text string, rangeTable *unicode.RangeTable) float64 {
	letters := 0
	matched := 0
	for _, r := range text {
		if !unicode.IsLetter(r) {
			continue
		}
		letters++
		if unicode.Is(rangeTable, r) {
			matched++
		}
	}
	if letters == 0 {
		return 0
	}
	return float64(matched) / float64(letters)
}

func countSupportLinks(text string, hints *keywordMatcher) []string {
	// Extract markdown/HTML-ish URLs and keep those near support hint keywords.
	var found []string
	lower := strings.ToLower(text)
	for _, hint := range hints.FindAll(lower, 20) {
		// Look for a URL within ~200 chars of the hint.
		idx := strings.Index(lower, strings.ToLower(hint))
		if idx < 0 {
			continue
		}
		windowStart := idx - 80
		if windowStart < 0 {
			windowStart = 0
		}
		windowEnd := idx + len(hint) + 200
		if windowEnd > len(text) {
			windowEnd = len(text)
		}
		window := text[windowStart:windowEnd]
		if url := firstURL(window); url != "" {
			found = append(found, url)
		} else {
			found = append(found, hint)
		}
	}
	// Dedup
	seen := map[string]struct{}{}
	var out []string
	for _, f := range found {
		k := strings.ToLower(f)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, f)
	}
	return out
}

func firstURL(s string) string {
	// Prefer markdown links [text](url)
	if i := strings.Index(s, "]("); i >= 0 {
		rest := s[i+2:]
		if j := strings.IndexByte(rest, ')'); j > 0 {
			u := strings.TrimSpace(rest[:j])
			if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "/") {
				return u
			}
		}
	}
	for _, prefix := range []string{"https://", "http://"} {
		if i := strings.Index(s, prefix); i >= 0 {
			end := i + len(prefix)
			for end < len(s) {
				c := s[end]
				if c == ' ' || c == '\n' || c == ')' || c == '"' || c == '<' || c == '>' {
					break
				}
				end++
			}
			return s[i:end]
		}
	}
	return ""
}

func firstModuleItems(snap CourseSnapshot) (module StructureItem, children []StructureItem, ok bool) {
	var mods []StructureItem
	for _, it := range snap.StructureItems {
		if it.Archived {
			continue
		}
		if it.Kind == "module" && it.ParentID == nil {
			mods = append(mods, it)
		}
	}
	if len(mods) == 0 {
		return StructureItem{}, nil, false
	}
	// Lowest sort_order wins.
	mod := mods[0]
	for _, m := range mods[1:] {
		if m.SortOrder < mod.SortOrder {
			mod = m
		}
	}
	for _, it := range snap.StructureItems {
		if it.Archived || it.ParentID == nil {
			continue
		}
		if *it.ParentID == mod.ID {
			children = append(children, it)
		}
	}
	return mod, children, true
}

func assignmentGroupWeightSum(groups []AssignmentGroupSnap) float64 {
	var sum float64
	for _, g := range groups {
		if g.Weight != nil {
			sum += *g.Weight
		}
	}
	return sum
}

func hasPrintBreakingEmbeds(text string) []string {
	lower := strings.ToLower(text)
	var out []string
	for _, tag := range []string{"<iframe", "<video", "<embed", "<object"} {
		if strings.Contains(lower, tag) {
			out = append(out, tag+">")
		}
	}
	return out
}
