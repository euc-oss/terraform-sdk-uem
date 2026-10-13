package tests

import (
	"regexp"
	"strconv"
	"testing"
)

// A fixture's relative path ships in the public package (as
// testdata/mock-responses/<rel>), so a real tenant id embedded in a file
// name leaks the same way one in a body would. Any run of 4+ digits in the
// path, whether or not separators (/ _ . -) bound it ("profile40105.json"
// counts like "profile_40105.json"), must be either a documentation
// placeholder id or an allowlisted version token.
var fixturePathIDToken = regexp.MustCompile(`\d{4,}`)

// fixturePathAllowedTokens are digit runs that are not ids:
// "26400" is the 26.4.0.0 version marker in smartgroups_{get,post}_26400.json.
var fixturePathAllowedTokens = map[string]bool{"26400": true}

// fixturePathPlaceholderIDs is the documentation placeholder id range the
// anonymizer uses (12345..12352).
func fixturePathPlaceholderID(n int) bool { return n >= 12345 && n <= 12352 }

// fixturePathIDViolations returns the digit tokens in rel that are neither
// placeholder ids nor allowlisted tokens.
func fixturePathIDViolations(rel string) []string {
	var bad []string
	for _, tok := range fixturePathIDToken.FindAllString(rel, -1) {
		n, err := strconv.Atoi(tok)
		if !fixturePathAllowedTokens[tok] && (err != nil || !fixturePathPlaceholderID(n)) {
			bad = append(bad, tok)
		}
	}
	return bad
}

func TestFixturePathIDGuard_PlantedControls(t *testing.T) {
	cases := map[string]bool{ // path -> want violation
		"profiles/get_apple_scep_40105.json":                         true,
		"profiles/get_profile_12345.json":                            false,
		"smartgroups/smartgroups_get_26400.json":                     false,
		"profiles/get_apple_12348.json":                              false,
		"profiles/get_appleosx_credentials_23156_264sandbox_v2.json": true,
		"profiles/get_appleosx_credentials_12352_264sandbox_v2.json": false,
		"profiles/12353/get.json":                                    true,
		"profiles/get_12344.json":                                    true,
		"updates/updates_get_details.json":                           false,
		"profiles/linux/create-v4.json":                              false,
		"groups_get_id_baselines_264sandbox_v1.json":                 false,
		"a/b_123.json":         false, // 3 digits: not an id token
		"a/b_12345_99999.json": true,  // second token is a violation
		// Digit runs not bounded by separators.
		"profiles/profile40105.json":        true,  // glued to a word
		"profiles/x40105y.json":             true,  // glued on both sides
		"profiles/get_apple40105_v2.json":   true,  // glued before, separator after
		"profiles/40105profile.json":        true,  // glued after
		"profiles/profile12345.json":        false, // glued, but the placeholder id
		"profiles/get_apple_1234567.json":   true,  // longer than a placeholder
		"profiles/a123456b.json":            true,
		"smartgroups/smartgroups26400.json": false, // glued version marker
		"profiles/v264sandbox.json":         false, // 3 digits
		"profiles/get_8021.json":            true,  // four digits is the shortest token
	}
	for path, want := range cases {
		got := len(fixturePathIDViolations(path)) > 0
		if got != want {
			t.Errorf("fixturePathIDViolations(%q) = %v, want violation=%v", path, fixturePathIDViolations(path), want)
		}
	}
}

// TestFixturePathIDGuard_VendoredCorpus walks the real vendored corpus and
// fails on any fixture path carrying a non-placeholder numeric id. Skips
// when MOCK_RESPONSES_DIR is unset (see walkVendoredFixtureCorpus).
func TestFixturePathIDGuard_VendoredCorpus(t *testing.T) {
	for _, e := range walkVendoredFixtureCorpus(t) {
		if bad := fixturePathIDViolations(e.relPath); len(bad) > 0 {
			t.Errorf("%s: numeric token(s) %v in the path look like a real id; rename the file to the fixture's own anonymized placeholder id (12345..12352) or a neutral name", e.relPath, bad)
		}
	}
}
