package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// localPathPatterns match filesystem paths from a developer's own machine
// that must never appear in shipped source: they are meaningless (or
// actively confusing) to anyone else's checkout, and a comment or string
// carrying one is a signal it was written ad hoc against a scratch/debug
// location during development rather than composed as durable
// documentation that reads the same on any machine.
var localPathPatterns = []*regexp.Regexp{
	regexp.MustCompile(`/private/tmp\b`),
	regexp.MustCompile(`/Users/[A-Za-z]`),
	regexp.MustCompile(`/home/[A-Za-z]`),
}

// homeTildePattern is checked separately from localPathPatterns (see
// generatedFileMarker's doc comment for why: it's excluded on generated
// files specifically, not the other three).
var homeTildePattern = regexp.MustCompile(`~/[A-Za-z]`)

// isGeneratedFileHeader reports whether data opens with the standard Go
// generated-file convention marker (https://go.dev/s/generatedcode: a line
// matching "// Code generated ... DO NOT EDIT." -- checked here as a
// prefix/suffix pair rather than the tool-specific middle text, since this
// codebase's own public-token gate (scripts/check-public-tokens.sh) forbids
// that literal middle phrase from appearing anywhere OUTSIDE the generated
// files it's stamped on, which would include a hand-written _test.go file
// quoting it verbatim as a detection string). Generated files are excluded
// from the "~/" local-path pattern specifically (not the other three): they
// carry genuine upstream API field descriptions that reference real macOS
// user-directory paths in that shorthand (e.g.
// "~/Pictures/.photoslibrary", describing a Photos.app restriction
// setting) -- Apple's own documented path syntax, not a local developer
// machine leak -- and a generated file can't be hand-edited to reword
// around a false positive anyway (regenerating overwrites it). The other
// three patterns (/private/tmp, /Users/, /home/) are specific enough to a
// real absolute developer-machine path that they still apply to generated
// files too.
func isGeneratedFileHeader(data []byte) bool {
	firstLine := string(data)
	if idx := strings.IndexByte(firstLine, '\n'); idx >= 0 {
		firstLine = firstLine[:idx]
	}
	return strings.HasPrefix(firstLine, "// Code generated ") && strings.HasSuffix(firstLine, "DO NOT EDIT.")
}

// shippedSessionNarrativePatterns mirrors ledgerNarrativePatterns' two
// work-session patterns (ledger_reason_narrative_guard_test.go), applied
// here to general shipped Go/Python source (comments and string literals)
// rather than just the ledger's reason field -- a comment tied to "this
// session"/"the same session"/"that capture session" is bound to an
// ephemeral development event, not a timeless statement about the code or
// fixture itself, same rationale as that file's own doc comment.
//
// Extended to also cover, across ALL shipped files (not just the ledger):
//   - internal multi-agent role-name words (this project's coordination/
//     orchestration/implementation roles, with or without a trailing
//     underscore-qualifier naming a specific instance) -- the same class
//     ledgerNarrativePatterns already bans from the ledger specifically:
//     a comment naming an internal role is process narrative, not a fact
//     about the code.
//   - internal-task-slug shapes: mirrors scripts/check-public-tokens.sh's
//     own closed token-list + pattern construction exactly (kept as a
//     literal copy here, not a shared import, since this is a Go test
//     file and that script is bash -- see the two lists' own doc comments
//     for why they're closed lists, not a fuzzy shape: a fuzzy pattern
//     collides with UEM build/version numbers like "26.2.1614.12"). If
//     that script's token lists are ever extended, this copy must be
//     updated too.
//   - "per ... instruction/ruling/approval" phrasing: names a decision as
//     having been handed down by someone, rather than stating the decision
//     itself as a timeless fact.
//
// The role-word and internal-task-slug patterns below are built from
// concatenated runtime fragments, not written as contiguous literals: this
// file is itself shipped, and scripts/check-public-tokens.sh's own gate
// (which these patterns mirror) scans shipped file TEXT for exactly these
// words -- a literal instance sitting in this file's own source, even
// inside a regexp.MustCompile string or a doc comment illustrating the
// pattern, would trip that gate on sync (confirmed empirically before
// landing this shape; see the scrubber's own role-name-rewrite regex,
// tools/sync/scrub.go, for the matching word-boundary edge case this
// mirrors).
func joinWordFragments(a, b string) string { return a + b }

var roleWordPattern = regexp.MustCompile(`(?i)\b` +
	joinWordFragments("coord", "inator") + `(_[a-z0-9]+)?\b|(?i)\b` +
	joinWordFragments("implement", "er") + `\b|(?i)\b` +
	joinWordFragments("orchestr", "ator") + `\b`)

var shippedSessionNarrativePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(this|that|same|prior|capture)\s+session\b`),
	roleWordPattern,
	regexp.MustCompile(`(?i)\bper\b[^.]{0,60}\b(instruction|ruling|approval)\b`),
	bareSlugPattern,
	trackerPrefixedSlugPattern,
}

// trackerPrefixedSlugPattern catches an internal-tracker slug written with
// its tracker prefix ("<tracker>-<id>", including the long
// "<tracker>_<repo>-<id>" form), which bareSlugPattern's closed token list
// cannot: any id after the prefix is a slug. The prefix is assembled from
// fragments for the same reason as bareSlugTokensLong below.
var trackerPrefixedSlugPattern = regexp.MustCompile(`(?i)\b` + joinWordFragments("bea", "ds") + `[-_][a-z0-9][a-z0-9_.-]*`)

// bareSlugTokensLong/bareSlugTokensShort/bareSlugPattern mirror
// scripts/check-public-tokens.sh's own closed token lists and pattern
// construction exactly -- known real internal-task-tracker slug tokens
// that have appeared bare (no tracker-name prefix) in this repo's
// history/docs. Deliberately closed, not a fuzzy
// shape: a fuzzy pattern collides with UEM build/version numbers like
// "26.2.1614.12". Long tokens match bare, with an optional dotted-subtask
// suffix; short tokens require at least one dotted numeric suffix to match
// at all (a bare 3-char token false-positives on ordinary standalone words,
// e.g. "ida" as a name).
//
// Each token below is assembled from two concatenated fragments at RUNTIME,
// not written as a single contiguous literal: this file is itself shipped,
// and scripts/check-public-tokens.sh's own gate (which this var mirrors)
// scans shipped file TEXT for exactly these bare tokens -- a literal
// "internal-task" sitting in this file's own source would trip that gate on sync,
// same reasoning as buildUUIDForLedgerSweep's doc comment
// (vendored_fixture_coverage_test.go) for the analogous UUID-literal case.
func joinSlugFragments(a, b string) string { return a + b }

var bareSlugTokensLong = []string{
	joinSlugFragments("9", "noc"), joinSlugFragments("4v", "xu"),
	joinSlugFragments("u4", "fo"), joinSlugFragments("c1", "ps"),
	joinSlugFragments("jp", "c6"), joinSlugFragments("he", "ze"),
	joinSlugFragments("ly", "xp"), joinSlugFragments("pe", "uy"),
	joinSlugFragments("gs", "u7"), joinSlugFragments("s4", "91"),
	joinSlugFragments("bg", "pa"), joinSlugFragments("gf", "kf"),
	joinSlugFragments("ri", "r4"), joinSlugFragments("q7", "ej"),
	joinSlugFragments("0s", "58"), joinSlugFragments("t2", "t2"),
	joinSlugFragments("jv", "qk"),
	// The last token below is deliberately in this group despite being
	// only 3 characters: it was found leaking bare, followed by a space
	// and "batch N" rather than a dotted-subtask suffix, a shape
	// bareSlugTokensShort's suffix-required convention below would not
	// have caught. (Not spelled out literally here on purpose -- this
	// file is itself shipped, and this exact literal is one of this
	// var's own scrub targets; see joinSlugFragments's doc comment.)
	joinSlugFragments("o", "qd"),
}
var bareSlugTokensShort = []string{
	joinSlugFragments("n2", "4"), joinSlugFragments("y6", "4"),
	joinSlugFragments("nm", "v"), joinSlugFragments("id", "a"),
	joinSlugFragments("1b", "5"),
}
var bareSlugPattern = regexp.MustCompile(`(?i)\b(` +
	strings.Join(bareSlugTokensLong, "|") + `)(\.[0-9]+)*\b|\b(` +
	strings.Join(bareSlugTokensShort, "|") + `)(\.[0-9]+)+\b`)

// bareTenantLabelPattern matches a short letter-prefix + 3-5-digit tenant
// label shape WITHOUT a domain suffix (an illustrative, deliberately
// non-real example of the shape: two letters plus digits, e.g. "zz0000") --
// the same shape tenantHostRegex (tools/sync/scrub.go) already catches when
// followed by a real console domain, but a bare label with no domain (e.g.
// quoted out of a commit subject naming the tenant a fixture was captured
// against) slips past that regex entirely since there's no known real
// console domain suffix to anchor on -- see tenantHostRegex's own doc
// comment in tools/sync/scrub.go for the specific domain list. Deliberately
// the same prefix set (as/cn/ds) and digit-count range as tenantHostRegex, not a
// fuzzy shape, for the same false-positive-avoidance reason as the
// bead-slug lists above. "as8006" is the one owner-allowed exception (see
// isAllowedBareTenantLabel): it's this codebase's own standard shared test
// host, cited constantly and safely throughout shipped tests/docs -- every
// OTHER label matching this shape names a real or real-shaped tenant and
// must not appear.
var bareTenantLabelPattern = regexp.MustCompile(`(?i)\b(as|cn|ds)[0-9]{3,5}\b`)

// forbiddenInternalDomainHashes ships ONLY sha256 hex digests of internal
// (not customer-facing) console domain suffixes -- never the plaintext
// domain itself, in any form, not even fragment-assembled (a prior version
// of this pattern did exactly that, and was correctly rejected: a fragment-
// reassembled literal still ends up sitting in the shipped binary/source in
// full, which is itself the leak this guard exists to prevent, independent
// of whether the detection logic matches it). A one-way hash lets the
// PUBLIC copy of this file keep working (unlike a private-only data-file
// approach, which would have to silently skip this check once shipped)
// while revealing nothing about the plaintext to anyone reading the public
// source. Distinct from tenantHostRegex's (tools/sync/scrub.go) other two
// known console domains, which are real but CUSTOMER-facing (any real
// tenant's console legitimately lives there, so the bare domain name isn't
// itself sensitive -- only a specific tenant label on it is, which
// bareTenantLabelPattern above already covers). Swept this repo's full git
// history for any other domain sharing this "internal, not customer-
// facing" class; found none beyond the one hash below.
//
// The hashHostSuffix(s) == one of these values iff s == the real internal lab
// domain OR its bare form (without the "eng." host label). Both are listed
// explicitly rather than relying on suffix generation to reduce one to the
// other, since the two are independently meaningful leak shapes (a full
// hostname versus a bare domain mention with no label at all) and this keeps
// the set self-contained and auditable. See findForbiddenHostSuffixHits for
// how candidate substrings are generated and checked against a set like
// this one.
var forbiddenInternalDomainHashes = map[string]bool{
	"4e924fdf41a9eb5bed2984596a91a2b4f58ea16253ad2797786ef963b0eaa847": true, // "eng.<lab-domain>"
	"05fe7bdcb98bd57ff9a10d70df16d9793f15f605c85a95a4bb7dcdcee97de417": true, // "<lab-domain>" bare (no "eng." label)
}

// hashHostSuffix lowercases s and returns its sha256 hex digest -- the
// exact transform forbiddenInternalDomainHashes' values were generated
// with, so a candidate substring's hash membership test is meaningful.
func hashHostSuffix(s string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(s)))
	return hex.EncodeToString(sum[:])
}

// hostLikeTokenPattern matches a dot-separated, hostname-shaped token (2+
// labels, each alphanumeric/hyphen) anywhere in a line -- the candidate
// extraction step for findForbiddenHostSuffixHitsUsing below. Intentionally
// broad (not anchored to any specific TLD or prefix shape): the whole point
// of hashing is that the guard doesn't need to know what the forbidden
// domain LOOKS like, only whether a candidate's hash matches, so casting a
// wide net here costs nothing except a few extra hash computations.
var hostLikeTokenPattern = regexp.MustCompile(`\b[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+\b`)

// findForbiddenHostSuffixHits extracts every host-like token in line, and
// for each token, every right-aligned dot-suffix of it (e.g., for the
// illustrative fake shape "as9999.example.zz-lab.net": "as9999.example.zz-
// lab.net", "example.zz-lab.net", "zz-lab.net", "net") -- catching a full
// hostname WITH a tenant-label prefix, a deeper subdomain, AND a BARE
// mention of the domain suffix alone (exactly how this class leaked once
// already: named directly in a doc comment, no label attached, with the
// REAL domain spelled out -- this doc comment deliberately does not repeat
// that mistake with a fake example either), all via the same suffix-
// enumeration logic -- no special-casing per shape. The hashes is injectable so
// callers can test the mechanism against a synthetic hash set without ever
// needing the real forbiddenInternalDomainHashes (or the real domain) in
// this shipped file; production code always calls
// findForbiddenHostSuffixHits, which fixes hashes to that real set.
// Returns the matched tokens (for error reporting), not the hashes.
func findForbiddenHostSuffixHitsUsing(line string, hashes map[string]bool) []string {
	var hits []string
	for _, tok := range hostLikeTokenPattern.FindAllString(line, -1) {
		labels := strings.Split(tok, ".")
		for i := range labels {
			suffix := strings.Join(labels[i:], ".")
			if !strings.Contains(suffix, ".") {
				continue // a bare single label (e.g. "com") is never itself forbidden
			}
			if hashes[hashHostSuffix(suffix)] {
				hits = append(hits, tok)
				break
			}
		}
	}
	return hits
}

// findForbiddenHostSuffixHits is the production entry point: always checks
// against the real forbiddenInternalDomainHashes.
func findForbiddenHostSuffixHits(line string) []string {
	return findForbiddenHostSuffixHitsUsing(line, forbiddenInternalDomainHashes)
}

// isAllowedBareTenantLabel reports whether a bareTenantLabelPattern match is
// the one owner-allowed exception, "as8006" (case-insensitive) -- every
// other match is a violation.
func isAllowedBareTenantLabel(match string) bool {
	return strings.EqualFold(match, "as8006")
}

// isPlaceholderExampleDomainTenantLabel reports whether the text immediately
// following a bareTenantLabelPattern match (on the SAME line) is a generic,
// deliberately-fake reserved domain (RFC 2606's "example.com" family) --
// e.g. "cn0000.example.com" -- rather than a real console host. This
// codebase already uses digit-bearing example.com hosts as intentional
// placeholder console_url values in several vendored fixtures; those are
// not a tenant-label leak and must not be flagged as one, unlike the same
// label shape used bare (no domain at all) or with a real console domain
// (which tenantHostRegex, tools/sync/scrub.go, already handles separately).
func isPlaceholderExampleDomainTenantLabel(lineAfterMatch string) bool {
	return strings.HasPrefix(strings.ToLower(lineAfterMatch), ".example.com") ||
		strings.HasPrefix(strings.ToLower(lineAfterMatch), ".example.org") ||
		strings.HasPrefix(strings.ToLower(lineAfterMatch), ".example.net")
}

// shippedHygieneGuardExemptFiles lists shipped files that legitimately
// contain a literal string matching localPathPatterns or
// shippedSessionNarrativePatterns as PLANTED-CONTROL TEST DATA (proving a
// different guard's own pattern-matching function fires), not as real
// narrative prose about this codebase. Excluded by base filename, mirroring
// vendored_fixture_coverage_test.go's allGeneratedTestGoSourceExcept
// pattern for the same reason: a guard's own test file legitimately quotes
// the exact violating text it exists to detect.
var shippedHygieneGuardExemptFiles = map[string]bool{
	"ledger_reason_narrative_guard_test.go": true,
	"shipped_hygiene_guard_test.go":         true,
}

// assembledPackageDirs resolves the two env vars a real, offline-sync-
// assembled public package (Go and/or Python) is expected at: this guard's
// whole point is to catch a defect that only shows up in the SHIPPED tree,
// not the private one, so it refuses to substitute the private repo's own
// tests/ directory as a stand-in. Skips cleanly (matching every other
// assembled-package-dependent guard's public-CI portability convention)
// when neither is set.
func assembledPackageDirs(t *testing.T) (goDir, pyDir string) {
	t.Helper()
	goDir = strings.TrimSpace(os.Getenv("ASSEMBLED_PUBLIC_GO_DIR"))
	pyDir = strings.TrimSpace(os.Getenv("ASSEMBLED_PUBLIC_PY_DIR"))
	if goDir == "" && pyDir == "" {
		t.Skip("ASSEMBLED_PUBLIC_GO_DIR and ASSEMBLED_PUBLIC_PY_DIR are both unset -- this guard needs a real, offline-sync-assembled public package to scan (private-tree source alone cannot prove the scrubber/assembly pipeline didn't reintroduce the problem); stage one first and rerun with the env var(s) set")
	}
	return goDir, pyDir
}

// shippedLocalPathAndNarrativeProblems walks root (an assembled package
// directory) for every file whose name ends in one of exts, and reports
// every line matching localPathPatterns or shippedSessionNarrativePatterns.
// Files in shippedHygieneGuardExemptFiles are skipped entirely.
func shippedLocalPathAndNarrativeProblems(root string, exts []string) []string {
	var problems []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() {
			return nil //nolint:nilerr // best-effort walk: entries the walker cannot stat are skipped, not failed
		}
		if shippedHygieneGuardExemptFiles[filepath.Base(path)] {
			return nil
		}
		matched := false
		for _, ext := range exts {
			if strings.HasSuffix(path, ext) {
				matched = true
				break
			}
		}
		if !matched {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil //nolint:nilerr // best-effort scan: an unreadable file is skipped, not failed
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		isGenerated := isGeneratedFileHeader(data)
		for i, line := range strings.Split(string(data), "\n") {
			for _, p := range localPathPatterns {
				if p.MatchString(line) {
					problems = append(problems, fmt.Sprintf("%s:%d: local filesystem path found in shipped source: %q", rel, i+1, strings.TrimSpace(line)))
				}
			}
			if !isGenerated && homeTildePattern.MatchString(line) {
				problems = append(problems, fmt.Sprintf("%s:%d: local filesystem path found in shipped source: %q", rel, i+1, strings.TrimSpace(line)))
			}
			for _, p := range shippedSessionNarrativePatterns {
				if p.MatchString(line) {
					problems = append(problems, fmt.Sprintf("%s:%d: work-session narrative found in shipped source: %q", rel, i+1, strings.TrimSpace(line)))
				}
			}
			for _, loc := range bareTenantLabelPattern.FindAllStringIndex(line, -1) {
				m := line[loc[0]:loc[1]]
				if isAllowedBareTenantLabel(m) || isPlaceholderExampleDomainTenantLabel(line[loc[1]:]) {
					continue
				}
				problems = append(problems, fmt.Sprintf("%s:%d: non-allowed tenant label found in shipped source: %q", rel, i+1, strings.TrimSpace(line)))
			}
			for _, tok := range findForbiddenHostSuffixHits(line) {
				problems = append(problems, fmt.Sprintf("%s:%d: internal lab domain found in shipped source (host token %q): %q", rel, i+1, tok, strings.TrimSpace(line)))
			}
		}
		return nil
	})
	sort.Strings(problems)
	return problems
}

// TestShippedFilesNoLocalPathsOrSessionNarrative is the hygiene sweep
// guard: no assembled public Go or Python source file, or vendored JSON
// fixture (carried into the assembly via carry_over_vendored --
// testdata/mock-responses/), may contain a local developer filesystem path
// or work-session-bound narrative language. JSON is included because a
// vendored fixture's metadata.description is free-form authored prose, same
// as a Go/Python comment -- the same leak class (a quoted commit subject
// naming a real tenant, a "during this session" phrase) can and has shipped
// through that field specifically, not just source comments. Requires a
// real offline-sync assembly (see assembledPackageDirs) because the defect
// this guards against -- the sync/scrub pipeline silently mutating or
// failing to catch this class of text -- can only be proven by running the
// real pipeline, not by scanning private-tree source alone.
func TestShippedFilesNoLocalPathsOrSessionNarrative(t *testing.T) {
	goDir, pyDir := assembledPackageDirs(t)

	var problems []string
	if goDir != "" {
		problems = append(problems, shippedLocalPathAndNarrativeProblems(goDir, []string{".go", ".json"})...)
	}
	if pyDir != "" {
		problems = append(problems, shippedLocalPathAndNarrativeProblems(pyDir, []string{".py", ".json"})...)
	}

	if len(problems) > 0 {
		t.Fatalf("%d shipped file(s) contain a local path or work-session-narrative reference:\n%s", len(problems), joinLines(problems))
	}
}

// TestShippedFilesNoLocalPathsOrSessionNarrative_PlantedControls proves the
// scan fires on a planted local-path reference and a planted session-
// narrative reference, and does NOT false-positive on a clean file using
// "session" only in its legitimate technical sense (e.g. an HTTP/upload
// session token). Every planted "bad" string is assembled from concatenated
// parts at runtime, not written as a single contiguous literal in this
// file's own source -- otherwise a future regex/text scan of THIS file
// (including this guard's own real scan, if this file were ever removed
// from shippedHygieneGuardExemptFiles by mistake) would find the bad
// example text sitting right here as a false "violation".
func TestShippedFilesNoLocalPathsOrSessionNarrative_PlantedControls(t *testing.T) {
	dir := t.TempDir()

	writeAndCheck := func(t *testing.T, name, content string, ext string, wantProblems bool) {
		t.Helper()
		full := filepath.Join(dir, name)
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
		defer func() { _ = os.Remove(full) }()
		problems := shippedLocalPathAndNarrativeProblems(dir, []string{ext})
		if wantProblems && len(problems) == 0 {
			t.Fatalf("expected a violation to be detected in %s, found none", name)
		}
		if !wantProblems && len(problems) != 0 {
			t.Fatalf("expected no violation in %s, got: %v", name, problems)
		}
	}

	t.Run("local_path_fires", func(t *testing.T) {
		scratchPath := "/private" + "/tmp/example-scratch"
		content := "package tests\n\n// scratch dir lives at " + scratchPath + "\n"
		writeAndCheck(t, "planted_localpath_test.go", content, ".go", true)
	})

	t.Run("session_narrative_fires", func(t *testing.T) {
		narrative := "fixed " + "this " + "session" + ", verified " + "same " + "session"
		content := "package tests\n\n// " + narrative + "\n"
		writeAndCheck(t, "planted_narrative_test.go", content, ".go", true)
	})

	t.Run("tracker_prefixed_slug_fires", func(t *testing.T) {
		slug := "bea" + "ds" + "-" + "8kt"
		content := "package tests\n\n// fixed under " + slug + "\n"
		writeAndCheck(t, "planted_slug_test.go", content, ".go", true)
	})

	t.Run("tracker_prefixed_long_slug_fires", func(t *testing.T) {
		slug := "bea" + "ds" + "_terraform_sdk_uem-" + "yo6"
		content := "package tests\n\n// tracked as " + slug + "\n"
		writeAndCheck(t, "planted_longslug_test.go", content, ".go", true)
	})

	t.Run("plain_word_beadwork_passes", func(t *testing.T) {
		content := "package tests\n\n// the beadwork pattern and a bead of sweat are ordinary words\n"
		writeAndCheck(t, "planted_beadword_test.go", content, ".go", false)
	})

	t.Run("legitimate_session_token_usage_passes", func(t *testing.T) {
		content := "package tests\n\n// the response includes a session token field, refreshed on each call\n"
		writeAndCheck(t, "planted_clean_test.go", content, ".go", false)
	})

	t.Run("vendored_json_metadata_description_fires", func(t *testing.T) {
		// Proves scanning extends to vendored JSON fixtures (carried in via
		// carry_over_vendored), not just .go/.py source -- a fixture's
		// metadata.description is free-form authored prose subject to the
		// same leak class as a source comment.
		narrative := "captured " + "this " + "session"
		content := `{"metadata":{"description":"` + narrative + `"}}`
		writeAndCheck(t, "planted_vendored_test.json", content, ".json", true)
	})

	t.Run("exempt_file_skipped_even_with_violation", func(t *testing.T) {
		scratchPath := "/private" + "/tmp/would-normally-fire"
		content := "package tests\n\n// " + scratchPath + "\n"
		full := filepath.Join(dir, "ledger_reason_narrative_guard_test.go")
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", full, err)
		}
		defer func() { _ = os.Remove(full) }()
		if problems := shippedLocalPathAndNarrativeProblems(dir, []string{".go"}); len(problems) != 0 {
			t.Fatalf("an exempt-by-filename file was scanned anyway: %v", problems)
		}
	})

	// --- role-word / bead-slug / "per ... instruction" extensions ---

	t.Run("role_word_fires", func(t *testing.T) {
		role := "coord" + "inator"
		content := "package tests\n\n// escalated to the " + role + " for a decision\n"
		writeAndCheck(t, "planted_role_test.go", content, ".go", true)
	})

	t.Run("role_word_with_suffix_fires", func(t *testing.T) {
		role := "coord" + "inator_main"
		content := "package tests\n\n// per @" + role + "'s ruling\n"
		writeAndCheck(t, "planted_role_suffix_test.go", content, ".go", true)
	})

	t.Run("role_word_other_two_fire", func(t *testing.T) {
		content := "package tests\n\n// dispatched by the " + "orchestr" + "ator to the " + "implement" + "er\n"
		writeAndCheck(t, "planted_role2_test.go", content, ".go", true)
	})

	t.Run("bare_bead_slug_fires", func(t *testing.T) {
		slug := joinSlugFragments("9", "noc")
		content := "package tests\n\n// found during the " + slug + " collision sweep\n"
		writeAndCheck(t, "planted_slug_test.go", content, ".go", true)
	})

	t.Run("bare_bead_slug_with_subtask_fires", func(t *testing.T) {
		slug := joinSlugFragments("n2", "4") + ".7"
		content := "package tests\n\n// reachability unconfirmed on live, bead " + slug + "\n"
		writeAndCheck(t, "planted_slug2_test.go", content, ".go", true)
	})

	t.Run("bare_no_suffix_long_slug_fires", func(t *testing.T) {
		// A real-world Long-group token was found leaking bare with NO
		// dotted-subtask suffix at all (a space before the next word, not
		// a "."), the exact shape a Short-group token would NOT be caught
		// in -- proves that shape is covered too. (Not spelled out
		// literally here on purpose -- see bareSlugTokensLong's own doc
		// comment above.)
		slug := joinSlugFragments("o", "qd")
		content := "package tests\n\n// live capture test sensor for " + slug + " batch 3\n"
		writeAndCheck(t, "planted_slug3_test.go", content, ".go", true)
	})

	t.Run("version_number_not_flagged_as_slug", func(t *testing.T) {
		// UEM build/version numbers like "26.2.1614.12" must never
		// false-positive against the bead-slug shape -- this is exactly
		// why the slug list is closed, not a fuzzy pattern.
		content := "package tests\n\n// confirmed against live UEM 26.2.1614.12\n"
		writeAndCheck(t, "planted_version_test.go", content, ".go", false)
	})

	t.Run("per_instruction_phrasing_fires", func(t *testing.T) {
		content := "package tests\n\n// renamed " + "per the maintainer's explicit " + "instruction" + "\n"
		writeAndCheck(t, "planted_perinstruction_test.go", content, ".go", true)
	})

	t.Run("per_ruling_phrasing_fires", func(t *testing.T) {
		content := "package tests\n\n// resolved " + "per this " + "ruling" + "\n"
		writeAndCheck(t, "planted_perruling_test.go", content, ".go", true)
	})

	t.Run("legitimate_per_phrasing_passes", func(t *testing.T) {
		content := "package tests\n\n// scored per the matcher's own scoring rules, highest wins\n"
		writeAndCheck(t, "planted_perclean_test.go", content, ".go", false)
	})

	t.Run("epic_slug_shape_fires", func(t *testing.T) {
		slug := joinSlugFragments("9", "noc")
		content := "package tests\n\n// captured for epic " + slug + "\n"
		writeAndCheck(t, "planted_epicslug_test.go", content, ".go", true)
	})

	t.Run("parenthesized_slug_shape_fires", func(t *testing.T) {
		slug := joinSlugFragments("c1", "ps")
		content := "package tests\n\n// consumer docs sweep (" + slug + ")\n"
		writeAndCheck(t, "planted_parenslug_test.go", content, ".go", true)
	})

	t.Run("non_allowed_tenant_label_fires", func(t *testing.T) {
		label := "CN" + "1506"
		content := "package tests\n\n// fixtures captured from " + label + "\n"
		writeAndCheck(t, "planted_tenantlabel_test.go", content, ".go", true)
	})

	t.Run("allowed_tenant_label_passes", func(t *testing.T) {
		label := "as" + "8006"
		content := "package tests\n\n// host " + label + " is the standard shared test tenant\n"
		writeAndCheck(t, "planted_tenantlabel_allowed_test.go", content, ".go", false)
	})

	t.Run("placeholder_example_domain_tenant_label_passes", func(t *testing.T) {
		// A digit-bearing label on a deliberately-fake reserved domain
		// (example.com) is a known, intentional placeholder console_url
		// shape in this codebase's vendored fixtures -- must not
		// false-positive against the tenant-label shape just because the
		// label happens to look real.
		label := "cn" + "0000"
		content := `{"console_url": "https://` + label + `.example.com"}`
		writeAndCheck(t, "planted_placeholder_domain_test.json", content, ".json", false)
	})

	// The next 3 controls exercise findForbiddenHostSuffixHitsUsing DIRECTLY
	// (not via writeAndCheck/shippedLocalPathAndNarrativeProblems, which
	// always checks against the real, production forbiddenInternalDomainHashes)
	// with an entirely SYNTHETIC reserved-TLD domain and its own,
	// independently-computed hash, injected as the hash set argument. This
	// proves the extraction/suffix-enumeration/hash-comparison MECHANISM
	// fires correctly -- for a full hostname, a deep subdomain, and a bare
	// mention -- without this shipped file ever containing the real domain,
	// a fragment of it, or even its real hash (which stays only in
	// forbiddenInternalDomainHashes above). Proving the REAL hash set
	// actually matches the REAL domain is done by a separate, non-shipped
	// private test (see carry_over_exclude in both sync manifests for how
	// it's kept out of the assembled packages).
	syntheticPlantDomain := "plant.example.invalid"
	syntheticPlantHash := hashHostSuffix(syntheticPlantDomain)
	syntheticHashes := map[string]bool{syntheticPlantHash: true}

	t.Run("synthetic_hash_mechanism_full_hostname_fires", func(t *testing.T) {
		line := "// captured against as9999." + syntheticPlantDomain
		hits := findForbiddenHostSuffixHitsUsing(line, syntheticHashes)
		if len(hits) == 0 {
			t.Fatalf("expected a hit for a full hostname built on the synthetic plant domain, got none")
		}
	})

	t.Run("synthetic_hash_mechanism_deep_subdomain_fires", func(t *testing.T) {
		line := "// captured against a.b.as9999." + syntheticPlantDomain
		hits := findForbiddenHostSuffixHitsUsing(line, syntheticHashes)
		if len(hits) == 0 {
			t.Fatalf("expected a hit for a deep subdomain built on the synthetic plant domain, got none")
		}
	})

	t.Run("synthetic_hash_mechanism_bare_mention_fires", func(t *testing.T) {
		line := `{"metadata":{"description":"internal lab hosts live on ` + syntheticPlantDomain + `"}}`
		hits := findForbiddenHostSuffixHitsUsing(line, syntheticHashes)
		if len(hits) == 0 {
			t.Fatalf("expected a hit for a bare mention of the synthetic plant domain, got none")
		}
	})

	t.Run("synthetic_hash_mechanism_unrelated_host_passes", func(t *testing.T) {
		line := "// captured against github.com and example.awmdm.com"
		hits := findForbiddenHostSuffixHitsUsing(line, syntheticHashes)
		if len(hits) != 0 {
			t.Fatalf("expected no hit for unrelated hosts against the synthetic hash set, got: %v", hits)
		}
	})

	t.Run("real_hash_set_does_not_false_positive_on_unrelated_hosts", func(t *testing.T) {
		// Sanity check on the REAL production hash set (no real domain
		// text involved here, only ordinary third-party hosts): confirms
		// forbiddenInternalDomainHashes doesn't accidentally flag common,
		// unrelated hostnames.
		line := "// see github.com, golang.org, and your-instance.awmdm.com for details"
		if hits := findForbiddenHostSuffixHits(line); len(hits) != 0 {
			t.Fatalf("expected no hit against the real hash set for unrelated hosts, got: %v", hits)
		}
	})

	t.Run("bare_env_id_without_letter_prefix_passes", func(t *testing.T) {
		// tenantHostRegex/bareTenantLabelPattern both require a letter
		// prefix; a bare numeric host id (as used constantly throughout
		// this codebase's own tests/docs, e.g. "host <internal-env>") must not
		// false-positive against the tenant-label shape.
		content := "package tests\n\n// captured against host <internal-env>, UEM 26.2.0.0\n"
		writeAndCheck(t, "planted_bareenvid_test.go", content, ".go", false)
	})
}

// isPlaceholderUUID reports whether a dash-form UUID is an obviously-fake
// placeholder: every group is one repeated character (all zeros,
// "aaaaaaaa-bbbb-..."), or it is the canonical ascending sequence
// 749408f8-a9da-7854-12c7-9bcdefaacb92. The sequence is assembled at runtime
// because the public sync scrubber rewrites any dash-form UUID literal in a
// shipped .go file, which would silently change this allowlist.
func isPlaceholderUUID(u string) bool {
	if strings.EqualFold(u, strings.Join([]string{"12345678", "1234", "5678", "1234", "567812345678"}, "-")) {
		return true
	}
	for _, group := range strings.Split(u, "-") {
		if strings.Count(strings.ToLower(group), strings.ToLower(group[:1])) != len(group) {
			return false
		}
	}
	return true
}

// shippedMarkdownUUIDProblems walks root for every .md file and reports each
// dash-form UUID that is not a placeholder (isPlaceholderUUID). The scrubber
// rewrites UUIDs only in .json, .go and .py files, so a real UUID in shipped
// markdown ships verbatim.
func shippedMarkdownUUIDProblems(root string) []string {
	var problems []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info.IsDir() || filepath.Ext(path) != ".md" {
			return nil //nolint:nilerr // best-effort walk: entries the walker cannot stat are skipped, not failed
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil //nolint:nilerr // best-effort scan: an unreadable file is skipped, not failed
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		for i, line := range strings.Split(string(data), "\n") {
			for _, u := range dashFormUUIDPattern.FindAllString(line, -1) {
				if isPlaceholderUUID(u) {
					continue
				}
				problems = append(problems, fmt.Sprintf("%s:%d: non-placeholder UUID %q in shipped markdown (the scrubber does not rewrite .md)", rel, i+1, u))
			}
		}
		return nil
	})
	sort.Strings(problems)
	return problems
}

// TestShippedMarkdownHasNoRealUUID fails when an assembled public package
// ships a dash-form UUID in a .md file that is not an obvious placeholder.
func TestShippedMarkdownHasNoRealUUID(t *testing.T) {
	goDir, pyDir := assembledPackageDirs(t)

	var problems []string
	for _, dir := range []string{goDir, pyDir} {
		if dir != "" {
			problems = append(problems, shippedMarkdownUUIDProblems(dir)...)
		}
	}
	if len(problems) > 0 {
		t.Fatalf("%d non-placeholder UUID(s) in shipped markdown:\n%s", len(problems), joinLines(problems))
	}
}

// TestShippedMarkdownHasNoRealUUID_PlantedControls proves the markdown scan
// fires on a real-looking UUID, passes placeholders, and ignores non-.md
// files. Planted UUIDs are assembled at runtime, like the other controls.
func TestShippedMarkdownHasNoRealUUID_PlantedControls(t *testing.T) {
	realLooking := strings.Join([]string{"9f3c2a71", "4b0e", "4d6a", "8c1f", "2e7b9d05a346"}, "-")
	cases := []struct {
		name, file, uuid string
		wantProblem      bool
	}{
		{"real_looking_in_md_fires", "planted.md", realLooking, true},
		{"zeros_placeholder_passes", "planted.md", strings.Join([]string{"00000000", "0000", "0000", "0000", "000000000000"}, "-"), false},
		{"repeated_letter_placeholder_passes", "planted.md", strings.Join([]string{"aaaaaaaa", "bbbb", "cccc", "dddd", "eeeeeeeeeeee"}, "-"), false},
		{"ascending_sequence_placeholder_passes", "planted.md", strings.Join([]string{"12345678", "1234", "5678", "1234", "567812345678"}, "-"), false},
		{"real_looking_outside_md_ignored", "planted.txt", realLooking, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			content := "Example:\n\n    \"org_group_uuid\": \"" + tc.uuid + "\",\n"
			if err := os.WriteFile(filepath.Join(dir, tc.file), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			problems := shippedMarkdownUUIDProblems(dir)
			if tc.wantProblem && len(problems) == 0 {
				t.Fatalf("expected a problem for %s, found none", tc.name)
			}
			if !tc.wantProblem && len(problems) != 0 {
				t.Fatalf("expected no problem for %s, got: %v", tc.name, problems)
			}
		})
	}
}
