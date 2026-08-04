package coursechecklist

import (
	"sort"
	"strings"

	"github.com/google/uuid"
)

func listModules(snap CourseSnapshot) []StructureItem {
	var mods []StructureItem
	for _, it := range snap.StructureItems {
		if it.Archived {
			continue
		}
		if it.Kind == "module" && it.ParentID == nil {
			mods = append(mods, it)
		}
	}
	sort.SliceStable(mods, func(i, j int) bool {
		if mods[i].SortOrder == mods[j].SortOrder {
			return mods[i].Title < mods[j].Title
		}
		return mods[i].SortOrder < mods[j].SortOrder
	})
	return mods
}

func childrenByParent(snap CourseSnapshot) map[uuid.UUID][]StructureItem {
	out := map[uuid.UUID][]StructureItem{}
	for _, it := range snap.StructureItems {
		if it.Archived || it.ParentID == nil {
			continue
		}
		out[*it.ParentID] = append(out[*it.ParentID], it)
	}
	for id := range out {
		sort.SliceStable(out[id], func(i, j int) bool {
			if out[id][i].SortOrder == out[id][j].SortOrder {
				return out[id][i].Title < out[id][j].Title
			}
			return out[id][i].SortOrder < out[id][j].SortOrder
		})
	}
	return out
}

func structureByID(snap CourseSnapshot) map[uuid.UUID]StructureItem {
	out := make(map[uuid.UUID]StructureItem, len(snap.StructureItems))
	for _, it := range snap.StructureItems {
		out[it.ID] = it
	}
	return out
}

func sortStructureItems(items []StructureItem) []StructureItem {
	out := append([]StructureItem(nil), items...)
	// Parent module order, then item sort order, then title.
	parentOrder := map[uuid.UUID]int{}
	for _, it := range items {
		if it.Kind == "module" {
			parentOrder[it.ID] = it.SortOrder
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		pi, pj := 0, 0
		if out[i].ParentID != nil {
			pi = parentOrder[*out[i].ParentID]
		}
		if out[j].ParentID != nil {
			pj = parentOrder[*out[j].ParentID]
		}
		if pi != pj {
			return pi < pj
		}
		if out[i].SortOrder != out[j].SortOrder {
			return out[i].SortOrder < out[j].SortOrder
		}
		return out[i].Title < out[j].Title
	})
	return out
}

func sortEvidenceByLabel(rows []EvidenceRow) []EvidenceRow {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Label == rows[j].Label {
			return rows[i].Sublabel < rows[j].Sublabel
		}
		return rows[i].Label < rows[j].Label
	})
	return rows
}

func humanKind(kind string) string {
	switch kind {
	case "content_page":
		return "Page"
	case "external_link":
		return "External link"
	case "library_resource":
		return "Library resource"
	case "textbook_resource":
		return "Textbook"
	case "lti_link":
		return "LTI"
	case "vibe_activity":
		return "Activity"
	default:
		if kind == "" {
			return "Item"
		}
		return strings.ToUpper(kind[:1]) + kind[1:]
	}
}

func itemEditorRoute(kind string) string {
	switch kind {
	case "assignment":
		return "/courses/{courseCode}/modules/assignment/{itemId}"
	case "quiz":
		return "/courses/{courseCode}/modules/quiz/{itemId}"
	case "content_page":
		return "/courses/{courseCode}/modules/content/{itemId}"
	case "external_link":
		return "/courses/{courseCode}/modules/external-link/{itemId}"
	case "h5p":
		return "/courses/{courseCode}/modules/h5p/{itemId}"
	case "scorm":
		return "/courses/{courseCode}/modules/scorm/{itemId}"
	case "lti_link":
		return "/courses/{courseCode}/modules/lti/{itemId}"
	case "textbook_resource":
		return "/courses/{courseCode}/modules/textbook-resource/{itemId}"
	default:
		return "/courses/{courseCode}/modules"
	}
}
