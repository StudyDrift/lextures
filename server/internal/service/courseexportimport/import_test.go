package courseexportimport

import (
	"testing"

	"github.com/google/uuid"

	modelcourse "github.com/lextures/lextures/server/internal/models/course"
	"github.com/lextures/lextures/server/internal/models/courseexport"
	"github.com/lextures/lextures/server/internal/models/coursesyllabus"
	"github.com/lextures/lextures/server/internal/repos/coursegrading"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

func TestValidateExportPayload_OK(t *testing.T) {
	modID := uuid.New()
	ex := &Bundle{
		FormatVersion: 1,
		CourseCode:    "C-SRC",
		Course:        courseexport.CourseExportSnapshot{Title: "T", ScheduleMode: "fixed", CourseType: "traditional"},
		Grading: coursegrading.SettingsResponse{
			GradingScale: modelcourse.GradingScales[0],
			AssignmentGroups: []coursegrading.AssignmentGroupPublic{
				{ID: uuid.New(), Name: "Homework", SortOrder: 1, WeightPercent: 100},
			},
		},
		Syllabus: []coursesyllabus.SyllabusSection{},
		Structure: []coursestructure.ItemResponse{
			{ID: modID.String(), Kind: "module", Title: "M1", SortOrder: 0, Published: true},
		},
		ContentPages: map[uuid.UUID]courseexport.ExportedContentPageBody{},
		Assignments:  map[uuid.UUID]courseexport.ExportedAssignmentBody{},
		Quizzes:      map[uuid.UUID]courseexport.ExportedQuizBody{},
		Enrollments:  nil,
	}
	if err := validateExportPayload(ex); err != nil {
		t.Fatalf("validateExportPayload: %v", err)
	}
}

func TestValidateExportPayload_BadVersion(t *testing.T) {
	ex := &Bundle{
		FormatVersion: 99,
		CourseCode:    "C-SRC",
		Grading:       coursegrading.SettingsResponse{GradingScale: modelcourse.GradingScales[0]},
	}
	err := validateExportPayload(ex)
	if !IsInvalidInput(err) {
		t.Fatalf("want InvalidInput, got %v", err)
	}
}

func TestInvalidInputMessage(t *testing.T) {
	err := InvalidInput("hello")
	if got := InvalidInputMessage(err); got != "hello" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateContentToolsExport_OK(t *testing.T) {
	itemID := uuid.New()
	instID := uuid.New()
	ex := &Bundle{
		FormatVersion: 1,
		CourseCode:    "C-SRC",
		Course:        courseexport.CourseExportSnapshot{Title: "T", ScheduleMode: "fixed", CourseType: "traditional"},
		Grading: coursegrading.SettingsResponse{
			GradingScale: modelcourse.GradingScales[0],
			AssignmentGroups: []coursegrading.AssignmentGroupPublic{
				{ID: uuid.New(), Name: "Homework", SortOrder: 1, WeightPercent: 100},
			},
		},
		Syllabus: []coursesyllabus.SyllabusSection{},
		Structure: []coursestructure.ItemResponse{
			{ID: uuid.New().String(), Kind: "module", Title: "M1", SortOrder: 0, Published: true},
		},
		ContentPages: map[uuid.UUID]courseexport.ExportedContentPageBody{},
		Assignments:  map[uuid.UUID]courseexport.ExportedAssignmentBody{},
		Quizzes:      map[uuid.UUID]courseexport.ExportedQuizBody{},
		ContentToolSettings: &courseexport.ExportedContentToolSettings{
			AllowedToolIDs:      []string{"flashcards"},
			MaxInstancesPerItem: 50,
			LinkIngestionMode:   "public",
		},
		ContentToolInstances: []courseexport.ExportedContentToolInstance{
			{
				ID:                  instID,
				StructureItemID:     &itemID,
				HostKind:            "content_page",
				ToolID:              "flashcards",
				ToolVersion:         "1.0.0",
				ConfigJSON:          []byte(`{"cards":[]}`),
				ConfigSchemaVersion: 1,
				Status:              "active",
			},
		},
	}
	if err := validateExportPayload(ex); err != nil {
		t.Fatalf("validateExportPayload: %v", err)
	}
}

func TestValidateContentToolsExport_BadHost(t *testing.T) {
	ex := &Bundle{
		FormatVersion: 1,
		CourseCode:    "C-SRC",
		Grading:       coursegrading.SettingsResponse{GradingScale: modelcourse.GradingScales[0]},
		ContentToolInstances: []courseexport.ExportedContentToolInstance{
			{
				ID:          uuid.New(),
				HostKind:    "not_a_host",
				ToolID:      "flashcards",
				ToolVersion: "1.0.0",
			},
		},
	}
	err := validateExportPayload(ex)
	if !IsInvalidInput(err) {
		t.Fatalf("want InvalidInput, got %v", err)
	}
}
