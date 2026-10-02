package tests

import (
	"encoding/json"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestGeneratedScriptResourceCatalogDisplay is a regression guard for
// internal-source/mdmv1.scripts.overlay.yaml's CatalogDisplay.name +
// CatalogDisplay.description gap-fill. The upstream mdmv1.json copy of the
// CatalogDisplay schema (shared by ScriptResourceV1/CreateScriptV1/
// UpdateScriptV1.catalog_display and ScriptDeploymentV1.DisplayAttributes)
// only declares display_name/display_desc/catalog_icon_url/... — it has no
// name or description property at all. The live wire, however, returns
// catalog_display as {"name": ..., "description": ...} (confirmed against
// testdata/mock-responses/scripts/get.json's catalog_display shape --
// SYNTHETIC: this fixture's body is a hand-authored mock (script_uuid is a
// hand-typed sequential-digit placeholder, name/description are literal
// "Sample script 1"/"Test script for SDK lifecycle" mock strings), not a
// live capture; the catalog_display.name/description KEY SHAPE this test
// guards is real regardless, independently confirmed against the live wire
// when the overlay fix was authored). Before the overlay fix, neither wire
// key matched a declared property, so catalog_display silently decoded as
// an empty object. If the overlay entry regresses, Name/Description
// disappear from the generated CatalogDisplayV1 struct and this file fails
// to compile -- same "compile failure is the proof" pattern used by
// tests/generated_deviceprofilev2_response_fields_test.go.
func TestGeneratedScriptResourceCatalogDisplay(t *testing.T) {
	resp, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/scripts/get.json"))
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}

	raw, err := json.Marshal(resp.Response.Body)
	if err != nil {
		t.Fatalf("re-marshal fixture body: %v", err)
	}

	var script sdk.ScriptResourceV1
	if err := json.Unmarshal(raw, &script); err != nil {
		t.Fatalf("unmarshal fixture body into ScriptResourceV1: %v", err)
	}

	if script.CatalogDisplay == nil {
		t.Fatal("expected non-nil CatalogDisplay")
	}
	if script.CatalogDisplay.Name == "" {
		t.Fatal("expected CatalogDisplay.Name to be present -- it was silently dropped before the fix")
	}
	if script.CatalogDisplay.Name != "Sample Script" {
		t.Errorf("CatalogDisplay.Name: got %q, want %q", script.CatalogDisplay.Name, "Sample Script")
	}
	if script.CatalogDisplay.Description == "" {
		t.Fatal("expected CatalogDisplay.Description to be present -- it was silently dropped before the fix")
	}
	if script.CatalogDisplay.Description != "Test" {
		t.Errorf("CatalogDisplay.Description: got %q, want %q", script.CatalogDisplay.Description, "Test")
	}
}
