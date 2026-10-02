package tests

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRootPackageDocLivesOnlyInDocGo pins that no root package file other than
// doc.go attaches a doc comment to its package clause: go doc concatenates every
// file's package doc, so a stray one (for example a hand-written file's internal
// note) would be published next to the real overview rendered from doc.go.
func TestRootPackageDocLivesOnlyInDocGo(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "*.go"))
	if err != nil || len(files) == 0 {
		t.Skip("root package sources not present")
	}
	fset := token.NewFileSet()
	for _, f := range files {
		base := filepath.Base(f)
		if base == "doc.go" || strings.HasSuffix(base, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := parser.ParseFile(fset, f, src, parser.PackageClauseOnly|parser.ParseComments)
		if err != nil {
			t.Fatalf("parsing %s: %v", base, err)
		}
		if parsed.Doc != nil {
			t.Errorf("%s attaches a package doc comment (%q); only doc.go may carry the package doc, so detach it with a blank line before the package clause", base, strings.TrimSpace(strings.SplitN(parsed.Doc.Text(), "\n", 2)[0]))
		}
	}
}
