package introcourse

import "testing"

func TestLoadSyllabus_EnglishFiveSections(t *testing.T) {
	syllabus, err := LoadSyllabus("en")
	if err != nil {
		t.Fatal(err)
	}
	if len(syllabus.Sections) != 5 {
		t.Fatalf("expected 5 sections, got %d", len(syllabus.Sections))
	}
	if err := ValidateSyllabus(syllabus); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSyllabusMerged_SpanishOverlay(t *testing.T) {
	merged, err := LoadSyllabusMerged("es")
	if err != nil {
		t.Fatal(err)
	}
	if merged.Sections[0].Heading != "Descripción del curso" {
		t.Fatalf("overview heading: got %q", merged.Sections[0].Heading)
	}
}

func TestLocalizeSyllabus_SpanishOverlay(t *testing.T) {
	en, err := LoadSyllabus("en")
	if err != nil {
		t.Fatal(err)
	}
	localized := LocalizeSyllabus(en.Sections, "es")
	if localized[0].Heading != "Descripción del curso" {
		t.Fatalf("heading: got %q", localized[0].Heading)
	}
	if localized[0].Markdown == en.Sections[0].Markdown {
		t.Fatal("expected Spanish markdown overlay")
	}
}

func TestValidateSyllabus_RejectsExternalLinks(t *testing.T) {
	syllabus, err := LoadSyllabus("en")
	if err != nil {
		t.Fatal(err)
	}
	syllabus.Sections[0].Markdown += "\n[bad](https://evil.example)"
	if err := ValidateSyllabus(syllabus); err == nil {
		t.Fatal("expected validation error for external link")
	}
}