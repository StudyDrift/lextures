package academicrecord

import (
	"testing"
)

func TestGradePoints_FourPoint(t *testing.T) {
	cases := []struct {
		grade string
		pts   float64
		in    bool
	}{
		{"A", 4.0, true},
		{"A-", 3.7, true},
		{"B+", 3.3, true},
		{"B", 3.0, true},
		{"C", 2.0, true},
		{"F", 0.0, true},
		{"P", 0, false},
		{"W", 0, false},
		{"IP", 0, false},
	}
	for _, tc := range cases {
		pts, in := GradePoints(tc.grade, ScaleFourPoint)
		if pts != tc.pts || in != tc.in {
			t.Errorf("GradePoints(%q)=(%v,%v) want (%v,%v)", tc.grade, pts, in, tc.pts, tc.in)
		}
	}
}

func TestComputeCumulative_ThreeTermFixture(t *testing.T) {
	// Golden fixture: 3 terms, known GPA.
	// Fall: MATH 101 (3cr, A=4.0) → 12 QP; ENG 101 (3cr, B=3.0) → 9 QP; term GPA 3.5
	// Spring: HIST 201 (3cr, A-=3.7) → 11.1 QP; term GPA 3.7
	// Summer: CHEM 110 (4cr, B+=3.3) → 13.2 QP; term GPA 3.3
	// Cumulative: 45.3 QP / 13 cr = 3.485 → 3.485
	lines := []CourseLine{
		{Code: "MATH101", Title: "Calc I", CreditsAttempted: 3, CreditsEarned: 3, Grade: "A"},
		{Code: "ENG101", Title: "Comp", CreditsAttempted: 3, CreditsEarned: 3, Grade: "B"},
		{Code: "HIST201", Title: "World", CreditsAttempted: 3, CreditsEarned: 3, Grade: "A-"},
		{Code: "CHEM110", Title: "Chem", CreditsAttempted: 4, CreditsEarned: 4, Grade: "B+"},
	}
	cum := ComputeCumulative(lines, ScaleFourPoint)
	if cum.CreditsEarned != 13 {
		t.Fatalf("credits earned=%v want 13", cum.CreditsEarned)
	}
	if cum.GPA == nil || *cum.GPA != 3.485 {
		t.Fatalf("gpa=%v want 3.485", cum.GPA)
	}
	if cum.QualityPoints != 45.3 {
		t.Fatalf("qp=%v want 45.3", cum.QualityPoints)
	}
}

func TestCreditsEarnedForGrade(t *testing.T) {
	if CreditsEarnedForGrade("A", 3) != 3 {
		t.Fatal("A should earn credits")
	}
	if CreditsEarnedForGrade("F", 3) != 0 {
		t.Fatal("F should earn 0")
	}
	if CreditsEarnedForGrade("W", 3) != 0 {
		t.Fatal("W should earn 0")
	}
}
