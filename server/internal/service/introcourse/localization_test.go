package introcourse

import "testing"

func TestLoadCurriculumMerged_SpanishFallback(t *testing.T) {
	merged, err := LoadCurriculumMerged("es")
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Modules) < 7 {
		t.Fatalf("modules: got %d want >= 7", len(merged.Modules))
	}
	var found bool
	for _, mod := range merged.Modules {
		if mod.Meta.Slug == "m1.welcome" {
			found = true
			if mod.Meta.Title != "Bienvenida y orientación" {
				t.Fatalf("module title: got %q", mod.Meta.Title)
			}
			for _, p := range mod.Pages {
				if p.Slug == "m1.welcome.what-is-lextures" && p.Title != "¿Qué es Lextures?" {
					t.Fatalf("page title: got %q", p.Title)
				}
			}
		}
	}
	if !found {
		t.Fatal("m1.welcome module missing from merged curriculum")
	}
}

func TestLocalizePage_EnglishPassthrough(t *testing.T) {
	title, md := LocalizePage("m1.welcome.what-is-lextures", "What is Lextures?", "English body", "en")
	if title != "What is Lextures?" || md != "English body" {
		t.Fatalf("unexpected localization: %q / %q", title, md)
	}
}

func TestLocalizePage_SpanishOverlay(t *testing.T) {
	InvalidateLocaleIndex()
	title, md := LocalizePage("m1.welcome.what-is-lextures", "What is Lextures?", "English body", "es")
	if title != "¿Qué es Lextures?" {
		t.Fatalf("title: got %q", title)
	}
	if md == "" || md == "English body" {
		t.Fatalf("expected Spanish markdown overlay, got %q", md)
	}
}

func TestListContentLocales_IncludesEnglishAndSpanish(t *testing.T) {
	locales, err := ListContentLocales()
	if err != nil {
		t.Fatal(err)
	}
	hasEn, hasEs := false, false
	for _, loc := range locales {
		switch loc {
		case "en":
			hasEn = true
		case "es":
			hasEs = true
		}
	}
	if !hasEn || !hasEs {
		t.Fatalf("locales: %v", locales)
	}
}