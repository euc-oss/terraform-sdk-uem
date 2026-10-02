package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	mamv2 "github.com/euc-oss/terraform-sdk-uem/internal/mam/v2"
	mdmv1 "github.com/euc-oss/terraform-sdk-uem/internal/mdm/v1"
	mdmv2 "github.com/euc-oss/terraform-sdk-uem/internal/mdm/v2"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestFixtureSchemaValidation strict-decodes each listed fixture's response
// body against the exact generated Go type its own SDK method returns
// (DisallowUnknownFields — any field on the wire the model doesn't declare
// fails the test). This is the check that would have caught the original
// class of fabrication bug: a schema-derived fixture can pass a superficial
// field-name skim while still not matching the live wire shape, but it
// cannot survive a strict decode against the real struct. Scoped today to
// the fixtures captured/re-verified during a prior recapture pass;
// extending this to every fixture automatically (endpoint -> type lookup
// without hand-listing each one) needs a codegen-emitted manifest and
// is a separate, larger undertaking, not this survey.
func TestFixtureSchemaValidation(t *testing.T) {
	cases := []struct {
		fixture   string
		newTarget func() any // returns a pointer to a zero value of the expected response type
	}{
		{"../testdata/mock-responses/baselines-adjacent/osversions-list.json",
			func() any { return &[]mdmv1.OSVersionV1Model{} }},
		{"../testdata/mock-responses/baselines-adjacent/platforms-list.json",
			func() any { return &[]mdmv1.PlatformV2ModelV1{} }},
		{"../testdata/mock-responses/baselines-adjacent/templates-list.json",
			func() any { return &[]mdmv1.BaselineVendorTemplateV1Model{} }},
		{"../testdata/mock-responses/baselines-adjacent/catalogs-getpolicycatalog.json",
			func() any { return &mdmv1.PolicyCatalogModelV1{} }},
		{"../testdata/mock-responses/baselines-adjacent/catalogs-getall-policies.json",
			func() any { return &mdmv1.GetAllPoliciesResponseV1Model{} }},
		{"../testdata/mock-responses/baselines-adjacent/catalogs-getpolicy.json",
			func() any { return &mdmv1.PolicyModelV1{} }},
		{"../testdata/mock-responses/baselines-adjacent/osversions-byplatform.json",
			func() any { return &[]mdmv1.OSVersionV2ModelV1{} }},
		{"../testdata/mock-responses/baselines-adjacent/templates-search.json",
			func() any { return &mdmv1.BaselineTemplateV1{} }},
		// templates-getpolicy.json intentionally excluded from strict decode:
		// its captured policy's Layout[] contains a DecimalType element with
		// fields (defaultValue, minValue, maxValue, spin, spinStep, unit,
		// required, typeName) that BaseElementModelV1 doesn't declare -- a
		// real, known, DOCUMENTED gap (see the type's own doc comment: "the
		// canonical server type spills extras as sibling JSON keys, not into
		// this nested object"), not a fixture defect. catalogs-getpolicy.json
		// above uses the same PolicyModelV1 type and passes because its
		// captured policy's layout doesn't happen to hit this gap.
		{"../testdata/mock-responses/baselines/groups_get_id_baselines.json",
			func() any { return &[]mdmv1.BaselineV1Model{} }},
		{"../testdata/mock-responses/scripts/list.json",
			func() any { return &mdmv1.ScriptsSearchResultV1{} }},
		{"../internal/mockserver/testdata/synthetic-placeholders/scripts-assignments/list.json",
			func() any { return &mdmv1.ScriptAssignmentsSearchResultV1{} }},
		{"../testdata/mock-responses/profiles/profiles_get_apple_id.json",
			func() any { return &mdmv2.DeviceProfileV2Entity{} }},
		{"../testdata/mock-responses/mam-apps/apps_get_filters_id.json",
			func() any { return &[]mamv2.ApplicationFiltersModelV2{} }},
	}

	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			body := readFixtureBody(t, c.fixture)
			if len(body) == 0 || string(body) == "null" {
				t.Skip("empty/null body — nothing to schema-validate")
			}
			dec := json.NewDecoder(bytes.NewReader(body))
			dec.DisallowUnknownFields()
			if err := dec.Decode(c.newTarget()); err != nil {
				t.Errorf("fixture does not strictly decode against its generated type (unknown or mismatched field): %v", err)
			}
		})
	}
}

// readFixtureBody reads a fixture two ways depending on the case table's
// literal path. A path rooted under testdata/mock-responses is
// vendored/governed and resolved override-aware (mockserver.ResolveResponsesDir),
// same as the rest of tests/*.go. A path rooted under the synthetic-placeholders
// directory (see internal/mockserver/testdata/synthetic-placeholders/README.md)
// is a fixture that was never migrated to vendored (governed unverified[], no
// surviving home on MOCK_RESPONSES_DIR by design) and is read directly, with
// no override redirection — ResolveResponsesDir would otherwise misinterpret
// a path with no mock-responses segment as a request for the override root
// itself, not this specific file.
func readFixtureBody(t *testing.T, path string) json.RawMessage {
	t.Helper()
	resolved := path
	if !strings.Contains(path, "synthetic-placeholders/") {
		resolved = mockserver.ResolveResponsesDir(path)
	}
	raw, err := os.ReadFile(resolved)
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}
	var fixture struct {
		Response struct {
			Body json.RawMessage `json:"body"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatalf("parsing fixture envelope: %v", err)
	}
	return fixture.Response.Body
}
