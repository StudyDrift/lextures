package grading

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestFormulaSafe_PrefixesFormulaChars(t *testing.T) {
	for _, in := range []string{"=SUM(A1:A10)", "+1", "-2", "@cmd"} {
		got := FormulaSafe(in)
		if !strings.HasPrefix(got, "'") {
			t.Fatalf("FormulaSafe(%q) = %q, want leading apostrophe", in, got)
		}
	}
	if FormulaSafe("Ada") != "Ada" {
		t.Fatal("plain text should be unchanged")
	}
}

func TestBuildGradebookCSV_RoundTripAndColumns(t *testing.T) {
	s1, s2, s3 := uuid.New(), uuid.New(), uuid.New()
	a1, a2 := uuid.New(), uuid.New()
	max10, max100 := 10, 100
	book := CSVBook{
		Students: []CSVStudent{
			{ID: s1, Name: "Ada Lovelace", Email: "ada@example.com"},
			{ID: s2, Name: "=cmd|'/c calc", Email: "bob@example.com"},
			{ID: s3, Name: "Grace Hopper", Email: "grace@example.com"},
		},
		Columns: []CSVColumn{
			{ID: a1, Title: "Quiz 1", MaxPoints: &max10},
			{ID: a2, Title: "Exam", MaxPoints: &max100},
		},
		Grades: map[uuid.UUID]map[uuid.UUID]string{
			s1: {a1: "8", a2: "90"},
			s2: {a1: "10"},
			s3: {},
		},
		Display: map[uuid.UUID]map[uuid.UUID]string{
			s1: {a1: "80%", a2: "A"},
			s2: {a1: "100%"},
		},
	}
	score, grade := StraightFinals(book)
	book.FinalScore = score
	book.FinalGrade = grade

	raw, err := BuildGradebookCSV(book)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 3 || raw[0] != 0xEF || raw[1] != 0xBB || raw[2] != 0xBF {
		t.Fatal("export missing UTF-8 BOM")
	}
	text := string(raw[3:])
	if !strings.Contains(text, "__lextures__,1") && !strings.Contains(text, "__lextures__") {
		t.Fatalf("missing metadata marker:\n%s", text)
	}
	if !strings.Contains(text, "student_id") || !strings.Contains(text, "final_score") || !strings.Contains(text, "final_grade") {
		t.Fatalf("missing required columns:\n%s", text)
	}
	if !strings.Contains(text, a1.String()) || !strings.Contains(text, a2.String()) {
		t.Fatal("metadata row should include assignment ids")
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	// metadata + header + 3 students
	if len(lines) != 5 {
		t.Fatalf("got %d lines, want 5:\n%s", len(lines), text)
	}
	if !strings.Contains(text, "'=cmd|'/c calc") && !strings.Contains(text, `"'=cmd|'/c calc'"`) {
		t.Fatalf("formula injection not prefixed:\n%s", text)
	}
	if !strings.Contains(text, "8") || !strings.Contains(text, "90") {
		t.Fatalf("scores missing:\n%s", text)
	}

	again, err := ValidateGradebookCSV(string(raw), book)
	if err != nil {
		t.Fatal(err)
	}
	if again.Preview.Stats.Errors != 0 || again.Preview.Stats.Updated != 0 || again.Preview.Stats.Added != 0 {
		t.Fatalf("round trip should be unchanged, got %+v", again.Preview.Stats)
	}
	if again.Preview.Confirmable {
		t.Fatal("unchanged import should not be confirmable")
	}
}

func TestValidateGradebookCSV_UnknownStudentAndOutOfRange(t *testing.T) {
	student := uuid.New()
	item := uuid.New()
	max := 100
	book := CSVBook{
		Students: []CSVStudent{{ID: student, Name: "Ada", Email: "ada@example.com"}},
		Columns:  []CSVColumn{{ID: item, Title: "Exam", MaxPoints: &max}},
		Grades:   map[uuid.UUID]map[uuid.UUID]string{student: {item: "80"}},
	}
	unknown := uuid.New()
	bad := "student_id,Exam\n" +
		unknown.String() + ",90\n" +
		student.String() + ",150\n" +
		student.String() + ",=SUM(A1)\n"
	got, err := ValidateGradebookCSV(bad, book)
	if err != nil {
		t.Fatal(err)
	}
	if got.Preview.Stats.Errors < 2 {
		t.Fatalf("want student + formula errors, got %+v rows=%+v", got.Preview.Stats, got.Preview.Rows)
	}
	if got.Preview.Confirmable || got.Preview.Token != nil {
		t.Fatal("errors must block confirm")
	}
	if got.Preview.Stats.Warnings != 1 {
		t.Fatalf("want 1 out-of-range warning, got %+v", got.Preview.Stats)
	}
	foundRange := false
	for _, row := range got.Preview.Rows {
		for _, c := range row.Cells {
			if c.OutOfRange {
				foundRange = true
			}
		}
	}
	if !foundRange {
		t.Fatal("expected an out-of-range cell")
	}
}

func TestValidateGradebookCSV_UpdateIsConfirmable(t *testing.T) {
	student := uuid.New()
	item := uuid.New()
	max := 100
	book := CSVBook{
		Students: []CSVStudent{{ID: student, Name: "Ada", Email: "ada@example.com"}},
		Columns:  []CSVColumn{{ID: item, Title: "Exam", MaxPoints: &max, ManualHoldBlind: true}},
		Grades:   map[uuid.UUID]map[uuid.UUID]string{student: {item: "80"}},
	}
	csvText := "student_id,Exam\n" + student.String() + ",91\n"
	got, err := ValidateGradebookCSV(csvText, book)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Preview.Confirmable || got.Preview.Stats.Updated != 1 || got.Preview.Stats.Errors != 0 {
		t.Fatalf("want one confirmable update, got %+v", got.Preview.Stats)
	}
	if !got.RequireAck || !got.Preview.RequireBlindManualHoldAck {
		t.Fatal("manual-hold blind column should require acknowledgement")
	}
	if got.Pending[student.String()][item.String()] != "91" {
		t.Fatalf("pending = %#v", got.Pending)
	}
}
