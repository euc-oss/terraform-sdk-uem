package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/client"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestOrgGroupDeleteV2Fixtures replays the live-captured v2 org group DELETE
// and its repeat through the shared mock server: the first returns 200 with
// no body, the repeat a 400 errorCode 400 "not found" (api-doctrine quirk 20).
// The repeat's message is not one client.IsNotFound recognizes; this pins
// that the SDK surfaces it as a plain *client.APIError.
func TestOrgGroupDeleteV2Fixtures(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)
	c := mockserver.NewMockClient(t, ms)
	ctx := context.Background()
	const accept = "application/json;version=2"

	if _, err := c.DoRequest(ctx, "DELETE", "/api/system/groups/037ac4dc-483c-a3cc-ac3e-4b729abb4178", accept, "application/json", nil, nil); err != nil {
		t.Fatalf("DELETE org group: want success from groups_delete_uuid_v2.json, got %v", err)
	}

	_, err := c.DoRequest(ctx, "DELETE", "/api/system/groups/6dbaeac2-e813-beca-6849-68c2e7302194", accept, "application/json", nil, nil)
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 400 || apiErr.ErrorCode != "400" {
		t.Fatalf("repeat DELETE org group: want *client.APIError 400/400 from groups_delete_uuid_v2_not_found.json, got %T %v", err, err)
	}
	var want struct {
		Message string `json:"message"`
	}
	loadFixtureResponseBodyForTest(t, "org-groups/groups_delete_uuid_v2_not_found.json", &want)
	if want.Message == "" || apiErr.Message != want.Message {
		t.Errorf("repeat DELETE message: got %q, want the fixture's %q", apiErr.Message, want.Message)
	}
}
