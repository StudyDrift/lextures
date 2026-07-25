package adaptivecontent

import (
	"strings"
	"testing"
)

func TestExtractFencedCodeBlocks(t *testing.T) {
	t.Parallel()
	md := "intro\n```go\nfmt.Println(\"hi\")\n```\nmore\n```\nplain\n```"
	got := ExtractFencedCodeBlocks(md)
	if len(got) != 2 {
		t.Fatalf("got %d blocks: %#v", len(got), got)
	}
	if got[0] != `fmt.Println("hi")` {
		t.Fatalf("first: %q", got[0])
	}
	if got[1] != "plain" {
		t.Fatalf("second: %q", got[1])
	}
}

func TestExtractLatexBlocks(t *testing.T) {
	t.Parallel()
	md := `Energy is $E=mc^2$ and display $$\int_0^1 x\,dx$$.`
	got := ExtractLatexBlocks(md)
	if len(got) < 2 {
		t.Fatalf("got %#v", got)
	}
	joined := strings.Join(got, "|")
	if !strings.Contains(joined, "E=mc^2") {
		t.Fatalf("missing inline: %s", joined)
	}
	if !strings.Contains(joined, `\int_0^1`) && !strings.Contains(joined, "int_0^1") {
		t.Fatalf("missing display: %s", joined)
	}
}

func TestCheckKeyTerms(t *testing.T) {
	t.Parallel()
	ok, missing := CheckKeyTerms("Photosynthesis converts light.", []string{"Photosynthesis", "chlorophyll"})
	if ok {
		t.Fatal("expected fail")
	}
	if len(missing) != 1 || missing[0] != "chlorophyll" {
		t.Fatalf("missing=%v", missing)
	}
	ok, missing = CheckKeyTerms("Photosynthesis and chlorophyll.", []string{"Photosynthesis", "chlorophyll"})
	if !ok || len(missing) != 0 {
		t.Fatalf("ok=%v missing=%v", ok, missing)
	}
}

func TestCheckCodeAndLatexSurvive(t *testing.T) {
	t.Parallel()
	base := "See:\n```python\nprint(42)\n```\nand $a^2+b^2=c^2$."
	good := "Rewritten:\n```python\nprint(42)\n```\nstill $a^2+b^2=c^2$."
	if ok, _ := CheckCodeBlocksSurvive(base, good); !ok {
		t.Fatal("code should survive")
	}
	if ok, _ := CheckLatexSurvives(base, good); !ok {
		t.Fatal("latex should survive")
	}
	bad := "Rewritten without code or math."
	if ok, missing := CheckCodeBlocksSurvive(base, bad); ok || len(missing) == 0 {
		t.Fatal("code should fail")
	}
	if ok, missing := CheckLatexSurvives(base, bad); ok || len(missing) == 0 {
		t.Fatal("latex should fail")
	}
}

func TestRunHardFidelityChecks_FabricationPath(t *testing.T) {
	t.Parallel()
	// Hard checks only care about terms/math/code; missing key term fails.
	base := "The mitochondria is the powerhouse. Key: ATP synthase."
	variant := "Cells make energy somehow."
	res := RunHardFidelityChecks(base, variant, []string{"ATP synthase"})
	if res.HardPass || res.Score != 0 {
		t.Fatalf("expected hard fail: %+v", res)
	}
}

func TestCompositeFidelityScore(t *testing.T) {
	t.Parallel()
	hard := FidelityResult{HardPass: true, Score: 1, JudgeScore: -1}
	got := CompositeFidelityScore(hard, 0.9)
	if got.Score != 0.9 {
		t.Fatalf("score=%v", got.Score)
	}
	hardFail := FidelityResult{HardPass: false, Score: 0}
	got = CompositeFidelityScore(hardFail, 0.99)
	if got.Score != 0 {
		t.Fatalf("hard fail should force 0, got %v", got.Score)
	}
}

func TestSanitizeAndSafety(t *testing.T) {
	t.Parallel()
	dirty := `Hello <script>alert(1)</script> [x](javascript:alert(1))`
	clean := SanitizeVariantMarkdown(dirty)
	if strings.Contains(strings.ToLower(clean), "<script") {
		t.Fatalf("script remained: %q", clean)
	}
	if strings.Contains(strings.ToLower(clean), "javascript:") {
		t.Fatalf("js url remained: %q", clean)
	}
	flags := SafetyScan(`<script>x</script>`)
	if len(flags) == 0 {
		t.Fatal("expected safety flags")
	}
}

func TestLintA11y_MissingAlt(t *testing.T) {
	t.Parallel()
	flags := LintA11y("See ![](http://example.com/x.png)")
	if len(flags) == 0 {
		t.Fatal("expected image_missing_alt")
	}
}

func TestLexicalClaimOverlap(t *testing.T) {
	t.Parallel()
	base := "The mitochondria produces ATP through cellular respiration in eukaryotic cells."
	same := base + " Additional scaffolding for learners."
	score := LexicalClaimOverlap(base, same)
	if score < 0.5 {
		t.Fatalf("expected high overlap, got %v", score)
	}
	unrelated := "The stock market closed higher today on strong earnings."
	low := LexicalClaimOverlap(base, unrelated)
	if low > 0.5 {
		t.Fatalf("expected low overlap, got %v", low)
	}
}

func TestParseFidelityJudgeJSON(t *testing.T) {
	t.Parallel()
	score, claims, err := ParseFidelityJudgeJSON(`{"supportScore":0.92,"unsupportedClaims":["fake stat 99%"],"addressesMisconceptions":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if score != 0.92 || len(claims) != 1 {
		t.Fatalf("score=%v claims=%v", score, claims)
	}
}
