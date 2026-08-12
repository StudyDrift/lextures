package validate

import (
	"strings"
	"testing"
)

func TestCJKPassageLengthDoesNotFailOnWordCount(t *testing.T) {
	t.Parallel()
	answer := strings.Repeat("这是一段足够长的中文直接回答用来覆盖字符阈值而不是英文词数。", 4)
	body := ":::key-takeaways\n- 一\n- 二\n- 三\n:::\n\n:::answer\n" + answer + "\n:::\n\n## 如何使用？\n\n" + strings.Repeat("这是正文。", 20)
	report := Article(Input{Kind: "doc", Locale: "zh", BodyMD: body, Metadata: validMetadata()})
	for _, f := range report.Findings {
		if f.Rule == "passage.length" || f.Rule == "passage.self-contained" {
			t.Fatalf("CJK article failed English word-count rule: %+v", f)
		}
	}
}

func TestEnglishPassageThresholdUnchanged(t *testing.T) {
	t.Parallel()
	report := Article(Input{Kind: "blog", Locale: "en", BodyMD: ":::answer\nshort\n:::\n:::key-takeaways\n- a\n- b\n- c\n:::", Metadata: validMetadata()})
	if !has(report, "passage.length", "warn") {
		t.Fatal("expected English short-answer warning")
	}
}
