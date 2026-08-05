package courseexportimport

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	modelcourse "github.com/lextures/lextures/server/internal/models/course"
	"github.com/lextures/lextures/server/internal/models/courseexport"
	"github.com/lextures/lextures/server/internal/repos/coursegrading"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

func TestRemapBundleIDsForCrossCourseImport_SameCourseNoop(t *testing.T) {
	groupID := uuid.New()
	modID := uuid.New()
	ex := &Bundle{
		FormatVersion: 1,
		CourseCode:    "C-SRC",
		Course:        courseexport.CourseExportSnapshot{Title: "T"},
		Grading: coursegrading.SettingsResponse{
			GradingScale: modelcourse.GradingScales[0],
			AssignmentGroups: []coursegrading.AssignmentGroupPublic{
				{ID: groupID, Name: "Homework", SortOrder: 1, WeightPercent: 100},
			},
		},
		Structure: []coursestructure.ItemResponse{
			{ID: modID.String(), Kind: "module", Title: "M1", SortOrder: 0, Published: true},
		},
	}
	remapBundleIDsForCrossCourseImport(ex, "C-SRC")
	if ex.Grading.AssignmentGroups[0].ID != groupID {
		t.Fatalf("same-course import must preserve assignment group id")
	}
	if ex.Structure[0].ID != modID.String() {
		t.Fatalf("same-course import must preserve structure id")
	}
}

func TestRemapBundleIDsForCrossCourseImport_RemapsAndRewrites(t *testing.T) {
	groupID := uuid.New()
	modID := uuid.New()
	pageID := uuid.New()
	parent := modID.String()
	groupStr := groupID.String()
	toolOld := uuid.New()
	fileOld := uuid.New()
	itemFileOld := uuid.New()

	ex := &Bundle{
		FormatVersion: 1,
		CourseCode:    "C-SRC",
		Course: courseexport.CourseExportSnapshot{
			Title: "T",
			HeroImageURL: func() *string {
				s := "/api/v1/courses/C-SRC/course-files/" + fileOld.String() + "/content"
				return &s
			}(),
		},
		Grading: coursegrading.SettingsResponse{
			GradingScale: modelcourse.GradingScales[0],
			AssignmentGroups: []coursegrading.AssignmentGroupPublic{
				{ID: groupID, Name: "Homework", SortOrder: 1, WeightPercent: 100},
			},
		},
		Structure: []coursestructure.ItemResponse{
			{ID: modID.String(), Kind: "module", Title: "M1", SortOrder: 0, Published: true},
			{
				ID: pageID.String(), Kind: "content_page", Title: "P1", SortOrder: 0, Published: true,
				ParentID: &parent, AssignmentGroupID: &groupStr,
			},
		},
		ContentPages: map[uuid.UUID]courseexport.ExportedContentPageBody{
			pageID: {
				Markdown: "See ```lex-tool\n{\"instanceId\":\"" + toolOld.String() + "\",\"toolId\":\"inline_questions\"}\n``` and /course-files/" + fileOld.String() + "/content",
			},
		},
		ContentToolInstances: []courseexport.ExportedContentToolInstance{
			{
				ID:              toolOld,
				StructureItemID: &pageID,
				HostKind:        "content_page",
				ToolID:          "inline_questions",
				ToolVersion:     "1.0.0",
				ConfigJSON:      []byte(`{}`),
				Status:          "active",
			},
		},
		CourseFiles: []courseexport.ExportedCourseFile{
			{ID: fileOld, OriginalFilename: "a.jpg", MimeType: "image/jpeg", ByteSize: 1},
		},
		FileItems: []courseexport.ExportedFileItem{
			{ID: itemFileOld, OriginalFilename: "b.jpg", DisplayName: "b.jpg", MimeType: "image/jpeg", ByteSize: 1},
		},
	}

	remapBundleIDsForCrossCourseImport(ex, "C-NEW")

	if ex.Grading.AssignmentGroups[0].ID == groupID {
		t.Fatalf("cross-course import must remap assignment group id")
	}
	if ex.Structure[0].ID == modID.String() {
		t.Fatalf("cross-course import must remap module id")
	}
	if ex.Structure[1].ID == pageID.String() {
		t.Fatalf("cross-course import must remap page id")
	}
	// parent + group FKs point at remapped ids
	if ex.Structure[1].ParentID == nil || *ex.Structure[1].ParentID != ex.Structure[0].ID {
		t.Fatalf("parent id not remapped: got %v want %s", ex.Structure[1].ParentID, ex.Structure[0].ID)
	}
	if ex.Structure[1].AssignmentGroupID == nil || *ex.Structure[1].AssignmentGroupID != ex.Grading.AssignmentGroups[0].ID.String() {
		t.Fatalf("assignment group fk not remapped: got %v", ex.Structure[1].AssignmentGroupID)
	}

	newPageID, err := uuid.Parse(ex.Structure[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ex.ContentPages[newPageID]; !ok {
		t.Fatalf("content page map not rekeyed to new page id")
	}
	if _, ok := ex.ContentPages[pageID]; ok {
		t.Fatalf("old content page key should be gone")
	}

	if ex.ContentToolInstances[0].ID == toolOld {
		t.Fatalf("content tool id not remapped")
	}
	if ex.ContentToolInstances[0].StructureItemID == nil || *ex.ContentToolInstances[0].StructureItemID != newPageID {
		t.Fatalf("content tool structureItemId not remapped")
	}

	md := ex.ContentPages[newPageID].Markdown
	if !strings.Contains(md, ex.ContentToolInstances[0].ID.String()) {
		t.Fatalf("markdown lex-tool fence not rewritten: %s", md)
	}
	if strings.Contains(md, toolOld.String()) {
		t.Fatalf("markdown still has old tool id")
	}
	if strings.Contains(md, fileOld.String()) {
		t.Fatalf("markdown still has old course-file id")
	}
	if !strings.Contains(md, ex.CourseFiles[0].ID.String()) {
		t.Fatalf("markdown missing new course-file id")
	}
	if ex.Course.HeroImageURL == nil || strings.Contains(*ex.Course.HeroImageURL, fileOld.String()) {
		t.Fatalf("hero url not rewritten: %v", ex.Course.HeroImageURL)
	}
	if ex.CourseFiles[0].ID == fileOld {
		t.Fatalf("course file id not remapped")
	}
	if ex.FileItems[0].ID == itemFileOld {
		t.Fatalf("file item id not remapped")
	}
}
