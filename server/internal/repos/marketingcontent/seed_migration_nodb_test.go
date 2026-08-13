package marketingcontent

import (
	"bytes"
	"testing"

	serverdata "github.com/lextures/lextures/server"
)

func TestMarketingContentSeedMigration(t *testing.T) {
	t.Parallel()
	b, err := serverdata.Migrations.ReadFile("migrations/485_marketing_content_seed.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range [][]byte{
		[]byte(`"kind":"blog"`),
		[]byte(`"kind":"doc"`),
		[]byte(`"slug":"chase-willden"`),
		[]byte(`079aacf8fef3b171a54b2063dce723471714b824`),
		[]byte(`ON CONFLICT DO NOTHING`),
		[]byte(`content_revisions`),
		[]byte(`content_route_hints`),
	} {
		if !bytes.Contains(b, want) {
			t.Fatalf("seed migration missing %q", want)
		}
	}
	if got := bytes.Count(b, []byte(`"kind":"blog"`)); got != 5 {
		t.Fatalf("blog seed count = %d, want 5", got)
	}
	if got := bytes.Count(b, []byte(`"kind":"doc"`)); got != 65 {
		t.Fatalf("doc seed count = %d, want 65", got)
	}
}
