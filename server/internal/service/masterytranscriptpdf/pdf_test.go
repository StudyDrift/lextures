package masterytranscriptpdf

import (
	"bytes"
	"compress/zlib"
	"io"
	"strings"
	"testing"
	"time"
)

func pdfPlain(t *testing.T, b []byte) string {
	t.Helper()
	var out strings.Builder
	rest := b
	for {
		i := bytes.Index(rest, []byte("stream\n"))
		if i < 0 {
			break
		}
		rest = rest[i+len("stream\n"):]
		j := bytes.Index(rest, []byte("\nendstream"))
		if j < 0 {
			break
		}
		raw := rest[:j]
		rest = rest[j+len("\nendstream"):]
		r, err := zlib.NewReader(bytes.NewReader(raw))
		if err != nil {
			continue
		}
		decoded, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			continue
		}
		out.Write(decoded)
	}
	if out.Len() == 0 {
		t.Fatal("no decodable PDF streams")
	}
	return out.String()
}

func TestBuild_ContainsCourseAndStudent(t *testing.T) {
	when := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	assessed := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	b, err := Build(Input{
		CourseTitle: "AI Essentials",
		CourseCode:  "C-HUPCNF",
		StudentName: "Ahmed Dhieb",
		GeneratedAt: when,
		Rows: []Row{{
			Code:         "CCSS.MATH.3.OA.A.1",
			Description:  "Interpret products of whole numbers.",
			Level:        "Meets",
			LastAssessed: &assessed,
		}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(b) < 5 || string(b[:5]) != "%PDF-" {
		t.Fatalf("expected PDF header, got %q", b[:min(12, len(b))])
	}
	body := pdfPlain(t, b)
	for _, want := range []string{"Mastery Transcript", "AI Essentials", "C-HUPCNF", "Ahmed Dhieb", "CCSS.MATH.3.OA.A.1", "Meets"} {
		if !strings.Contains(body, want) {
			t.Errorf("PDF missing %q", want)
		}
	}
}

func TestBuild_EmptyStandards(t *testing.T) {
	b, err := Build(Input{CourseCode: "EMPTY", StudentName: "Test Student"})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(b) < 5 || string(b[:5]) != "%PDF-" {
		t.Fatal("expected PDF header")
	}
	if !strings.Contains(pdfPlain(t, b), "No standards are attached to this course.") {
		t.Fatal("expected empty-standards sentence")
	}
}
