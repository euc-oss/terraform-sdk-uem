package tests

import (
	"context"
	"encoding/json"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	mdmv1 "github.com/euc-oss/terraform-sdk-uem/internal/mdm/v1"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// fixtureBodyLen reads a fixture's response body and returns the length of
// the JSON array found by walking one field name from the body root (an
// empty field means the body itself is the array, e.g. the bare-array
// baselines/osversions-shaped responses). Used to cross-check a decoded SDK
// response's slice length against the captured fixture's own actual
// content, so the assertion tracks whatever the fixture genuinely contains
// instead of a hardcoded count that goes stale the next time the fixture is
// recaptured (the previous version of this test hardcoded fabricated
// counts/values that broke the moment the fixtures were replaced with
// genuine live captures).
func fixtureBodyLen(t *testing.T, path, field string) int {
	t.Helper()
	body := readFixtureBody(t, path)
	var root interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		t.Fatalf("parsing fixture body %s: %v", path, err)
	}
	if field != "" {
		m, ok := root.(map[string]interface{})
		if !ok {
			t.Fatalf("fixture body %s: expected an object to look up field %q, got %T", path, field, root)
		}
		root = m[field]
	}
	arr, ok := root.([]interface{})
	if !ok {
		t.Fatalf("fixture body %s field %q: expected a JSON array, got %T", path, field, root)
	}
	return len(arr)
}

// Fixture collision resolved: baselines-adjacent/ is the sole canonical
// home for this endpoint family. The genuine osversions/platforms/templates
// captures were
// moved into baselines-adjacent/*-list.json and the accidental duplicates
// under testdata/mock-responses/baselines/ were deleted -- one fixture per
// endpoint, zero overlap.
const (
	fixtureCatalogsPolicyCatalog = "../testdata/mock-responses/baselines-adjacent/catalogs-getpolicycatalog.json"
	fixtureCatalogsAllPolicies   = "../testdata/mock-responses/baselines-adjacent/catalogs-getall-policies.json"
	fixtureOSVersionsList        = "../testdata/mock-responses/baselines-adjacent/osversions-list.json"
	fixtureOSVersionsByPlatform  = "../testdata/mock-responses/baselines-adjacent/osversions-byplatform.json"
	fixturePlatformsList         = "../testdata/mock-responses/baselines-adjacent/platforms-list.json"
	fixtureTemplatesList         = "../testdata/mock-responses/baselines-adjacent/templates-list.json"
)

// TestGeneratedBaselinesV1Adjacent exercises the 9 adjacent-controller
// endpoints (Catalogs, OSVersions, Platforms, Templates) against genuinely
// captured fixtures under testdata/mock-responses/baselines-adjacent/. These
// four controllers don't carry [InternalApi] in canonical and are
// unambiguously public per D7.
func TestGeneratedBaselinesV1Adjacent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	ctx := context.Background()

	// Any well-formed value works for the path params below -- the mock
	// server matches fixtures by endpoint TEMPLATE, not by the literal path
	// segment substituted into it, so these don't need to correspond to any
	// real captured UUID.
	const (
		osVersionUUID      = "0e55c4fa-3b5c-3850-00b8-32638670c77f"
		platformUUID       = "fea1c595-91f0-e37b-aef6-6769865efad2"
		policyUUID         = "8a8b2cc1-fcc3-f355-585b-1c17bbd08b8d"
		vendorTemplateUUID = "e7536274-79a2-0fe9-2833-1be1fcb0c7fa"
		baselineTemplateID = "9e8ac479-600d-db66-c209-56395e04a587"
	)

	// CATALOGS
	catalogs := sdk.NewCatalogsV1Service(c)

	_, policyCatalog, err := catalogs.GetPolicyCatalogAsync(ctx, osVersionUUID, &mdmv1.CatalogsV1GetPolicyCatalogAsyncOptions{})
	if err != nil {
		t.Fatalf("CatalogsV1.GetPolicyCatalogAsync failed: %v", err)
	}
	if policyCatalog == nil {
		t.Fatal("expected non-nil PolicyCatalog response")
	}
	if policyCatalog.OsVersion == nil || policyCatalog.OsVersion.Name == "" {
		t.Errorf("expected a non-nil OsVersion with a non-empty Name, got %+v", policyCatalog.OsVersion)
	}
	if want := fixtureBodyLen(t, fixtureCatalogsPolicyCatalog, "policyTree"); len(policyCatalog.PolicyTree) != want {
		t.Errorf("expected %d top-level policyTree categories (matching the captured fixture), got %d", want, len(policyCatalog.PolicyTree))
	}

	_, allPolicies, err := catalogs.GetAllPoliciesAsync(ctx, osVersionUUID, &mdmv1.CatalogsV1GetAllPoliciesAsyncOptions{})
	if err != nil {
		t.Fatalf("CatalogsV1.GetAllPoliciesAsync failed: %v", err)
	}
	if allPolicies == nil {
		t.Fatal("expected non-nil GetAllPolicies response")
	}
	if want := fixtureBodyLen(t, fixtureCatalogsAllPolicies, "results"); len(allPolicies.Results) != want {
		t.Fatalf("expected %d policy results (matching the captured fixture), got %d", want, len(allPolicies.Results))
	}
	for i, p := range allPolicies.Results {
		if p.Name == "" || p.UUID == "" {
			t.Errorf("results[%d]: expected non-empty Name and UUID, got %+v", i, p)
		}
	}

	_, policy, err := catalogs.GetPolicyAsync(ctx, policyUUID, &mdmv1.CatalogsV1GetPolicyAsyncOptions{})
	if err != nil {
		t.Fatalf("CatalogsV1.GetPolicyAsync failed: %v", err)
	}
	if policy == nil || policy.Name == "" || policy.UUID == "" || policy.Class == "" || policy.Path == "" {
		t.Errorf("expected non-empty Name/UUID/Class/Path, got %+v", policy)
	}

	// OS VERSIONS
	osv := sdk.NewOSVersionsV1Service(c)

	_, allOSVersions, err := osv.GetActiveOSVersionsAsync(ctx)
	if err != nil {
		t.Fatalf("OSVersionsV1.GetActiveOSVersionsAsync failed: %v", err)
	}
	if allOSVersions == nil {
		t.Fatal("expected non-nil OS versions response")
	}
	if want := fixtureBodyLen(t, fixtureOSVersionsList, ""); len(*allOSVersions) != want {
		t.Fatalf("expected %d OS versions (matching the captured fixture), got %d", want, len(*allOSVersions))
	}
	for i, v := range *allOSVersions {
		if v.Name == "" || v.UUID == "" {
			t.Errorf("osVersions[%d]: expected non-empty Name and UUID, got %+v", i, v)
		}
	}

	_, platformOSVersions, err := osv.GetActiveOSVersionsForPlatformAsync(ctx, platformUUID)
	if err != nil {
		t.Fatalf("OSVersionsV1.GetActiveOSVersionsForPlatformAsync failed: %v", err)
	}
	if platformOSVersions == nil {
		t.Fatal("expected non-nil platform OS versions response")
	}
	if want := fixtureBodyLen(t, fixtureOSVersionsByPlatform, ""); len(*platformOSVersions) != want {
		t.Fatalf("expected %d platform OS versions (matching the captured fixture), got %d", want, len(*platformOSVersions))
	}
	for i, v := range *platformOSVersions {
		if v.Name == "" || v.UUID == "" {
			t.Errorf("platformOSVersions[%d]: expected non-empty Name and UUID, got %+v", i, v)
		}
	}

	// PLATFORMS
	plat := sdk.NewPlatformsV1Service(c)

	_, platforms, err := plat.GetActivePlatformsAsync(ctx)
	if err != nil {
		t.Fatalf("PlatformsV1.GetActivePlatformsAsync failed: %v", err)
	}
	if platforms == nil {
		t.Fatal("expected non-nil platforms response")
	}
	if want := fixtureBodyLen(t, fixturePlatformsList, ""); len(*platforms) != want {
		t.Fatalf("expected %d platforms (matching the captured fixture), got %d", want, len(*platforms))
	}
	for i, p := range *platforms {
		if p.Name == "" || p.UUID == "" {
			t.Errorf("platforms[%d]: expected non-empty Name and UUID, got %+v", i, p)
		}
	}

	// TEMPLATES
	tpl := sdk.NewTemplatesV1Service(c)

	_, allTemplates, err := tpl.GetAllAsync(ctx)
	if err != nil {
		t.Fatalf("TemplatesV1.GetAllAsync failed: %v", err)
	}
	if allTemplates == nil {
		t.Fatal("expected non-nil templates response")
	}
	if want := fixtureBodyLen(t, fixtureTemplatesList, ""); len(*allTemplates) != want {
		t.Fatalf("expected %d templates (matching the captured fixture), got %d", want, len(*allTemplates))
	}
	for i, tm := range *allTemplates {
		if tm.Name == "" || tm.UUID == "" {
			t.Errorf("templates[%d]: expected non-empty Name and UUID, got %+v", i, tm)
		}
	}

	// PolicyTree must be explicitly true: that's how the fixture was
	// captured (needed to get a non-empty policyTree in the response body),
	// and the mock server's matcher treats every key recorded in a fixture's
	// query_params as required on the request (internal/mockserver/matcher.go
	// matchQueryParams) -- omitting it here would 404 against this fixture.
	wantPolicyTree := true
	searchOpts := &mdmv1.TemplatesV1SearchTemplateAsyncOptions{OsVersionUUID: osVersionUUID, PolicyTree: &wantPolicyTree}
	_, searchedTemplate, err := tpl.SearchTemplateAsync(ctx, vendorTemplateUUID, searchOpts)
	if err != nil {
		t.Fatalf("TemplatesV1.SearchTemplateAsync failed: %v", err)
	}
	if searchedTemplate == nil || searchedTemplate.UUID == "" || searchedTemplate.VendorTemplate == nil {
		t.Fatalf("expected a non-nil searched template with a UUID and VendorTemplate, got %+v", searchedTemplate)
	}

	_, templatePolicy, err := tpl.GetPolicyAsync(ctx, baselineTemplateID, policyUUID, &mdmv1.TemplatesV1GetPolicyAsyncOptions{})
	if err != nil {
		t.Fatalf("TemplatesV1.GetPolicyAsync failed: %v", err)
	}
	if templatePolicy == nil || templatePolicy.Name == "" || templatePolicy.UUID == "" || templatePolicy.Class == "" {
		t.Errorf("expected non-empty Name/UUID/Class, got %+v", templatePolicy)
	}
}
