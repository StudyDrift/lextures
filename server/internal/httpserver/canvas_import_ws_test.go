package httpserver

import (
	"strings"
	"testing"
)

func TestCanvasEnrollmentTypeToRole_Teacher(t *testing.T) {
	for _, typ := range []string{"TeacherEnrollment", "teacher"} {
		if got := canvasEnrollmentTypeToRole(typ); got != "teacher" {
			t.Errorf("canvasEnrollmentTypeToRole(%q) = %q, want teacher", typ, got)
		}
	}
}

func TestCanvasEnrollmentTypeToRole_TA(t *testing.T) {
	for _, typ := range []string{"TaEnrollment", "ta", "head_ta"} {
		if got := canvasEnrollmentTypeToRole(typ); got != "instructor" {
			t.Errorf("canvasEnrollmentTypeToRole(%q) = %q, want instructor", typ, got)
		}
	}
}

func TestCanvasEnrollmentTypeToRole_OtherMapsToStudent(t *testing.T) {
	for _, typ := range []string{"StudentEnrollment", "DesignerEnrollment", "ObserverEnrollment", "", "unknown"} {
		if got := canvasEnrollmentTypeToRole(typ); got != "student" {
			t.Errorf("canvasEnrollmentTypeToRole(%q) = %q, want student", typ, got)
		}
	}
}

func TestCanvasImportInclude_WithDefaults_AllFalseGivesAll(t *testing.T) {
	got := (canvasImportInclude{}).withDefaults()
	want := canvasImportInclude{Modules: true, Assignments: true, Quizzes: true, Enrollments: true, Grades: true, Settings: true, Files: true}
	if got != want {
		t.Fatalf("withDefaults on zero include = %+v, want %+v", got, want)
	}
}

func TestCanvasImportInclude_WithDefaults_PartialUnchanged(t *testing.T) {
	partial := canvasImportInclude{Modules: true, Enrollments: true}
	if got := partial.withDefaults(); got != partial {
		t.Fatalf("withDefaults on partial should return as-is, got %+v", got)
	}
}

func TestCanvasImportInclude_WithDefaults_LegacyAllExceptFiles(t *testing.T) {
	legacy := canvasImportInclude{Modules: true, Assignments: true, Quizzes: true, Enrollments: true, Grades: true, Settings: true}
	got := legacy.withDefaults()
	if !got.Files {
		t.Fatalf("legacy all-true include should default Files=true, got %+v", got)
	}
}

func TestMarkdownFromHTML_ConvertsBasicFormatting(t *testing.T) {
	md := markdownFromHTML("<h2>Title</h2><p>Hello <strong>world</strong> and <a href=\"https://example.com\">link</a>.</p>")
	if !strings.Contains(md, "## Title") {
		t.Fatalf("expected heading markdown, got: %q", md)
	}
	if !strings.Contains(md, "**world**") {
		t.Fatalf("expected bold markdown, got: %q", md)
	}
	if !strings.Contains(md, "[link](https://example.com)") {
		t.Fatalf("expected link markdown, got: %q", md)
	}
}

func TestMarkdownFromHTML_CanvasEmbeddedFileIframe(t *testing.T) {
	html := `<p><iframe src="/courses/12345/files/67890/preview" title="doc.pdf" width="100%" height="600"></iframe></p>`
	md := markdownFromHTML(html)
	if md == "" {
		t.Fatal("iframe embed produced empty markdown")
	}
	if !strings.Contains(md, "doc.pdf") {
		t.Fatalf("expected embedded file title in markdown, got: %q", md)
	}
	if !strings.Contains(md, "/courses/12345/files/67890") {
		t.Fatalf("expected canvas file URL in markdown, got: %q", md)
	}
}

func TestMarkdownFromHTML_CanvasEmbeddedFileIframe_AbsoluteURL(t *testing.T) {
	html := `<iframe src="https://school.instructure.com/courses/7/files/99/preview" title="slides.pptx"></iframe>`
	md := markdownFromHTML(html)
	if !strings.Contains(md, "slides.pptx") {
		t.Fatalf("expected title in markdown, got: %q", md)
	}
	if !strings.Contains(md, "instructure.com/courses/7/files/99") {
		t.Fatalf("expected absolute canvas file URL in markdown, got: %q", md)
	}
}

func TestMapCanvasTypeToKind_FileIsContentPage(t *testing.T) {
	kind, body := mapCanvasTypeToKind("File")
	if kind != "content_page" || body != "content" {
		t.Fatalf("mapCanvasTypeToKind(File) = (%q, %q), want (content_page, content)", kind, body)
	}
}

func TestMarkdownFromHTML_CanvasFileLink(t *testing.T) {
	html := `<p><a class="instructure_file_link" href="/courses/12345/files/67890/download?wrap=1">doc.pdf</a></p>`
	md := markdownFromHTML(html)
	if !strings.Contains(md, "doc.pdf") {
		t.Fatalf("expected file link preserved, got: %q", md)
	}
	if !strings.Contains(md, "/courses/12345/files/67890") {
		t.Fatalf("expected canvas file URL in markdown, got: %q", md)
	}
}

func TestHTMLToPlainText_StripsTagsAndNormalizesBreaks(t *testing.T) {
	plain := htmlToPlainText("<p>One</p><p>Two<br/>Three</p>")
	if plain != "One\n\nTwo\nThree" {
		t.Fatalf("unexpected plain output: %q", plain)
	}
}
