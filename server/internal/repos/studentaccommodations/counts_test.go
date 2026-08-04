package studentaccommodations

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Guard: AggregateActiveTypesForCourse SQL must never SELECT user identifiers (CC.5 AC-10).
func TestAggregateActiveTypesSQLHasNoPII(t *testing.T) {
	src, err := os.ReadFile("counts.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(src)
	fn := "func AggregateActiveTypesForCourse"
	idx := strings.Index(text, fn)
	if idx < 0 {
		t.Fatalf("%s missing", fn)
	}
	chunk := strings.ToLower(text[idx:])
	if end := strings.Index(chunk, "\nfunc "); end > 0 {
		chunk = chunk[:end]
	}
	for _, banned := range []string{
		"user_id", "sa.user_id", "email", "display_name", "created_by", "updated_by",
	} {
		if strings.Contains(chunk, banned) {
			t.Fatalf("%s must not reference %q (privacy)", fn, banned)
		}
	}
}

func TestTypeAggregateExportedFields(t *testing.T) {
	fset := token.NewFileSet()
	path := filepath.Join("counts.go")
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != "TypeAggregate" {
				continue
			}
			found = true
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				t.Fatal("TypeAggregate not a struct")
			}
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					switch name.Name {
					case "Type", "Count":
					default:
						t.Fatalf("unexpected field %s on TypeAggregate (privacy)", name.Name)
					}
				}
			}
		}
	}
	if !found {
		t.Fatal("TypeAggregate missing")
	}
}
