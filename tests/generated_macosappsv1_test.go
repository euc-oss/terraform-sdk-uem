package tests

import (
	"context"
	"strconv"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/client"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// TestGeneratedMacOsAppsV1Create exercises MacOsAppsV1Service.CreateMacOSApplication
// against the testdata/mock-responses/mam-apps/macos_create.json fixture.
//
// Canonical CreateMacOSApplication (MacOsAppsV1Controller, see
// docs/guides/apps/canonical-models/apps-response-v1.md:153-169) returns
// 201 + Location header only — .WithLocationHeader(...), never .WithModel(...).
// There is no response body to decode. The overlay in
// internal-source/mamv1.overlay.yaml previously fabricated an integer
// response schema for this operation (since fixed),
// which produced a 3-value (http.Header, int, error) signature that silently
// returned 0 for the new app ID on every real call (empty body, decode
// guard skips json.Unmarshal). The fix mirrors the ScriptsV1 Pattern B
// precedent (see generated_scriptsv1_test.go): headers-only signature,
// caller extracts the ID via client.ParseLocationID.
func TestGeneratedMacOsAppsV1Create(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewMacOsAppsV1Service(c)
	ctx := context.Background()

	createReq := &sdk.MacOsCreateApplicationRequestV1Model{
		Version: "1.0",
	}

	// POST-FIX signature: (http.Header, error) — no int return, since the
	// real response body is empty.
	headers, err := svc.CreateMacOSApplication(ctx, 10, createReq)
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	location := headers.Get("Location")
	if location == "" {
		t.Fatal("expected Location header on 201 create response, got empty")
	}

	idStr, err := client.ParseLocationID(location)
	if err != nil {
		t.Fatalf("parse Location header failed: %v", err)
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		t.Fatalf("expected numeric app ID from Location header, got %q: %v", idStr, err)
	}

	if id != 424242 {
		t.Fatalf("expected app ID 424242 from fixture Location header, got %d", id)
	}
}
