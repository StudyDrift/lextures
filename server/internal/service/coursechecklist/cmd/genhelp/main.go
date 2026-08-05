package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lextures/lextures/server/internal/service/coursechecklist"
)

type helpItem struct {
	ItemID        string   `json:"itemId"`
	HelpRef       string   `json:"helpRef"`
	Title         string   `json:"title"`
	What          string   `json:"what"`
	Why           string   `json:"why"`
	How           string   `json:"how"`
	WhenToDismiss string   `json:"whenToDismiss"`
	Sources       []string `json:"sources"`
}

type helpCatalog struct {
	Version int                 `json:"version"`
	Comment string              `json:"comment"`
	Items   map[string]helpItem `json:"items"`
}

func howFor(id, title string) string {
	switch {
	case strings.HasPrefix(id, "course."):
		return "Open Course settings and complete the field or control this check names. Save, then re-check the item from the checklist."
	case strings.HasPrefix(id, "orientation."):
		return "Add the orientation content (welcome post, Start Here page, syllabus block, or course feed) so learners see it on day one. Use the checklist target link to jump to the control."
	case strings.HasPrefix(id, "syllabus."):
		return "Edit the syllabus and include the missing policy or block. Keep a printable/exportable version when required."
	case strings.HasPrefix(id, "people."):
		return "Use People / enrollments / sections settings to fix the listed rows, or dismiss if the course intentionally has no students yet."
	case strings.HasPrefix(id, "structure."):
		return "Open Modules and fix the modules or items listed in the evidence table (empty modules, placeholders, unpublished items, etc.)."
	case strings.HasPrefix(id, "outcomes."):
		return "Define or edit learning outcomes under Course settings → Outcomes, then map assessments and modules as needed. Use assisted mapping when offered."
	case strings.HasPrefix(id, "assessment."), strings.HasPrefix(id, "grading."):
		return "Open the assignment or quiz (or grading settings) linked from evidence and set dates, points, groups, or policies as required."
	case strings.HasPrefix(id, "feedback."):
		return "Add rubrics, written criteria, review settings, or formative checks on the listed items so learners know how they will be evaluated."
	case strings.HasPrefix(id, "integrity."):
		return "Review high-stakes assessment integrity settings and course AI policy so they match institutional expectations."
	case strings.HasPrefix(id, "accommodations."):
		return "Review accommodation workflows with your accessibility office. This check reports process readiness only — never student details."
	case strings.HasPrefix(id, "interaction."):
		return "Enable discussions, office hours, groups, or presence plans so learners have structured ways to engage."
	case strings.HasPrefix(id, "a11y."), strings.HasPrefix(id, "udl."):
		return "Fix the listed content using the editor accessibility tools (alt text, headings, captions, contrast). Accessibility rules are machine-checkable heuristics, not a full WCAG audit."
	case strings.HasPrefix(id, "links."):
		return "Replace or remove dead external links listed in evidence. Re-check after publishing fixes."
	case strings.HasPrefix(id, "launch."):
		return "Complete the pre-term readiness step (preview as student, clear drafts after start, calendar sanity, or backup export)."
	default:
		return fmt.Sprintf("Follow the checklist target for “%s” and re-check when finished.", title)
	}
}

func dismissFor(id string) string {
	switch {
	case strings.HasPrefix(id, "people."):
		return "Dismiss when the course is a template, not yet open for enrollment, or deliberately staff-only. Prefer “later” if enrollments are imminent."
	case strings.HasPrefix(id, "accommodations."):
		return "Dismiss only if your institution handles accommodations entirely outside this LMS and you have confirmed process coverage."
	case strings.HasPrefix(id, "a11y."):
		return "Dismiss only when the flagged content is non-instructional decorative media or when a human accessibility review supersedes this heuristic — never to skip genuine access barriers."
	case strings.HasPrefix(id, "outcomes."):
		return "Dismiss when outcomes live in an external system of record and this course only mirrors grades (document where mappings live in the note)."
	case id == "course.hero-image" || id == "course.language":
		return "Dismiss if institutional branding or language defaults already cover this course."
	default:
		return "Dismiss when the check truly does not apply to this course design, the work is done outside the LMS, or you disagree with the heuristic — pick the matching reason and add a short note for co-teachers."
	}
}

func whatFor(title string) string {
	return fmt.Sprintf("This check looks at whether the course satisfies: %s. It evaluates structured course data (settings, modules, syllabus, outcomes, assessments) using a deterministic rule — not an AI judgment.", title)
}

func main() {
	reg, err := coursechecklist.BuildBuiltinRegistry()
	if err != nil {
		panic(err)
	}
	items := map[string]helpItem{}
	for _, it := range reg.List() {
		ref := strings.TrimSpace(it.HelpRef)
		if ref == "" {
			continue
		}
		items[ref] = helpItem{
			ItemID:        string(it.ID),
			HelpRef:       ref,
			Title:         it.TitleDefault,
			What:          whatFor(it.TitleDefault),
			Why:           it.WhyDefault + " Sources: " + strings.Join(it.Sources, ", ") + ".",
			How:           howFor(string(it.ID), it.TitleDefault),
			WhenToDismiss: dismissFor(string(it.ID)),
			Sources:       append([]string(nil), it.Sources...),
		}
	}
	keys := make([]string, 0, len(items))
	for k := range items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ordered := make(map[string]helpItem, len(items))
	for _, k := range keys {
		ordered[k] = items[k]
	}
	cat := helpCatalog{
		Version: 1,
		Comment: "CC.10 per-item help catalog. HelpRef = course-checklist#<slug>. Support URL: /help/course-checklist#<slug>",
		Items:   ordered,
	}
	payload, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		panic(err)
	}
	payload = append(payload, '\n')

	// Resolve repo root from this file's package path convention: server/internal/...
	wd, _ := os.Getwd()
	// When run via `go run ./internal/service/coursechecklist/cmd/genhelp` from server/
	root := filepath.Clean(filepath.Join(wd, ".."))
	if filepath.Base(wd) == "server" {
		root = filepath.Clean(filepath.Join(wd, ".."))
	}
	// Prefer walking up until we find docs/
	for d := wd; d != "/" && d != "."; d = filepath.Dir(d) {
		if st, err := os.Stat(filepath.Join(d, "docs")); err == nil && st.IsDir() {
			root = d
			break
		}
		if st, err := os.Stat(filepath.Join(d, "server")); err == nil && st.IsDir() {
			if st2, err2 := os.Stat(filepath.Join(d, "docs")); err2 == nil && st2.IsDir() {
				root = d
				break
			}
		}
	}

	helpDir := filepath.Join(root, "docs", "help", "course-checklist")
	if err := os.MkdirAll(helpDir, 0o755); err != nil {
		panic(err)
	}
	path := filepath.Join(helpDir, "items.json")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		panic(err)
	}
	clientPath := filepath.Join(root, "clients", "packages", "checklist-help.json")
	if err := os.MkdirAll(filepath.Dir(clientPath), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(clientPath, payload, 0o644); err != nil {
		panic(err)
	}
	webPath := filepath.Join(root, "clients", "web", "src", "lib", "checklist-help-data.json")
	if err := os.MkdirAll(filepath.Dir(webPath), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(webPath, payload, 0o644); err != nil {
		panic(err)
	}

	// Keep the in-app research page content in sync with product help (CC.10 FR-4).
	// Source lives under docs/help (not docs/plan) so plan cleanup cannot break the app.
	researchSrc := filepath.Join(root, "docs", "help", "course-checklist", "research.md")
	researchDst := filepath.Join(root, "clients", "web", "src", "content", "checklist", "research.md")
	if raw, err := os.ReadFile(researchSrc); err != nil {
		fmt.Printf("warn: could not sync checklist research.md: %v\n", err)
	} else {
		if err := os.MkdirAll(filepath.Dir(researchDst), 0o755); err != nil {
			panic(err)
		}
		if err := os.WriteFile(researchDst, raw, 0o644); err != nil {
			panic(err)
		}
	}

	fmt.Printf("wrote %d help items\n  %s\n  %s\n  %s\n  %s\n", len(items), path, clientPath, webPath, researchDst)
}
