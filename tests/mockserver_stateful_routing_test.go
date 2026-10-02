package tests

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// sharedMockCall sends one request to the shared mock server and returns the
// status and the decoded JSON body (nil for an empty body). The headers override
// the default Accept/Content-Type.
func sharedMockCall(t *testing.T, ms *mockserver.MockServer, method, path, version string, headers map[string]string) (int, interface{}) {
	t.Helper()
	var body io.Reader
	if method == http.MethodPut {
		body = strings.NewReader(`{}`)
	}
	req, err := http.NewRequest(method, ms.URL()+path, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "application/json;version="+version)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	var decoded interface{}
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("%s %s: body is not JSON: %v", method, path, err)
		}
	}
	return resp.StatusCode, decoded
}

// fixtureBody loads a fixture and returns its status, its body round-tripped
// through JSON (comparable with sharedMockCall's result) and the request
// headers the mock server validates for it.
func fixtureBody(t *testing.T, rel string) (int, interface{}, map[string]string) {
	t.Helper()
	fx, err := mockserver.LoadResponseFromFile(filepath.Join(mockserver.ResolveResponsesDir("../testdata/mock-responses"), rel))
	if err != nil {
		t.Fatalf("load %s: %v", rel, err)
	}
	raw, err := json.Marshal(fx.Response.Body)
	if err != nil {
		t.Fatal(err)
	}
	var decoded interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	return fx.Response.StatusCode, decoded, fx.Request.Headers
}

func assertServes(t *testing.T, ms *mockserver.MockServer, method, path, version, fixture string) {
	t.Helper()
	wantStatus, wantBody, headers := fixtureBody(t, fixture)
	gotStatus, gotBody := sharedMockCall(t, ms, method, path, version, headers)
	if gotStatus != wantStatus || !reflect.DeepEqual(gotBody, wantBody) {
		t.Fatalf("%s %s: got status %d body %v, want %s (status %d)", method, path, gotStatus, gotBody, fixture, wantStatus)
	}
}

// TestSharedMockServer_ProfileDeleteDefersToConcreteFixture proves a profile
// DELETE captured for an exact id is served from its fixture instead of the
// stateful 204, and that a non-2xx fixture does not mark the profile deleted:
// the GET that follows still serves the captured not-found fixture.
func TestSharedMockServer_ProfileDeleteDefersToConcreteFixture(t *testing.T) {
	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")

	assertServes(t, ms, http.MethodDelete, "/api/mdm/profiles/68990", "1", "profiles/profiles_delete_not_found.json")
	assertServes(t, ms, http.MethodGet, "/api/mdm/profiles/68990", "2", "profiles/profiles_get_not_found.json")

	// An id with no concrete fixture keeps the stateful 204.
	if status, _ := sharedMockCall(t, ms, http.MethodDelete, "/api/mdm/profiles/12345", "1", nil); status != http.StatusNoContent {
		t.Fatalf("DELETE of an id without a concrete fixture: status %d, want 204", status)
	}
}

// TestSharedMockServer_SmartGroupGetFollowsObservedLifecycle proves GET
// /api/mdm/smartgroups/{id}, which three fixtures share, serves the
// lifecycle stage matching the mutations the server has observed for that id.
func TestSharedMockServer_SmartGroupGetFollowsObservedLifecycle(t *testing.T) {
	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	const path = "/api/mdm/smartgroups/12346"

	assertServes(t, ms, http.MethodGet, path, "1", "smartgroups/smartgroups_get_after_create.json")
	assertServes(t, ms, http.MethodPut, path, "1", "smartgroups/smartgroups_update.json")
	assertServes(t, ms, http.MethodGet, path, "1", "smartgroups/smartgroups_get_after_update.json")
	assertServes(t, ms, http.MethodDelete, path, "1", "smartgroups/smartgroups_delete.json")
	assertServes(t, ms, http.MethodGet, path, "1", "smartgroups/smartgroups_get_after_delete.json")

	// A captured not-found DELETE (non-2xx) is served from its concrete
	// fixture and leaves no state behind, so the id's GET is unchanged.
	assertServes(t, ms, http.MethodDelete, "/api/mdm/smartgroups/46813", "1", "smartgroups/smartgroups_delete_not_found.json")
	assertServes(t, ms, http.MethodGet, "/api/mdm/smartgroups/46813", "1", "smartgroups/smartgroups_get_after_create.json")
}
