package coursechecklist

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/service/readinglevel"
)

func ruleA11yColorContrast() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yColorContrast,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.color-contrast.title",
		TitleDefault: "Fix custom theme contrast",
		WhyKey:       "coursechecklist.item.a11y.color-contrast.why",
		WhyDefault:   a11yWhy("Custom theme colors must meet WCAG contrast so text stays readable."),
		HelpRef:      "course-checklist#a11y-color-contrast",
		Tier:         TierRecommended,
		Sources:      []string{"WCAG 1.4.3", "OSCQR 18"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Applies: func(snap CourseSnapshot) bool {
			return strings.EqualFold(snap.MarkdownThemePreset, "custom") && len(snap.MarkdownThemeCustom) > 0
		},
		Evaluate: evalA11yColorContrast,
		Target:   NavTarget{Surface: "web", Route: "/courses/{courseCode}/settings/general"},
	}
}

func evalA11yColorContrast(_ context.Context, snap CourseSnapshot) (Finding, error) {
	fails := themeContrastFailures(snap.MarkdownThemeCustom)
	if len(fails) == 0 {
		return Finding{Status: StatusDone, DetailDefault: "Custom theme contrast meets WCAG thresholds."}, nil
	}
	detail := formatContrastDetail(fails[0])
	// Prefer the body pair wording from AC-5 when present.
	for _, f := range fails {
		if strings.HasPrefix(f.Pair, "body") {
			detail = fmt.Sprintf("%.1f:1 (needs %.1f:1)", f.Ratio, f.Need)
			break
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.a11y.color-contrast.detail.todo",
		DetailDefault: detail,
		DetailFields:  map[string]any{"failures": len(fails)},
	}, nil
}

func ruleA11yTextFormatting() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yTextFormatting,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.text-formatting.title",
		TitleDefault: "Remove blinking text and all-caps blocks",
		WhyKey:       "coursechecklist.item.a11y.text-formatting.why",
		WhyDefault:   a11yWhy("Blinking/marquee and long all-caps blocks make content harder to read."),
		HelpRef:      "course-checklist#a11y-text-formatting",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 22", "OSCQR 23"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedSyllabus},
		Evaluate:     evalA11yTextFormatting,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"},
		EvidenceShape: &EvidenceShape{Columns: []string{"Page", "Issue", "Location"}},
	}
}

func evalA11yTextFormatting(_ context.Context, snap CourseSnapshot) (Finding, error) {
	doc := contentDocFor(snap)
	var evidence []EvidenceRow
	for _, p := range doc.Pages {
		if p.HasBlinking {
			evidence = append(evidence, EvidenceRow{
				Label: pageLabel(p), Sublabel: "blinking/marquee", Status: StatusTodo,
				TargetOverride: &NavTarget{Surface: "web", Route: p.Route, EntityKey: p.ItemID.String()},
			})
		}
		for _, block := range p.AllCaps {
			evidence = append(evidence, EvidenceRow{
				Label: pageLabel(p), Sublabel: "all-caps block · " + truncateRunes(block, 40), Status: StatusTodo,
				TargetOverride: &NavTarget{Surface: "web", Route: p.Route, EntityKey: p.ItemID.String()},
			})
		}
	}
	if len(evidence) == 0 {
		return Finding{Status: StatusDone, DetailDefault: "No blinking or long all-caps blocks found."}, nil
	}
	return Finding{Status: StatusTodo, Evidence: evidence, DetailDefault: fmt.Sprintf("%d formatting issues found.", len(evidence))}, nil
}

func ruleA11yDocumentAccessibility() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yDocumentAccessibility,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.document-accessibility.title",
		TitleDefault: "Replace image-only PDFs",
		WhyKey:       "coursechecklist.item.a11y.document-accessibility.why",
		WhyDefault:   a11yWhy("PDFs without a text layer need a text alternative so they can be read aloud."),
		HelpRef:      "course-checklist#a11y-document-accessibility",
		Tier:         TierRecommended,
		Sources:      []string{"WCAG 1.1.1", "OSCQR 34"},
		DataNeeds:    []DataNeed{DataNeedFiles, DataNeedStructure, DataNeedItemMeta},
		Evaluate:     evalA11yDocumentAccessibility,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}/files"},
		EvidenceShape: &EvidenceShape{Columns: []string{"File", "Issue", "Hint"}},
	}
}

func evalA11yDocumentAccessibility(_ context.Context, snap CourseSnapshot) (Finding, error) {
	var evidence []EvidenceRow
	unknown := 0
	referenced := referencedFileIDs(snap)
	for _, f := range snap.Files {
		if !strings.Contains(strings.ToLower(f.ContentType), "pdf") &&
			!strings.HasSuffix(strings.ToLower(f.DisplayName), ".pdf") {
			continue
		}
		if len(referenced) > 0 {
			if _, ok := referenced[f.ID]; !ok {
				continue
			}
		}
		switch f.TextLayer {
		case "has_text":
			continue
		case "image_only":
			evidence = append(evidence, EvidenceRow{
				Label: f.DisplayName, Sublabel: "image-only PDF · replace or add a text alternative", Status: StatusTodo,
			})
		default:
			unknown++
		}
	}
	if len(evidence) == 0 && unknown == 0 {
		return Finding{Status: StatusDone, DetailDefault: "Referenced PDFs have a text layer."}, nil
	}
	if len(evidence) == 0 && unknown > 0 {
		return Finding{
			Status:        StatusUnknown,
			DetailKey:     "coursechecklist.item.a11y.document-accessibility.detail.unknown",
			DetailDefault: "Could not open one or more PDFs to check for a text layer.",
		}, nil
	}
	return Finding{
		Status: StatusTodo, Evidence: evidence,
		DetailDefault: fmt.Sprintf("%d image-only PDFs need a text alternative.", len(evidence)),
	}, nil
}

func referencedFileIDs(snap CourseSnapshot) map[uuid.UUID]struct{} {
	out := map[uuid.UUID]struct{}{}
	for _, meta := range snap.ItemMeta {
		for _, id := range meta.EmbeddedFileIDs {
			out[id] = struct{}{}
		}
	}
	return out
}

func ruleA11yMediaAlternatives() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yMediaAlternatives,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.media-alternatives.title",
		TitleDefault: "Add a text alternative beside video modules",
		WhyKey:       "coursechecklist.item.a11y.media-alternatives.why",
		WhyDefault:   a11yWhy("When a module is primarily video, offer a transcript, notes, or text page in the same module."),
		HelpRef:      "course-checklist#a11y-media-alternatives",
		Tier:         TierRecommended,
		Sources:      []string{"WCAG 1.2.x", "UDL Representation"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta},
		Evaluate:     evalA11yMediaAlternatives,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"},
		EvidenceShape: &EvidenceShape{Columns: []string{"Module", "Issue"}},
	}
}

func evalA11yMediaAlternatives(_ context.Context, snap CourseSnapshot) (Finding, error) {
	doc := contentDocFor(snap)
	byModule := map[string][]ContentPage{}
	for _, p := range doc.Pages {
		if p.Kind == "syllabus" {
			continue
		}
		key := p.ModuleTitle
		if key == "" {
			key = "(no module)"
		}
		byModule[key] = append(byModule[key], p)
	}
	var evidence []EvidenceRow
	for mod, pages := range byModule {
		mediaPages, textPages := 0, 0
		for _, p := range pages {
			if len(p.Media) > 0 {
				mediaPages++
			}
			if len(p.PlainText) > 200 && len(p.Media) == 0 {
				textPages++
			}
			for _, m := range p.Media {
				if m.HasCaptionsOrTranscript {
					textPages++
				}
			}
		}
		if mediaPages > 0 && textPages == 0 && mediaPages >= len(pages) {
			evidence = append(evidence, EvidenceRow{
				Label: mod, Sublabel: "video-primary module without text alternative", Status: StatusTodo,
			})
		}
	}
	if len(evidence) == 0 {
		return Finding{Status: StatusDone, DetailDefault: "Video-primary modules offer a text alternative."}, nil
	}
	return Finding{Status: StatusTodo, Evidence: evidence, DetailDefault: fmt.Sprintf("%d modules need a text alternative.", len(evidence))}, nil
}

func ruleA11yEnforcementSettings() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yEnforcementSettings,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.enforcement-settings.title",
		TitleDefault: "Review accessibility settings",
		WhyKey:       "coursechecklist.item.a11y.enforcement-settings.why",
		WhyDefault:   a11yWhy("Confirm alt-text enforcement and caption requirements for this course."),
		HelpRef:      "course-checklist#a11y-enforcement-settings",
		Tier:         TierRecommended,
		Sources:      []string{"QM 8.x"},
		DataNeeds:    []DataNeed{DataNeedCourse},
		Evaluate:     evalA11yEnforcementSettings,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}/settings/accessibility"},
	}
}

func evalA11yEnforcementSettings(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if snap.A11yReviewedAt != nil {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.a11y.enforcement-settings.detail.done",
			DetailDefault: "Accessibility settings have been reviewed.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.a11y.enforcement-settings.detail.todo",
		DetailDefault: "Open accessibility settings and confirm alt-text and caption choices.",
	}, nil
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func udlRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleUDLMultipleRepresentations(),
		ruleUDLExpressionChoice(),
		ruleUDLEngagementRelevance(),
		ruleA11yPlainLanguage(),
	}
}

func ruleUDLMultipleRepresentations() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemUDLMultipleRepresentations,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.udl.multiple-representations.title",
		TitleDefault: "Offer multiple representations per module",
		WhyKey:       "coursechecklist.item.udl.multiple-representations.why",
		WhyDefault:   a11yWhy("Most modules should mix modalities (text + media or interactive) so learners can choose a path."),
		HelpRef:      "course-checklist#udl-multiple-representations",
		Tier:         TierRecommended,
		Sources:      []string{"UDL Representation", "QM 4.5", "OSCQR 29"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedContentTools},
		Evaluate:     evalUDLMultipleRepresentations,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"},
	}
}

func evalUDLMultipleRepresentations(_ context.Context, snap CourseSnapshot) (Finding, error) {
	doc := contentDocFor(snap)
	type modStat struct {
		modalities map[string]bool
	}
	mods := map[string]*modStat{}
	for _, it := range snap.StructureItems {
		if it.Kind == "module" && !it.Archived {
			mods[it.ID.String()] = &modStat{modalities: map[string]bool{}}
		}
	}
	parentOf := map[string]string{}
	for _, it := range snap.StructureItems {
		if it.ParentID != nil {
			parentOf[it.ID.String()] = it.ParentID.String()
		}
		if it.Kind == "h5p" || it.Kind == "scorm" || it.Kind == "lti" {
			if pid := parentOf[it.ID.String()]; pid != "" {
				if m := mods[pid]; m != nil {
					m.modalities["interactive"] = true
				}
			}
		}
		if _, ok := snap.ContentToolItemIDs[it.ID]; ok {
			if pid, ok := parentOf[it.ID.String()]; ok {
				if m := mods[pid]; m != nil {
					m.modalities["interactive"] = true
				}
			}
		}
	}
	for _, p := range doc.Pages {
		pid := ""
		for _, it := range snap.StructureItems {
			if it.ID == p.ItemID && it.ParentID != nil {
				pid = it.ParentID.String()
				break
			}
		}
		if pid == "" {
			continue
		}
		m := mods[pid]
		if m == nil {
			continue
		}
		for k, v := range p.Modalities {
			if v {
				m.modalities[k] = true
			}
		}
		if len(p.Media) > 0 {
			m.modalities["media"] = true
		}
	}
	if len(mods) == 0 {
		return Finding{Status: StatusNotApplicable, DetailDefault: "No modules to evaluate."}, nil
	}
	multi := 0
	for _, m := range mods {
		n := 0
		for _, v := range m.modalities {
			if v {
				n++
			}
		}
		if n >= 2 {
			multi++
		}
	}
	ratio := float64(multi) / float64(len(mods))
	detail := fmt.Sprintf("%.0f%% of modules offer ≥2 modalities.", ratio*100)
	if ratio+0.001 >= 0.60 {
		return Finding{
			Status: StatusDone, DetailDefault: detail,
			DetailFields: map[string]any{"ratio": ratio, "multi": multi, "total": len(mods)},
		}, nil
	}
	return Finding{
		Status: StatusTodo, DetailDefault: detail + " Aim for at least 60%.",
		DetailFields: map[string]any{"ratio": ratio, "multi": multi, "total": len(mods)},
	}, nil
}

func ruleUDLExpressionChoice() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemUDLExpressionChoice,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.udl.expression-choice.title",
		TitleDefault: "Allow a choice of submission format",
		WhyKey:       "coursechecklist.item.udl.expression-choice.why",
		WhyDefault:   a11yWhy("At least one assessment should let learners choose how to demonstrate learning."),
		HelpRef:      "course-checklist#udl-expression-choice",
		Tier:         TierRecommended,
		Sources:      []string{"UDL Action & Expression", "QM 3.4"},
		DataNeeds:    []DataNeed{DataNeedAssessmentItems},
		Evaluate:     evalUDLExpressionChoice,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"},
	}
}

func evalUDLExpressionChoice(_ context.Context, snap CourseSnapshot) (Finding, error) {
	lex := lexiconForSnap(snap)
	for _, a := range assessmentItemsFor(snap) {
		types := 0
		if a.AllowTextSubmission {
			types++
		}
		if a.AllowFileUpload {
			types++
		}
		if a.AllowURLSubmission {
			types++
		}
		if types >= 2 {
			return Finding{
				Status:        StatusDone,
				DetailDefault: fmt.Sprintf("%q accepts multiple submission types.", a.Title),
			}, nil
		}
		body := strings.ToLower(a.Title + " " + a.BodyMarkdown)
		if lex != nil && lex.ExpressionChoice != nil && lex.ExpressionChoice.Match(body) {
			return Finding{
				Status:        StatusDone,
				DetailDefault: fmt.Sprintf("%q invites a choice of format.", a.Title),
			}, nil
		}
		if strings.Contains(body, "choose") && (strings.Contains(body, "format") || strings.Contains(body, "submit") || strings.Contains(body, "video or")) {
			return Finding{Status: StatusDone, DetailDefault: fmt.Sprintf("%q invites a choice of format.", a.Title)}, nil
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailDefault: "Add one assessment that accepts more than one submission type or offers a format choice.",
	}, nil
}

func ruleUDLEngagementRelevance() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemUDLEngagementRelevance,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.udl.engagement-relevance.title",
		TitleDefault: "Include an authentic or applied activity",
		WhyKey:       "coursechecklist.item.udl.engagement-relevance.why",
		WhyDefault:   a11yWhy("At least one authentic activity (project, case study, portfolio, capstone) helps engagement."),
		HelpRef:      "course-checklist#udl-engagement-relevance",
		Tier:         TierRecommended,
		Sources:      []string{"UDL Engagement", "QM 4.x"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedAssessmentItems},
		Evaluate:     evalUDLEngagementRelevance,
		Target:       NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"},
	}
}

func evalUDLEngagementRelevance(_ context.Context, snap CourseSnapshot) (Finding, error) {
	lex := lexiconForSnap(snap)
	match := func(text string) bool {
		if lex != nil && lex.AuthenticActivity != nil && lex.AuthenticActivity.Match(text) {
			return true
		}
		lower := strings.ToLower(text)
		for _, kw := range []string{"project", "case study", "portfolio", "capstone", "authentic", "real-world", "applied"} {
			if strings.Contains(lower, kw) {
				return true
			}
		}
		return false
	}
	for _, it := range snap.StructureItems {
		if it.Archived {
			continue
		}
		body := it.Title
		if meta, ok := snap.ItemMeta[it.ID]; ok {
			body += " " + meta.BodyMarkdown
		}
		if match(body) {
			return Finding{Status: StatusDone, DetailDefault: fmt.Sprintf("Found authentic activity: %s", it.Title)}, nil
		}
	}
	for _, a := range assessmentItemsFor(snap) {
		if match(a.Title + " " + a.BodyMarkdown) {
			return Finding{Status: StatusDone, DetailDefault: fmt.Sprintf("Found authentic activity: %s", a.Title)}, nil
		}
	}
	return Finding{
		Status:        StatusTodo,
		DetailDefault: "Add a project, case study, portfolio, or capstone activity.",
	}, nil
}

func ruleA11yPlainLanguage() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemA11yPlainLanguage,
		Category:     CategoryAccessibility,
		TitleKey:     "coursechecklist.item.a11y.plain-language.title",
		TitleDefault: "Keep page reading level near the grade band",
		WhyKey:       "coursechecklist.item.a11y.plain-language.why",
		WhyDefault:   a11yWhy("Pages much harder than the course grade band can shut learners out — this is a readability estimate, not a judgment."),
		HelpRef:      "course-checklist#a11y-plain-language",
		Tier:         TierRecommended,
		Sources:      []string{"QM 8.x"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedStructure, DataNeedItemMeta, DataNeedSyllabus},
		Applies: func(snap CourseSnapshot) bool {
			if _, ok := gradeBandMaxFKGL(snap.GradeLevels); !ok {
				return false
			}
			loc := strings.ToLower(snapLocale(snap))
			return strings.HasPrefix(loc, "en")
		},
		Evaluate: evalA11yPlainLanguage,
		Target:   NavTarget{Surface: "web", Route: "/courses/{courseCode}/modules"},
		EvidenceShape: &EvidenceShape{Columns: []string{"Page", "Grade level", "Issue"}},
	}
}

func evalA11yPlainLanguage(_ context.Context, snap CourseSnapshot) (Finding, error) {
	band, ok := gradeBandMaxFKGL(snap.GradeLevels)
	if !ok {
		return Finding{Status: StatusNotApplicable, DetailDefault: "No grade band declared."}, nil
	}
	doc := contentDocFor(snap)
	var evidence []EvidenceRow
	for _, p := range doc.Pages {
		if p.Kind == "syllabus" {
			continue
		}
		score := readinglevel.Analyze(p.PlainText)
		if !score.Sufficient {
			continue
		}
		if score.FKGL > band+3 {
			evidence = append(evidence, EvidenceRow{
				Label:    pageLabel(p),
				Sublabel: fmt.Sprintf("FKGL %.1f (band %.0f + 3)", score.FKGL, band),
				Status:   StatusTodo,
				TargetOverride: &NavTarget{
					Surface: "web", Route: p.Route, EntityKey: p.ItemID.String(),
				},
			})
		}
	}
	if len(evidence) == 0 {
		return Finding{Status: StatusDone, DetailDefault: "Page reading levels are within tolerance of the grade band."}, nil
	}
	return Finding{
		Status: StatusTodo, Evidence: evidence,
		DetailDefault: fmt.Sprintf("%d pages exceed the grade band by more than 3 levels.", len(evidence)),
	}, nil
}
