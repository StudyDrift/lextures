package platformconfig

import (
	"bytes"
	"testing"

	"github.com/lextures/lextures/server/internal/config"
	"github.com/lextures/lextures/server/internal/crypto/appsecrets"
)

func TestMerge_OpenRouterEmptyDBStaysEmpty(t *testing.T) {
	env := config.Config{OpenRouterAPIKey: "env-key"}
	db := Row{OpenRouterAPIKey: ptr("")}
	got := Merge(env, &db)
	if got.OpenRouterAPIKey != "" {
		t.Fatalf("OpenRouter: got %q want empty (not loaded from env)", got.OpenRouterAPIKey)
	}
}

func TestMerge_OpenRouterFromDB(t *testing.T) {
	env := config.Config{OpenRouterAPIKey: "env-key"}
	db := Row{OpenRouterAPIKey: ptr("db-key")}
	got := Merge(env, &db)
	if got.OpenRouterAPIKey != "db-key" {
		t.Fatalf("OpenRouter: got %q want db", got.OpenRouterAPIKey)
	}
}

func TestMerge_SMTPPasswordDecryptsFromDB(t *testing.T) {
	key := bytes.Repeat([]byte{11}, 32)
	blob, err := appsecrets.Encrypt([]byte("db-secret"), key)
	if err != nil {
		t.Fatal(err)
	}
	env := config.Config{SMTPHost: "h1", SMTPPassword: "envpw", PlatformSecretsKey: key}
	db := Row{SMTPPasswordCiphertext: blob}
	got := Merge(env, &db)
	if got.SMTPPassword != "db-secret" {
		t.Fatalf("SMTPPassword: got %q", got.SMTPPassword)
	}
	if got.SMTPHost != "h1" {
		t.Fatalf("SMTPHost: got %q", got.SMTPHost)
	}
}

func TestMerge_SMTPHostFromDB(t *testing.T) {
	env := config.Config{SMTPHost: "env-host", SMTPPort: 25}
	db := Row{SMTPHost: ptr("db-host")}
	got := Merge(env, &db)
	if got.SMTPHost != "db-host" || got.SMTPPort != 25 {
		t.Fatalf("got host=%q port=%d", got.SMTPHost, got.SMTPPort)
	}
}

func TestMerge_H5PFromDB(t *testing.T) {
	env := config.Config{}
	on := true
	got := Merge(env, &Row{H5PEnabled: &on})
	if !got.H5PEnabled {
		t.Fatal("expected H5P enabled from DB")
	}
}

func TestMerge_H5PDefaultOff(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if got.H5PEnabled {
		t.Fatal("expected H5P off when DB unset")
	}
}

func TestMerge_BookstoreIntegrationDefaultsOffWhenDBUnset(t *testing.T) {
	// Feature flags are DB-managed; a process-env/config value must not leak through when the
	// settings row is unset — the documented default (off) wins.
	got := Merge(config.Config{FFBookstoreIntegration: true}, nil)
	if got.FFBookstoreIntegration {
		t.Fatal("expected bookstore integration off (default) when DB unset, ignoring config/env")
	}
}

func TestMerge_BookstoreIntegrationDBOverridesEnv(t *testing.T) {
	off := false
	got := Merge(config.Config{FFBookstoreIntegration: true}, &Row{FFBookstoreIntegration: &off})
	if got.FFBookstoreIntegration {
		t.Fatal("expected DB false to override env true")
	}
}

// Plan MKT1 AC-1: FFCourseMarketplace defaults ON when platform settings row is unset.
func TestMerge_CourseMarketplaceDefaultOnWhenDBUnset(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if !got.FFCourseMarketplace {
		t.Fatal("expected FFCourseMarketplace true (default ON) when DB unset")
	}
	// Env/config seed must not win over the documented default when DB is unset.
	got = Merge(config.Config{FFCourseMarketplace: false}, nil)
	if !got.FFCourseMarketplace {
		t.Fatal("expected default ON to override env false when DB unset")
	}
}

func TestMerge_CourseMarketplaceDBOverridesDefault(t *testing.T) {
	off := false
	got := Merge(config.Config{}, &Row{FFCourseMarketplace: &off})
	if got.FFCourseMarketplace {
		t.Fatal("expected DB false to disable course marketplace")
	}
	on := true
	got = Merge(config.Config{}, &Row{FFCourseMarketplace: &on})
	if !got.FFCourseMarketplace {
		t.Fatal("expected DB true to enable course marketplace")
	}
}

func TestMerge_EmailSESDefaultOff(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if got.FFEmailSES {
		t.Fatal("expected FFEmailSES false (default OFF) when DB unset")
	}
	if got.EmailProvider != "" && got.EmailProvider != "smtp" {
		// Empty env leaves EmailProvider empty; runtime normalizes to smtp.
		t.Fatalf("unexpected EmailProvider %q", got.EmailProvider)
	}
	on := true
	prov := "ses"
	region := "us-west-2"
	from := "noreply@example.edu"
	got = Merge(config.Config{EmailProvider: "smtp", SESRegion: "us-east-1"}, &Row{
		FFEmailSES:    &on,
		EmailProvider: &prov,
		SESRegion:     &region,
		SESFrom:       &from,
	})
	if !got.FFEmailSES {
		t.Fatal("expected FFEmailSES true from DB")
	}
	if got.EmailProvider != "ses" || got.SESRegion != "us-west-2" || got.SESFrom != "noreply@example.edu" {
		t.Fatalf("got provider=%q region=%q from=%q", got.EmailProvider, got.SESRegion, got.SESFrom)
	}
}

// Plan FB0: FFFeedback defaults ON when platform settings row is unset.
func TestMerge_FeedbackDefaultOnWhenDBUnset(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if !got.FFFeedback {
		t.Fatal("expected FFFeedback true (default ON) when DB unset")
	}
	off := false
	got = Merge(config.Config{}, &Row{FFFeedback: &off})
	if got.FFFeedback {
		t.Fatal("expected DB false to disable feedback")
	}
}

// Collaboration boards are course-scoped only; platform master is always on (ignored).
func TestMerge_VisualBoardsAlwaysOn(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if !got.FFVisualBoards {
		t.Fatal("expected FFVisualBoards true (always on; course flag is the gate)")
	}
	off := false
	got = Merge(config.Config{}, &Row{FFVisualBoards: &off})
	if !got.FFVisualBoards {
		t.Fatal("expected FFVisualBoards true even when DB stores false")
	}
}

// Plan VC.6: FFBoardsExternalSharing defaults OFF when platform settings row is unset.
func TestMerge_FFBoardsExternalSharingDefaultOff(t *testing.T) {
	got := Merge(config.Config{}, &Row{})
	if got.FFBoardsExternalSharing {
		t.Fatal("expected FFBoardsExternalSharing false (default OFF) when DB unset")
	}
	on := true
	got = Merge(config.Config{}, &Row{FFBoardsExternalSharing: &on})
	if !got.FFBoardsExternalSharing {
		t.Fatal("expected FFBoardsExternalSharing true when DB set")
	}
}

// Plan VC.4: FFBoardsRealtime defaults ON when platform settings row is unset.
func TestMerge_BoardsRealtimeDefaultOnWhenDBUnset(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if !got.FFBoardsRealtime {
		t.Fatal("expected FFBoardsRealtime true (default ON) when DB unset")
	}
	off := false
	got = Merge(config.Config{}, &Row{FFBoardsRealtime: &off})
	if got.FFBoardsRealtime {
		t.Fatal("expected explicit DB false to disable boards realtime")
	}
	on := true
	got = Merge(config.Config{}, &Row{FFBoardsRealtime: &on})
	if !got.FFBoardsRealtime {
		t.Fatal("expected DB true to enable boards realtime")
	}
}

// Plan IQ.1: FFInteractiveQuizzes defaults OFF when platform settings row is unset.
func TestMerge_InteractiveQuizzesDefaultOffWhenDBUnset(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if got.FFInteractiveQuizzes {
		t.Fatal("expected FFInteractiveQuizzes false (default OFF) when DB unset")
	}
	on := true
	got = Merge(config.Config{}, &Row{FFInteractiveQuizzes: &on})
	if !got.FFInteractiveQuizzes {
		t.Fatal("expected DB true to enable interactive quizzes")
	}
}

// Plan IQ.3: FFIqLiveHosting defaults OFF when platform settings row is unset.
func TestMerge_IqLiveHostingDefaultOffWhenDBUnset(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if got.FFIqLiveHosting {
		t.Fatal("expected FFIqLiveHosting false (default OFF) when DB unset")
	}
	on := true
	got = Merge(config.Config{}, &Row{FFIqLiveHosting: &on})
	if !got.FFIqLiveHosting {
		t.Fatal("expected DB true to enable iq live hosting")
	}
}

// Plan IQ.6: mode sub-flags default OFF when platform settings row is unset.
func TestMerge_IqModeFlagsDefaultOffWhenDBUnset(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if got.FFIqTeamMode || got.FFIqStudentPaced || got.FFIqHomework {
		t.Fatal("expected IQ.6 mode flags false when DB unset")
	}
	on := true
	got = Merge(config.Config{}, &Row{FFIqTeamMode: &on, FFIqStudentPaced: &on, FFIqHomework: &on})
	if !got.FFIqTeamMode || !got.FFIqStudentPaced || !got.FFIqHomework {
		t.Fatal("expected DB true to enable IQ.6 mode flags")
	}
}

// Plan IQ.7: gradebook push sub-flag defaults OFF when platform settings row is unset.
func TestMerge_IqGradebookPushDefaultOffWhenDBUnset(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if got.FFIqGradebookPush {
		t.Fatal("expected FFIqGradebookPush false when DB unset")
	}
	on := true
	got = Merge(config.Config{}, &Row{FFIqGradebookPush: &on})
	if !got.FFIqGradebookPush {
		t.Fatal("expected DB true to enable IQ.7 gradebook push")
	}
}

// Plan IQ.8: public kit catalog defaults OFF when platform settings row is unset.
func TestMerge_IqPublicKitCatalogDefaultOffWhenDBUnset(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if got.FFIqPublicKitCatalog {
		t.Fatal("expected FFIqPublicKitCatalog false when DB unset")
	}
	on := true
	got = Merge(config.Config{}, &Row{FFIqPublicKitCatalog: &on})
	if !got.FFIqPublicKitCatalog {
		t.Fatal("expected DB true to enable IQ.8 public kit catalog")
	}
}

// Plan IQ.9: guest join defaults OFF when platform settings row is unset.
func TestMerge_IqGuestJoinDefaultOffWhenDBUnset(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if got.FFIqGuestJoin {
		t.Fatal("expected FFIqGuestJoin false when DB unset")
	}
	on := true
	got = Merge(config.Config{}, &Row{FFIqGuestJoin: &on})
	if !got.FFIqGuestJoin {
		t.Fatal("expected DB true to enable IQ.9 guest join")
	}
}

// Plan IQ.10: AI generation defaults OFF when platform settings row is unset.
func TestMerge_IqAiGenerationDefaultOffWhenDBUnset(t *testing.T) {
	got := Merge(config.Config{}, nil)
	if got.FFIqAiGeneration {
		t.Fatal("expected FFIqAiGeneration false when DB unset")
	}
	on := true
	got = Merge(config.Config{}, &Row{FFIqAiGeneration: &on})
	if !got.FFIqAiGeneration {
		t.Fatal("expected DB true to enable IQ.10 AI generation")
	}
}

func ptr(s string) *string { return &s }
