package starterlibrary

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	maxModules        = 4
	maxPagesPerModule = 3
	maxTitleRunes     = 120
	maxMarkdownRunes  = 4_000
	maxFocusRunes     = 280
)

// OutlineSystemPrompt asks the model for a short editable starter course.
const OutlineSystemPrompt = `You write a short starter course for a homeschool family.
Respond with ONLY valid JSON (no markdown fences).

Shape:
{"title":"","summary":"","estimatedMinutes":45,"standards":["CODE"],"modules":[{"title":"","pages":[{"title":"","markdown":""}]}]}

Rules:
- 2 modules, 2 pages each, unless the topic truly needs 3 modules.
- markdown is learner-facing Markdown (paragraphs and lists). No headings inside markdown.
- Match the grade band's reading level. K-2 uses short sentences.
- estimatedMinutes is between 30 and 90.
- standards may be an empty array when you are not sure of a code.
- The family will edit the result, so teach one clear idea per page.`

// TemplateCourse clones the curated starter for a band and subject and weaves in an optional focus.
func TemplateCourse(band, subject, focus string) (Course, error) {
	band = strings.TrimSpace(band)
	subject = strings.TrimSpace(subject)
	if !ValidBand(band) {
		return Course{}, fmt.Errorf("grade band must be one of %s", strings.Join(GradeBands, ", "))
	}
	if !ValidSubject(subject) {
		return Course{}, fmt.Errorf("subject must be one of math, ela, science, history, elective")
	}
	focus = trimRunes(strings.TrimSpace(focus), maxFocusRunes)
	var found Course
	ok := false
	for _, c := range catalog() {
		if c.GradeBand != band || c.Subject != subject {
			continue
		}
		found = c
		ok = true
		break
	}
	if !ok {
		return Course{}, fmt.Errorf("no starter for %s %s", band, subject)
	}
	return ApplyFocus(found, focus), nil
}

// ApplyFocus appends a family focus line to the summary and the first page.
func ApplyFocus(c Course, focus string) Course {
	focus = trimRunes(strings.TrimSpace(focus), maxFocusRunes)
	if focus == "" {
		return c
	}
	c.Summary = strings.TrimSpace(c.Summary + " Family focus: " + focus)
	if len(c.Modules) == 0 || len(c.Modules[0].Pages) == 0 {
		return c
	}
	c.Modules[0].Pages[0].Markdown = strings.TrimSpace(c.Modules[0].Pages[0].Markdown) +
		"\n\n**Your focus**\n\n" + focus
	return c
}

type aiOutline struct {
	Title            string   `json:"title"`
	Summary          string   `json:"summary"`
	EstimatedMinutes int      `json:"estimatedMinutes"`
	Standards        []string `json:"standards"`
	Modules          []struct {
		Title string `json:"title"`
		Pages []struct {
			Title    string `json:"title"`
			Markdown string `json:"markdown"`
		} `json:"pages"`
	} `json:"modules"`
}

// ParseOutline turns model JSON into a course for the requested band and subject.
func ParseOutline(raw, band, subject string) (Course, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	var parsed aiOutline
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return Course{}, fmt.Errorf("parse starter outline: %w", err)
	}
	title := trimRunes(strings.TrimSpace(parsed.Title), maxTitleRunes)
	if title == "" {
		return Course{}, fmt.Errorf("outline title is required")
	}
	if len(parsed.Modules) == 0 || len(parsed.Modules) > maxModules {
		return Course{}, fmt.Errorf("outline must include 1 to %d modules", maxModules)
	}
	minutes := parsed.EstimatedMinutes
	if minutes < 20 || minutes > 120 {
		minutes = 45
	}
	course := Course{
		Title:            title,
		Summary:          trimRunes(strings.TrimSpace(parsed.Summary), 500),
		GradeBand:        band,
		Subject:          subject,
		EstimatedMinutes: minutes,
		Standards:        cleanStandards(parsed.Standards),
	}
	if course.Summary == "" {
		course.Summary = "A starter course you can edit."
	}
	for _, mod := range parsed.Modules {
		mt := trimRunes(strings.TrimSpace(mod.Title), maxTitleRunes)
		if mt == "" || len(mod.Pages) == 0 || len(mod.Pages) > maxPagesPerModule {
			return Course{}, fmt.Errorf("each module needs a title and 1 to %d pages", maxPagesPerModule)
		}
		pages := make([]Page, 0, len(mod.Pages))
		for _, p := range mod.Pages {
			pt := trimRunes(strings.TrimSpace(p.Title), maxTitleRunes)
			body := trimRunes(strings.TrimSpace(p.Markdown), maxMarkdownRunes)
			if pt == "" || body == "" {
				return Course{}, fmt.Errorf("each page needs a title and markdown")
			}
			pages = append(pages, Page{Title: pt, Markdown: body})
		}
		course.Modules = append(course.Modules, Module{Title: mt, Pages: pages})
	}
	return course, nil
}

func cleanStandards(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = trimRunes(strings.TrimSpace(s), 80)
		if s == "" {
			continue
		}
		out = append(out, s)
		if len(out) == 6 {
			break
		}
	}
	return out
}

func trimRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}
