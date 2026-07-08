package introcourse

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

//go:embed content/**
var contentFS embed.FS

const defaultLocale = "en"

// ModuleMeta is parsed from content/<locale>/<module>/module.yaml.
type ModuleMeta struct {
	Slug         string
	Title        string
	SortOrder    int
	RequiresFlag string
	Dir          string
}

// GradingConfig is grading front-matter shared by quizzes and assignments (IC04).
type GradingConfig struct {
	Points              int
	Group               string
	GradePolicy         string
	MaxAttempts         int
	GradeAttemptPolicy  string
	SubmissionModes     []string
}

// PageFixture is a content page markdown fixture with front-matter.
type PageFixture struct {
	Slug         string
	Title        string
	SortOrder    int
	RequiresFlag string
	Markdown     string
	I18nKey      string
	ModuleDir    string
	FilePath     string
}

// AssignmentFixture is a gradable assignment markdown fixture with front-matter.
type AssignmentFixture struct {
	Slug         string
	Title        string
	SortOrder    int
	RequiresFlag string
	Markdown     string
	I18nKey      string
	Grading      GradingConfig
	ModuleDir    string
	FilePath     string
}

// QuizFixture is a knowledge-check quiz fixture (JSON).
type QuizFixture struct {
	Slug         string
	Title        string
	SortOrder    int
	RequiresFlag string
	Markdown     string
	Questions    []coursemodulequiz.QuizQuestion
	Grading      GradingConfig
	ModuleDir    string
	FilePath     string
}

// Curriculum is the parsed source curriculum for a locale.
type Curriculum struct {
	Locale  string
	Modules []ModuleSpec
}

// ModuleSpec is a module and its child items after parsing fixtures.
type ModuleSpec struct {
	Meta        ModuleMeta
	Pages       []PageFixture
	Assignments []AssignmentFixture
	Quizzes     []QuizFixture
}

// LoadCurriculum parses embedded fixtures for locale (falls back to English).
func LoadCurriculum(locale string) (*Curriculum, error) {
	loc := strings.TrimSpace(locale)
	if loc == "" {
		loc = defaultLocale
	}
	root := path.Join("content", loc)
	if _, err := contentFS.ReadDir(root); err != nil {
		if loc != defaultLocale {
			return LoadCurriculum(defaultLocale)
		}
		return nil, fmt.Errorf("intro course content: locale %q missing", loc)
	}
	modules, err := loadModules(root)
	if err != nil {
		return nil, err
	}
	return &Curriculum{Locale: loc, Modules: modules}, nil
}

func loadModules(root string) ([]ModuleSpec, error) {
	entries, err := contentFS.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			dirs = append(dirs, e.Name())
		}
	}
	sort.Strings(dirs)

	var modules []ModuleSpec
	for _, dir := range dirs {
		modRoot := path.Join(root, dir)
		meta, err := loadModuleMeta(modRoot, dir)
		if err != nil {
			return nil, err
		}
		pages, assignments, quizzes, err := loadModuleChildren(modRoot, dir)
		if err != nil {
			return nil, err
		}
		modules = append(modules, ModuleSpec{
			Meta: meta, Pages: pages, Assignments: assignments, Quizzes: quizzes,
		})
	}
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Meta.SortOrder < modules[j].Meta.SortOrder
	})
	return modules, nil
}

func loadModuleMeta(modRoot, dir string) (ModuleMeta, error) {
	b, err := contentFS.ReadFile(path.Join(modRoot, "module.yaml"))
	if err != nil {
		return ModuleMeta{}, fmt.Errorf("%s: read module.yaml: %w", modRoot, err)
	}
	fields, err := parseSimpleYAML(string(b))
	if err != nil {
		return ModuleMeta{}, fmt.Errorf("%s: %w", modRoot, err)
	}
	meta := ModuleMeta{
		Slug:         strings.TrimSpace(fields["slug"]),
		Title:        strings.TrimSpace(fields["title"]),
		SortOrder:    atoiDefault(fields["sort_order"], -1),
		RequiresFlag: strings.TrimSpace(fields["requires_flag"]),
		Dir:          dir,
	}
	if meta.Slug == "" {
		return ModuleMeta{}, fmt.Errorf("%s: module slug is required", modRoot)
	}
	if meta.Title == "" {
		return ModuleMeta{}, fmt.Errorf("%s: module title is required", modRoot)
	}
	if meta.SortOrder < 0 {
		return ModuleMeta{}, fmt.Errorf("%s: module sort_order is required", modRoot)
	}
	return meta, nil
}

func loadModuleChildren(modRoot, dir string) ([]PageFixture, []AssignmentFixture, []QuizFixture, error) {
	if err := fs.WalkDir(contentFS, modRoot, func(_ string, _ fs.DirEntry, walkErr error) error {
		return walkErr
	}); err != nil {
		return nil, nil, nil, err
	}
	entries, err := contentFS.ReadDir(modRoot)
	if err != nil {
		return nil, nil, nil, err
	}

	var pages []PageFixture
	var assignments []AssignmentFixture
	var quizzes []QuizFixture
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == "module.yaml" || strings.HasPrefix(name, ".") {
			continue
		}
		full := path.Join(modRoot, name)
		switch {
		case strings.HasSuffix(name, ".md"):
			fm, body, ferr := readMarkdownFixture(full)
			if ferr != nil {
				return nil, nil, nil, ferr
			}
			if strings.TrimSpace(fm["kind"]) == "assignment" {
				assign, aerr := assignmentFromFrontMatter(full, dir, fm, body)
				if aerr != nil {
					return nil, nil, nil, aerr
				}
				assignments = append(assignments, assign)
			} else {
				page, perr := pageFromFrontMatter(full, dir, fm, body)
				if perr != nil {
					return nil, nil, nil, perr
				}
				pages = append(pages, page)
			}
		case strings.HasSuffix(name, ".json"):
			quiz, qerr := loadQuizFixture(full, dir)
			if qerr != nil {
				return nil, nil, nil, qerr
			}
			quizzes = append(quizzes, quiz)
		}
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].SortOrder < pages[j].SortOrder })
	sort.Slice(assignments, func(i, j int) bool { return assignments[i].SortOrder < assignments[j].SortOrder })
	sort.Slice(quizzes, func(i, j int) bool { return quizzes[i].SortOrder < quizzes[j].SortOrder })
	return pages, assignments, quizzes, nil
}

func readMarkdownFixture(filePath string) (map[string]string, string, error) {
	b, err := contentFS.ReadFile(filePath)
	if err != nil {
		return nil, "", err
	}
	fm, body, err := splitFrontMatter(string(b))
	if err != nil {
		return nil, "", fmt.Errorf("%s: %w", filePath, err)
	}
	return fm, strings.TrimSpace(body), nil
}

func pageFromFrontMatter(filePath, moduleDir string, fm map[string]string, body string) (PageFixture, error) {
	page := PageFixture{
		Slug:         strings.TrimSpace(fm["slug"]),
		Title:        strings.TrimSpace(fm["title"]),
		SortOrder:    atoiDefault(fm["sort_order"], -1),
		RequiresFlag: strings.TrimSpace(fm["requires_flag"]),
		I18nKey:      strings.TrimSpace(fm["i18n_key"]),
		Markdown:     body,
		ModuleDir:    moduleDir,
		FilePath:     filePath,
	}
	if page.Slug == "" || page.Title == "" || page.SortOrder < 0 {
		return PageFixture{}, fmt.Errorf("%s: slug, title, and sort_order are required", filePath)
	}
	if page.Markdown == "" {
		return PageFixture{}, fmt.Errorf("%s: page body is empty", filePath)
	}
	return page, nil
}

func assignmentFromFrontMatter(filePath, moduleDir string, fm map[string]string, body string) (AssignmentFixture, error) {
	assign := AssignmentFixture{
		Slug:         strings.TrimSpace(fm["slug"]),
		Title:        strings.TrimSpace(fm["title"]),
		SortOrder:    atoiDefault(fm["sort_order"], -1),
		RequiresFlag: strings.TrimSpace(fm["requires_flag"]),
		I18nKey:      strings.TrimSpace(fm["i18n_key"]),
		Markdown:     body,
		Grading:      parseGradingFrontMatter(fm, GradePolicyCompletionFull),
		ModuleDir:    moduleDir,
		FilePath:     filePath,
	}
	if assign.Slug == "" || assign.Title == "" || assign.SortOrder < 0 {
		return AssignmentFixture{}, fmt.Errorf("%s: slug, title, and sort_order are required", filePath)
	}
	if assign.Markdown == "" {
		return AssignmentFixture{}, fmt.Errorf("%s: assignment body is empty", filePath)
	}
	if assign.Grading.Points <= 0 {
		return AssignmentFixture{}, fmt.Errorf("%s: points is required for assignments", filePath)
	}
	if assign.Grading.Group == "" {
		assign.Grading.Group = "Assignments"
	}
	if assign.Grading.GradePolicy == "" {
		assign.Grading.GradePolicy = GradePolicyCompletionFull
	}
	if len(assign.Grading.SubmissionModes) == 0 {
		assign.Grading.SubmissionModes = []string{"text"}
	}
	return assign, nil
}

func loadQuizFixture(filePath, moduleDir string) (QuizFixture, error) {
	b, err := contentFS.ReadFile(filePath)
	if err != nil {
		return QuizFixture{}, err
	}
	var raw struct {
		Slug         string                          `json:"slug"`
		Title        string                          `json:"title"`
		SortOrder    int                             `json:"sort_order"`
		RequiresFlag string                          `json:"requires_flag"`
		Markdown     string                          `json:"markdown"`
		Points       int                             `json:"points"`
		Group        string                          `json:"group"`
		GradePolicy  string                          `json:"grade_policy"`
		MaxAttempts  int                             `json:"max_attempts"`
		GradeAttemptPolicy string                    `json:"grade_attempt_policy"`
		Questions    []coursemodulequiz.QuizQuestion `json:"questions"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return QuizFixture{}, fmt.Errorf("%s: invalid JSON: %w", filePath, err)
	}
	q := QuizFixture{
		Slug:         strings.TrimSpace(raw.Slug),
		Title:        strings.TrimSpace(raw.Title),
		SortOrder:    raw.SortOrder,
		RequiresFlag: strings.TrimSpace(raw.RequiresFlag),
		Markdown:     strings.TrimSpace(raw.Markdown),
		Questions:    raw.Questions,
		Grading: quizGradingFromJSON(struct {
			Points             int    `json:"points"`
			Group              string `json:"group"`
			GradePolicy        string `json:"grade_policy"`
			MaxAttempts        int    `json:"max_attempts"`
			GradeAttemptPolicy string `json:"grade_attempt_policy"`
		}{
			Points:             raw.Points,
			Group:              raw.Group,
			GradePolicy:        raw.GradePolicy,
			MaxAttempts:        raw.MaxAttempts,
			GradeAttemptPolicy: raw.GradeAttemptPolicy,
		}),
		ModuleDir:    moduleDir,
		FilePath:     filePath,
	}
	if q.Slug == "" || q.Title == "" || q.SortOrder < 0 {
		return QuizFixture{}, fmt.Errorf("%s: slug, title, and sort_order are required", filePath)
	}
	if q.Grading.Points <= 0 {
		q.Grading.Points = sumQuestionPoints(q.Questions)
	}
	if q.Grading.GradePolicy == "" {
		q.Grading.GradePolicy = GradePolicyQuizAutoscore
	}
	if q.Grading.Group == "" {
		q.Grading.Group = "Quizzes"
	}
	if q.Grading.MaxAttempts <= 0 {
		q.Grading.MaxAttempts = 3
	}
	if q.Grading.GradeAttemptPolicy == "" {
		q.Grading.GradeAttemptPolicy = "highest"
	}
	return q, nil
}

func quizGradingFromJSON(raw struct {
	Points             int    `json:"points"`
	Group              string `json:"group"`
	GradePolicy        string `json:"grade_policy"`
	MaxAttempts        int    `json:"max_attempts"`
	GradeAttemptPolicy string `json:"grade_attempt_policy"`
}) GradingConfig {
	return GradingConfig{
		Points:             raw.Points,
		Group:              strings.TrimSpace(raw.Group),
		GradePolicy:        strings.TrimSpace(raw.GradePolicy),
		MaxAttempts:        raw.MaxAttempts,
		GradeAttemptPolicy: strings.TrimSpace(raw.GradeAttemptPolicy),
	}
}

func parseGradingFrontMatter(fm map[string]string, defaultPolicy string) GradingConfig {
	modes := strings.Split(strings.TrimSpace(fm["submission_modes"]), ",")
	var cleaned []string
	for _, m := range modes {
		if s := strings.TrimSpace(m); s != "" {
			cleaned = append(cleaned, s)
		}
	}
	return GradingConfig{
		Points:             atoiDefault(fm["points"], 0),
		Group:              strings.TrimSpace(fm["group"]),
		GradePolicy:        strings.TrimSpace(fm["grade_policy"]),
		MaxAttempts:        atoiDefault(fm["max_attempts"], 0),
		GradeAttemptPolicy: strings.TrimSpace(fm["grade_attempt_policy"]),
		SubmissionModes:    cleaned,
	}
}

func sumQuestionPoints(questions []coursemodulequiz.QuizQuestion) int {
	var total int
	for _, q := range questions {
		pts := int(q.Points)
		if pts <= 0 {
			pts = 1
		}
		total += pts
	}
	if total <= 0 {
		return 1
	}
	return total
}

func splitFrontMatter(src string) (map[string]string, string, error) {
	s := strings.TrimPrefix(src, "\ufeff")
	if !strings.HasPrefix(s, "---") {
		return nil, "", fmt.Errorf("missing front-matter opener ---")
	}
	rest := s[3:]
	rest = strings.TrimLeft(rest, "\r\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, "", fmt.Errorf("missing front-matter closer ---")
	}
	header := rest[:end]
	body := strings.TrimLeft(rest[end+4:], "\r\n")
	fm, err := parseSimpleYAML(header)
	if err != nil {
		return nil, "", err
	}
	return fm, body, nil
}

func parseSimpleYAML(header string) (map[string]string, error) {
	out := make(map[string]string)
	for _, line := range strings.Split(header, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid YAML line %q", line)
		}
		out[strings.TrimSpace(key)] = unquoteYAML(strings.TrimSpace(val))
	}
	return out, nil
}

func unquoteYAML(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func atoiDefault(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return def
	}
	return n
}

// FilterCurriculum returns modules/items whose requires_flag gates pass for cfg.
func FilterCurriculum(cur *Curriculum, cfg config.Config) []ModuleSpec {
	if cur == nil {
		return nil
	}
	var out []ModuleSpec
	for _, mod := range cur.Modules {
		if !FlagEnabled(cfg, mod.Meta.RequiresFlag) {
			continue
		}
		spec := ModuleSpec{Meta: mod.Meta}
		for _, p := range mod.Pages {
			if FlagEnabled(cfg, p.RequiresFlag) {
				spec.Pages = append(spec.Pages, p)
			}
		}
		for _, a := range mod.Assignments {
			if FlagEnabled(cfg, a.RequiresFlag) {
				spec.Assignments = append(spec.Assignments, a)
			}
		}
		for _, q := range mod.Quizzes {
			if FlagEnabled(cfg, q.RequiresFlag) {
				spec.Quizzes = append(spec.Quizzes, q)
			}
		}
		out = append(out, spec)
	}
	return out
}

// AllDesiredSlugs returns slugs that would be synced for cfg.
func AllDesiredSlugs(cur *Curriculum, cfg config.Config) []string {
	mods := FilterCurriculum(cur, cfg)
	var slugs []string
	for _, mod := range mods {
		slugs = append(slugs, mod.Meta.Slug)
		for _, p := range mod.Pages {
			slugs = append(slugs, p.Slug)
		}
		for _, a := range mod.Assignments {
			slugs = append(slugs, a.Slug)
		}
		for _, q := range mod.Quizzes {
			slugs = append(slugs, q.Slug)
		}
	}
	return slugs
}

// ContentFS exposes the embedded fixture tree for validation tooling.
func ContentFS() embed.FS {
	return contentFS
}