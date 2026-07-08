package introcourse

import (
	"strings"
	"testing"
)

func TestIntroHeroBannerAsset_Embedded(t *testing.T) {
	if len(introHeroBannerJPEG) == 0 {
		t.Fatal("expected embedded intro course banner")
	}
	if len(introHeroBannerJPEG) < 4 {
		t.Fatal("banner JPEG too small")
	}
	// JFIF / JPEG magic
	if introHeroBannerJPEG[0] != 0xff || introHeroBannerJPEG[1] != 0xd8 {
		t.Fatalf("banner should be JPEG, got %q", introHeroBannerJPEG[:2])
	}
	if introHeroBannerMIME != "image/jpeg" {
		t.Fatalf("unexpected mime: %s", introHeroBannerMIME)
	}
	if !strings.HasSuffix(introHeroBannerFilename, ".jpg") {
		t.Fatalf("unexpected filename: %s", introHeroBannerFilename)
	}
}