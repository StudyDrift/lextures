package officepreview

import (
	"os"
	"strings"
	"testing"
)

func TestDocxSprintCardRendersStampBoxes(t *testing.T) {
	path := "../../../../data/course-files/managed-files/C-VIDVCN/1ab54df5-b727-46ea-820a-68450d38dcb2.docx"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("sample docx not available:", err)
	}
	html, err := ConvertToHTML(data, "sprint.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"CSE 199 Sprint Card",
		"Participation Stamps",
		"Day 1",
		"Planning Meeting",
		"Day 2",
		"Stand-Up Meeting",
		"Day 3",
		"Day 4",
		"Review Meeting",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Count(html, `class="docx-table"`) < 5 {
		t.Fatalf("expected nested stamp tables, got %d tables", strings.Count(html, `class="docx-table"`))
	}
	// Stamp boxes must render inside their cell, not before the page header.
	headerIdx := strings.Index(html, "CSE 199 Sprint Card")
	day1Idx := strings.Index(html, "Day 1")
	if day1Idx >= 0 && day1Idx < headerIdx {
		t.Fatalf("Day 1 stamp box rendered before page header (absolute positioning leak)")
	}
	stampIdx := strings.Index(html, "Participation Stamps")
	if stampIdx >= 0 && day1Idx >= 0 && day1Idx < stampIdx {
		t.Fatalf("Day 1 stamp box rendered before Participation Stamps header")
	}
}
