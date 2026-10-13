package mockserver

import (
	"net/http"
	"regexp"
	"testing"
)

var generatedGUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func correlationFixture() []*MockResponse {
	return []*MockResponse{{
		Metadata: ResponseMetadata{Endpoint: "/api/test", Method: "GET"},
		Response: ResponseSpec{StatusCode: 200, Body: map[string]string{"ok": "1"}},
	}}
}

func getWith(t *testing.T, url, id string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if id != "" {
		req.Header.Set("X-Correlation-ID", id)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

// Like UEM: reuse the request's id when it is a valid GUID, generate one otherwise,
// on matched and unmatched routes alike.
func TestMockServer_AnswersEveryRequestWithACorrelationID(t *testing.T) {
	ms := NewMockServer(correlationFixture())
	defer ms.Close()

	const valid = "feaccd25-ce86-7dc9-60f6-1b9477954856"
	if got := getWith(t, ms.URL()+"/api/test", valid).Header.Get("X-Correlation-ID"); got != valid {
		t.Errorf("valid GUID not reused: got %q, want %q", got, valid)
	}
	for _, bad := range []string{"", "not-a-guid", "12345", "feaccd25-ce86-7dc9-60f6-1b9477954856X", "aaaaaaaabbbb4ccc8dddeeeeeeeeeeee"} {
		got := getWith(t, ms.URL()+"/api/test", bad).Header.Get("X-Correlation-ID")
		if !generatedGUID.MatchString(got) {
			t.Errorf("request id %q: response id %q is not a generated v4 GUID", bad, got)
		}
	}
	// An unmatched route answers 404 and still carries the header.
	resp := getWith(t, ms.URL()+"/api/unknown", "")
	if resp.StatusCode != http.StatusNotFound || !generatedGUID.MatchString(resp.Header.Get("X-Correlation-ID")) {
		t.Errorf("404 route: status %d, id %q", resp.StatusCode, resp.Header.Get("X-Correlation-ID"))
	}
}

func TestMockServer_FixtureHeaderOverridesTheCorrelationID(t *testing.T) {
	fx := correlationFixture()
	fx[0].Response.Headers = map[string]string{"X-Correlation-ID": "fixture-value"}
	ms := NewMockServer(fx)
	defer ms.Close()
	if got := getWith(t, ms.URL()+"/api/test", "").Header.Get("X-Correlation-ID"); got != "fixture-value" {
		t.Errorf("got %q, want the fixture's header", got)
	}
}
