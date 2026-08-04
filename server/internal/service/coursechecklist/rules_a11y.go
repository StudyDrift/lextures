package coursechecklist

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

func a11yRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleA11yImageAltText(),
		ruleA11yVideoCaptions(),
		ruleA11yHeadingStructure(),
		ruleA11yLinkText(),
		ruleA11yTableHeaders(),
		ruleA11yTablesForLayout(),
		ruleA11yColorContrast(),
		ruleA11yTextFormatting(),
		ruleA11yDocumentAccessibility(),
		ruleA11yMediaAlternatives(),
		ruleA11yEnforcementSettings(),
	}
}

func ruleA11yImageAltText() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yImageAltText,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.image-alt-text.title",
		TitleDefault: "Add alt text to images",
		WhyKey:       "coursechecklist.item.a11y.image-alt-text.why",
		WhyDefault:   a11yWhy("Every image needs a short description or a decorative mark so screen readers can skip or announce it."),
		HelpRef:      "course-checklist#a11y-image-alt-text",
		Tier:         TierRecommended,
		Sources:      []string{"WCAG 1.1.1", "QM 8.2", "OSCQR 36"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedSyllabus},
		Evaluate:     evalA11yImageAltText,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: a11yEvidenceColumns},
	}
}

func evalA11yImageAltText(_ context.Context, snap CourseSnapshot) (Finding, error) {
	doc := contentDocFor(snap)
	var evidence []EvidenceRow
	total, withAlt := 0, 0
	for _, p := range doc.Pages {
		for _, img := range p.Images {
			if img.Decorative {
				continue
			}
			total++
			if img.HasValidAlt {
				withAlt++
				continue
			}
			src := img.Src
			if len(src) > 48 {
				src = src[:45] + "…"
			}
			evidence = append(evidence, EvidenceRow{
				Label:    pageLabel(p),
				Sublabel: fmt.Sprintf("%s · line %d", src, img.Line),
				Status:   StatusTodo,
				TargetOverride: &NavTarget{
					Surface: "web", Route: p.Route, Anchor: "content.image-alt",
					EntityKey: p.ItemID.String(),
				},
			})
		}
	}
	if total == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.a11y.image-alt-text.detail.none",
			DetailDefault: "No images need alt text in authored content.",
		}, nil
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			Progress:      &Progress{Done: withAlt, Total: total},
			DetailKey:     "coursechecklist.item.a11y.image-alt-text.detail.done",
			DetailDefault: fmt.Sprintf("All %d images have alt text or are decorative.", total),
		}, nil
	}
	st := StatusInProgress
	if withAlt == 0 {
		st = StatusTodo
	}
	return Finding{
		Status:   st,
		Progress: &Progress{Done: withAlt, Total: total},
		Evidence: evidence,
		DetailKey: "coursechecklist.item.a11y.image-alt-text.detail.progress",
		DetailDefault: fmt.Sprintf("%d of %d images have alt text.", withAlt, total),
		DetailFields:  map[string]any{"done": withAlt, "total": total},
	}, nil
}

func ruleA11yVideoCaptions() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yVideoCaptions,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.video-captions.title",
		TitleDefault: "Add captions or transcripts to media",
		WhyKey:       "coursechecklist.item.a11y.video-captions.why",
		WhyDefault:   a11yWhy("Time-based media needs captions or a transcript so learners can follow along without audio."),
		HelpRef:      "course-checklist#a11y-video-captions",
		Tier:         TierRecommended,
		Sources:      []string{"WCAG 1.2.2", "QM 8.3", "OSCQR 35"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedSyllabus, DataNeedCourse},
		Evaluate:     evalA11yVideoCaptions,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/settings/accessibility",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Page", "Media", "Location"}},
	}
}

func evalA11yVideoCaptions(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.Features.RequireCaptions {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.a11y.video-captions.detail.require",
			DetailDefault: "Course requires captions for media.",
		}, nil
	}
	doc := contentDocFor(snap)
	var evidence []EvidenceRow
	total := 0
	for _, p := range doc.Pages {
		for _, m := range p.Media {
			total++
			if m.HasCaptionsOrTranscript {
				continue
			}
			evidence = append(evidence, EvidenceRow{
				Label:    pageLabel(p),
				Sublabel: m.Kind,
				Status:   StatusTodo,
				TargetOverride: &NavTarget{
					Surface: "web", Route: p.Route, EntityKey: p.ItemID.String(),
				},
			})
		}
	}
	if total == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.a11y.video-captions.detail.na",
			DetailDefault: "This course embeds no time-based media.",
		}, nil
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.a11y.video-captions.detail.done",
			DetailDefault: "Embedded media has captions or transcripts.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		Evidence:      evidence,
		DetailKey:     "coursechecklist.item.a11y.video-captions.detail.todo",
		DetailDefault: fmt.Sprintf("%d media embeds need captions or a transcript.", len(evidence)),
	}, nil
}

func ruleA11yHeadingStructure() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yHeadingStructure,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.heading-structure.title",
		TitleDefault: "Fix heading structure on pages",
		WhyKey:       "coursechecklist.item.a11y.heading-structure.why",
		WhyDefault:   a11yWhy("Headings should start at H2 and never skip levels so assistive tech can navigate the page."),
		HelpRef:      "course-checklist#a11y-heading-structure",
		Tier:         TierRecommended,
		Sources:      []string{"WCAG 1.3.1", "WCAG 2.4.6", "OSCQR 21"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedSyllabus},
		Evaluate:     evalA11yHeadingStructure,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Page", "Heading", "Issue"}},
	}
}

func evalA11yHeadingStructure(_ context.Context, snap CourseSnapshot) (Finding, error) {
	doc := contentDocFor(snap)
	var evidence []EvidenceRow
	for _, p := range doc.Pages {
		if issue, heading := firstHeadingIssue(p.Headings); issue != "" {
			evidence = append(evidence, EvidenceRow{
				Label:    pageLabel(p),
				Sublabel: fmt.Sprintf("%s · %s", heading, issue),
				Status:   StatusTodo,
				TargetOverride: &NavTarget{
					Surface: "web", Route: p.Route, EntityKey: p.ItemID.String(),
				},
			})
		}
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.a11y.heading-structure.detail.done",
			DetailDefault: "Heading levels look sequential.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		Evidence:      evidence,
		DetailKey:     "coursechecklist.item.a11y.heading-structure.detail.todo",
		DetailDefault: fmt.Sprintf("%d pages need heading fixes.", len(evidence)),
	}, nil
}

func firstHeadingIssue(heads []ContentHeading) (issue, heading string) {
	if len(heads) == 0 {
		return "", ""
	}
	prev := 0
	for _, h := range heads {
		if h.BoldAsHeading {
			return "bold used as heading", h.Text
		}
		if h.Level == 1 {
			return "starts at H1 (use H2+)", h.Text
		}
		if prev == 0 {
			if h.Level > 2 {
				return "starts below H2", h.Text
			}
			prev = h.Level
			continue
		}
		if h.Level > prev+1 {
			return fmt.Sprintf("skips from H%d to H%d", prev, h.Level), h.Text
		}
		prev = h.Level
	}
	return "", ""
}

const a11yAutomatedDisclaimer = "This automated check is partial and does not certify WCAG conformance; see docs/accessibility/course-checklist-scope.md for what still needs manual testing."

var a11yEvidenceColumns = []string{"Page", "Image", "Location"}

var hexColorRE = regexp.MustCompile(`(?i)^#?([0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})$`)

func a11yWhy(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return a11yAutomatedDisclaimer
	}
	if strings.Contains(base, "automated") || strings.Contains(base, "partial") {
		return base
	}
	return base + " " + a11yAutomatedDisclaimer
}

// relativeLuminance implements WCAG 2.x relative luminance for sRGB.
func relativeLuminance(r, g, b float64) float64 {
	lin := func(c float64) float64 {
		c /= 255
		if c <= 0.04045 {
			return c / 12.92
		}
		return math.Pow((c+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(r) + 0.7152*lin(g) + 0.0722*lin(b)
}

// contrastRatio returns (lighter+0.05)/(darker+0.05).
func contrastRatio(l1, l2 float64) float64 {
	lighter, darker := l1, l2
	if l2 > l1 {
		lighter, darker = l2, l1
	}
	return (lighter + 0.05) / (darker + 0.05)
}

func parseHexColor(s string) (r, g, b float64, ok bool) {
	s = strings.TrimSpace(s)
	m := hexColorRE.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, 0, false
	}
	h := m[1]
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) == 8 {
		h = h[:6]
	}
	ri, err1 := strconv.ParseUint(h[0:2], 16, 8)
	gi, err2 := strconv.ParseUint(h[2:4], 16, 8)
	bi, err3 := strconv.ParseUint(h[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return float64(ri), float64(gi), float64(bi), true
}

type themeColors struct {
	BodyColor    string
	HeadingColor string
	LinkColor    string
	CodeBg       string
	Background   string // assumed white when unset
}

func parseThemeCustom(raw json.RawMessage) themeColors {
	tc := themeColors{Background: "#ffffff"}
	if len(raw) == 0 {
		return tc
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return tc
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
		return ""
	}
	tc.BodyColor = get("bodyColor", "body_color")
	tc.HeadingColor = get("headingColor", "heading_color")
	tc.LinkColor = get("linkColor", "link_color")
	tc.CodeBg = get("codeBackground", "code_background")
	return tc
}

type contrastFailure struct {
	Pair  string
	Ratio float64
	Need  float64
}

func themeContrastFailures(raw json.RawMessage) []contrastFailure {
	tc := parseThemeCustom(raw)
	bgR, bgG, bgB, ok := parseHexColor(tc.Background)
	if !ok {
		bgR, bgG, bgB = 255, 255, 255
	}
	bgLum := relativeLuminance(bgR, bgG, bgB)
	var fails []contrastFailure
	check := func(name, color string, need float64) {
		if strings.TrimSpace(color) == "" {
			return
		}
		r, g, b, ok := parseHexColor(color)
		if !ok {
			return
		}
		ratio := contrastRatio(relativeLuminance(r, g, b), bgLum)
		if ratio+0.001 < need {
			fails = append(fails, contrastFailure{Pair: name, Ratio: ratio, Need: need})
		}
	}
	check("body/background", tc.BodyColor, 4.5)
	check("heading/background", tc.HeadingColor, 3.0) // large text
	check("link/background", tc.LinkColor, 4.5)
	return fails
}

func formatContrastDetail(f contrastFailure) string {
	return fmt.Sprintf("%s %.1f:1 (needs %.1f:1)", f.Pair, f.Ratio, f.Need)
}

// gradeBandMaxFKGL maps course grade-level tokens to an approximate FKGL ceiling.
func gradeBandMaxFKGL(levels []string) (max float64, ok bool) {
	if len(levels) == 0 {
		return 0, false
	}
	highest := -1.0
	for _, lv := range levels {
		lv = strings.ToLower(strings.TrimSpace(lv))
		var g float64
		switch {
		case lv == "k" || lv == "kindergarten":
			g = 1
		case strings.HasPrefix(lv, "grade"):
			n := strings.TrimSpace(strings.TrimPrefix(lv, "grade"))
			n = strings.TrimPrefix(n, "-")
			n = strings.TrimPrefix(n, "_")
			n = strings.TrimSpace(n)
			v, err := strconv.Atoi(n)
			if err != nil {
				continue
			}
			g = float64(v)
		default:
			v, err := strconv.Atoi(lv)
			if err != nil {
				continue
			}
			g = float64(v)
		}
		if g > highest {
			highest = g
		}
	}
	if highest < 0 {
		return 0, false
	}
	return highest, true
}

func pageLabel(p ContentPage) string {
	if p.ModuleTitle != "" {
		return p.ModuleTitle + " · " + p.Title
	}
	return p.Title
}

func ruleA11yLinkText() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yLinkText,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.link-text.title",
		TitleDefault: "Rewrite vague or bare link text",
		WhyKey:       "coursechecklist.item.a11y.link-text.why",
		WhyDefault:   a11yWhy("Link text should describe the destination — avoid “click here” and long bare URLs."),
		HelpRef:      "course-checklist#a11y-link-text",
		Tier:         TierRecommended,
		Sources:      []string{"WCAG 2.4.4", "OSCQR 37"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedSyllabus},
		Evaluate:     evalA11yLinkText,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Page", "Link text", "Location"}},
	}
}

func evalA11yLinkText(_ context.Context, snap CourseSnapshot) (Finding, error) {
	doc := contentDocFor(snap)
	lex := lexiconForSnap(snap)
	var evidence []EvidenceRow
	for _, p := range doc.Pages {
		for _, link := range p.Links {
			if vagueLinkText(link.Text, lex) {
				evidence = append(evidence, EvidenceRow{
					Label:    pageLabel(p),
					Sublabel: truncateRunes(link.Text, 60),
					Status:   StatusTodo,
					TargetOverride: &NavTarget{
						Surface: "web", Route: p.Route, EntityKey: p.ItemID.String(),
					},
				})
			}
		}
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.a11y.link-text.detail.done",
			DetailDefault: "Link text looks descriptive.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		Evidence:      evidence,
		DetailKey:     "coursechecklist.item.a11y.link-text.detail.todo",
		DetailDefault: fmt.Sprintf("%d links need clearer text.", len(evidence)),
	}, nil
}

func vagueLinkText(text string, lex *Lexicon) bool {
	t := strings.TrimSpace(strings.ToLower(text))
	if t == "" {
		return true
	}
	if lex != nil && lex.VagueLinkText != nil && lex.VagueLinkText.Match(t) {
		return true
	}
	// Fallback English defaults when lexicon field absent.
	for _, bad := range []string{"click here", "read more", "link", "here", "more"} {
		if t == bad {
			return true
		}
	}
	if (strings.HasPrefix(t, "http://") || strings.HasPrefix(t, "https://")) && len(t) > 40 {
		return true
	}
	return false
}

func ruleA11yTableHeaders() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yTableHeaders,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.table-headers.title",
		TitleDefault: "Add headers to data tables",
		WhyKey:       "coursechecklist.item.a11y.table-headers.why",
		WhyDefault:   a11yWhy("Data tables need a header row or column so screen readers can associate cells."),
		HelpRef:      "course-checklist#a11y-table-headers",
		Tier:         TierRecommended,
		Sources:      []string{"WCAG 1.3.1", "OSCQR 25", "OSCQR 26"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedSyllabus},
		Evaluate:     evalA11yTableHeaders,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"},
		EvidenceShape: &EvidenceShape{Columns: []string{"Page", "Table", "Issue"}},
	}
}

func evalA11yTableHeaders(_ context.Context, snap CourseSnapshot) (Finding, error) {
	doc := contentDocFor(snap)
	var evidence []EvidenceRow
	for _, p := range doc.Pages {
		for i, tbl := range p.Tables {
			if tbl.LayoutOnly {
				continue
			}
			if !tbl.HasHeader {
				evidence = append(evidence, EvidenceRow{
					Label:    pageLabel(p),
					Sublabel: fmt.Sprintf("table %d", i+1),
					Status:   StatusTodo,
					TargetOverride: &NavTarget{
						Surface: "web", Route: p.Route, EntityKey: p.ItemID.String(),
					},
				})
			}
		}
	}
	if len(evidence) == 0 {
		return Finding{Status: StatusDone, DetailDefault: "Data tables have headers."}, nil
	}
	return Finding{Status: StatusTodo, Evidence: evidence, DetailDefault: fmt.Sprintf("%d tables lack headers.", len(evidence))}, nil
}

func ruleA11yTablesForLayout() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yTablesForLayout,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.tables-for-layout.title",
		TitleDefault: "Replace layout tables with structured content",
		WhyKey:       "coursechecklist.item.a11y.tables-for-layout.why",
		WhyDefault:   a11yWhy("Single-row or single-column tables without headers are usually layout — prefer headings and lists."),
		HelpRef:      "course-checklist#a11y-tables-for-layout",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 24"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedSyllabus},
		Evaluate:     evalA11yTablesForLayout,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"},
		EvidenceShape: &EvidenceShape{Columns: []string{"Page", "Table", "Issue"}},
	}
}

func evalA11yTablesForLayout(_ context.Context, snap CourseSnapshot) (Finding, error) {
	doc := contentDocFor(snap)
	var evidence []EvidenceRow
	for _, p := range doc.Pages {
		for i, tbl := range p.Tables {
			if !tbl.LayoutOnly {
				continue
			}
			evidence = append(evidence, EvidenceRow{
				Label:    pageLabel(p),
				Sublabel: fmt.Sprintf("table %d · layout", i+1),
				Status:   StatusTodo,
				TargetOverride: &NavTarget{
					Surface: "web", Route: p.Route, EntityKey: p.ItemID.String(),
				},
			})
		}
	}
	if len(evidence) == 0 {
		return Finding{Status: StatusDone, DetailDefault: "No layout-only tables found."}, nil
	}
	return Finding{Status: StatusTodo, Evidence: evidence, DetailDefault: fmt.Sprintf("%d layout tables found.", len(evidence))}, nil
}

