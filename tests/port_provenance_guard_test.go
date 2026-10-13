package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// A fixture carried over from the 26.2 corpus without re-verification on this
// release is marked in its metadata:
//
//	"provenance": {"class": "PORTED-FROM-26.2", "source_tenant": "...",
//	  "source_version": "...", "source_commit": "<40 hex>", "reverified_on": null}
//
// and its description starts with portedDescriptionPrefix. A later recapture
// REPLACES the fixture (it becomes genuine), so a port never carries a
// reverified_on date. A port is never evidence that an endpoint is verified on
// this release.

const (
	portedClass             = "PORTED-FROM-26.2"
	portedDescriptionPrefix = "PORTED-FROM-26.2 (not re-verified on 26.4): "
)

var fortyHex = regexp.MustCompile(`^[0-9a-f]{40}$`)

type fixtureProvenanceMeta struct {
	Description string           `json:"description"`
	Provenance  *json.RawMessage `json:"provenance"`
}

type fixtureProvenanceFields struct {
	Class         *string `json:"class"`
	SourceTenant  *string `json:"source_tenant"`
	SourceVersion *string `json:"source_version"`
	SourceCommit  *string `json:"source_commit"`
	ReverifiedOn  any     `json:"reverified_on"`
}

// portProvenanceProblems validates one fixture's metadata. It returns the
// problems found and whether the fixture is a port.
func portProvenanceProblems(raw []byte) ([]string, bool) {
	var doc struct {
		Metadata fixtureProvenanceMeta `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return []string{fmt.Sprintf("not JSON: %v", err)}, false
	}
	m := doc.Metadata
	hasPrefix := strings.HasPrefix(m.Description, portedDescriptionPrefix)
	if m.Provenance == nil {
		if hasPrefix {
			return []string{"description has the PORTED-FROM-26.2 prefix but metadata.provenance is missing"}, false
		}
		return nil, false
	}
	var p fixtureProvenanceFields
	if err := json.Unmarshal(*m.Provenance, &p); err != nil {
		return []string{fmt.Sprintf("metadata.provenance is not an object: %v", err)}, false
	}
	var bad []string
	if p.Class == nil || *p.Class != portedClass {
		bad = append(bad, fmt.Sprintf("provenance.class must be %q", portedClass))
	}
	if p.SourceTenant == nil || strings.TrimSpace(*p.SourceTenant) == "" {
		bad = append(bad, "provenance.source_tenant must be a non-empty string")
	}
	if p.SourceVersion == nil || strings.TrimSpace(*p.SourceVersion) == "" {
		bad = append(bad, "provenance.source_version must be a non-empty string")
	}
	if p.SourceCommit == nil || !fortyHex.MatchString(*p.SourceCommit) {
		bad = append(bad, "provenance.source_commit must be a full 40-hex commit SHA")
	}
	if p.ReverifiedOn != nil {
		bad = append(bad, "provenance.reverified_on must be null (a recapture replaces the fixture instead)")
	}
	if !hasPrefix {
		bad = append(bad, fmt.Sprintf("description must start with %q", portedDescriptionPrefix))
	}
	return bad, true
}

func TestPortProvenance_PlantedControls(t *testing.T) {
	const sha = "a4b67af4b43fa6e94365fcae2ae565376dfde61a"
	good := `{"metadata":{"description":"` + portedDescriptionPrefix + `x","provenance":{"class":"PORTED-FROM-26.2","source_tenant":"lab-tenant","source_version":"26.2.0.0","source_commit":"` + sha + `","reverified_on":null}}}`
	cases := []struct {
		name     string
		doc      string
		wantBad  bool
		wantPort bool
	}{
		{"valid port", good, false, true},
		{"genuine, no provenance", `{"metadata":{"description":"Captured GET"}}`, false, false},
		{"prefix without provenance", `{"metadata":{"description":"` + portedDescriptionPrefix + `x"}}`, true, false},
		{"wrong class", strings.Replace(good, `"class":"PORTED-FROM-26.2"`, `"class":"PORTED"`, 1), true, true},
		{"short sha", strings.Replace(good, sha, "a4b67af4b", 1), true, true},
		{"reverified date", strings.Replace(good, `"reverified_on":null`, `"reverified_on":"2026-09-24"`, 1), true, true},
		{"empty tenant", strings.Replace(good, `"source_tenant":"lab-tenant"`, `"source_tenant":""`, 1), true, true},
		{"missing tenant key", strings.Replace(good, `"source_tenant":"lab-tenant",`, ``, 1), true, true},
		{"missing version key", strings.Replace(good, `"source_version":"26.2.0.0",`, ``, 1), true, true},
		{"missing commit key", strings.Replace(good, `"source_commit":"`+sha+`",`, ``, 1), true, true},
		{"missing prefix", strings.Replace(good, portedDescriptionPrefix, "", 1), true, true},
	}
	for _, c := range cases {
		bad, port := portProvenanceProblems([]byte(c.doc))
		if (len(bad) > 0) != c.wantBad || port != c.wantPort {
			t.Errorf("%s: problems=%v port=%v, want problems=%v port=%v", c.name, bad, port, c.wantBad, c.wantPort)
		}
	}
}

// TestPortProvenance_VendoredCorpus validates every fixture's provenance.
func TestPortProvenance_VendoredCorpus(t *testing.T) {
	ports := 0
	for _, e := range walkVendoredFixtureCorpus(t) {
		raw, err := os.ReadFile(e.fullPath)
		if err != nil {
			t.Fatal(err)
		}
		bad, port := portProvenanceProblems(raw)
		if port {
			ports++
		}
		for _, b := range bad {
			t.Errorf("%s: %s", e.relPath, b)
		}
	}
	t.Logf("%d PORTED-FROM-26.2 fixtures", ports)
}

// ledgerFixturePaths splits a ledger row's fixture field ("a.json, b.json
// (note)") into corpus-relative paths.
func ledgerFixturePaths(field string) []string {
	var out []string
	for _, part := range strings.Split(field, ",") {
		tok := strings.Fields(strings.TrimSpace(part))
		if len(tok) > 0 && strings.HasSuffix(tok[0], ".json") {
			out = append(out, tok[0])
		}
	}
	return out
}

// portedFixtureSet returns the corpus-relative paths of every ported fixture.
func portedFixtureSet(t *testing.T) map[string]bool {
	t.Helper()
	ported := map[string]bool{}
	for _, e := range walkVendoredFixtureCorpus(t) {
		raw, err := os.ReadFile(e.fullPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, port := portProvenanceProblems(raw); port {
			ported[e.relPath] = true
		}
	}
	return ported
}

// TestPortProvenance_PortBackedRowsAreLabelled: a verified[] row whose
// fixtures are all ports must say so, with its reason starting
// "PORTED-FROM-26.2: "; a row with a genuine fixture must not carry the label.
func TestPortProvenance_PortBackedRowsAreLabelled(t *testing.T) {
	ported := portedFixtureSet(t)
	data, err := os.ReadFile("verified-endpoints.json")
	if err != nil {
		t.Skipf("tests/verified-endpoints.json is unreadable (%v) -- deliberately excluded from the assembled public package", err)
	}
	var ledger struct {
		Verified []struct {
			Service string `json:"service"`
			Method  string `json:"method"`
			Fixture string `json:"fixture"`
			Reason  string `json:"reason"`
		} `json:"verified"`
	}
	if err := json.Unmarshal(data, &ledger); err != nil {
		t.Fatal(err)
	}
	labeled := 0
	for _, row := range ledger.Verified {
		paths := ledgerFixturePaths(row.Fixture)
		if len(paths) == 0 {
			continue
		}
		allPorted := true
		for _, p := range paths {
			if !ported[p] {
				allPorted = false
			}
		}
		hasLabel := strings.HasPrefix(row.Reason, portedClass+": ")
		switch {
		case allPorted && !hasLabel:
			t.Errorf("verified[] %s.%s is backed only by ports (%s); its reason must start %q", row.Service, row.Method, strings.Join(paths, ", "), portedClass+": ")
		case !allPorted && hasLabel:
			t.Errorf("verified[] %s.%s has a genuine fixture but its reason carries the %q label", row.Service, row.Method, portedClass+": ")
		}
		if hasLabel {
			labeled++
		}
	}
	t.Logf("%d of %d verified[] rows are port-backed pending recapture", labeled, len(ledger.Verified))
}

// portBackedVerifiedRows lists verified[] rows whose fixtures are all ports and still await a
// genuine recapture. It is empty: a row backed only by ports belongs in unverified[]. A new
// port-backed verified row fails the test below, and so does a listed row that gained a
// genuine fixture.
var portBackedVerifiedRows = map[string]bool{}

// TestPortProvenance_VerifiedRowsNeedAGenuineFixture blocks a verified[] row
// whose fixtures are all ports (a port is not evidence for this release)
// unless the row is in portBackedVerifiedRows.
func TestPortProvenance_VerifiedRowsNeedAGenuineFixture(t *testing.T) {
	entries := walkVendoredFixtureCorpus(t)
	ported := map[string]bool{}
	for _, e := range entries {
		raw, err := os.ReadFile(e.fullPath)
		if err != nil {
			t.Fatal(err)
		}
		if _, port := portProvenanceProblems(raw); port {
			ported[e.relPath] = true
		}
	}
	data, err := os.ReadFile("verified-endpoints.json")
	if err != nil {
		t.Skipf("tests/verified-endpoints.json is unreadable (%v) -- deliberately excluded from the assembled public package", err)
	}
	var ledger struct {
		Verified []struct {
			Service string `json:"service"`
			Method  string `json:"method"`
			Fixture string `json:"fixture"`
		} `json:"verified"`
	}
	if err := json.Unmarshal(data, &ledger); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, row := range ledger.Verified {
		paths := ledgerFixturePaths(row.Fixture)
		if len(paths) == 0 {
			continue
		}
		allPorted := true
		for _, p := range paths {
			if !ported[p] {
				allPorted = false
			}
		}
		key := row.Service + "." + row.Method
		if allPorted {
			seen[key] = true
			if !portBackedVerifiedRows[key] {
				t.Errorf("verified[] %s: every fixture is PORTED-FROM-26.2 (%s); move the row to unverified[] with a \"PORTED-FROM-26.2: \" reason until it is recaptured on this release", key, strings.Join(paths, ", "))
			}
		}
	}
	for key := range portBackedVerifiedRows {
		if !seen[key] {
			t.Errorf("portBackedVerifiedRows lists %s, which is no longer a verified[] row backed only by ports; remove it from the list", key)
		}
	}
}

func TestLedgerFixturePaths(t *testing.T) {
	got := ledgerFixturePaths("a/x.json, b/y.json (documentation-only -- see reason), c/z.json")
	if strings.Join(got, "|") != "a/x.json|b/y.json|c/z.json" {
		t.Errorf("ledgerFixturePaths = %v", got)
	}
}
