package courseexportimport

import (
	"strings"

	"github.com/google/uuid"

	"github.com/lextures/lextures/server/internal/models/courseexport"
	ctsvc "github.com/lextures/lextures/server/internal/service/contenttools"
)

// remapBundleIDsForCrossCourseImport rewrites entity UUIDs when importing a bundle
// into a different course than it was exported from.
//
// Assignment groups, structure items, content tools, and files all use global
// primary keys on id. Preserving export ids while the source course still exists
// causes PK collisions (and ON CONFLICT updates would steal rows from the source).
// Same-course restore (export.courseCode == target) keeps ids so markdown/file
// URLs and ```lex-tool fences continue to resolve.
func remapBundleIDsForCrossCourseImport(ex *Bundle, targetCourseCode string) {
	if ex == nil {
		return
	}
	source := strings.TrimSpace(ex.CourseCode)
	target := strings.TrimSpace(targetCourseCode)
	if source == "" || target == "" || strings.EqualFold(source, target) {
		return
	}

	groupMap := map[uuid.UUID]uuid.UUID{}
	for i := range ex.Grading.AssignmentGroups {
		old := ex.Grading.AssignmentGroups[i].ID
		if old == uuid.Nil {
			continue
		}
		if _, ok := groupMap[old]; !ok {
			groupMap[old] = uuid.New()
		}
		ex.Grading.AssignmentGroups[i].ID = groupMap[old]
	}

	itemMap := map[uuid.UUID]uuid.UUID{}
	for i := range ex.Structure {
		it := &ex.Structure[i]
		oldID, err := uuid.Parse(strings.TrimSpace(it.ID))
		if err != nil {
			continue
		}
		if _, ok := itemMap[oldID]; !ok {
			itemMap[oldID] = uuid.New()
		}
		newID := itemMap[oldID]
		it.ID = newID.String()

		if it.ParentID != nil && strings.TrimSpace(*it.ParentID) != "" {
			if pid, perr := uuid.Parse(strings.TrimSpace(*it.ParentID)); perr == nil {
				if mapped, ok := itemMap[pid]; ok {
					s := mapped.String()
					it.ParentID = &s
				}
			}
		}
		if it.AssignmentGroupID != nil && strings.TrimSpace(*it.AssignmentGroupID) != "" {
			if gid, gerr := uuid.Parse(strings.TrimSpace(*it.AssignmentGroupID)); gerr == nil {
				if mapped, ok := groupMap[gid]; ok {
					s := mapped.String()
					it.AssignmentGroupID = &s
				} else {
					// Group not in export grading list — drop rather than keep a foreign id.
					it.AssignmentGroupID = nil
				}
			}
		}
	}

	if len(ex.ContentPages) > 0 {
		next := make(map[uuid.UUID]courseexport.ExportedContentPageBody, len(ex.ContentPages))
		for old, body := range ex.ContentPages {
			next[mapOrSame(itemMap, old)] = body
		}
		ex.ContentPages = next
	}
	if len(ex.Assignments) > 0 {
		next := make(map[uuid.UUID]courseexport.ExportedAssignmentBody, len(ex.Assignments))
		for old, body := range ex.Assignments {
			next[mapOrSame(itemMap, old)] = body
		}
		ex.Assignments = next
	}
	if len(ex.Quizzes) > 0 {
		next := make(map[uuid.UUID]courseexport.ExportedQuizBody, len(ex.Quizzes))
		for old, body := range ex.Quizzes {
			if len(body.AdaptiveSourceItemIDs) > 0 {
				mapped := make([]uuid.UUID, 0, len(body.AdaptiveSourceItemIDs))
				for _, src := range body.AdaptiveSourceItemIDs {
					mapped = append(mapped, mapOrSame(itemMap, src))
				}
				body.AdaptiveSourceItemIDs = mapped
			}
			next[mapOrSame(itemMap, old)] = body
		}
		ex.Quizzes = next
	}

	toolMapStr := map[string]string{}
	for i := range ex.ContentToolInstances {
		in := &ex.ContentToolInstances[i]
		if in.ID != uuid.Nil {
			newID := uuid.New()
			toolMapStr[in.ID.String()] = newID.String()
			in.ID = newID
		}
		if in.StructureItemID != nil {
			mapped := mapOrSame(itemMap, *in.StructureItemID)
			in.StructureItemID = &mapped
		}
	}

	fileMapStr := map[string]string{}
	for i := range ex.CourseFiles {
		old := ex.CourseFiles[i].ID
		if old == uuid.Nil {
			continue
		}
		newID := uuid.New()
		fileMapStr[old.String()] = newID.String()
		ex.CourseFiles[i].ID = newID
	}

	folderMap := map[uuid.UUID]uuid.UUID{}
	for i := range ex.FileFolders {
		old := ex.FileFolders[i].ID
		if old == uuid.Nil {
			continue
		}
		if _, ok := folderMap[old]; !ok {
			folderMap[old] = uuid.New()
		}
		ex.FileFolders[i].ID = folderMap[old]
	}
	for i := range ex.FileFolders {
		if ex.FileFolders[i].ParentID != nil {
			if mapped, ok := folderMap[*ex.FileFolders[i].ParentID]; ok {
				ex.FileFolders[i].ParentID = &mapped
			} else {
				ex.FileFolders[i].ParentID = nil
			}
		}
	}

	for i := range ex.FileItems {
		old := ex.FileItems[i].ID
		if old != uuid.Nil {
			newID := uuid.New()
			fileMapStr[old.String()] = newID.String()
			ex.FileItems[i].ID = newID
		}
		if ex.FileItems[i].FolderID != nil {
			if mapped, ok := folderMap[*ex.FileItems[i].FolderID]; ok {
				ex.FileItems[i].FolderID = &mapped
			} else {
				ex.FileItems[i].FolderID = nil
			}
		}
	}

	// Adaptive content units (if present) — remap structure FKs; units are not
	// applied by ApplyImport today, but keep the bundle consistent for future use.
	for i := range ex.AdaptiveContentUnits {
		u := &ex.AdaptiveContentUnits[i]
		if u.TargetModuleItemID != nil {
			m := mapOrSame(itemMap, *u.TargetModuleItemID)
			u.TargetModuleItemID = &m
		}
		u.BaseContentItemID = mapOrSame(itemMap, u.BaseContentItemID)
		if u.PreAssessmentItemID != nil {
			m := mapOrSame(itemMap, *u.PreAssessmentItemID)
			u.PreAssessmentItemID = &m
		}
		if u.PostAssessmentItemID != nil {
			m := mapOrSame(itemMap, *u.PostAssessmentItemID)
			u.PostAssessmentItemID = &m
		}
	}

	rewriteMarkdownRefsInBundle(ex, toolMapStr, fileMapStr)
}

func mapOrSame(m map[uuid.UUID]uuid.UUID, old uuid.UUID) uuid.UUID {
	if old == uuid.Nil {
		return old
	}
	if n, ok := m[old]; ok {
		return n
	}
	return old
}

func rewriteMarkdownRefsInBundle(ex *Bundle, toolMapStr, fileMapStr map[string]string) {
	rewrite := func(md string) string {
		if md == "" {
			return md
		}
		if len(toolMapStr) > 0 {
			md = ctsvc.RewriteLexToolFences(md, toolMapStr)
		}
		if len(fileMapStr) > 0 {
			md = rewriteFileIDsInText(md, fileMapStr)
		}
		return md
	}

	if ex.Course.HeroImageURL != nil {
		u := rewriteFileIDsInText(*ex.Course.HeroImageURL, fileMapStr)
		ex.Course.HeroImageURL = &u
	}
	for i := range ex.Syllabus {
		ex.Syllabus[i].Markdown = rewrite(ex.Syllabus[i].Markdown)
	}
	for id, body := range ex.ContentPages {
		body.Markdown = rewrite(body.Markdown)
		ex.ContentPages[id] = body
	}
	for id, body := range ex.Assignments {
		body.Markdown = rewrite(body.Markdown)
		ex.Assignments[id] = body
	}
	for id, body := range ex.Quizzes {
		body.Markdown = rewrite(body.Markdown)
		ex.Quizzes[id] = body
	}
}

// rewriteFileIDsInText replaces known file UUIDs in course-file / file-item URL paths.
func rewriteFileIDsInText(s string, fileMapStr map[string]string) string {
	if s == "" || len(fileMapStr) == 0 {
		return s
	}
	out := s
	for old, neu := range fileMapStr {
		if old == "" || neu == "" || old == neu {
			continue
		}
		// Path segments: /course-files/{id}/ and /files/items/{id}/
		out = strings.ReplaceAll(out, "/course-files/"+old, "/course-files/"+neu)
		out = strings.ReplaceAll(out, "/files/items/"+old, "/files/items/"+neu)
	}
	return out
}
