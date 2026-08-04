package coursechecklist

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

func ruleStructurePacingSignal() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructurePacingSignal,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.pacing-signal.title",
		TitleDefault: "Add pacing signals to modules",
		WhyKey:       "coursechecklist.item.structure.pacing-signal.why",
		WhyDefault:   "Date ranges or due dates help learners plan their week.",
		HelpRef:      "course-checklist#structure-pacing-signal",
		Tier:         TierRecommended,
		Sources:      []string{"QM 8.6", "OSCQR 30"},
		DataNeeds:    []DataNeed{DataNeedStructure},
		Evaluate:     evalStructurePacingSignal,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Module"}},
	}
}

func evalStructurePacingSignal(_ context.Context, snap CourseSnapshot) (Finding, error) {
	mods := listModules(snap)
	if len(mods) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.structure.pacing-signal.detail.na",
			DetailDefault: "No modules to check.",
		}, nil
	}
	children := childrenByParent(snap)
	var missing []EvidenceRow
	for _, m := range mods {
		if moduleHasPacing(children[m.ID]) {
			continue
		}
		missing = append(missing, EvidenceRow{
			Label:  m.Title,
			Status: StatusTodo,
			TargetOverride: &NavTarget{
				Surface: "web",
				Route:   "/courses/{courseCode}/modules",
				Anchor:  "module:" + m.ID.String(),
			},
		})
	}
	missing = sortEvidenceByLabel(missing)
	if len(missing) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.structure.pacing-signal.detail.done",
			DetailDefault: "Every module has a pacing signal.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.structure.pacing-signal.detail.todo",
		DetailDefault: fmt.Sprintf("%d modules have no due dates or date range.", len(missing)),
		Evidence:      missing,
	}, nil
}

func moduleHasPacing(children []StructureItem) bool {
	var minDue, maxDue *time.Time
	for _, c := range children {
		if c.Archived || c.DueAt == nil {
			continue
		}
		d := *c.DueAt
		if minDue == nil || d.Before(*minDue) {
			minDue = &d
		}
		if maxDue == nil || d.After(*maxDue) {
			maxDue = &d
		}
	}
	return minDue != nil
}

func ruleStructureContentVariety() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructureContentVariety,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.content-variety.title",
		TitleDefault: "Use more than one content type",
		WhyKey:       "coursechecklist.item.structure.content-variety.why",
		WhyDefault:   "A mix of activities supports different ways of learning.",
		HelpRef:      "course-checklist#structure-content-variety",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 29", "UDL Representation"},
		DataNeeds:    []DataNeed{DataNeedStructure},
		Evaluate:     evalStructureContentVariety,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Module"}},
	}
}

func evalStructureContentVariety(_ context.Context, snap CourseSnapshot) (Finding, error) {
	varietyKinds := map[string]struct{}{
		"assignment": {}, "quiz": {}, "external_link": {}, "h5p": {}, "scorm": {},
		"lti_link": {}, "library_resource": {}, "textbook_resource": {},
		"vibe_activity": {}, "survey": {},
	}
	kinds := map[string]struct{}{}
	children := childrenByParent(snap)
	var textOnly []EvidenceRow
	for _, m := range listModules(snap) {
		nonPage := 0
		for _, c := range children[m.ID] {
			if c.Archived {
				continue
			}
			if _, ok := varietyKinds[c.Kind]; ok {
				kinds[c.Kind] = struct{}{}
				nonPage++
			}
		}
		if nonPage == 0 && len(children[m.ID]) > 0 {
			textOnly = append(textOnly, EvidenceRow{Label: m.Title, Status: StatusTodo})
		}
	}
	textOnly = sortEvidenceByLabel(textOnly)
	kindList := make([]string, 0, len(kinds))
	for k := range kinds {
		kindList = append(kindList, k)
	}
	sort.Strings(kindList)
	if len(kinds) >= 2 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.structure.content-variety.detail.done",
			DetailDefault: fmt.Sprintf("Observed mix: %s.", strings.Join(kindList, ", ")),
			DetailFields:  map[string]any{"kinds": kindList},
		}, nil
	}
	detail := "Add at least two item types beyond plain pages (assignment, quiz, discussion, link, H5P…)."
	if len(kindList) == 1 {
		detail = fmt.Sprintf("Only one activity type so far (%s).", kindList[0])
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.structure.content-variety.detail.todo",
		DetailDefault: detail,
		DetailFields:  map[string]any{"kinds": kindList},
		Evidence:      textOnly,
	}, nil
}

func ruleStructureInteractiveElements() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructureInteractiveElements,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.interactive-elements.title",
		TitleDefault: "Add interactive elements to pages",
		WhyKey:       "coursechecklist.item.structure.interactive-elements.why",
		WhyDefault:   "Interactive tools keep learners engaged beyond static reading.",
		HelpRef:      "course-checklist#structure-interactive-elements",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 30", "OSCQR 31", "UDL Engagement"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedContentTools},
		Evaluate:     evalStructureInteractiveElements,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
	}
}

func evalStructureInteractiveElements(_ context.Context, snap CourseSnapshot) (Finding, error) {
	var pages []StructureItem
	h5pOrQuizEmbed := false
	for _, it := range snap.StructureItems {
		if it.Archived {
			continue
		}
		if it.Kind == "content_page" {
			pages = append(pages, it)
		}
		if it.Kind == "h5p" || it.Kind == "quiz" {
			h5pOrQuizEmbed = true
		}
	}
	toolsEnabled := snap.Features.ContentToolsEnabled
	if !toolsEnabled && !h5pOrQuizEmbed && len(snap.ContentToolItemIDs) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.structure.interactive-elements.detail.na",
			DetailDefault: "Content tools are off and no H5P/quiz embeds exist.",
		}, nil
	}
	if len(pages) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.structure.interactive-elements.detail.nopages",
			DetailDefault: "No content pages to check.",
		}, nil
	}
	interactive := 0
	for _, p := range pages {
		if _, ok := snap.ContentToolItemIDs[p.ID]; ok {
			interactive++
		}
	}
	// Also count sibling H5P items under the same modules as contributing engagement.
	// Threshold: ≥ 50% of content pages have a content-tool instance.
	ratio := float64(interactive) / float64(len(pages))
	if ratio >= 0.5 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.structure.interactive-elements.detail.done",
			DetailDefault: fmt.Sprintf("%d of %d content pages include an interactive element.", interactive, len(pages)),
			Progress:      &Progress{Done: interactive, Total: len(pages)},
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.structure.interactive-elements.detail.todo",
		DetailDefault: fmt.Sprintf("%d of %d content pages include an interactive element (aim for half).", interactive, len(pages)),
		Progress:      &Progress{Done: interactive, Total: len(pages)},
	}, nil
}

func ruleStructureAttribution() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructureAttribution,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.attribution.title",
		TitleDefault: "Attribute external and library resources",
		WhyKey:       "coursechecklist.item.structure.attribution.why",
		WhyDefault:   "Attribution keeps shared materials copyright-safe and transparent.",
		HelpRef:      "course-checklist#structure-attribution",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 32", "OSCQR 33"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta},
		Evaluate:     evalStructureAttribution,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Item", "Type"}},
	}
}

func evalStructureAttribution(_ context.Context, snap CourseSnapshot) (Finding, error) {
	needKinds := map[string]struct{}{
		"external_link": {}, "library_resource": {}, "textbook_resource": {},
	}
	var evidence []EvidenceRow
	for _, it := range sortStructureItems(snap.StructureItems) {
		if it.Archived {
			continue
		}
		if _, ok := needKinds[it.Kind]; !ok {
			continue
		}
		meta := snap.ItemMeta[it.ID]
		if strings.TrimSpace(meta.Attribution) != "" {
			continue
		}
		evidence = append(evidence, EvidenceRow{
			Label:    it.Title,
			Sublabel: humanKind(it.Kind),
			Status:   StatusTodo,
			TargetOverride: &NavTarget{
				Surface: "web",
				Route:   itemEditorRoute(it.Kind),
				Anchor:  "attribution:" + it.ID.String(),
			},
		})
	}
	if len(evidence) == 0 {
		// N/A when course has none of these kinds.
		hasAny := false
		for _, it := range snap.StructureItems {
			if it.Archived {
				continue
			}
			if _, ok := needKinds[it.Kind]; ok {
				hasAny = true
				break
			}
		}
		if !hasAny {
			return Finding{
				Status:        StatusNotApplicable,
				DetailKey:     "coursechecklist.item.structure.attribution.detail.na",
				DetailDefault: "No external, library, or textbook resources to attribute.",
			}, nil
		}
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.structure.attribution.detail.done",
			DetailDefault: "External and library resources include attribution.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.structure.attribution.detail.todo",
		DetailDefault: fmt.Sprintf("%d resources are missing attribution.", len(evidence)),
		Evidence:      evidence,
	}, nil
}

func ruleStructureFileReferences() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructureFileReferences,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.file-references.title",
		TitleDefault: "Fix broken file references",
		WhyKey:       "coursechecklist.item.structure.file-references.why",
		WhyDefault:   "Broken file links inside pages leave learners stuck.",
		HelpRef:      "course-checklist#structure-file-references",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 37"},
		DataNeeds:    []DataNeed{DataNeedStructure, DataNeedItemMeta, DataNeedFiles},
		Evaluate:     evalStructureFileReferences,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/files",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"File", "Page"}},
	}
}

func evalStructureFileReferences(_ context.Context, snap CourseSnapshot) (Finding, error) {
	fileIDs := make(map[uuid.UUID]struct{}, len(snap.Files))
	for _, f := range snap.Files {
		fileIDs[f.ID] = struct{}{}
	}
	byID := structureByID(snap)
	var evidence []EvidenceRow
	for id, meta := range snap.ItemMeta {
		for _, fid := range meta.EmbeddedFileIDs {
			if _, ok := fileIDs[fid]; ok {
				continue
			}
			pageTitle := fid.String()
			if it, ok := byID[id]; ok {
				pageTitle = it.Title
			}
			evidence = append(evidence, EvidenceRow{
				Label:    fid.String(),
				Sublabel: pageTitle,
				Status:   StatusTodo,
				TargetOverride: &NavTarget{
					Surface:   "web",
					Route:     "/courses/{courseCode}/modules/content/{itemId}",
					Anchor:    "item:" + id.String(),
					EntityKey: id.String(),
				},
			})
		}
	}
	sort.SliceStable(evidence, func(i, j int) bool {
		if evidence[i].Sublabel == evidence[j].Sublabel {
			return evidence[i].Label < evidence[j].Label
		}
		return evidence[i].Sublabel < evidence[j].Sublabel
	})
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.structure.file-references.detail.done",
			DetailDefault: "Embedded file references resolve.",
		}, nil
	}
	total := len(evidence)
	detail := fmt.Sprintf("%d broken internal file references.", total)
	if total > MaxEvidenceRows {
		detail = fmt.Sprintf("%d broken internal file references (showing first %d).", total, MaxEvidenceRows)
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.structure.file-references.detail.todo",
		DetailDefault: detail,
		DetailFields:  map[string]any{"brokenCount": total},
		Evidence:      evidence,
	}, nil
}

func ruleStructureGatingReview() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructureGatingReview,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.gating-review.title",
		TitleDefault: "Review module gating for dead ends",
		WhyKey:       "coursechecklist.item.structure.gating-review.why",
		WhyDefault:   "Prerequisite cycles or unpublished gates can permanently lock learners out.",
		HelpRef:      "course-checklist#structure-gating-review",
		Tier:         TierRecommended,
		Sources:      []string{"Product"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedStructure, DataNeedModulePrerequisites},
		Applies: func(snap CourseSnapshot) bool {
			return snap.ModuleGatingEnabled || len(snap.ModulePrerequisiteEdges) > 0 || snap.HasItemCompletionRules
		},
		Evaluate: evalStructureGatingReview,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Module", "Issue"}},
	}
}

func evalStructureGatingReview(_ context.Context, snap CourseSnapshot) (Finding, error) {
	byID := structureByID(snap)
	adj := map[uuid.UUID][]uuid.UUID{}
	for _, e := range snap.ModulePrerequisiteEdges {
		adj[e.ModuleID] = append(adj[e.ModuleID], e.PrerequisiteModuleID)
	}
	var evidence []EvidenceRow
	cycles := findPrerequisiteCycles(adj)
	for _, cyc := range cycles {
		label := "Prerequisite cycle"
		if len(cyc) > 0 {
			if m, ok := byID[cyc[0]]; ok {
				label = m.Title
			}
		}
		evidence = append(evidence, EvidenceRow{
			Label:    label,
			Sublabel: "cycle",
			Status:   StatusTodo,
		})
	}
	// Unpublished prerequisite modules.
	for _, e := range snap.ModulePrerequisiteEdges {
		prereq, ok := byID[e.PrerequisiteModuleID]
		if !ok || prereq.Archived || !prereq.Published {
			modTitle := e.ModuleID.String()
			if m, ok := byID[e.ModuleID]; ok {
				modTitle = m.Title
			}
			evidence = append(evidence, EvidenceRow{
				Label:    modTitle,
				Sublabel: "unpublished prerequisite",
				Status:   StatusTodo,
			})
		}
	}
	evidence = sortEvidenceByLabel(evidence)
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.structure.gating-review.detail.done",
			DetailDefault: "Gating paths look satisfiable.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.structure.gating-review.detail.todo",
		DetailDefault: fmt.Sprintf("%d gating issues found.", len(evidence)),
		Evidence:      evidence,
	}, nil
}

// findPrerequisiteCycles returns cycles using DFS with a visited-set + recursion stack (AC-9).
func findPrerequisiteCycles(adj map[uuid.UUID][]uuid.UUID) [][]uuid.UUID {
	const depthCap = 10_000
	state := map[uuid.UUID]int{} // 0=unseen, 1=visiting, 2=done
	var cycles [][]uuid.UUID
	var stack []uuid.UUID

	var dfs func(uuid.UUID, int)
	dfs = func(n uuid.UUID, depth int) {
		if depth > depthCap {
			return
		}
		state[n] = 1
		stack = append(stack, n)
		for _, next := range adj[n] {
			switch state[next] {
			case 0:
				dfs(next, depth+1)
			case 1:
				// Extract cycle from stack.
				var cyc []uuid.UUID
				for i := len(stack) - 1; i >= 0; i-- {
					cyc = append(cyc, stack[i])
					if stack[i] == next {
						break
					}
				}
				cycles = append(cycles, cyc)
			}
		}
		stack = stack[:len(stack)-1]
		state[n] = 2
	}
	for n := range adj {
		if state[n] == 0 {
			dfs(n, 0)
		}
	}
	return cycles
}
