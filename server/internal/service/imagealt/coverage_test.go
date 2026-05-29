package imagealt

import "testing"

func TestScanMarkdown(t *testing.T) {
	md := `# Page
![Diagram of cell](https://example.com/cell.png)
![](https://example.com/missing.png)
![](https://example.com/dec.png "lex-decorative")
`
	imgs := ScanMarkdown(md)
	if len(imgs) != 3 {
		t.Fatalf("expected 3 images, got %d", len(imgs))
	}
	if !imgs[0].HasValidAlt || imgs[0].Alt != "Diagram of cell" {
		t.Fatalf("first image should have alt: %+v", imgs[0])
	}
	if imgs[1].HasValidAlt {
		t.Fatalf("second image should be missing alt: %+v", imgs[1])
	}
	if !imgs[2].HasValidAlt || !imgs[2].Decorative {
		t.Fatalf("third image should be decorative: %+v", imgs[2])
	}
	cov := Summarize(imgs)
	if cov.WithAlt != 2 || cov.Total != 3 {
		t.Fatalf("coverage: %+v", cov)
	}
}
