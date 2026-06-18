package httpserver

import (
	"strings"
	"testing"
)

func TestCanvasAvatarURLFromMaps_prefersUserObject(t *testing.T) {
	enrollment := map[string]any{"avatar_url": "https://canvas.example/enrollment.png"}
	user := map[string]any{"avatar_url": "https://canvas.example/user.png"}
	if got := canvasAvatarURLFromMaps(enrollment, user); got != "https://canvas.example/user.png" {
		t.Fatalf("canvasAvatarURLFromMaps() = %q, want user avatar", got)
	}
}

func TestCanvasAvatarURLFromMaps_fallsBackToEnrollment(t *testing.T) {
	enrollment := map[string]any{"avatar_url": "https://canvas.example/enrollment.png"}
	if got := canvasAvatarURLFromMaps(enrollment, nil); got != "https://canvas.example/enrollment.png" {
		t.Fatalf("canvasAvatarURLFromMaps() = %q, want enrollment avatar", got)
	}
}

func TestCanvasImageBytesToDataURL_encodesImage(t *testing.T) {
	data := []byte{0xff, 0xd8, 0xff}
	got, err := canvasImageBytesToDataURL(data, "image/jpeg")
	if err != nil {
		t.Fatalf("canvasImageBytesToDataURL: %v", err)
	}
	if !strings.HasPrefix(got, "data:image/jpeg;base64,") {
		t.Fatalf("unexpected data url prefix: %q", got)
	}
}

func TestCanvasImageBytesToDataURL_rejectsNonImage(t *testing.T) {
	if _, err := canvasImageBytesToDataURL([]byte("x"), "text/plain"); err == nil {
		t.Fatal("expected non-image content type to fail")
	}
}

func TestCanvasIsDefaultAvatarURL_detectsCanvasPlaceholder(t *testing.T) {
	if !canvasIsDefaultAvatarURL("https://canvas.example.edu/images/messages/avatar-50.png") {
		t.Fatal("expected Canvas default avatar URL to be treated as blank")
	}
	if canvasIsDefaultAvatarURL("https://canvas.example.edu/files/12345/download?ver=1") {
		t.Fatal("custom Canvas file URL should not be treated as blank")
	}
}

func TestCanvasCanvasUserIDFromEnrollment_prefersEmbeddedUser(t *testing.T) {
	enrollment := map[string]any{"user_id": float64(99)}
	user := map[string]any{"id": float64(42)}
	if got := canvasCanvasUserIDFromEnrollment(enrollment, user); got != 42 {
		t.Fatalf("canvasCanvasUserIDFromEnrollment() = %d, want 42", got)
	}
}

func TestCanvasCanvasUserIDFromEnrollment_fallsBackToEnrollmentUserID(t *testing.T) {
	enrollment := map[string]any{"user_id": float64(77)}
	if got := canvasCanvasUserIDFromEnrollment(enrollment, nil); got != 77 {
		t.Fatalf("canvasCanvasUserIDFromEnrollment() = %d, want 77", got)
	}
}

func TestCanvasResolveProvisioningEmail_usesRosterEmail(t *testing.T) {
	got := canvasResolveProvisioningEmail(42, map[string]any{"login_id": "jdoe"}, "Jane@School.edu")
	if got != "jane@school.edu" {
		t.Fatalf("canvasResolveProvisioningEmail() = %q, want normalized roster email", got)
	}
}

func TestCanvasResolveProvisioningEmail_usesLoginIDWhenEmailMissing(t *testing.T) {
	got := canvasResolveProvisioningEmail(42, map[string]any{"login_id": "jdoe"}, "")
	want := "canvas+jdoe-42@canvas-import.invalid"
	if got != want {
		t.Fatalf("canvasResolveProvisioningEmail() = %q, want %q", got, want)
	}
}

func TestCanvasResolveProvisioningEmail_usesCanvasUIDFallback(t *testing.T) {
	got := canvasResolveProvisioningEmail(12345, nil, "")
	if got != "canvas+12345@canvas-import.invalid" {
		t.Fatalf("canvasResolveProvisioningEmail() = %q, want canvas uid fallback", got)
	}
}

func TestCanvasSanitizeEmailLocal_stripsUnsafeCharacters(t *testing.T) {
	if got := canvasSanitizeEmailLocal("Jane O'Neil"); got != "jane-o-neil" {
		t.Fatalf("canvasSanitizeEmailLocal() = %q, want jane-o-neil", got)
	}
}

func TestCanvasEnrollmentListQuery_includesConcludedStates(t *testing.T) {
	q := canvasEnrollmentListQuery()
	states := q["state[]"]
	for _, want := range []string{"active", "invited", "creation_pending", "completed", "inactive"} {
		found := false
		for _, s := range states {
			if s == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("state[] = %v, want to include %q", states, want)
		}
	}
}

func TestCanvasEnrollmentRowsFromCourseUsers_synthesizesStudentWhenNoEnrollments(t *testing.T) {
	rows := canvasEnrollmentRowsFromCourseUsers([]map[string]any{{
		"id": float64(42),
		"name": "Pat Student",
		"login_id": "pat",
	}})
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if got := canvasCanvasUserIDFromEnrollment(rows[0], objAt(rows[0], "user")); got != 42 {
		t.Fatalf("canvas user id = %d, want 42", got)
	}
	if got := canvasEnrollmentTypeFromRow(rows[0]); got != "StudentEnrollment" {
		t.Fatalf("type = %q, want StudentEnrollment", got)
	}
}

func TestCanvasEnrollmentRowsFromCourseUsers_usesNestedEnrollments(t *testing.T) {
	rows := canvasEnrollmentRowsFromCourseUsers([]map[string]any{{
		"id": float64(7),
		"name": "Alex Teacher",
		"enrollments": []any{
			map[string]any{
				"type":             "TeacherEnrollment",
				"enrollment_state": "completed",
			},
		},
	}})
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if got := canvasEnrollmentTypeFromRow(rows[0]); got != "TeacherEnrollment" {
		t.Fatalf("type = %q, want TeacherEnrollment", got)
	}
}

func TestCanvasFilterImportableEnrollmentRows_skipsDeleted(t *testing.T) {
	rows := canvasFilterImportableEnrollmentRows([]map[string]any{
		{"enrollment_state": "active"},
		{"enrollment_state": "deleted"},
	})
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1 active enrollment", len(rows))
	}
}

func TestCanvasEnrollmentTypeFromRow_fallsBackToRole(t *testing.T) {
	if got := canvasEnrollmentTypeFromRow(map[string]any{"role": "TaEnrollment"}); got != "TaEnrollment" {
		t.Fatalf("canvasEnrollmentTypeFromRow() = %q, want TaEnrollment", got)
	}
}

func TestCanvasEnrollmentListQuery_includesAvatarURL(t *testing.T) {
	q := canvasEnrollmentListQuery()
	includes := q["include[]"]
	foundUser := false
	foundAvatar := false
	for _, v := range includes {
		if v == "user" {
			foundUser = true
		}
		if v == "avatar_url" {
			foundAvatar = true
		}
	}
	if !foundUser || !foundAvatar {
		t.Fatalf("include[] = %v, want user and avatar_url", includes)
	}
}