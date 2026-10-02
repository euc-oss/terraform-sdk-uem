package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// dashFormUUIDPattern is a byte-for-byte mirror of tools/sync/scrub.go's own
// uuidRegex: the public sync scrubber rewrites ANY substring matching this
// shape, anywhere in a .json/.go/.py file, to a deterministic synthetic
// UUID (same input always maps to the same output, but the output shares
// none of the input's visual properties -- "sequential", "ascending-hex",
// etc. claims about a specific dashed literal become false once scrubbed).
// If tools/sync/scrub.go's own regex ever changes, this one must be updated
// to match -- TestDashFormUUIDPatternMirrorsScrubber pins the two literal
// pattern strings together so drift is caught mechanically, not just by
// convention.
var dashFormUUIDPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// dashFormUUIDPatternSource is dashFormUUIDPattern's literal source string,
// kept separately so TestDashFormUUIDPatternMirrorsScrubber can compare it
// character-for-character against scrub.go's own source text without
// needing to import the tools/sync package (a test-only import of a
// non-test package from here would be unusual, and the two need to be
// compared as raw text anyway since this file cannot see scrub.go's
// unexported uuidRegex variable across package boundaries).
const dashFormUUIDPatternSource = `[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`

// dashUUIDScrubberExemptValues lists dash-form UUID substrings the
// scrubber itself is known NOT to rewrite (currently empty -- scrub.go's
// uuidRegex has no exemption list of its own, so nothing is expected here).
// If the scrubber ever grows an exemption, add the exact value here so this
// guard stops flagging it, with a comment citing the scrubber code that
// exempts it.
var dashUUIDScrubberExemptValues = map[string]bool{}

// dashUUIDProblems scans text line-by-line for a dash-form UUID substring,
// skipping any exact match present in dashUUIDScrubberExemptValues.
func dashUUIDProblems(label, text string) []string {
	var problems []string
	for i, line := range strings.Split(text, "\n") {
		for _, match := range dashFormUUIDPattern.FindAllString(line, -1) {
			if dashUUIDScrubberExemptValues[match] {
				continue
			}
			problems = append(problems, fmt.Sprintf("%s:%d: dash-form UUID %q found -- the public sync scrubber rewrites this, invalidating any claim made about its literal value or shape: %q", label, i+1, match, strings.TrimSpace(line)))
		}
	}
	return problems
}

// vendoredDescriptionDashUUIDProblems walks the resolved vendored corpus
// (same MOCK_RESPONSES_DIR convention as this file's other corpus-dependent
// guards) and reports every fixture whose metadata.description contains a
// dash-form UUID.
func vendoredDescriptionDashUUIDProblems(t *testing.T) []string {
	t.Helper()
	corpus := walkVendoredFixtureCorpus(t)
	var problems []string
	for _, e := range corpus {
		data, err := os.ReadFile(e.fullPath)
		if err != nil {
			t.Fatalf("reading fixture %s: %v", e.fullPath, err)
		}
		var fixture struct {
			Metadata struct {
				Description string `json:"description"`
			} `json:"metadata"`
		}
		if err := json.Unmarshal(data, &fixture); err != nil {
			t.Fatalf("parsing fixture %s: %v", e.fullPath, err)
		}
		problems = append(problems, dashUUIDProblems(e.relPath+" (metadata.description)", fixture.Metadata.Description)...)
	}
	return problems
}

// TestVendoredDescriptionsHaveNoDashFormUUID is the PRIVATE-TREE half of the
// dash-form-UUID guard: no vendored fixture's metadata.description may quote
// a dash-form UUID literal. Runs against the resolved vendored corpus
// (MOCK_RESPONSES_DIR), skipping cleanly when unset/too small, matching
// this file's other corpus-dependent guards.
func TestVendoredDescriptionsHaveNoDashFormUUID(t *testing.T) {
	if problems := vendoredDescriptionDashUUIDProblems(t); len(problems) > 0 {
		sort.Strings(problems)
		t.Fatalf("%d vendored fixture description(s) quote a dash-form UUID the scrubber would rewrite:\n%s", len(problems), joinLines(problems))
	}
}

// shippedCommentLinesDashUUIDProblems walks root for every file ending in
// ext, extracts only COMMENT lines (goCommentPrefix "//" or pyCommentPrefix
// "#", matching allGeneratedTestGoSourceExcept's own line-comment-only
// convention elsewhere in this file -- functional string literals used for
// exact-match test assertions are a different, already-handled class, see
// buildUUIDForLedgerSweep's doc comment), and reports every dash-form UUID
// found in one.
func shippedCommentLinesDashUUIDProblems(root, ext, commentPrefix string) []string {
	var problems []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() || !strings.HasSuffix(path, ext) {
			return nil //nolint:nilerr // best-effort walk: entries the walker cannot stat are skipped, not failed
		}
		problems = append(problems, shippedFileCommentLinesDashUUIDProblems(root, path, commentPrefix)...)
		return nil
	})
	return problems
}

// shippedFileCommentLinesDashUUIDProblems reports every dash-form UUID found
// in a comment line of the single file at path, labeling hits relative to
// root. Files in shippedHygieneGuardExemptFiles, unreadable files and
// generated files yield no problems.
func shippedFileCommentLinesDashUUIDProblems(root, path, commentPrefix string) []string {
	if shippedHygieneGuardExemptFiles[filepath.Base(path)] {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // best-effort scan: an unreadable file is skipped, not failed
	}
	// Generated files (codegen doc comments carrying an upstream
	// swagger field description) are excluded here: an "E.g.
	// <uuid>" illustrative example value in a generated doc comment
	// makes no claim that becomes FALSE once scrubbed (unlike a
	// fixture description's "sequential-digit"/"ascending-hex"
	// pattern claim about a SPECIFIC captured value) -- it just shows
	// a different, equally-valid-looking example UUID. Fixing this
	// class requires an upstream swagger-overlay change and a
	// codegen re-run, out of scope for this pass; flagged separately
	// rather than silently suppressed or hand-edited into a file that
	// regenerates over any hand edit anyway.
	if isGeneratedFileHeader(data) {
		return nil
	}
	rel, relErr := filepath.Rel(root, path)
	if relErr != nil {
		rel = path
	}
	var problems []string
	for i, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, commentPrefix) {
			continue
		}
		problems = append(problems, dashUUIDProblems(fmt.Sprintf("%s:%d", rel, i+1), trimmed)...)
	}
	return problems
}

// TestNoDashFormUUIDInProseOrComments is the PRIVATE-TREE half of the
// dash-form-UUID guard covering shipped Go test source: no comment line in
// any *_test.go file under tests/ may quote a dash-form UUID literal (the
// same "public text becomes false once scrubbed" defect, applied to
// comments rather than fixture JSON).
func TestNoDashFormUUIDInProseOrComments(t *testing.T) {
	problems := shippedCommentLinesDashUUIDProblems(".", "_test.go", "//")
	sort.Strings(problems)
	if len(problems) > 0 {
		t.Fatalf("%d shipped test-file comment(s) quote a dash-form UUID the scrubber would rewrite:\n%s", len(problems), joinLines(problems))
	}
}

// TestCarriedHandWrittenGoCommentsHaveNoDashFormUUID is the PRIVATE-TREE
// guard for the hand-written Go source the public sync carries (client/,
// models/, resources/ and the root *.go files): no comment in any of them may
// quote a dash-form UUID literal, because the sync scrubber rewrites it in the
// published text and the assembled-package guard then fails the release cut.
// Generated files and shippedHygieneGuardExemptFiles are skipped, exactly as
// in the assembled-package scan.
func TestCarriedHandWrittenGoCommentsHaveNoDashFormUUID(t *testing.T) {
	var problems []string
	for _, dir := range []string{"../client", "../models", "../resources"} {
		problems = append(problems, shippedCommentLinesDashUUIDProblems(dir, ".go", "//")...)
	}
	rootEntries, err := os.ReadDir("..")
	if err != nil {
		t.Fatalf("reading repository root: %v", err)
	}
	for _, e := range rootEntries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		problems = append(problems, shippedFileCommentLinesDashUUIDProblems("..", filepath.Join("..", e.Name()), "//")...)
	}
	sort.Strings(problems)
	if len(problems) > 0 {
		t.Fatalf("%d carried hand-written Go comment(s) quote a dash-form UUID the scrubber would rewrite:\n%s", len(problems), joinLines(problems))
	}
}

// TestAssembledPackagesHaveNoDashFormUUIDInProse is the ASSEMBLED-PACKAGE
// half of this guard: a private-tree-only check proves the SOURCE text is
// clean, but cannot prove the sync/scrub pipeline itself didn't reintroduce
// or fail to catch the problem, since the scrubber only runs during
// assembly -- this test re-runs the same scan against a REAL offline-sync-
// assembled package's fixture JSON
// (metadata.description) and Go/Python comment lines. Skips cleanly
// (ASSEMBLED_PUBLIC_GO_DIR / ASSEMBLED_PUBLIC_PY_DIR unset) when no real
// assembly has been staged -- see assembledPackageDirs.
func TestAssembledPackagesHaveNoDashFormUUIDInProse(t *testing.T) {
	goDir, pyDir := assembledPackageDirs(t)

	var problems []string
	if goDir != "" {
		problems = append(problems, shippedCommentLinesDashUUIDProblems(goDir, ".go", "//")...)
		// Assembled fixture JSON ships under testdata/mock-responses/ in the
		// Go package -- scan every metadata.description there too, not just
		// Go source comments.
		fixturesRoot := filepath.Join(goDir, "testdata", "mock-responses")
		if info, err := os.Stat(fixturesRoot); err == nil && info.IsDir() {
			_ = filepath.Walk(fixturesRoot, func(path string, fi os.FileInfo, walkErr error) error {
				if walkErr != nil || fi.IsDir() || filepath.Ext(path) != ".json" {
					return nil //nolint:nilerr // best-effort walk: entries the walker cannot stat are skipped, not failed
				}
				data, rerr := os.ReadFile(path)
				if rerr != nil {
					return nil //nolint:nilerr // best-effort scan: an unreadable file is skipped, not failed
				}
				var fixture struct {
					Metadata struct {
						Description string `json:"description"`
					} `json:"metadata"`
				}
				if json.Unmarshal(data, &fixture) != nil {
					return nil //nolint:nilerr // not every JSON file here is a capture fixture; non-fixture JSON is skipped
				}
				rel, _ := filepath.Rel(goDir, path)
				problems = append(problems, dashUUIDProblems(rel+" (metadata.description)", fixture.Metadata.Description)...)
				return nil
			})
		}
	}
	if pyDir != "" {
		problems = append(problems, shippedCommentLinesDashUUIDProblems(pyDir, ".py", "#")...)
		fixturesRoot := filepath.Join(pyDir, "tests", "fixtures", "mock-responses")
		if info, err := os.Stat(fixturesRoot); err == nil && info.IsDir() {
			_ = filepath.Walk(fixturesRoot, func(path string, fi os.FileInfo, walkErr error) error {
				if walkErr != nil || fi.IsDir() || filepath.Ext(path) != ".json" {
					return nil //nolint:nilerr // best-effort walk: entries the walker cannot stat are skipped, not failed
				}
				data, rerr := os.ReadFile(path)
				if rerr != nil {
					return nil //nolint:nilerr // best-effort scan: an unreadable file is skipped, not failed
				}
				var fixture struct {
					Metadata struct {
						Description string `json:"description"`
					} `json:"metadata"`
				}
				if json.Unmarshal(data, &fixture) != nil {
					return nil //nolint:nilerr // not every JSON file here is a capture fixture; non-fixture JSON is skipped
				}
				rel, _ := filepath.Rel(pyDir, path)
				problems = append(problems, dashUUIDProblems(rel+" (metadata.description)", fixture.Metadata.Description)...)
				return nil
			})
		}
	}

	sort.Strings(problems)
	if len(problems) > 0 {
		t.Fatalf("%d assembled shipped file(s)/fixture(s) quote a dash-form UUID the scrubber rewrote or would rewrite:\n%s", len(problems), joinLines(problems))
	}
}

// TestDashUUIDGuard_PlantedControls proves dashUUIDProblems fires on a
// planted dash-form UUID and does not false-positive on a clean underscore-
// form equivalent or an explicitly-exempted value. The planted "bad" string
// is assembled from concatenated parts at runtime so it never exists as a
// contiguous literal in this file's own shipped source (this file is not in
// shippedHygieneGuardExemptFiles, so its own comments ARE scanned by
// TestNoDashFormUUIDInProseOrComments -- a static literal here would
// self-flag).
func TestDashUUIDGuard_PlantedControls(t *testing.T) {
	t.Run("dash_form_fires", func(t *testing.T) {
		group1, group2, group3, group4, group5 := "a1b2c3d4", "e5f6", "7890", "abcd", "ef1234567890"
		planted := strings.Join([]string{group1, group2, group3, group4, group5}, "-")
		if problems := dashUUIDProblems("scratch", "example text quoting "+planted+" inline"); len(problems) == 0 {
			t.Fatal("a planted dash-form UUID was not detected")
		}
	})

	t.Run("underscore_form_passes", func(t *testing.T) {
		group1, group2, group3, group4, group5 := "a1b2c3d4", "e5f6", "7890", "abcd", "ef1234567890"
		planted := strings.Join([]string{group1, group2, group3, group4, group5}, "_")
		if problems := dashUUIDProblems("scratch", "example text quoting "+planted+" inline"); len(problems) != 0 {
			t.Fatalf("an underscore-form UUID (which the scrubber does not match) was wrongly flagged: %v", problems)
		}
	})

	t.Run("exempt_value_passes", func(t *testing.T) {
		group1, group2, group3, group4, group5 := "00000000", "0000", "0000", "0000", "000000000000"
		planted := strings.Join([]string{group1, group2, group3, group4, group5}, "-")
		defer delete(dashUUIDScrubberExemptValues, planted)
		dashUUIDScrubberExemptValues[planted] = true
		if problems := dashUUIDProblems("scratch", "example text quoting "+planted+" inline"); len(problems) != 0 {
			t.Fatalf("an explicitly-exempted dash-form UUID was wrongly flagged: %v", problems)
		}
	})

	t.Run("no_uuid_passes", func(t *testing.T) {
		if problems := dashUUIDProblems("scratch", "plain text with no UUID-shaped substring at all"); len(problems) != 0 {
			t.Fatalf("plain text with no UUID was wrongly flagged: %v", problems)
		}
	})
}

// TestDashFormUUIDPatternMirrorsScrubber pins dashFormUUIDPatternSource
// against tools/sync/scrub.go's own uuidRegex source line, read as raw
// text (not imported -- this is a _test.go file in a different package,
// and comparing raw source text is the same portable approach this file's
// other cross-package pins already use). Fails loudly if the two patterns
// drift apart, so a future scrubber regex change doesn't silently leave
// this guard checking the wrong shape.
func TestDashFormUUIDPatternMirrorsScrubber(t *testing.T) {
	data, err := os.ReadFile("../tools/sync/scrub.go")
	if err != nil {
		t.Skipf("../tools/sync/scrub.go is unreadable (%v) -- tools/sync is excluded from the assembled public package, so this cross-check is structurally unavailable there", err)
	}
	if !strings.Contains(string(data), dashFormUUIDPatternSource) {
		t.Fatalf("tools/sync/scrub.go's uuidRegex source no longer contains the exact pattern %q this guard mirrors -- update dashFormUUIDPattern/dashFormUUIDPatternSource to match", dashFormUUIDPatternSource)
	}
}
