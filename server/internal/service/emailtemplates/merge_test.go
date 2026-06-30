package emailtemplates

import "testing"

func TestMerge_replacesKnownTokens(t *testing.T) {
	got := Merge("Hello {{user.first_name}} in {{course.title}}", map[string]string{
		"user.first_name": "Alex",
		"course.title":    "Biology",
	})
	want := "Hello Alex in Biology"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFindUnknownTokens(t *testing.T) {
	allowed := map[string]string{"user.first_name": "x"}
	unknown := FindUnknownTokens("Hi {{user.first_name}} and {{foo.bar}}", allowed)
	if len(unknown) != 1 || unknown[0] != "foo.bar" {
		t.Fatalf("unknown=%v", unknown)
	}
}

func TestStripHTMLTags(t *testing.T) {
	got := StripHTMLTags(`<p>Hello <strong>world</strong></p><script>alert(1)</script>`)
	if got != "Hello world" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeHTML_stripsScript(t *testing.T) {
	got := SanitizeHTML(`<p>Hi</p><script>alert(1)</script>`)
	if got != `<p>Hi</p>` {
		t.Fatalf("got %q", got)
	}
}

func TestMapJobVars(t *testing.T) {
	out := MapJobVars(map[string]string{
		"courseName":     "Bio",
		"assignmentName": "Lab 1",
		"dueAt":          "Friday",
		"unsubscribeUrl": "https://example.edu/unsub",
	})
	if out["course.title"] != "Bio" || out["assignment.title"] != "Lab 1" {
		t.Fatalf("map=%v", out)
	}
}
