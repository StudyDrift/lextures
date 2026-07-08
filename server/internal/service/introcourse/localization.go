package introcourse

import (
	"fmt"
	"strings"
	"sync"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

var localeIndexMu sync.RWMutex
var localeIndexCache = map[string]map[string]localizedEntry{}

type localizedEntry struct {
	Title     string
	Markdown  string
	Questions []coursemodulequiz.QuizQuestion
}

// ListContentLocales returns locale directories under content/ (always includes en).
func ListContentLocales() ([]string, error) {
	entries, err := contentFS.ReadDir("content")
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{"en": {}}
	var locales []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		name := strings.TrimSpace(e.Name())
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		locales = append(locales, name)
	}
	if len(locales) == 0 {
		return []string{"en"}, nil
	}
	// en first, then sorted remainder
	out := []string{"en"}
	for _, loc := range locales {
		if loc != "en" {
			out = append(out, loc)
		}
	}
	return out, nil
}

// LoadCurriculumMerged loads locale fixtures overlaid on English (IC08 FR-4).
func LoadCurriculumMerged(locale string) (*Curriculum, error) {
	base, err := LoadCurriculum(defaultLocale)
	if err != nil {
		return nil, err
	}
	loc := normalizeIntroLocale(locale)
	if loc == defaultLocale {
		return base, nil
	}
	overlay, err := LoadCurriculum(loc)
	if err != nil {
		return base, nil
	}
	return mergeCurriculum(base, overlay), nil
}

func normalizeIntroLocale(locale string) string {
	loc := strings.TrimSpace(locale)
	if loc == "" {
		return defaultLocale
	}
	if idx := strings.Index(loc, "-"); idx > 0 {
		loc = loc[:idx]
	}
	return strings.ToLower(loc)
}

func mergeCurriculum(base, overlay *Curriculum) *Curriculum {
	if base == nil {
		return overlay
	}
	if overlay == nil {
		return base
	}
	out := &Curriculum{Locale: overlay.Locale, Modules: make([]ModuleSpec, len(base.Modules))}
	overlayMods := make(map[string]ModuleSpec, len(overlay.Modules))
	for _, m := range overlay.Modules {
		overlayMods[m.Meta.Slug] = m
	}
	for i, mod := range base.Modules {
		merged := mod
		if ov, ok := overlayMods[mod.Meta.Slug]; ok {
			if strings.TrimSpace(ov.Meta.Title) != "" {
				merged.Meta.Title = ov.Meta.Title
			}
			merged.Pages = mergePages(mod.Pages, ov.Pages)
			merged.Assignments = mergeAssignments(mod.Assignments, ov.Assignments)
			merged.Quizzes = mergeQuizzes(mod.Quizzes, ov.Quizzes)
		}
		out.Modules[i] = merged
	}
	return out
}

func mergePages(base, overlay []PageFixture) []PageFixture {
	bySlug := make(map[string]PageFixture, len(overlay))
	for _, p := range overlay {
		bySlug[p.Slug] = p
	}
	out := make([]PageFixture, len(base))
	for i, p := range base {
		out[i] = p
		if ov, ok := bySlug[p.Slug]; ok {
			if strings.TrimSpace(ov.Title) != "" {
				out[i].Title = ov.Title
			}
			if strings.TrimSpace(ov.Markdown) != "" {
				out[i].Markdown = ov.Markdown
			}
		}
	}
	return out
}

func mergeAssignments(base, overlay []AssignmentFixture) []AssignmentFixture {
	bySlug := make(map[string]AssignmentFixture, len(overlay))
	for _, a := range overlay {
		bySlug[a.Slug] = a
	}
	out := make([]AssignmentFixture, len(base))
	for i, a := range base {
		out[i] = a
		if ov, ok := bySlug[a.Slug]; ok {
			if strings.TrimSpace(ov.Title) != "" {
				out[i].Title = ov.Title
			}
			if strings.TrimSpace(ov.Markdown) != "" {
				out[i].Markdown = ov.Markdown
			}
		}
	}
	return out
}

func mergeQuizzes(base, overlay []QuizFixture) []QuizFixture {
	bySlug := make(map[string]QuizFixture, len(overlay))
	for _, q := range overlay {
		bySlug[q.Slug] = q
	}
	out := make([]QuizFixture, len(base))
	for i, q := range base {
		out[i] = q
		if ov, ok := bySlug[q.Slug]; ok {
			if strings.TrimSpace(ov.Title) != "" {
				out[i].Title = ov.Title
			}
			if strings.TrimSpace(ov.Markdown) != "" {
				out[i].Markdown = ov.Markdown
			}
			if len(ov.Questions) > 0 {
				out[i].Questions = ov.Questions
			}
		}
	}
	return out
}

func buildLocaleIndex(locale string) (map[string]localizedEntry, error) {
	cur, err := LoadCurriculumMerged(locale)
	if err != nil {
		return nil, err
	}
	idx := make(map[string]localizedEntry)
	for _, mod := range cur.Modules {
		idx[mod.Meta.Slug] = localizedEntry{Title: mod.Meta.Title}
		for _, p := range mod.Pages {
			idx[p.Slug] = localizedEntry{Title: p.Title, Markdown: p.Markdown}
		}
		for _, a := range mod.Assignments {
			idx[a.Slug] = localizedEntry{Title: a.Title, Markdown: a.Markdown}
		}
		for _, q := range mod.Quizzes {
			idx[q.Slug] = localizedEntry{Title: q.Title, Markdown: q.Markdown, Questions: q.Questions}
		}
	}
	return idx, nil
}

func localeIndex(locale string) map[string]localizedEntry {
	loc := normalizeIntroLocale(locale)
	localeIndexMu.RLock()
	if idx, ok := localeIndexCache[loc]; ok {
		localeIndexMu.RUnlock()
		return idx
	}
	localeIndexMu.RUnlock()

	idx, err := buildLocaleIndex(loc)
	if err != nil {
		return nil
	}
	localeIndexMu.Lock()
	localeIndexCache[loc] = idx
	localeIndexMu.Unlock()
	return idx
}

// InvalidateLocaleIndex clears the fixture overlay cache (tests and after deploy).
func InvalidateLocaleIndex() {
	localeIndexMu.Lock()
	localeIndexCache = map[string]map[string]localizedEntry{}
	localeIndexMu.Unlock()
}

// LocalizePage applies locale fixture overlays with English fallback.
func LocalizePage(slug, title, markdown, locale string) (string, string) {
	if normalizeIntroLocale(locale) == defaultLocale || strings.TrimSpace(slug) == "" {
		return title, markdown
	}
	entry, ok := localeIndex(locale)[slug]
	if !ok {
		return title, markdown
	}
	if strings.TrimSpace(entry.Title) != "" {
		title = entry.Title
	}
	if strings.TrimSpace(entry.Markdown) != "" {
		markdown = entry.Markdown
	}
	return title, markdown
}

// LocalizeQuiz applies locale fixture overlays to quiz presentation fields.
func LocalizeQuiz(slug, title, markdown string, questions []coursemodulequiz.QuizQuestion, locale string) (string, string, []coursemodulequiz.QuizQuestion) {
	if normalizeIntroLocale(locale) == defaultLocale || strings.TrimSpace(slug) == "" {
		return title, markdown, questions
	}
	entry, ok := localeIndex(locale)[slug]
	if !ok {
		return title, markdown, questions
	}
	if strings.TrimSpace(entry.Title) != "" {
		title = entry.Title
	}
	if strings.TrimSpace(entry.Markdown) != "" {
		markdown = entry.Markdown
	}
	if len(entry.Questions) > 0 {
		questions = entry.Questions
	}
	return title, markdown, questions
}

// LocaleCoverage reports the fraction of English slugs with non-empty locale overlays.
func LocaleCoverage(locale string) (float64, error) {
	base, err := LoadCurriculum(defaultLocale)
	if err != nil {
		return 0, err
	}
	loc := normalizeIntroLocale(locale)
	if loc == defaultLocale {
		return 1, nil
	}
	overlay, err := LoadCurriculum(loc)
	if err != nil {
		return 0, nil
	}
	merged := mergeCurriculum(base, overlay)
	total := 0
	translated := 0
	for _, mod := range base.Modules {
		total++
		if modTitleTranslated(mod, merged, mod.Meta.Slug) {
			translated++
		}
		for _, p := range mod.Pages {
			total++
			if pageTranslated(p, merged, p.Slug) {
				translated++
			}
		}
		for _, a := range mod.Assignments {
			total++
			if assignmentTranslated(a, merged, a.Slug) {
				translated++
			}
		}
		for _, q := range mod.Quizzes {
			total++
			if quizTranslated(q, merged, q.Slug) {
				translated++
			}
		}
	}
	if total == 0 {
		return 0, nil
	}
	return float64(translated) / float64(total), nil
}

func modTitleTranslated(base ModuleSpec, merged *Curriculum, slug string) bool {
	for _, m := range merged.Modules {
		if m.Meta.Slug == slug && m.Meta.Title != base.Meta.Title && strings.TrimSpace(m.Meta.Title) != "" {
			return true
		}
	}
	return false
}

func pageTranslated(base PageFixture, merged *Curriculum, slug string) bool {
	for _, m := range merged.Modules {
		for _, p := range m.Pages {
			if p.Slug != slug {
				continue
			}
			return (p.Title != base.Title && strings.TrimSpace(p.Title) != "") ||
				(p.Markdown != base.Markdown && strings.TrimSpace(p.Markdown) != "")
		}
	}
	return false
}

func assignmentTranslated(base AssignmentFixture, merged *Curriculum, slug string) bool {
	for _, m := range merged.Modules {
		for _, a := range m.Assignments {
			if a.Slug != slug {
				continue
			}
			return (a.Title != base.Title && strings.TrimSpace(a.Title) != "") ||
				(a.Markdown != base.Markdown && strings.TrimSpace(a.Markdown) != "")
		}
	}
	return false
}

func quizTranslated(base QuizFixture, merged *Curriculum, slug string) bool {
	for _, m := range merged.Modules {
		for _, q := range m.Quizzes {
			if q.Slug != slug {
				continue
			}
			return (q.Title != base.Title && strings.TrimSpace(q.Title) != "") ||
				(q.Markdown != base.Markdown && strings.TrimSpace(q.Markdown) != "") ||
				(len(q.Questions) > 0 && !quizQuestionsEqual(q.Questions, base.Questions))
		}
	}
	return false
}

func quizQuestionsEqual(a, b []coursemodulequiz.QuizQuestion) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Prompt != b[i].Prompt {
			return false
		}
	}
	return true
}

// ValidateAllLocales checks English curriculum and locale overlay structure.
func ValidateAllLocales() error {
	en, err := LoadCurriculum(defaultLocale)
	if err != nil {
		return err
	}
	if err := ValidateCurriculum(en); err != nil {
		return err
	}
	locales, err := ListContentLocales()
	if err != nil {
		return err
	}
	var ve ValidationError
	for _, loc := range locales {
		if loc == defaultLocale {
			continue
		}
		overlay, err := LoadCurriculum(loc)
		if err != nil {
			ve.add(fmt.Sprintf("locale %q: %v", loc, err))
			continue
		}
		if err := validateLocaleOverlay(en, overlay); err != nil {
			ve.add(fmt.Sprintf("locale %q: %v", loc, err))
		}
	}
	return ve.Err()
}

func validateLocaleOverlay(en, overlay *Curriculum) error {
	enSlugs := allSlugs(en)
	var ve ValidationError
	for _, mod := range overlay.Modules {
		if _, ok := enSlugs[mod.Meta.Slug]; !ok {
			ve.add(fmt.Sprintf("unknown module slug %q", mod.Meta.Slug))
		}
		for _, p := range mod.Pages {
			if _, ok := enSlugs[p.Slug]; !ok {
				ve.add(fmt.Sprintf("unknown page slug %q", p.Slug))
			}
		}
		for _, a := range mod.Assignments {
			if _, ok := enSlugs[a.Slug]; !ok {
				ve.add(fmt.Sprintf("unknown assignment slug %q", a.Slug))
			}
		}
		for _, q := range mod.Quizzes {
			if _, ok := enSlugs[q.Slug]; !ok {
				ve.add(fmt.Sprintf("unknown quiz slug %q", q.Slug))
			}
		}
	}
	if len(ve.Errors) > 0 {
		return ve.Err()
	}
	merged := mergeCurriculum(en, overlay)
	return ValidateCurriculum(merged)
}

func allSlugs(cur *Curriculum) map[string]struct{} {
	out := make(map[string]struct{})
	for _, mod := range cur.Modules {
		out[mod.Meta.Slug] = struct{}{}
		for _, p := range mod.Pages {
			out[p.Slug] = struct{}{}
		}
		for _, a := range mod.Assignments {
			out[a.Slug] = struct{}{}
		}
		for _, q := range mod.Quizzes {
			out[q.Slug] = struct{}{}
		}
	}
	return out
}