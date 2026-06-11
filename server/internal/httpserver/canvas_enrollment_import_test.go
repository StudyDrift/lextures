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