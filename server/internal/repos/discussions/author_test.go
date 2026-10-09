package discussions

import (
	"database/sql"
	"testing"
)

func TestAuthorFromNulls(t *testing.T) {
	name, avatar := authorFromNulls(
		sql.NullString{String: "QA Test Kid", Valid: true},
		sql.NullString{String: "https://cdn.example/a.png", Valid: true},
	)
	if name == nil || *name != "QA Test Kid" {
		t.Fatalf("display name = %v", name)
	}
	if avatar == nil || *avatar != "https://cdn.example/a.png" {
		t.Fatalf("avatar = %v", avatar)
	}

	name, avatar = authorFromNulls(sql.NullString{}, sql.NullString{})
	if name != nil || avatar != nil {
		t.Fatalf("missing profile = %v %v, want nils", name, avatar)
	}
}
