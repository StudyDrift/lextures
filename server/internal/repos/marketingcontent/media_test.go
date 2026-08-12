package marketingcontent

import (
	"testing"

	"github.com/google/uuid"
)

func TestExtractMediaIDs(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	body := "![one](/api/v1/public/content/media/" + a.String() + "/original.png)\n![two](/api/v1/public/content/media/" + b.String() + "/800w.webp)"
	got := ExtractMediaIDs(body)
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Fatalf("ExtractMediaIDs() = %v", got)
	}
}

func TestExtractMediaIDsIgnoresMalformedReferences(t *testing.T) {
	if got := ExtractMediaIDs("![](/api/v1/public/content/media/not-a-uuid/original.png)"); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}
