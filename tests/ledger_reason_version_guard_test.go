package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// A verified[] row's reason tells where its fixture came from. When the
// fixture is a capture from one UEM line, a reason that cites another line
// as the capture evidence is wrong: it keeps describing the previous
// corpus after the fixture was replaced. This guard compares the UEM
// versions a reason names with the versions the fixture's own metadata
// names (description, provenance.source_version), and with the corpus line
// when the metadata names none and the fixture is not a port.
//
// A mention of a different line is accepted only inside a sentence that
// marks it as history (an earlier or replaced capture, a port, ...).

// corpusUEMLine is the UEM line of the vendored corpus this layer is for.
const corpusUEMLine = "26.4"

var uemLineMention = regexp.MustCompile(`(?:^|[^0-9.])(26\.[0-9])(?:\.[0-9]+){0,2}\b`)

var historyMarker = regexp.MustCompile(`(?i)\b(earlier|prior|previous|previously|replac\w*|superseded|ported|formerly)\b|PORTED-FROM`)

// fixtureUEMLines returns the UEM lines (such as "26.4") a fixture's
// metadata names, or the corpus line for a genuine fixture whose metadata
// names none.
func fixtureUEMLines(raw []byte) (map[string]bool, error) {
	var doc struct {
		Metadata struct {
			Description string `json:"description"`
			Provenance  *struct {
				Class         string `json:"class"`
				SourceVersion string `json:"source_version"`
			} `json:"provenance"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	lines := map[string]bool{}
	text := doc.Metadata.Description
	if p := doc.Metadata.Provenance; p != nil {
		text += " " + p.SourceVersion
	}
	for _, m := range uemLineMention.FindAllStringSubmatch(text, -1) {
		lines[m[1]] = true
	}
	// A description's "PORTED-FROM-26.2 (not re-verified on 26.4)" names both
	// lines; for a port the source line is the evidence.
	if p := doc.Metadata.Provenance; p != nil && p.Class == portedClass {
		lines = map[string]bool{}
		for _, m := range uemLineMention.FindAllStringSubmatch(p.SourceVersion, -1) {
			lines[m[1]] = true
		}
	}
	if len(lines) == 0 {
		lines[corpusUEMLine] = true
	}
	return lines, nil
}

// ledgerReasonVersionProblems reports the UEM lines a reason cites outside
// history sentences that no fixture of the row carries.
func ledgerReasonVersionProblems(reason string, fixtureLines map[string]bool) []string {
	var bad []string
	seen := map[string]bool{}
	for _, sentence := range strings.Split(reason, ". ") {
		for _, m := range uemLineMention.FindAllStringSubmatch(sentence, -1) {
			line := m[1]
			if fixtureLines[line] || seen[line] || historyMarker.MatchString(sentence) {
				continue
			}
			seen[line] = true
			bad = append(bad, line)
		}
	}
	sort.Strings(bad)
	return bad
}

func TestLedgerReasonVersionsMatchFixtureVersions_PlantedControls(t *testing.T) {
	v := func(lines ...string) map[string]bool {
		m := map[string]bool{}
		for _, l := range lines {
			m[l] = true
		}
		return m
	}
	for _, c := range []struct {
		name, reason string
		lines        map[string]bool
		want         []string
	}{
		{"matching line", "GENUINE: captured on the 26.4 lab tenant, UEM 26.4.0.0.", v("26.4"), nil},
		{"stale line as evidence", "Live-recaptured against UEM 26.2.0.0: GET /x.", v("26.4"), []string{"26.2"}},
		{"stale line in a clause after a good sentence", "GENUINE 26.4 capture. Captured against the 26.2 test tenant.", v("26.4"), []string{"26.2"}},
		{"history sentence is accepted", "GENUINE 26.4 recapture, replacing the 26.2 capture. Body anonymized.", v("26.4"), nil},
		{"earlier capture is accepted", "GENUINE 26.4 capture; an earlier 26.2 entry was live-captured.", v("26.4"), nil},
		{"port cites its source line", "PORTED-FROM-26.2: carried over from 26.2.0.0 without re-verification.", v("26.2"), nil},
		{"port claiming the new line", "Captured against UEM 26.4.0.0.", v("26.2"), []string{"26.4"}},
		{"a plain past-tense verb is not history", "Live-captured against UEM 26.2.0.0; the body was anonymized.", v("26.4"), []string{"26.2"}},
		{"had is not history", "Captured on the 26.2 tenant, which had one enrolled device.", v("26.4"), []string{"26.2"}},
		{"original is not history", "Captured on UEM 26.2.0.0 from the original org group.", v("26.4"), []string{"26.2"}},
		{"old is not history", "Captured on UEM 26.2.0.0, old-style body.", v("26.4"), []string{"26.2"}},
		{"no version mention", "GENUINE: captured with the capture tool.", v("26.4"), nil},
		{"build number of the fixture line", "Captured on build 26.4.1795.16.", v("26.4"), nil},
		{"build number of another line", "Captured on build 26.2.1614.12.", v("26.4"), []string{"26.2"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := ledgerReasonVersionProblems(c.reason, c.lines)
			if fmt.Sprint(got) != fmt.Sprint(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestFixtureUEMLines_PlantedControls(t *testing.T) {
	for _, c := range []struct {
		name, doc string
		want      string
	}{
		{"description names the line", `{"metadata":{"description":"Captured GET /x on the 26.4 lab tenant (UEM 26.4), 2026-09-24"}}`, "[26.4]"},
		{"generic description falls back to the corpus line", `{"metadata":{"description":"Captured GET /x"}}`, "[26.4]"},
		{"port uses the source version", `{"metadata":{"description":"PORTED-FROM-26.2 (not re-verified on 26.4): x","provenance":{"class":"PORTED-FROM-26.2","source_version":"26.2.0.0"}}}`, "[26.2]"},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, err := fixtureUEMLines([]byte(c.doc))
			if err != nil {
				t.Fatal(err)
			}
			var ls []string
			for l := range got {
				ls = append(ls, l)
			}
			sort.Strings(ls)
			if fmt.Sprint(ls) != c.want {
				t.Fatalf("got %v, want %s", ls, c.want)
			}
		})
	}
}

// TestLedgerReasonVersionsMatchFixtureVersions: no verified[] reason cites a
// UEM line as evidence that its fixtures' own metadata does not carry.
func TestLedgerReasonVersionsMatchFixtureVersions(t *testing.T) {
	entries := walkVendoredFixtureCorpus(t)
	full := map[string]string{}
	for _, e := range entries {
		full[e.relPath] = e.fullPath
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
			Reason  string `json:"reason"`
		} `json:"verified"`
	}
	if err := json.Unmarshal(data, &ledger); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, row := range ledger.Verified {
		lines := map[string]bool{}
		for _, p := range ledgerFixturePaths(row.Fixture) {
			fp, ok := full[p]
			if !ok {
				continue
			}
			raw, err := os.ReadFile(fp)
			if err != nil {
				t.Fatal(err)
			}
			ls, err := fixtureUEMLines(raw)
			if err != nil {
				t.Errorf("%s: %v", p, err)
				continue
			}
			for l := range ls {
				lines[l] = true
			}
		}
		if len(lines) == 0 {
			continue
		}
		checked++
		for _, bad := range ledgerReasonVersionProblems(row.Reason, lines) {
			t.Errorf("verified[] %s.%s: the reason cites UEM %s as evidence but its fixture(s) %q carry %v", row.Service, row.Method, bad, row.Fixture, keysOf(lines))
		}
	}
	t.Logf("checked %d verified[] rows", checked)
}

func keysOf(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
