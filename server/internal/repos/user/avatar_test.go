package user

import "testing"

func TestIsMissingOrDefaultBlankAvatarURL_empty(t *testing.T) {
	for _, raw := range []string{"", "  ", "\t"} {
		if !IsMissingOrDefaultBlankAvatarURL(raw) {
			t.Fatalf("IsMissingOrDefaultBlankAvatarURL(%q) = false, want true", raw)
		}
	}
}

func TestIsMissingOrDefaultBlankAvatarURL_custom(t *testing.T) {
	custom := "https://cdn.example.com/users/alice.png"
	if IsMissingOrDefaultBlankAvatarURL(custom) {
		t.Fatalf("custom avatar URL should not be blank: %q", custom)
	}
	customData := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAEAAAABACUSTOM"
	if IsMissingOrDefaultBlankAvatarURL(customData) {
		t.Fatalf("custom data URL should not be blank: %q", customData)
	}
}

func TestIsMissingOrDefaultBlankAvatarURL_canvasHTTPDefault(t *testing.T) {
	u := "https://canvas.example.edu/images/messages/avatar-50.png"
	if !IsMissingOrDefaultBlankAvatarURL(u) {
		t.Fatalf("Canvas default HTTP avatar should be blank: %q", u)
	}
}

func TestIsMissingOrDefaultBlankAvatarURL_defaultPNG(t *testing.T) {
	if !IsMissingOrDefaultBlankAvatarURL(defaultBlankAvatarDataURL) {
		t.Fatal("canonical default PNG placeholder should be blank")
	}
}

func TestIsMissingOrDefaultBlankAvatarURL_canvasJPEGDefault(t *testing.T) {
	u := "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/" + canvasDefaultAvatarJPEGBase64Prefix
	if !IsMissingOrDefaultBlankAvatarURL(u) {
		t.Fatal("Canvas default JPEG placeholder should be blank")
	}
}

func TestIsMissingOrDefaultBlankAvatarURL_blankJPEGFixture(t *testing.T) {
	u := "data:image/jpeg;base64,/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAYEBQYFBAYGBQYHBwYIChAKCgkJChQODwwQFxQYGBcUFhYaHSUfGhsjHBYWICwgIyYnKSopGR8tMC0oMCUoKSj/2wBDAQcHBwoIChMKChMoGhYaKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCgoKCj/" + blankAvatarJPEGDimensionMarker + "ASIAAhEBAxEB/8QAFQABAQAAAAAAAAAAAAAAAAAAAAj/xAAUEAEAAAAAAAAAAAAAAAAAAAAA/8QAFAEBAAAAAAAAAAAAAAAAAAAAAP/EABQRAQAAAAAAAAAAAAAAAAAAAAD/2gAMAwEAAhEDEQA/AJrAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAB//9k="
	if !IsMissingOrDefaultBlankAvatarURL(u) {
		t.Fatal("blank 120x120 JPEG fixture should be blank")
	}
}