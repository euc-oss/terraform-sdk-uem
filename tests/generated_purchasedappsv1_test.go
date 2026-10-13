package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// TestGeneratedPurchasedAppsV1SearchEmpty exercises the generated
// PurchasedAppsV1Service.VppAppSearchAsync against
// testdata/mock-responses/mam-apps/apps_get_purchased_search.json.
//
// The fixture is a genuine 204 No Content capture (no VPP token on the
// capturing tenant), so this test asserts on the empty-result path rather
// than a populated search response -- that is the real, fixture-backed
// behavior of this operation on the captured tenant.
//
// The fixture apps_get_purchased_search.json is tagged provenance.class=PORTED-FROM-26.2,
// and the sibling apps_get_purchased_search_populated.json (a genuine 26.4
// capture of the SAME endpoint) is now present in the corpus. The matcher's
// genuine-over-ported preference (RequestMatcher.MatchPreferring) would make
// the populated fixture win a full-tree LoadMockResponses load, defeating
// the point of this empty-204 test. So this loads ONLY the 204 fixture on
// an isolated single-fixture mock server, via a literal LoadResponseFromFile
// path (so the bypass-governance scan sees it).
func TestGeneratedPurchasedAppsV1SearchEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/mam-apps/apps_get_purchased_search.json"))
	if err != nil {
		t.Fatalf("load apps_get_purchased_search fixture: %v", err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewPurchasedAppsV1Service(c)

	_, result, err := svc.VppAppSearchAsync(context.Background(), nil)
	if err != nil {
		t.Fatalf("VppAppSearchAsync failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected a non-nil (zero-value) response struct for 204 No Content, got nil")
	}
	if len(result.Application) != 0 {
		t.Fatalf("expected empty Application slice for 204 No Content fixture, got %+v", result.Application)
	}
}

// TestGeneratedPurchasedAppsV1SearchPopulated exercises the generated
// PurchasedAppsV1Service.VppAppSearchAsync against the new genuine 26.4
// capture mam-apps/apps_get_purchased_search_populated.json (a tenant with a
// configured VPP token, 12 purchased apps) -- the populated counterpart to
// TestGeneratedPurchasedAppsV1SearchEmpty's 204 case. Loaded on an isolated
// single-fixture mock server via a literal LoadResponseFromFile path (so the
// bypass-governance scan sees it), same pattern as the empty test above.
// Row count and the first app's name are read from the fixture's own body,
// not hardcoded.
func TestGeneratedPurchasedAppsV1SearchPopulated(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const fixture = "mam-apps/apps_get_purchased_search_populated.json"
	var want struct {
		Application []struct {
			ApplicationName string `json:"ApplicationName"`
		} `json:"Application"`
	}
	loadFixtureResponseBodyForTest(t, fixture, &want)
	if len(want.Application) == 0 {
		t.Fatalf("%s has no Application rows -- cannot derive expected values", fixture)
	}

	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/" + fixture))
	if err != nil {
		t.Fatalf("load %s: %v", fixture, err)
	}
	ms := mockserver.NewMockServer([]*mockserver.MockResponse{resp})
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewPurchasedAppsV1Service(c)

	_, result, err := svc.VppAppSearchAsync(context.Background(), nil)
	if err != nil {
		t.Fatalf("VppAppSearchAsync failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil response")
	}
	if len(result.Application) != len(want.Application) {
		t.Fatalf("Application: got %d rows, want the fixture's %d", len(result.Application), len(want.Application))
	}
	if result.Application[0].ApplicationName != want.Application[0].ApplicationName {
		t.Errorf("Application[0].ApplicationName: got %q, want the fixture's %q", result.Application[0].ApplicationName, want.Application[0].ApplicationName)
	}
}
