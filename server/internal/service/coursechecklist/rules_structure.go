package coursechecklist

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func structureRules() []ItemDescriptor {
	return []ItemDescriptor{
		ruleStructureModulesExist(),
		ruleStructureEmptyModules(),
		ruleStructurePlaceholderTitles(),
		ruleStructureModuleOverviews(),
		ruleStructureUnpublishedItems(),
		ruleStructureOrphanItems(),
		ruleStructurePacingSignal(),
		ruleStructureContentVariety(),
		ruleStructureInteractiveElements(),
		ruleStructureAttribution(),
		ruleStructureFileReferences(),
		ruleStructureGatingReview(),
	}
}

func ruleStructureModulesExist() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructureModulesExist,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.modules-exist.title",
		TitleDefault: "Add at least one module",
		WhyKey:       "coursechecklist.item.structure.modules-exist.why",
		WhyDefault:   "Learners need a module outline before they can navigate the course.",
		HelpRef:      "course-checklist#structure-modules-exist",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.2", "OSCQR 16", "NSQ C"},
		DataNeeds:    []DataNeed{DataNeedStructure},
		Evaluate:     evalStructureModulesExist,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
	}
}

func evalStructureModulesExist(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if len(listModules(snap)) >= 1 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.structure.modules-exist.detail.done",
			DetailDefault: "At least one module exists.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.structure.modules-exist.detail.todo",
		DetailDefault: "Add a module to organize course content.",
	}, nil
}

func ruleStructureEmptyModules() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructureEmptyModules,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.empty-modules.title",
		TitleDefault: "Fill in or remove empty modules",
		WhyKey:       "coursechecklist.item.structure.empty-modules.why",
		WhyDefault:   "Empty modules look unfinished and confuse learners on day one.",
		HelpRef:      "course-checklist#structure-empty-modules",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 16"},
		DataNeeds:    []DataNeed{DataNeedStructure},
		Evaluate:     evalStructureEmptyModules,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Module"}},
	}
}

func evalStructureEmptyModules(_ context.Context, snap CourseSnapshot) (Finding, error) {
	mods := listModules(snap)
	if len(mods) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.structure.empty-modules.detail.na",
			DetailDefault: "No modules to check.",
		}, nil
	}
	children := childrenByParent(snap)
	var evidence []EvidenceRow
	for _, m := range mods {
		if len(children[m.ID]) == 0 {
			evidence = append(evidence, EvidenceRow{
				Label:  m.Title,
				Status: StatusTodo,
				TargetOverride: &NavTarget{
					Surface: "web",
					Route:   "/courses/{courseCode}/modules",
					Anchor:  "module:" + m.ID.String(),
				},
			})
		}
	}
	evidence = sortEvidenceByLabel(evidence)
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.structure.empty-modules.detail.done",
			DetailDefault: "Every module has at least one item.",
		}, nil
	}
	total := len(evidence)
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.structure.empty-modules.detail.todo",
		DetailDefault: fmt.Sprintf("%d modules have no items.", total),
		DetailFields:  map[string]any{"emptyCount": total},
		Evidence:      evidence,
	}, nil
}

func ruleStructurePlaceholderTitles() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructurePlaceholderTitles,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.placeholder-titles.title",
		TitleDefault: "Replace placeholder item titles",
		WhyKey:       "coursechecklist.item.structure.placeholder-titles.why",
		WhyDefault:   "Titles like Untitled page tell learners the course is unfinished.",
		HelpRef:      "course-checklist#structure-placeholder-titles",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 20"},
		DataNeeds:    []DataNeed{DataNeedStructure},
		Evaluate:     evalStructurePlaceholderTitles,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Item", "Type"}},
	}
}

func evalStructurePlaceholderTitles(_ context.Context, snap CourseSnapshot) (Finding, error) {
	placeholders := PlaceholderLexiconFor(snapLocale(snap))
	var evidence []EvidenceRow
	for _, it := range sortStructureItems(snap.StructureItems) {
		if it.Archived || it.Kind == "module" || it.Kind == "heading" {
			continue
		}
		if !isStructurePlaceholderTitle(it.Title, placeholders) {
			continue
		}
		evidence = append(evidence, EvidenceRow{
			Label:    it.Title,
			Sublabel: humanKind(it.Kind),
			Status:   StatusTodo,
			TargetOverride: &NavTarget{
				Surface: "web",
				Route:   itemEditorRoute(it.Kind),
				Anchor:  "item:" + it.ID.String(),
			},
		})
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.structure.placeholder-titles.detail.done",
			DetailDefault: "No placeholder titles found.",
		}, nil
	}
	total := len(evidence)
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.structure.placeholder-titles.detail.todo",
		DetailDefault: fmt.Sprintf("%d items still use placeholder titles.", total),
		DetailFields:  map[string]any{"count": total},
		Evidence:      evidence,
	}, nil
}

func ruleStructureModuleOverviews() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructureModuleOverviews,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.module-overviews.title",
		TitleDefault: "Add an overview to every module",
		WhyKey:       "coursechecklist.item.structure.module-overviews.why",
		WhyDefault:   "Learners need to know what a week is for before they start it.",
		HelpRef:      "course-checklist#structure-module-overviews",
		Tier:         TierRecommended,
		Sources:      []string{"QM 1.2", "OSCQR 2", "NSQ A"},
		DataNeeds:    []DataNeed{DataNeedStructure},
		Evaluate:     evalStructureModuleOverviews,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Module"}},
	}
}

func evalStructureModuleOverviews(_ context.Context, snap CourseSnapshot) (Finding, error) {
	mods := listModules(snap)
	if len(mods) == 0 {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.structure.module-overviews.detail.na",
			DetailDefault: "No modules to check.",
		}, nil
	}
	lx := lexiconForSnap(snap)
	children := childrenByParent(snap)
	var missing []EvidenceRow
	done := 0
	for _, m := range mods {
		if moduleHasOverview(m, children[m.ID], lx) {
			done++
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
			DetailKey:     "coursechecklist.item.structure.module-overviews.detail.done",
			DetailDefault: "Every module has an overview.",
			Progress:      &Progress{Done: done, Total: len(mods)},
		}, nil
	}
	return Finding{
		Status:        StatusInProgress,
		DetailKey:     "coursechecklist.item.structure.module-overviews.detail.partial",
		DetailDefault: fmt.Sprintf("%d of %d modules have an overview.", done, len(mods)),
		Progress:      &Progress{Done: done, Total: len(mods)},
		Evidence:      missing,
	}, nil
}

func moduleHasOverview(mod StructureItem, children []StructureItem, lx *Lexicon) bool {
	// No module description column today — rely on first child content page title heuristic.
	var firstPage *StructureItem
	for i := range children {
		c := children[i]
		if c.Archived || c.Kind != "content_page" {
			continue
		}
		if firstPage == nil || c.SortOrder < firstPage.SortOrder {
			cp := c
			firstPage = &cp
		}
	}
	if firstPage == nil {
		return false
	}
	if lx != nil && TitleMatches(lx.ModuleOverviewTitles, firstPage.Title) {
		return true
	}
	// Also accept "overview"/"introduction" as a whole-title English fallback.
	t := strings.ToLower(strings.TrimSpace(firstPage.Title))
	return t == "overview" || strings.HasPrefix(t, "overview ") ||
		t == "introduction" || strings.HasPrefix(t, "introduction ")
}

func ruleStructureUnpublishedItems() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructureUnpublishedItems,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.unpublished-items.title",
		TitleDefault: "Publish items in live modules",
		WhyKey:       "coursechecklist.item.structure.unpublished-items.why",
		WhyDefault:   "Unpublished items inside a live module look like broken links to learners.",
		HelpRef:      "course-checklist#structure-unpublished-items",
		Tier:         TierRecommended,
		Sources:      []string{"OSCQR 7"},
		DataNeeds:    []DataNeed{DataNeedCourse, DataNeedStructure},
		Evaluate:     evalStructureUnpublishedItems,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Item", "Module"}},
	}
}

func evalStructureUnpublishedItems(_ context.Context, snap CourseSnapshot) (Finding, error) {
	if !snap.Published {
		return Finding{
			Status:        StatusNotApplicable,
			DetailKey:     "coursechecklist.item.structure.unpublished-items.detail.na",
			DetailDefault: "Course is not published yet.",
		}, nil
	}
	now := time.Now().UTC()
	byID := structureByID(snap)
	var evidence []EvidenceRow
	for _, it := range sortStructureItems(snap.StructureItems) {
		if it.Archived || it.Published || it.Kind == "module" || it.Kind == "heading" {
			continue
		}
		if it.ParentID == nil {
			continue
		}
		mod, ok := byID[*it.ParentID]
		if !ok || mod.Kind != "module" || mod.Archived || !mod.Published {
			continue
		}
		if mod.VisibleFrom != nil && mod.VisibleFrom.After(now) {
			continue
		}
		evidence = append(evidence, EvidenceRow{
			Label:    it.Title,
			Sublabel: mod.Title,
			Status:   StatusTodo,
			TargetOverride: &NavTarget{
				Surface: "web",
				Route:   itemEditorRoute(it.Kind),
				Anchor:  "publish:" + it.ID.String(),
			},
		})
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.structure.unpublished-items.detail.done",
			DetailDefault: "No unpublished items sit in live modules.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.structure.unpublished-items.detail.todo",
		DetailDefault: fmt.Sprintf("%d unpublished items are inside published modules.", len(evidence)),
		Evidence:      evidence,
	}, nil
}

func ruleStructureOrphanItems() ItemDescriptor {
	return ItemDescriptor{
		ID:           ItemStructureOrphanItems,
		Category:     CategoryStructure,
		TitleKey:     "coursechecklist.item.structure.orphan-items.title",
		TitleDefault: "Repair orphaned module items",
		WhyKey:       "coursechecklist.item.structure.orphan-items.why",
		WhyDefault:   "Items whose parent module is gone or archived cannot be reached by learners.",
		HelpRef:      "course-checklist#structure-orphan-items",
		Tier:         TierRecommended,
		Sources:      []string{"Product"},
		DataNeeds:    []DataNeed{DataNeedStructure},
		Evaluate:     evalStructureOrphanItems,
		Target: NavTarget{
			Surface: "web",
			Route:   "/courses/{courseCode}/modules",
		},
		EvidenceShape: &EvidenceShape{Columns: []string{"Item", "Type"}},
	}
}

func evalStructureOrphanItems(_ context.Context, snap CourseSnapshot) (Finding, error) {
	byID := structureByID(snap)
	var evidence []EvidenceRow
	for _, it := range sortStructureItems(snap.StructureItems) {
		if it.Archived || it.Kind == "module" || it.ParentID == nil {
			continue
		}
		parent, ok := byID[*it.ParentID]
		if !ok || parent.Archived {
			evidence = append(evidence, EvidenceRow{
				Label:    it.Title,
				Sublabel: humanKind(it.Kind),
				Status:   StatusTodo,
			})
		}
	}
	if len(evidence) == 0 {
		return Finding{
			Status:        StatusDone,
			DetailKey:     "coursechecklist.item.structure.orphan-items.detail.done",
			DetailDefault: "No orphaned items found.",
		}, nil
	}
	return Finding{
		Status:        StatusTodo,
		DetailKey:     "coursechecklist.item.structure.orphan-items.detail.todo",
		DetailDefault: fmt.Sprintf("%d items point at a missing or archived module.", len(evidence)),
		Evidence:      evidence,
	}, nil
}
