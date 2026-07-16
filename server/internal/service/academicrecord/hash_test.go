package academicrecord

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestContentHash_Stable(t *testing.T) {
	gen := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	params := AssembleParams{
		UserID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Variant:         VariantUnofficial,
		InstitutionName: "Test University",
		StudentName:     "Ada Lovelace",
		StudentID:       "S100",
		Scale:           ScaleFourPoint,
		GeneratedAt:     gen,
	}
	gradeA := "A"
	gradeB := "B"
	termID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	termName := "Fall 2025"
	start := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	rows := []CourseRow{
		{
			CourseID: uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			CourseCode: "MATH101", Title: "Calc I", Credits: 3, FinalGrade: &gradeA,
			TermID: &termID, TermName: &termName, TermStart: &start, EnrollmentState: "active",
		},
		{
			CourseID: uuid.MustParse("44444444-4444-4444-4444-444444444444"),
			CourseCode: "ENG101", Title: "Comp", Credits: 3, FinalGrade: &gradeB,
			TermID: &termID, TermName: &termName, TermStart: &start, EnrollmentState: "active",
		},
	}
	r1, err := Assemble(params, rows)
	if err != nil {
		t.Fatal(err)
	}
	r2, err := Assemble(params, rows)
	if err != nil {
		t.Fatal(err)
	}
	h1, _, err := ContentHash(r1)
	if err != nil {
		t.Fatal(err)
	}
	h2, _, err := ContentHash(r2)
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("hash not stable: %s vs %s", h1, h2)
	}
	ok, err := VerifyHash(r1, h1)
	if err != nil || !ok {
		t.Fatalf("verify failed: ok=%v err=%v", ok, err)
	}
	r1.Student.Name = "Changed"
	ok, err = VerifyHash(r1, h1)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("tampered record should fail hash verify")
	}
}

func TestAssemble_UnofficialIncludesInProgress(t *testing.T) {
	params := AssembleParams{
		Variant:         VariantUnofficial,
		InstitutionName: "U",
		StudentName:     "S",
		GeneratedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	rows := []CourseRow{
		{CourseCode: "BIO1", Title: "Bio", Credits: 4, EnrollmentState: "active"},
	}
	rec, err := Assemble(params, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Terms) != 1 || len(rec.Terms[0].Courses) != 1 {
		t.Fatalf("want 1 course, got %+v", rec.Terms)
	}
	if !rec.Terms[0].Courses[0].InProgress || rec.Terms[0].Courses[0].Grade != "IP" {
		t.Fatalf("want IP line, got %+v", rec.Terms[0].Courses[0])
	}
}

func TestAssemble_OfficialOmitsInProgress(t *testing.T) {
	params := AssembleParams{
		Variant:         VariantOfficial,
		InstitutionName: "U",
		StudentName:     "S",
		GeneratedAt:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	grade := "A"
	rows := []CourseRow{
		{CourseCode: "BIO1", Title: "Bio", Credits: 4, EnrollmentState: "active"},
		{CourseCode: "CHEM1", Title: "Chem", Credits: 3, FinalGrade: &grade, EnrollmentState: "active"},
	}
	rec, err := Assemble(params, rows)
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.Terms) != 1 || len(rec.Terms[0].Courses) != 1 {
		t.Fatalf("official should omit IP, got %+v", rec.Terms)
	}
	if rec.Terms[0].Courses[0].Code != "CHEM1" {
		t.Fatalf("want CHEM1, got %s", rec.Terms[0].Courses[0].Code)
	}
}
