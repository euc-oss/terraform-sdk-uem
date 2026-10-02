package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"testing"
)

// ledgerNarrativeEntry mirrors the subset of tests/verified-endpoints.json's
// row schema this guard inspects.
type ledgerNarrativeEntry struct {
	Reason       string `json:"reason"`
	TestFunction string `json:"testFunction"`
}

type ledgerNarrativeFile struct {
	Verified   []ledgerNarrativeEntry `json:"verified"`
	Unverified []ledgerNarrativeEntry `json:"unverified"`
}

// ledgerNarrativePatterns are the internal-process reference shapes a public
// fixture governance ledger must never carry: labels tied to an ephemeral
// development process (a work session, a numbered round or phase of that
// process, a role performing it, an internal tracking identifier) rather
// than a timeless, self-contained statement of fact about the fixture
// itself. A reason should read the same whether it's read today or in five
// years, by someone with no knowledge of how or when the work happened.
//
// Each pattern is deliberately narrow (a specific phrase or a word bound to
// a following digit/context) rather than a bare keyword, so that the same
// word used in a legitimate technical sense -- "session token", "task
// queue", a plain capture date -- does not false-positive. See
// TestLedgerReasonsAreTimeless's clean-but-trigger-adjacent subtests for the
// cases this was tuned against.
var ledgerNarrativePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(this|that|same|prior|capture)\s+session\b`),
	regexp.MustCompile(`(?i)\bsame\s+session\b`),
	regexp.MustCompile(`\bE5\b`),
	regexp.MustCompile(`(?i)\brun[\s-]\d+\b`),
	regexp.MustCompile(`(?i)\bphase[\s-]?\d+\b`),
	regexp.MustCompile(`(?i)report item`),
	regexp.MustCompile(`(?i)\btask\s+\d+\b`),
	regexp.MustCompile(`(?i)follow-up task`),
	regexp.MustCompile(`(?i)candidate for a new bead`),
	regexp.MustCompile(`(?i)\bbead\b`),
	regexp.MustCompile(`\bmsg_[A-Za-z0-9]+\b`),
	regexp.MustCompile(`(?i)\b(implementer|maintainer|orchestrator)\b`),
	regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}:`),
	regexp.MustCompile(`(?i)\bfeature/[\w.-]+`),
	regexp.MustCompile(`(?i)after the tag\b`),
	regexp.MustCompile(`\b(CTRLP|GHA|ATL|TSDKUEM|UEMSDK)-\d+\b`),
	regexp.MustCompile(`(?i)\bmyself\b`),
	regexp.MustCompile(`(?i)\bour own\b`),
	regexp.MustCompile(`\bQ\d+(-[ivxIVX]+)?\b`),
	regexp.MustCompile(`(?i)stale entry`),
	regexp.MustCompile(`(?i)this entry previously`),
	regexp.MustCompile(`(?i)\bupdate\s*\(\d{4}-\d{2}-\d{2}`),
	regexp.MustCompile(`\b[A-Z][a-z]+-staged\b`),
}

// ledgerVendoredProvenanceAllowed matches the one intentional, universal
// branch-name-shaped citation this corpus uses throughout: pointing at the
// exact vendored commit a fixture came from. It is provenance, not process
// narrative, and must never be flagged by a branch-name-shaped check.
var ledgerVendoredProvenanceAllowed = regexp.MustCompile(`\bchore/vendored-swagger-uem-[\w.-]+\b`)

// ledgerBranchNamePattern flags any OTHER branch-name-shaped reference (a
// feature/topic branch, a numbered work branch) once the allowed vendored-
// provenance citation above has been removed from consideration.
var ledgerBranchNamePattern = regexp.MustCompile(`(?i)\bchore/[\w.-]+\b`)

// ledgerNarrativeViolations returns every pattern-class match found in s,
// after stripping the one allowed vendored-provenance citation shape.
func ledgerNarrativeViolations(s string) []string {
	var hits []string
	for _, p := range ledgerNarrativePatterns {
		if p.MatchString(s) {
			hits = append(hits, p.String())
		}
	}
	stripped := ledgerVendoredProvenanceAllowed.ReplaceAllString(s, "")
	if ledgerBranchNamePattern.MatchString(stripped) {
		hits = append(hits, ledgerBranchNamePattern.String())
	}
	return hits
}

// ledgerUnverifiedCategoryPrefixes are the only category leads an
// unverified[] reason may open with. Every unverified row must state, in
// its first words, WHY the row isn't verified, as one of these fixed
// categories -- not as narrative about how or when that was discovered.
var ledgerUnverifiedCategoryPrefixes = []string{
	"Blocked (tenant precondition)",
	"Blocked (SDK gap)",
	"Deliberate edge case",
}

func ledgerUnverifiedHasCategoryLead(reason string) bool {
	for _, prefix := range ledgerUnverifiedCategoryPrefixes {
		if len(reason) >= len(prefix) && reason[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// TestLedgerReasonsAreTimeless is the mechanical replacement for a manual
// narrative sweep: every reason/testFunction string in the real ledger must
// be free of internal-process references, and every unverified[] reason
// must lead with a fixed category. Skips (not fails) when the ledger is
// absent, matching every other ledger-governance test's public-CI
// portability convention -- tests/verified-endpoints.json is deliberately
// excluded from the assembled public package.
func TestLedgerReasonsAreTimeless(t *testing.T) {
	data, err := os.ReadFile("verified-endpoints.json")
	if err != nil {
		t.Skipf("tests/verified-endpoints.json is unreadable (%v) -- deliberately excluded from the assembled public package", err)
	}
	var ledger ledgerNarrativeFile
	if err := json.Unmarshal(data, &ledger); err != nil {
		t.Fatalf("parse tests/verified-endpoints.json: %v", err)
	}
	if len(ledger.Verified) == 0 {
		t.Fatal("control check failed: verified[] is empty -- the ledger read itself is broken")
	}
	if len(ledger.Unverified) == 0 {
		t.Fatal("control check failed: unverified[] is empty -- the ledger read itself is broken")
	}

	var violations []string
	for i, e := range ledger.Verified {
		for _, field := range []struct {
			name, value string
		}{{"reason", e.Reason}, {"testFunction", e.TestFunction}} {
			if hits := ledgerNarrativeViolations(field.value); len(hits) > 0 {
				violations = append(violations, fmt.Sprintf("verified[%d].%s: %v", i, field.name, hits))
			}
		}
	}
	for i, e := range ledger.Unverified {
		for _, field := range []struct {
			name, value string
		}{{"reason", e.Reason}, {"testFunction", e.TestFunction}} {
			if hits := ledgerNarrativeViolations(field.value); len(hits) > 0 {
				violations = append(violations, fmt.Sprintf("unverified[%d].%s: %v", i, field.name, hits))
			}
		}
		if !ledgerUnverifiedHasCategoryLead(e.Reason) {
			violations = append(violations, fmt.Sprintf("unverified[%d].reason: does not lead with a category prefix %v", i, ledgerUnverifiedCategoryPrefixes))
		}
	}

	if len(violations) > 0 {
		t.Fatalf("%d ledger narrative/category violation(s):\n%s", len(violations), joinLines(violations))
	}
}

func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

// TestLedgerNarrativeViolations_PlantedControlsFire proves the guard
// actually catches each pattern class it claims to, by planting a
// synthetic violation of that exact shape (never against the real ledger
// file) and asserting detection fires.
func TestLedgerNarrativeViolations_PlantedControlsFire(t *testing.T) {
	cases := []struct {
		name   string
		reason string
	}{
		{"this_session", "Genuine live capture this session, real UUIDs."},
		{"same_session", "Corrected same session after review."},
		{"capture_session", "Captured during this capture session."},
		{"E5_token", "Confirmed as part of E5 milestone work."},
		{"run_number", "Fixed during run-4 of the gate."},
		{"phase_number", "Recaptured in Phase 3 of the vendored re-capture."},
		{"report_item", "See adjacent-followup report item #5 for detail."},
		{"task_number", "Reattributed from the fixture field, Task 3."},
		{"follow_up_task", "A follow-up re-capture is a follow-up task, not done here."},
		{"bead_candidate", "Confirmed SDK defect (candidate for a new bead)."},
		{"bead_bare", "Tracked as a bead for later triage."},
		{"msg_id", "See " + "msg_" + "01KYZ4XFFSPRP79MM2QTE8RVNT" + " for the full finding."},
		{"implementer_role", "Independently re-verified (implementer, not the capture agent)."},
		{"maintainer", "Escalated to the maintainer for a ruling."},
		{"orchestrator_role", "Dispatched by the orchestrator for live capture."},
		{"dated_task_prefix", "2026-08-06: relocated to its synthetic-placeholder location."},
		{"feature_branch", "Covered by feature/groups-codegen-262, landing later."},
		{"after_the_tag", "This lands after the tag, once the branch merges."},
		{"other_chore_branch", "See chore/some-other-topic-branch for the fix."},
		{"ticket_id", "Matches the finding tracked as " + "CTRLP" + "-13339" + " in the internal tracker."},
		{"first_person", "Confirmed by hashing the file myself before upload."},
		{"our_own", "No cleanup needed since this is our own designated test OG."},
		{"question_number", "Behavior preserved per Q7-ii of the design review."},
		{"stale_entry", "UPDATE (2026-08-01, STALE ENTRY CORRECTED): the claim below no longer holds."},
		{"entry_previously", "This entry previously claimed the route was unsupported."},
		{"dated_update_label", "UPDATE (2026-08-01): the blocked create was subsequently unblocked."},
		{"named_individual_staged", "The second asset was Leon-staged, requiring no extra conversion."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if hits := ledgerNarrativeViolations(tc.reason); len(hits) == 0 {
				t.Fatalf("planted violation %q was not detected", tc.reason)
			}
		})
	}
}

// TestLedgerNarrativeViolations_CleanTextPasses proves the guard does NOT
// false-positive on legitimate technical language that happens to contain a
// trigger word as a substring, or on the one allowed provenance citation
// shape.
func TestLedgerNarrativeViolations_CleanTextPasses(t *testing.T) {
	cases := []struct {
		name   string
		reason string
	}{
		{"session_token", "The response includes a session token field, refreshed on each call."},
		{"provenance_date", "Live-captured 2026-08-01 against host <internal-env>, UEM 26.2.0.0."},
		{"task_queue", "The endpoint processes requests from an internal task queue."},
		{"vendored_provenance", "Vendored provenance: d14f6ad9b on chore/vendored-swagger-uem-26.2.0.0."},
		{"quirk_number", "Same root cause as doctrine quirk 13 (response-vs-request classification)."},
		{"http_status", "Returns HTTP 400 with errorCode 5206, a genuine conflict response."},
		{"commit_sha", "Fixed the BaselinesV1 Content-Type codegen bug in commit 33cfbfbfa."},
		{"bare_ctrl_word", "The CTRL key modifier is unrelated to this field's encoding."},
		{"http_header_name", "The response sets a Content-Type header, not covered by this fixture."},
		{"og_itself", "No cleanup was needed because the OG itself was already disposable."},
		{"quirk_reference", "Same root cause as doctrine Quirk 7, a swagger-declared type mismatch."},
		{"update_endpoint_lowercase", "The generated method calls the update endpoint with a PUT verb."},
		{"question_mark_prose", "Is the field required? The swagger schema says no, but the wire disagrees."},
		{"queue_word", "Requests are processed from an internal task queue before this response is returned."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if hits := ledgerNarrativeViolations(tc.reason); len(hits) != 0 {
				t.Fatalf("clean text %q was false-flagged: %v", tc.reason, hits)
			}
		})
	}
}

// TestLedgerUnverifiedCategoryLead_PlantedControls proves the category-lead
// check fires on a reason missing a category prefix and passes on each real
// category prefix in use.
func TestLedgerUnverifiedCategoryLead_PlantedControls(t *testing.T) {
	if ledgerUnverifiedHasCategoryLead("Own description is honest about being a deliberate double.") {
		t.Fatal("planted uncategorized reason was not detected")
	}
	for _, prefix := range ledgerUnverifiedCategoryPrefixes {
		if !ledgerUnverifiedHasCategoryLead(prefix + ": the rest of the reason follows.") {
			t.Fatalf("real category prefix %q was not recognized", prefix)
		}
	}
}
