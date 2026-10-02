package tests

import (
	"context"
	"errors"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/client"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestNotFoundFixturesAreRecognised replays each live-captured not-found
// response (GET after delete, and a repeat DELETE) through the shared mock
// server and the real client error path, and asserts client.IsNotFound
// recognizes it. The fixtures are genuine captures; see each fixture's
// metadata.description. Each request sends the Accept version its fixture
// records, which the mock server validates.
func TestNotFoundFixturesAreRecognised(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	cases := []struct {
		name, method, path, version string
		// setup runs first on the same server, for fixtures served by
		// observed state (the smart group GET needs a prior 2xx DELETE).
		setup      func(t *testing.T, c *client.Client)
		wantStatus int
	}{
		{name: "profile GET", method: "GET", path: "/api/mdm/profiles/68990", version: "2", wantStatus: 400},
		{name: "profile DELETE", method: "DELETE", path: "/api/mdm/profiles/68990", version: "2", wantStatus: 400},
		{
			name: "smart group GET", method: "GET", path: "/api/mdm/smartgroups/12345", version: "1", wantStatus: 400,
			setup: func(t *testing.T, c *client.Client) {
				if _, err := c.DoRequest(context.Background(), "DELETE", "/api/mdm/smartgroups/12345", "application/json;version=1", "application/json", nil, nil); err != nil {
					t.Fatalf("setup DELETE /api/mdm/smartgroups/12345: %v", err)
				}
			},
		},
		{name: "smart group DELETE", method: "DELETE", path: "/api/mdm/smartgroups/46813", version: "1", wantStatus: 400},
		{name: "macOS internal app GET", method: "GET", path: "/api/mam/apps/internal/67090", version: "1", wantStatus: 401},
		{name: "macOS internal app DELETE", method: "DELETE", path: "/api/mam/apps/internal/67090", version: "1", wantStatus: 400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
			t.Cleanup(ms.Close)
			c := mockserver.NewMockClient(t, ms)
			if tc.setup != nil {
				tc.setup(t, c)
			}
			_, err := c.DoRequest(context.Background(), tc.method, tc.path, "application/json;version="+tc.version, "application/json", nil, nil)
			var apiErr *client.APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != tc.wantStatus {
				t.Fatalf("%s %s: got %T %v, want *client.APIError with status %d", tc.method, tc.path, err, err, tc.wantStatus)
			}
			if !client.IsNotFound(err) {
				t.Errorf("%s %s: IsNotFound = false for the captured not-found response %+v", tc.method, tc.path, apiErr)
			}
		})
	}
}
