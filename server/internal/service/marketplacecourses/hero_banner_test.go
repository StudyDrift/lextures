package marketplacecourses

import (
	"strings"
	"testing"
)

func TestAIEssentialsHeroBannerAsset_Embedded(t *testing.T) {
	asset, ok := heroBannerForSlug("ai-essentials")
	if !ok {
		t.Fatal("expected ai-essentials banner mapping")
	}
	if len(asset.jpeg) == 0 {
		t.Fatal("expected embedded AI Essentials banner")
	}
	if len(asset.jpeg) < 4 {
		t.Fatal("banner JPEG too small")
	}
	if asset.jpeg[0] != 0xff || asset.jpeg[1] != 0xd8 {
		t.Fatalf("banner should be JPEG, got %q", asset.jpeg[:2])
	}
	if heroBannerMIME != "image/jpeg" {
		t.Fatalf("unexpected mime: %s", heroBannerMIME)
	}
	if !strings.HasSuffix(asset.filename, ".jpg") {
		t.Fatalf("unexpected filename: %s", asset.filename)
	}
}

func TestIntroductionToPythonHeroBannerAsset_Embedded(t *testing.T) {
	asset, ok := heroBannerForSlug("introduction-to-python")
	if !ok {
		t.Fatal("expected introduction-to-python banner mapping")
	}
	if len(asset.jpeg) == 0 {
		t.Fatal("expected embedded Introduction to Python banner")
	}
	if len(asset.jpeg) < 4 {
		t.Fatal("banner JPEG too small")
	}
	if asset.jpeg[0] != 0xff || asset.jpeg[1] != 0xd8 {
		t.Fatalf("banner should be JPEG, got %q", asset.jpeg[:2])
	}
	if asset.filename != "introduction-to-python-banner.jpg" {
		t.Fatalf("unexpected filename: %s", asset.filename)
	}
}

func TestPersonalFinanceHeroBannerAsset_Embedded(t *testing.T) {
	asset, ok := heroBannerForSlug("personal-finance")
	if !ok {
		t.Fatal("expected personal-finance banner mapping")
	}
	if len(asset.jpeg) == 0 {
		t.Fatal("expected embedded Personal Finance banner")
	}
	if len(asset.jpeg) < 4 {
		t.Fatal("banner JPEG too small")
	}
	if asset.jpeg[0] != 0xff || asset.jpeg[1] != 0xd8 {
		t.Fatalf("banner should be JPEG, got %q", asset.jpeg[:2])
	}
	if asset.filename != "personal-finance-banner.jpg" {
		t.Fatalf("unexpected filename: %s", asset.filename)
	}
}

func TestHeroBannerForSlug_Unknown(t *testing.T) {
	if _, ok := heroBannerForSlug("harness-smoke"); ok {
		t.Fatal("harness-smoke should not have a banner asset")
	}
	if _, ok := heroBannerForSlug(""); ok {
		t.Fatal("empty slug should not have a banner asset")
	}
}
