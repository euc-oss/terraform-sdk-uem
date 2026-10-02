package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestPublicTargetsRunTheAssembledPackageGuards pins the wiring the sync tests
// cannot see: both make public-* targets pass --staging-check with the assembled-
// package guards (so they run before any branch or commit exists), and every guard
// named in the Makefile still exists in tests/ (a renamed guard would otherwise turn
// `go test -run` into a silent no-op).
func TestPublicTargetsRunTheAssembledPackageGuards(t *testing.T) {
	mk, err := os.ReadFile(filepath.Join("..", "Makefile"))
	if err != nil {
		t.Skipf("Makefile not present (%v); this guard needs the private checkout", err)
	}
	files, err := filepath.Glob("*.go")
	if err != nil || len(files) == 0 {
		t.Skip("tests/ not present")
	}
	var src strings.Builder
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		src.Write(b)
	}
	skip, problems := publicTargetsWiringVerdict(string(mk), src.String())
	if skip != "" {
		t.Skip(skip)
	}
	for _, p := range problems {
		t.Error(p)
	}
}

// assembledGuardsLine matches the Makefile line that lists the assembled-package
// guards, which only the private checkout's Makefile carries.
var assembledGuardsLine = regexp.MustCompile(`(?m)^ASSEMBLED_GUARDS = (.+)$`)

// publicTargetsWiringVerdict is the decision logic of
// TestPublicTargetsRunTheAssembledPackageGuards as a pure function of the
// Makefile text and the concatenated tests/ Go source. A non-empty skip means the
// Makefile is not the private checkout's (the assembled public tree ships its own
// consumer Makefile with neither the guard list nor the public-* targets), so the
// wiring does not apply. Otherwise problems lists every wiring defect; exactly one
// of the two markers being present is partial wiring and is a defect, not a skip.
func publicTargetsWiringVerdict(makefile, testSrc string) (skip string, problems []string) {
	m := assembledGuardsLine.FindStringSubmatch(makefile)
	hasTarget := regexp.MustCompile(`(?m)^public-sdk-pre-release:`).MatchString(makefile)
	switch {
	case m == nil && !hasTarget:
		return "Makefile has neither an ASSEMBLED_GUARDS list nor a public-sdk-pre-release target; it is not the private checkout's Makefile, so this wiring guard does not apply", nil
	case m == nil:
		return "", []string{"Makefile has the public-sdk-pre-release target but no ASSEMBLED_GUARDS list"}
	case !hasTarget:
		return "", []string{"Makefile has an ASSEMBLED_GUARDS list but no public-sdk-pre-release target"}
	}

	guards := strings.Split(m[1], "|")
	if len(guards) < 5 {
		return "", []string{fmt.Sprintf("ASSEMBLED_GUARDS lists only %v", guards)}
	}
	for _, g := range guards {
		if !strings.Contains(testSrc, "func "+g+"(t *testing.T)") {
			problems = append(problems, fmt.Sprintf("ASSEMBLED_GUARDS names %s, which does not exist in tests/", g))
		}
	}
	for _, must := range []string{"TestAssembledPackagesHaveNoLeakedOGUUID", "TestAssembledPackagesHaveNoDashFormUUIDInProse"} {
		if !strings.Contains(m[1], must) {
			problems = append(problems, fmt.Sprintf("ASSEMBLED_GUARDS dropped %s", must))
		}
	}

	for target, env := range map[string]string{
		"public-sdk-pre-release":        "ASSEMBLED_PUBLIC_GO_DIR",
		"public-python-sdk-pre-release": "ASSEMBLED_PUBLIC_PY_DIR",
	} {
		recipe := makefileTargetRecipe(makefile, target)
		if recipe == "" {
			problems = append(problems, fmt.Sprintf("target %s not found", target))
			continue
		}
		var syncLine string
		for _, line := range strings.Split(recipe, "\n") {
			if strings.Contains(line, "tools/sync/cmd/sync") {
				syncLine = line
			}
		}
		if !strings.Contains(syncLine, "--staging-check=") || !strings.Contains(syncLine, env+`="$$ASSEMBLED_STAGING_DIR"`) || !strings.Contains(syncLine, "$(ASSEMBLED_GUARDS)") {
			problems = append(problems, fmt.Sprintf("%s: the sync invocation does not run the assembled-package guards on $ASSEMBLED_STAGING_DIR:\n%s", target, syncLine))
		}
	}
	if py := makefileTargetRecipe(makefile, "public-python-sdk-pre-release"); !strings.Contains(py, `ARGS="--lang go,python"`) {
		problems = append(problems, fmt.Sprintf("the Python target must generate Go too (the guards are Go tests): %s", py))
	}
	return "", problems
}

// makefileTargetRecipe returns the recipe lines of a Makefile target.
func makefileTargetRecipe(makefile, target string) string {
	lines := strings.Split(makefile, "\n")
	var out []string
	in := false
	for _, l := range lines {
		if strings.HasPrefix(l, target+":") {
			in = true
			continue
		}
		if in {
			if strings.HasPrefix(l, "\t") {
				out = append(out, l)
				continue
			}
			break
		}
	}
	return strings.Join(out, "\n")
}

// TestPublicTargetsWiringVerdict pins the decision logic on planted Makefiles:
// the assembled public tree's consumer Makefile is skipped, partial wiring fails,
// and a complete private-style wiring passes.
func TestPublicTargetsWiringVerdict(t *testing.T) {
	const publicMakefile = "GO ?= go\n\n.PHONY: build\nbuild: ## Compile\n\t$(GO) build ./...\n\n.PHONY: test\ntest: ## Run tests\n\tTEST_MODE=mock $(GO) test ./...\n"

	guardNames := []string{"TestGuardA", "TestGuardB", "TestGuardC", "TestGuardD", "TestAssembledPackagesHaveNoLeakedOGUUID", "TestAssembledPackagesHaveNoDashFormUUIDInProse"}
	var testSrc strings.Builder
	for _, g := range guardNames {
		testSrc.WriteString("func " + g + "(t *testing.T) {}\n")
	}
	guardsLine := "ASSEMBLED_GUARDS = " + strings.Join(guardNames, "|") + "\n"
	const staging = `--staging-check='ASSEMBLED_PUBLIC_%s_DIR="$$ASSEMBLED_STAGING_DIR" go test ./tests/ -run "^($(ASSEMBLED_GUARDS))$$"'`
	privateMakefile := guardsLine +
		"\npublic-sdk-pre-release:\n\tgo run ./tools/sync/cmd/sync " + fmt.Sprintf(staging, "GO") + "\n" +
		"\npublic-python-sdk-pre-release:\n\t$(MAKE) public-sdk-pre-release ARGS=\"--lang go,python\" # go run ./tools/sync/cmd/sync " + fmt.Sprintf(staging, "PY") + "\n"

	t.Run("public_makefile_skips", func(t *testing.T) {
		skip, problems := publicTargetsWiringVerdict(publicMakefile, testSrc.String())
		if skip == "" || len(problems) != 0 {
			t.Fatalf("the assembled public tree's Makefile must skip, got skip=%q problems=%v", skip, problems)
		}
	})
	t.Run("the_shipped_public_makefile_skips", func(t *testing.T) {
		shipped, err := os.ReadFile(filepath.Join("..", "public", "Makefile"))
		if err != nil {
			t.Skipf("public/Makefile not present (%v); the assembled tree has no public/ directory", err)
		}
		skip, problems := publicTargetsWiringVerdict(string(shipped), testSrc.String())
		if skip == "" || len(problems) != 0 {
			t.Fatalf("the shipped public/Makefile must skip, got skip=%q problems=%v", skip, problems)
		}
	})
	t.Run("guards_without_target_fails", func(t *testing.T) {
		skip, problems := publicTargetsWiringVerdict(guardsLine+publicMakefile, testSrc.String())
		if skip != "" || len(problems) == 0 {
			t.Fatalf("an ASSEMBLED_GUARDS list without the target is partial wiring and must fail, got skip=%q problems=%v", skip, problems)
		}
	})
	t.Run("target_without_guards_fails", func(t *testing.T) {
		skip, problems := publicTargetsWiringVerdict(publicMakefile+"\npublic-sdk-pre-release:\n\t@true\n", testSrc.String())
		if skip != "" || len(problems) == 0 {
			t.Fatalf("the target without an ASSEMBLED_GUARDS list is partial wiring and must fail, got skip=%q problems=%v", skip, problems)
		}
	})
	t.Run("complete_private_wiring_passes", func(t *testing.T) {
		skip, problems := publicTargetsWiringVerdict(privateMakefile, testSrc.String())
		if skip != "" || len(problems) != 0 {
			t.Fatalf("complete wiring must pass, got skip=%q problems=%v", skip, problems)
		}
	})
	t.Run("renamed_guard_fails", func(t *testing.T) {
		skip, problems := publicTargetsWiringVerdict(privateMakefile, strings.Replace(testSrc.String(), "TestGuardA", "TestGuardRenamed", 1))
		if skip != "" || len(problems) == 0 {
			t.Fatalf("a guard named in the list but absent from tests/ must fail, got skip=%q problems=%v", skip, problems)
		}
	})
}
