package marketplacecourses

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

// AllowedDeepLinkPrefixes are same-origin relative routes permitted in page markdown.
var AllowedDeepLinkPrefixes = []string{
	"/courses",
	"/settings/",
	"/privacy-centre",
	"/notebooks",
	"/inbox",
	"/calendar",
	"/marketplace",
	"/",
}

var (
	markdownLinkRE  = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	htmlLinkRE      = regexp.MustCompile(`(?i)<a\s+[^>]*href=["']([^"']+)["']`)
	markdownImageRE = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)
	scriptTagRE     = regexp.MustCompile(`(?i)<script\b`)
	headingRE       = regexp.MustCompile(`(?m)^(#{1,6})\s+\S`)
)

// ValidationError collects fixture validation failures.
type ValidationError struct {
	Errors []string
}

func (e *ValidationError) add(msg string) {
	e.Errors = append(e.Errors, msg)
}

func (e *ValidationError) Err() error {
	if len(e.Errors) == 0 {
		return nil
	}
	return fmt.Errorf("marketplace course content validation failed (%d errors):\n- %s",
		len(e.Errors), strings.Join(e.Errors, "\n- "))
}

// ValidateCourseSpec checks manifest, structure, slugs, links, quiz schema, and syllabus.
func ValidateCourseSpec(spec *CourseSpec) error {
	if spec == nil {
		return fmt.Errorf("course spec is nil")
	}
	var ve ValidationError
	checkManifest(&ve, spec.Manifest)

	slugs := make(map[string]string)
	internal := make(map[string]struct{})
	for _, mod := range spec.Modules {
		checkSlug(&ve, slugs, mod.Meta.Slug, mod.Meta.Dir+"/module.yaml")
		internal[mod.Meta.Slug] = struct{}{}
		for _, p := range mod.Pages {
			checkSlug(&ve, slugs, p.Slug, p.FilePath)
			internal[p.Slug] = struct{}{}
			checkPageMarkdown(&ve, p)
		}
		for _, a := range mod.Assignments {
			checkSlug(&ve, slugs, a.Slug, a.FilePath)
			internal[a.Slug] = struct{}{}
			checkAssignment(&ve, a)
		}
		for _, q := range mod.Quizzes {
			checkSlug(&ve, slugs, q.Slug, q.FilePath)
			internal[q.Slug] = struct{}{}
			checkQuiz(&ve, q)
		}
		if len(mod.Pages) < 1 {
			ve.add(fmt.Sprintf("module %q must have at least one page", mod.Meta.Slug))
		}
		if len(mod.Quizzes) != 1 {
			ve.add(fmt.Sprintf("module %q must have exactly one knowledge-check quiz, got %d", mod.Meta.Slug, len(mod.Quizzes)))
		}
	}
	if len(spec.Modules) == 0 {
		ve.add("course must include at least one module")
	}

	if verr := ValidateSyllabus(spec.Syllabus, internal); verr != nil {
		ve.Errors = append(ve.Errors, verr.Errors...)
	}

	// Re-check page links with full internal slug set.
	for _, mod := range spec.Modules {
		for _, p := range mod.Pages {
			for _, href := range extractLinks(p.Markdown) {
				if err := validateLink(href, internal); err != nil {
					ve.add(fmt.Sprintf("%s: %v", p.FilePath, err))
				}
			}
		}
	}

	return ve.Err()
}

func checkManifest(ve *ValidationError, m CourseManifest) {
	src := "course.yaml"
	if m.DirSlug != "" {
		src = "content/" + m.DirSlug + "/course.yaml"
	}
	if strings.TrimSpace(m.Code) == "" {
		ve.add(src + ": code is required")
	} else if !courseCodeValid(m.Code) {
		ve.add(fmt.Sprintf("%s: code %q must match C-[A-Z0-9]{6}", src, m.Code))
	}
	if strings.TrimSpace(m.Title) == "" {
		ve.add(src + ": title is required")
	}
	if strings.TrimSpace(m.CatalogSlug) == "" {
		ve.add(src + ": catalog_slug is required")
	}
	if strings.TrimSpace(m.Summary) == "" {
		ve.add(src + ": summary is required")
	}
	switch m.DifficultyLevel {
	case "beginner", "intermediate", "advanced":
	default:
		ve.add(fmt.Sprintf("%s: difficulty_level must be beginner|intermediate|advanced, got %q", src, m.DifficultyLevel))
	}
	if m.PriceCents != 0 {
		ve.add(fmt.Sprintf("%s: price_cents must be 0 for official free courses, got %d", src, m.PriceCents))
	}
	if !m.MarketplaceListed {
		ve.add(src + ": marketplace_listed must be true")
	}
	if m.ContentVersion <= 0 {
		ve.add(src + ": content_version must be positive")
	}
	if m.EstimatedMinutes < 0 {
		ve.add(src + ": estimated_minutes must be >= 0")
	}
	if len(m.Outcomes) == 0 {
		ve.add(src + ": at least one outcome is required")
	}
}

func courseCodeValid(code string) bool {
	if len(code) != 8 {
		return false
	}
	if code[0] != 'C' || code[1] != '-' {
		return false
	}
	for i := 2; i < 8; i++ {
		c := code[i]
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func checkSlug(ve *ValidationError, seen map[string]string, slug, src string) {
	if slug == "" {
		ve.add(src + ": empty slug")
		return
	}
	if prev, ok := seen[slug]; ok {
		ve.add(fmt.Sprintf("duplicate slug %q in %s (first seen in %s)", slug, src, prev))
		return
	}
	seen[slug] = src
}

func checkPageMarkdown(ve *ValidationError, p PageFixture) {
	if scriptTagRE.MatchString(p.Markdown) {
		ve.add(fmt.Sprintf("%s: script tags are not allowed in content", p.FilePath))
	}
	for _, m := range markdownImageRE.FindAllStringSubmatch(p.Markdown, -1) {
		if len(m) > 1 && strings.TrimSpace(m[1]) == "" {
			ve.add(fmt.Sprintf("%s: images require alt text", p.FilePath))
		}
	}
	checkHeadingOrder(ve, p.FilePath, p.Markdown)
	checkClickHere(ve, p.FilePath, p.Markdown)
}

func checkClickHere(ve *ValidationError, src, markdown string) {
	re := regexp.MustCompile(`(?i)\[([^\]]*(?:click here|here)[^\]]*)\]\([^)]+\)`)
	for _, m := range re.FindAllStringSubmatch(markdown, -1) {
		if len(m) > 1 {
			ve.add(fmt.Sprintf("%s: link text %q is not descriptive (avoid \"click here\")", src, m[1]))
		}
	}
}

func checkHeadingOrder(ve *ValidationError, src, markdown string) {
	matches := headingRE.FindAllStringSubmatch(markdown, -1)
	prev := 0
	for _, m := range matches {
		level := len(m[1])
		if prev > 0 && level > prev+1 {
			ve.add(fmt.Sprintf("%s: heading levels skip from h%d to h%d", src, prev, level))
		}
		prev = level
	}
}

func extractLinks(markdown string) []string {
	var out []string
	for _, m := range markdownLinkRE.FindAllStringSubmatch(markdown, -1) {
		if len(m) > 1 && !strings.HasPrefix(m[0], "!") {
			out = append(out, m[1])
		}
	}
	for _, m := range htmlLinkRE.FindAllStringSubmatch(markdown, -1) {
		if len(m) > 1 {
			out = append(out, m[1])
		}
	}
	return out
}

func validateLink(href string, internalSlugs map[string]struct{}) error {
	h := strings.TrimSpace(href)
	if h == "" || strings.HasPrefix(h, "#") {
		return nil
	}
	lower := strings.ToLower(h)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") {
		return fmt.Errorf("link %q uses a disallowed scheme", href)
	}
	if strings.Contains(h, "://") {
		u, err := url.Parse(h)
		if err != nil {
			return fmt.Errorf("invalid URL %q: %w", href, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("external link %q must be http(s)", href)
		}
		// External URLs are allowed; CI link-checker verifies reachability.
		return nil
	}
	if strings.HasPrefix(h, "/") {
		if _, err := url.Parse(h); err != nil {
			return fmt.Errorf("invalid deep link %q: %w", href, err)
		}
		for _, prefix := range AllowedDeepLinkPrefixes {
			if prefix == "/" {
				if h == "/" {
					return nil
				}
				continue
			}
			if h == prefix || strings.HasPrefix(h, prefix) {
				return nil
			}
		}
		return fmt.Errorf("deep link %q is not in the allowed route list", href)
	}
	// Relative internal slug reference (e.g. module/page slug).
	if internalSlugs != nil {
		ref := strings.SplitN(h, "#", 2)[0]
		if _, ok := internalSlugs[ref]; !ok {
			return fmt.Errorf("internal link %q does not resolve to a known slug", href)
		}
	}
	return nil
}

func checkAssignment(ve *ValidationError, a AssignmentFixture) {
	if a.Grading.Points <= 0 {
		ve.add(fmt.Sprintf("%s: assignment points must be positive", a.FilePath))
	}
	switch a.Grading.GradePolicy {
	case "", GradePolicyCompletionFull, GradePolicyGraderAgent:
	default:
		ve.add(fmt.Sprintf("%s: invalid grade_policy %q", a.FilePath, a.Grading.GradePolicy))
	}
	if len(a.Grading.SubmissionModes) == 0 {
		ve.add(fmt.Sprintf("%s: submission_modes is required for assignments", a.FilePath))
	}
}

func checkQuiz(ve *ValidationError, q QuizFixture) {
	n := len(q.Questions)
	if n < 1 {
		ve.add(fmt.Sprintf("%s: quiz must have at least one question, got %d", q.FilePath, n))
	}
	allowedTypes := make(map[string]struct{}, len(coursemodulequiz.QuizQuestionTypes))
	for _, t := range coursemodulequiz.QuizQuestionTypes {
		allowedTypes[t] = struct{}{}
	}
	for i, question := range q.Questions {
		if strings.TrimSpace(question.ID) == "" {
			ve.add(fmt.Sprintf("%s: question %d missing id", q.FilePath, i))
		}
		if strings.TrimSpace(question.Prompt) == "" {
			ve.add(fmt.Sprintf("%s: question %d missing prompt", q.FilePath, i))
		}
		if _, ok := allowedTypes[question.QuestionType]; !ok {
			ve.add(fmt.Sprintf("%s: question %d has invalid questionType %q", q.FilePath, i, question.QuestionType))
		}
		switch question.QuestionType {
		case "multiple_choice":
			if question.CorrectChoiceIndex == nil {
				ve.add(fmt.Sprintf("%s: question %d multiple_choice requires correctChoiceIndex", q.FilePath, i))
			} else if len(question.Choices) > 0 && int(*question.CorrectChoiceIndex) >= len(question.Choices) {
				ve.add(fmt.Sprintf("%s: question %d correctChoiceIndex out of range", q.FilePath, i))
			}
		case "true_false":
			if question.CorrectChoiceIndex == nil {
				ve.add(fmt.Sprintf("%s: question %d true_false requires correctChoiceIndex", q.FilePath, i))
			}
		}
	}
	if q.Grading.GradePolicy != "" && q.Grading.GradePolicy != GradePolicyQuizAutoscore {
		ve.add(fmt.Sprintf("%s: quizzes must use grade_policy %q", q.FilePath, GradePolicyQuizAutoscore))
	}
}

// ValidateAllCourses loads and validates every embedded course.
func ValidateAllCourses() error {
	slugs, err := ListCourseSlugs()
	if err != nil {
		return err
	}
	if len(slugs) == 0 {
		return fmt.Errorf("no marketplace courses found under content/")
	}
	var ve ValidationError
	seenCodes := map[string]string{}
	seenCatalog := map[string]string{}
	for _, dir := range slugs {
		spec, err := LoadCourseSpec(dir)
		if err != nil {
			ve.add(fmt.Sprintf("%s: %v", dir, err))
			continue
		}
		if err := ValidateCourseSpec(spec); err != nil {
			ve.add(err.Error())
		}
		if prev, ok := seenCodes[spec.Manifest.Code]; ok {
			ve.add(fmt.Sprintf("duplicate course code %q in %s and %s", spec.Manifest.Code, prev, dir))
		}
		seenCodes[spec.Manifest.Code] = dir
		if prev, ok := seenCatalog[spec.Manifest.CatalogSlug]; ok {
			ve.add(fmt.Sprintf("duplicate catalog_slug %q in %s and %s", spec.Manifest.CatalogSlug, prev, dir))
		}
		seenCatalog[spec.Manifest.CatalogSlug] = dir
	}
	return ve.Err()
}

// ExtractExternalURLs returns unique http(s) URLs from all course content (for CI link-check).
func ExtractExternalURLs() ([]string, error) {
	slugs, err := ListCourseSlugs()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(href string) {
		h := strings.TrimSpace(href)
		if !strings.Contains(h, "://") {
			return
		}
		u, err := url.Parse(h)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
			return
		}
		if _, ok := seen[h]; ok {
			return
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	for _, dir := range slugs {
		spec, err := LoadCourseSpec(dir)
		if err != nil {
			return nil, err
		}
		for _, mod := range spec.Modules {
			for _, p := range mod.Pages {
				for _, href := range extractLinks(p.Markdown) {
					add(href)
				}
			}
			for _, a := range mod.Assignments {
				for _, href := range extractLinks(a.Markdown) {
					add(href)
				}
			}
		}
		if spec.Syllabus != nil {
			for _, sec := range spec.Syllabus.Sections {
				for _, href := range extractLinks(sec.Markdown) {
					add(href)
				}
			}
		}
	}
	sortStrings(out)
	return out, nil
}

func sortStrings(s []string) {
	for i := 0; i < len(s); i++ {
		for j := i + 1; j < len(s); j++ {
			if s[j] < s[i] {
				s[i], s[j] = s[j], s[i]
			}
		}
	}
}
