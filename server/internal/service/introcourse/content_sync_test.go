package introcourse

import (
	"testing"

	"github.com/lextures/lextures/server/internal/config"
)

func TestLoadCurriculum_EnglishSevenModules(t *testing.T) {
	cur, err := LoadCurriculum("en")
	if err != nil {
		t.Fatal(err)
	}
	if len(cur.Modules) != 7 {
		t.Fatalf("expected 7 modules, got %d", len(cur.Modules))
	}
	if err := ValidateCurriculum(cur); err != nil {
		t.Fatal(err)
	}
}

func TestValidateCurriculum_RejectsScript(t *testing.T) {
	cur, err := LoadCurriculum("en")
	if err != nil {
		t.Fatal(err)
	}
	cur.Modules[0].Pages[0].Markdown += "\n<script>alert(1)</script>"
	if err := ValidateCurriculum(cur); err == nil {
		t.Fatal("expected validation error for script tag")
	}
}

func TestValidateCurriculum_RejectsExternalLinks(t *testing.T) {
	cur, err := LoadCurriculum("en")
	if err != nil {
		t.Fatal(err)
	}
	cur.Modules[0].Pages[0].Markdown += "\n[bad](https://evil.example)"
	if err := ValidateCurriculum(cur); err == nil {
		t.Fatal("expected validation error for external link")
	}
}

func TestFilterCurriculum_LearnerProfileDisabled(t *testing.T) {
	cur, err := LoadCurriculum("en")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{
		LearnerProfileEnabled:        false,
		PushNotificationsEnabled:     true,
		AdaptiveLearnerModelEnabled:  true,
		SRSPracticeEnabled:           true,
		DiagnosticAssessmentsEnabled: true,
		SelfReflectionEnabled:        true,
		AiDisclosureEnabled:          true,
	}
	mods := FilterCurriculum(cur, cfg)
	for _, mod := range mods {
		if mod.Meta.Slug == "m4.learner-profile" {
			t.Fatal("learner profile module should be omitted when flag is off")
		}
	}
	slugs := AllDesiredSlugs(cur, cfg)
	for _, s := range slugs {
		if s == "m4.learner-profile" || len(s) > 3 && s[:3] == "m4." {
			t.Fatalf("unexpected learner profile slug %q when flag off", s)
		}
	}
}

func TestFilterCurriculum_CanvasImportEnabled(t *testing.T) {
	cfg := config.Config{}
	if !FlagEnabled(cfg, "canvas_import_enabled") || !CanvasImportEnabled(cfg) {
		t.Fatal("canvas import should be available in default deployments")
	}
}

func TestFlagEnabled_KnownFlags(t *testing.T) {
	cfg := config.Config{
		LearnerProfileEnabled: false,
		SelfReflectionEnabled: true,
	}
	if FlagEnabled(cfg, "learner_profile_enabled") {
		t.Fatal("expected false")
	}
	if !FlagEnabled(cfg, "self_reflection_enabled") {
		t.Fatal("expected true")
	}
	if !FlagEnabled(cfg, "") {
		t.Fatal("empty flag should pass")
	}
}

func TestSplitFrontMatter(t *testing.T) {
	fm, body, err := splitFrontMatter("---\nslug: test\n---\n\nHello")
	if err != nil {
		t.Fatal(err)
	}
	if fm["slug"] != "test" || body != "Hello" {
		t.Fatalf("got fm=%v body=%q", fm, body)
	}
}