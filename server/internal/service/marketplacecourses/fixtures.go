package marketplacecourses

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
)

//go:embed content/**
var contentFS embed.FS

// CourseManifest is parsed from content/<course-slug>/course.yaml.
type CourseManifest struct {
	Code              string   `yaml:"code" json:"code"`
	Title             string   `yaml:"title" json:"title"`
	CatalogSlug       string   `yaml:"catalog_slug" json:"catalog_slug"`
	CatalogCategory   string   `yaml:"catalog_category" json:"catalog_category"`
	DifficultyLevel   string   `yaml:"difficulty_level" json:"difficulty_level"`
	CatalogLanguage   string   `yaml:"catalog_language" json:"catalog_language"`
	Summary           string   `yaml:"summary" json:"summary"`
	Outcomes          []string `yaml:"outcomes" json:"outcomes"`
	EstimatedMinutes  int      `yaml:"estimated_minutes" json:"estimated_minutes"`
	PriceCents        int      `yaml:"price_cents" json:"price_cents"`
	IsPublic          bool     `yaml:"is_public" json:"is_public"`
	MarketplaceListed bool     `yaml:"marketplace_listed" json:"marketplace_listed"`
	HeroImage         string   `yaml:"hero_image" json:"hero_image"`
	ContentVersion    int      `yaml:"content_version" json:"content_version"`
	ShortCode         string   `yaml:"short_code" json:"short_code"`
	DirSlug           string   `json:"-"` // directory name under content/
}

// ModuleMeta is parsed from content/<course>/<locale>/<module>/module.yaml.
type ModuleMeta struct {
	Slug      string
	Title     string
	SortOrder int
	Dir       string
}

// GradingConfig is grading front-matter shared by quizzes and assignments.
type GradingConfig struct {
	Points             int
	Group              string
	GradePolicy        string
	MaxAttempts        int
	GradeAttemptPolicy string
	SubmissionModes    []string
}

// PageFixture is a content page markdown fixture with front-matter.
type PageFixture struct {
	Slug       string
	Title      string
	SortOrder  int
	Markdown   string
	ContentVer int
	ModuleDir  string
	FilePath   string
}

// AssignmentFixture is a gradable assignment markdown fixture with front-matter.
type AssignmentFixture struct {
	Slug       string
	Title      string
	SortOrder  int
	Markdown   string
	ContentVer int
	Grading    GradingConfig
	ModuleDir  string
	FilePath   string
}

// QuizFixture is a knowledge-check quiz fixture (JSON).
type QuizFixture struct {
	Slug       string
	Title      string
	SortOrder  int
	Markdown   string
	Questions  []coursemodulequiz.QuizQuestion
	ContentVer int
	Grading    GradingConfig
	ModuleDir  string
	FilePath   string
}

// ModuleSpec is a module and its child items after parsing fixtures.
type ModuleSpec struct {
	Meta        ModuleMeta
	Pages       []PageFixture
	Assignments []AssignmentFixture
	Quizzes     []QuizFixture
}

// Curriculum is the parsed source curriculum for a course locale.
type Curriculum struct {
	CourseSlug string
	Locale     string
	Modules    []ModuleSpec
}

// CourseSpec is a fully loaded official course (manifest + curriculum + syllabus).
type CourseSpec struct {
	Manifest CourseManifest
	Locale   string
	Modules  []ModuleSpec
	Syllabus *SyllabusFixture
}

// ListCourseSlugs returns course directory names under content/.
func ListCourseSlugs() ([]string, error) {
	entries, err := contentFS.ReadDir("content")
	if err != nil {
		return nil, err
	}
	var slugs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
			slugs = append(slugs, e.Name())
		}
	}
	sort.Strings(slugs)
	return slugs, nil
}

// LoadManifest parses course.yaml for a course directory slug.
func LoadManifest(courseDirSlug string) (CourseManifest, error) {
	filePath := path.Join("content", courseDirSlug, "course.yaml")
	b, err := contentFS.ReadFile(filePath)
	if err != nil {
		return CourseManifest{}, fmt.Errorf("%s: %w", filePath, err)
	}
	m, err := parseCourseYAML(string(b))
	if err != nil {
		return CourseManifest{}, fmt.Errorf("%s: %w", filePath, err)
	}
	m.DirSlug = courseDirSlug
	if m.CatalogSlug == "" {
		m.CatalogSlug = courseDirSlug
	}
	if m.CatalogLanguage == "" {
		m.CatalogLanguage = defaultLocale
	}
	if m.ShortCode == "" {
		m.ShortCode = "LEX-MC-" + strings.ToUpper(strings.ReplaceAll(m.CatalogSlug, "-", ""))
		if len(m.ShortCode) > 32 {
			m.ShortCode = m.ShortCode[:32]
		}
	}
	if m.ContentVersion <= 0 {
		m.ContentVersion = 1
	}
	return m, nil
}

// LoadCourseSpec loads manifest + English curriculum + syllabus for a course directory slug.
func LoadCourseSpec(courseDirSlug string) (*CourseSpec, error) {
	return LoadCourseSpecLocale(courseDirSlug, defaultLocale)
}

// LoadCourseSpecLocale loads a course for locale (falls back to English for curriculum).
func LoadCourseSpecLocale(courseDirSlug, locale string) (*CourseSpec, error) {
	manifest, err := LoadManifest(courseDirSlug)
	if err != nil {
		return nil, err
	}
	cur, err := LoadCurriculum(courseDirSlug, locale)
	if err != nil {
		return nil, err
	}
	syllabus, err := LoadSyllabus(courseDirSlug, locale)
	if err != nil {
		return nil, err
	}
	return &CourseSpec{
		Manifest: manifest,
		Locale:   cur.Locale,
		Modules:  cur.Modules,
		Syllabus: syllabus,
	}, nil
}

// LoadCurriculum parses embedded fixtures for a course locale (falls back to English).
func LoadCurriculum(courseDirSlug, locale string) (*Curriculum, error) {
	loc := strings.TrimSpace(locale)
	if loc == "" {
		loc = defaultLocale
	}
	root := path.Join("content", courseDirSlug, loc)
	if _, err := contentFS.ReadDir(root); err != nil {
		if loc != defaultLocale {
			return LoadCurriculum(courseDirSlug, defaultLocale)
		}
		return nil, fmt.Errorf("marketplace course %s: locale %q missing", courseDirSlug, loc)
	}
	modules, err := loadModules(root)
	if err != nil {
		return nil, err
	}
	return &Curriculum{CourseSlug: courseDirSlug, Locale: loc, Modules: modules}, nil
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
		Slug:      strings.TrimSpace(fields["slug"]),
		Title:     strings.TrimSpace(fields["title"]),
		SortOrder: atoiDefault(fields["sort_order"], -1),
		Dir:       dir,
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
		case strings.HasSuffix(name, ".json") && name != "syllabus.json":
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
		Slug:       strings.TrimSpace(fm["slug"]),
		Title:      strings.TrimSpace(fm["title"]),
		SortOrder:  atoiDefault(fm["sort_order"], -1),
		ContentVer: atoiDefault(fm["content_version"], 1),
		Markdown:   body,
		ModuleDir:  moduleDir,
		FilePath:   filePath,
	}
	if page.Slug == "" || page.Title == "" || page.SortOrder < 0 {
		return PageFixture{}, fmt.Errorf("%s: slug, title, and sort_order are required", filePath)
	}
	if page.Markdown == "" {
		return PageFixture{}, fmt.Errorf("%s: page body is empty", filePath)
	}
	if page.ContentVer <= 0 {
		page.ContentVer = 1
	}
	return page, nil
}

func assignmentFromFrontMatter(filePath, moduleDir string, fm map[string]string, body string) (AssignmentFixture, error) {
	assign := AssignmentFixture{
		Slug:       strings.TrimSpace(fm["slug"]),
		Title:      strings.TrimSpace(fm["title"]),
		SortOrder:  atoiDefault(fm["sort_order"], -1),
		ContentVer: atoiDefault(fm["content_version"], 1),
		Markdown:   body,
		Grading:    parseGradingFrontMatter(fm),
		ModuleDir:  moduleDir,
		FilePath:   filePath,
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
	if assign.ContentVer <= 0 {
		assign.ContentVer = 1
	}
	return assign, nil
}

func loadQuizFixture(filePath, moduleDir string) (QuizFixture, error) {
	b, err := contentFS.ReadFile(filePath)
	if err != nil {
		return QuizFixture{}, err
	}
	var raw struct {
		Slug               string                          `json:"slug"`
		Title              string                          `json:"title"`
		SortOrder          int                             `json:"sort_order"`
		Markdown           string                          `json:"markdown"`
		Points             int                             `json:"points"`
		Group              string                          `json:"group"`
		GradePolicy        string                          `json:"grade_policy"`
		MaxAttempts        int                             `json:"max_attempts"`
		GradeAttemptPolicy string                          `json:"grade_attempt_policy"`
		ContentVersion     int                             `json:"content_version"`
		Questions          []coursemodulequiz.QuizQuestion `json:"questions"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return QuizFixture{}, fmt.Errorf("%s: invalid JSON: %w", filePath, err)
	}
	q := QuizFixture{
		Slug:       strings.TrimSpace(raw.Slug),
		Title:      strings.TrimSpace(raw.Title),
		SortOrder:  raw.SortOrder,
		Markdown:   strings.TrimSpace(raw.Markdown),
		Questions:  raw.Questions,
		ContentVer: raw.ContentVersion,
		Grading: GradingConfig{
			Points:             raw.Points,
			Group:              strings.TrimSpace(raw.Group),
			GradePolicy:        strings.TrimSpace(raw.GradePolicy),
			MaxAttempts:        raw.MaxAttempts,
			GradeAttemptPolicy: strings.TrimSpace(raw.GradeAttemptPolicy),
		},
		ModuleDir: moduleDir,
		FilePath:  filePath,
	}
	if q.Slug == "" || q.Title == "" || q.SortOrder < 0 {
		return QuizFixture{}, fmt.Errorf("%s: slug, title, and sort_order are required", filePath)
	}
	if q.ContentVer <= 0 {
		q.ContentVer = 1
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

func parseGradingFrontMatter(fm map[string]string) GradingConfig {
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

// parseCourseYAML parses course.yaml including list fields (outcomes).
func parseCourseYAML(src string) (CourseManifest, error) {
	m := CourseManifest{
		MarketplaceListed: true, // default; may be overridden
		PriceCents:        -1,   // sentinel: unset
	}
	lines := strings.Split(src, "\n")
	var inOutcomes bool
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if inOutcomes {
			if strings.HasPrefix(trimmed, "- ") {
				m.Outcomes = append(m.Outcomes, unquoteYAML(strings.TrimSpace(trimmed[2:])))
				continue
			}
			// Non-list line ends the outcomes block; reprocess.
			inOutcomes = false
		}
		key, val, ok := strings.Cut(trimmed, ":")
		if !ok {
			return CourseManifest{}, fmt.Errorf("invalid YAML line %q", trimmed)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "code":
			m.Code = unquoteYAML(val)
		case "title":
			m.Title = unquoteYAML(val)
		case "catalog_slug":
			m.CatalogSlug = unquoteYAML(val)
		case "catalog_category":
			m.CatalogCategory = unquoteYAML(val)
		case "difficulty_level":
			m.DifficultyLevel = unquoteYAML(val)
		case "catalog_language":
			m.CatalogLanguage = unquoteYAML(val)
		case "summary":
			m.Summary = unquoteYAML(val)
		case "estimated_minutes":
			m.EstimatedMinutes = atoiDefault(val, 0)
		case "price_cents":
			m.PriceCents = atoiDefault(val, 0)
		case "is_public":
			m.IsPublic = parseBoolYAML(val)
		case "marketplace_listed":
			m.MarketplaceListed = parseBoolYAML(val)
		case "hero_image":
			m.HeroImage = unquoteYAML(val)
		case "content_version":
			m.ContentVersion = atoiDefault(val, 1)
		case "short_code":
			m.ShortCode = unquoteYAML(val)
		case "outcomes":
			if val == "" || val == "[]" {
				inOutcomes = true
			} else {
				return CourseManifest{}, fmt.Errorf("outcomes must be a YAML list")
			}
		default:
			// ignore unknown keys for forward compatibility
		}
	}
	if m.PriceCents < 0 {
		m.PriceCents = 0
	}
	return m, nil
}

func parseBoolYAML(s string) bool {
	switch strings.ToLower(strings.TrimSpace(unquoteYAML(s))) {
	case "true", "yes", "1", "on":
		return true
	default:
		return false
	}
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

// AllDesiredSlugs returns slugs that would be synced for a course spec.
func AllDesiredSlugs(spec *CourseSpec) []string {
	if spec == nil {
		return nil
	}
	var slugs []string
	for _, mod := range spec.Modules {
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

// ResolveCourseDir finds the content directory for a catalog slug or directory name.
func ResolveCourseDir(slugOrDir string) (string, error) {
	want := strings.TrimSpace(slugOrDir)
	if want == "" {
		return "", fmt.Errorf("course slug is required")
	}
	slugs, err := ListCourseSlugs()
	if err != nil {
		return "", err
	}
	for _, dir := range slugs {
		if dir == want {
			return dir, nil
		}
		m, err := LoadManifest(dir)
		if err != nil {
			continue
		}
		if m.CatalogSlug == want || m.Code == want || m.ShortCode == want {
			return dir, nil
		}
	}
	return "", fmt.Errorf("unknown marketplace course %q", want)
}
