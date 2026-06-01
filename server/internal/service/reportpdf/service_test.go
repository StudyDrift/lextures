package reportpdf

import (
	"testing"
	"time"
)

func TestBuildGradebookPDF_NonEmpty(t *testing.T) {
	in := GradebookInput{
		InstitutionName: "Test University",
		CourseName:      "Introduction to Go",
		CourseCode:      "CS101",
		GeneratedAt:     time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC),
		Students: []GradebookRow{
			{DisplayName: "Alice Smith", FinalGrade: "A", GradePercent: 95.5},
			{DisplayName: "Bob Jones", FinalGrade: "B+", GradePercent: 88.2},
		},
	}
	b, err := BuildGradebookPDF(in)
	if err != nil {
		t.Fatalf("BuildGradebookPDF: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}
	// PDF files start with %PDF-
	if string(b[:5]) != "%PDF-" {
		t.Errorf("output does not start with PDF header, got: %q", string(b[:10]))
	}
}

func TestBuildGradebookPDF_Empty(t *testing.T) {
	in := GradebookInput{
		CourseName:  "Empty Course",
		CourseCode:  "EMPTY",
		GeneratedAt: time.Now().UTC(),
	}
	b, err := BuildGradebookPDF(in)
	if err != nil {
		t.Fatalf("BuildGradebookPDF empty: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty PDF bytes even with no students")
	}
}

func TestBuildProgressPDF_NonEmpty(t *testing.T) {
	in := ProgressInput{
		InstitutionName: "Test University",
		CourseName:      "Introduction to Go",
		CourseCode:      "CS101",
		StudentName:     "Alice Smith",
		GeneratedAt:     time.Now().UTC(),
		CompletionPct:   75.0,
		Activities: []ProgressActivity{
			{ItemTitle: "Week 1 Quiz", ItemType: "quiz", Status: "completed", Grade: "A"},
			{ItemTitle: "Week 2 Assignment", ItemType: "assignment", Status: "submitted", Grade: "B+"},
		},
	}
	b, err := BuildProgressPDF(in)
	if err != nil {
		t.Fatalf("BuildProgressPDF: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}
	if string(b[:5]) != "%PDF-" {
		t.Errorf("output does not start with PDF header")
	}
}

func TestBuildLearningActivityPDF_NonEmpty(t *testing.T) {
	now := time.Now().UTC()
	in := LearningActivityInput{
		InstitutionName: "Test University",
		GeneratedAt:     now,
		From:            now.AddDate(0, 0, -30),
		To:              now,
		TotalEvents:     1500,
		UniqueUsers:     80,
		UniqueCourses:   12,
		ByDay: []LearningActivityDay{
			{Day: "2026-04-01", TotalEvents: 50},
			{Day: "2026-04-02", TotalEvents: 75},
		},
	}
	b, err := BuildLearningActivityPDF(in)
	if err != nil {
		t.Fatalf("BuildLearningActivityPDF: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}
	if string(b[:5]) != "%PDF-" {
		t.Errorf("output does not start with PDF header")
	}
}

func TestBuildLearningActivityPDF_Empty(t *testing.T) {
	now := time.Now().UTC()
	b, err := BuildLearningActivityPDF(LearningActivityInput{
		GeneratedAt: now,
		From:        now.AddDate(0, 0, -7),
		To:          now,
	})
	if err != nil {
		t.Fatalf("BuildLearningActivityPDF empty: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}
}

// ── Report Card PDF tests (plan 13.4) ─────────────────────────────────────────

func TestBuildReportCardPDF_WithComment(t *testing.T) {
	pct := 94.5
	letter := "A"
	comment := "Shows strong effort and improvement throughout the quarter."
	in := ReportCardInput{
		InstitutionName: "Springfield Elementary",
		CourseName:      "Mathematics Grade 5",
		CourseCode:      "MATH5",
		GradingPeriod:   "Q1-2026",
		GeneratedAt:     time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC),
		Student: ReportCardStudent{
			DisplayName:   "Alice Smith",
			StudentID:     "STU-001",
			FinalGradePct: &pct,
			LetterGrade:   letter,
			Comment:       comment,
			Absences:      2,
		},
	}
	b, err := BuildReportCardPDF(in)
	if err != nil {
		t.Fatalf("BuildReportCardPDF: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty PDF bytes")
	}
	if string(b[:5]) != "%PDF-" {
		t.Errorf("output does not start with PDF header, got: %q", string(b[:10]))
	}
}

func TestBuildReportCardPDF_NoGrade(t *testing.T) {
	in := ReportCardInput{
		CourseName:    "Science",
		CourseCode:    "SCI",
		GradingPeriod: "S1-2026",
		GeneratedAt:   time.Now().UTC(),
		Student: ReportCardStudent{
			DisplayName: "Bob Jones",
		},
	}
	b, err := BuildReportCardPDF(in)
	if err != nil {
		t.Fatalf("BuildReportCardPDF no grade: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("expected non-empty PDF bytes even without grade data")
	}
}
