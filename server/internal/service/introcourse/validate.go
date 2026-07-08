package introcourse

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/lextures/lextures/server/internal/config"
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
	"/",
}

var (
	markdownLinkRE = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
	htmlLinkRE     = regexp.MustCompile(`<a\s+[^>]*href=["']([^"']+)["']`)
	scriptTagRE    = regexp.MustCompile(`(?i)<script\b`)
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
	return fmt.Errorf("intro course content validation failed (%d errors):\n- %s",
		len(e.Errors), strings.Join(e.Errors, "\n- "))
}

// ValidateCurriculum checks fixture structure, slugs, flags, links, quiz schema, and syllabus.
func ValidateCurriculum(cur *Curriculum) error {
	if cur == nil {
		return fmt.Errorf("curriculum is nil")
	}
	syllabus, err := LoadSyllabus(defaultLocale)
	if err != nil {
		return err
	}
	if err := ValidateSyllabus(syllabus); err != nil {
		return err
	}
	var ve ValidationError
	slugs := make(map[string]string)

	for _, mod := range cur.Modules {
		checkSlug(&ve, slugs, mod.Meta.Slug, mod.Meta.Dir+"/module.yaml")
		checkRequiresFlag(&ve, mod.Meta.RequiresFlag, mod.Meta.Dir)
		if !strings.HasPrefix(mod.Meta.Slug, "m") {
			ve.add(fmt.Sprintf("module slug %q should start with mN", mod.Meta.Slug))
		}
		for _, p := range mod.Pages {
			checkSlug(&ve, slugs, p.Slug, p.FilePath)
			checkRequiresFlag(&ve, p.RequiresFlag, p.FilePath)
			checkPageMarkdown(&ve, p)
		}
		for _, a := range mod.Assignments {
			checkSlug(&ve, slugs, a.Slug, a.FilePath)
			checkRequiresFlag(&ve, a.RequiresFlag, a.FilePath)
			checkAssignment(&ve, a)
		}
		for _, q := range mod.Quizzes {
			checkSlug(&ve, slugs, q.Slug, q.FilePath)
			checkRequiresFlag(&ve, q.RequiresFlag, q.FilePath)
			checkQuiz(&ve, q)
		}
		if len(mod.Quizzes) != 1 {
			ve.add(fmt.Sprintf("module %q must have exactly one knowledge-check quiz, got %d", mod.Meta.Slug, len(mod.Quizzes)))
		}
	}

	var assignmentCount int
	for _, mod := range cur.Modules {
		assignmentCount += len(mod.Assignments)
	}
	if assignmentCount < 2 {
		ve.add(fmt.Sprintf("curriculum must include at least 2 assignments, got %d", assignmentCount))
	}

	cfg := config.Config{
		LearnerProfileEnabled:        true,
		PushNotificationsEnabled:     true,
		AdaptiveLearnerModelEnabled:  true,
		SRSPracticeEnabled:           true,
		DiagnosticAssessmentsEnabled: true,
		SelfReflectionEnabled:        true,
		AiDisclosureEnabled:          true,
	}
	desired := AllDesiredSlugs(cur, cfg)
	if len(desired) == 0 {
		ve.add("no curriculum items after filtering with all flags enabled")
	}
	return ve.Err()
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

func checkRequiresFlag(ve *ValidationError, flag, src string) {
	if flag == "" {
		return
	}
	if !IsKnownRequiresFlag(flag) {
		ve.add(fmt.Sprintf("%s: unknown requires_flag %q", src, flag))
	}
}

func checkPageMarkdown(ve *ValidationError, p PageFixture) {
	if scriptTagRE.MatchString(p.Markdown) {
		ve.add(fmt.Sprintf("%s: script tags are not allowed in content", p.FilePath))
	}
	for _, href := range extractLinks(p.Markdown) {
		if err := validateDeepLink(href); err != nil {
			ve.add(fmt.Sprintf("%s: %v", p.FilePath, err))
		}
	}
}

func extractLinks(markdown string) []string {
	var out []string
	for _, m := range markdownLinkRE.FindAllStringSubmatch(markdown, -1) {
		if len(m) > 1 {
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

func validateDeepLink(href string) error {
	h := strings.TrimSpace(href)
	if h == "" || strings.HasPrefix(h, "#") {
		return nil
	}
	if strings.Contains(h, "://") || strings.HasPrefix(strings.ToLower(h), "javascript:") {
		return fmt.Errorf("deep link %q must be a same-origin relative route", href)
	}
	if !strings.HasPrefix(h, "/") {
		return fmt.Errorf("deep link %q must start with /", href)
	}
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
	if n < 3 || n > 5 {
		ve.add(fmt.Sprintf("%s: quiz must have 3–5 questions, got %d", q.FilePath, n))
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
		if question.QuestionType == "multiple_choice" && question.CorrectChoiceIndex == nil {
			ve.add(fmt.Sprintf("%s: question %d multiple_choice requires correctChoiceIndex", q.FilePath, i))
		}
	}
	if q.Grading.GradePolicy != "" && q.Grading.GradePolicy != GradePolicyQuizAutoscore {
		ve.add(fmt.Sprintf("%s: quizzes must use grade_policy %q", q.FilePath, GradePolicyQuizAutoscore))
	}
}