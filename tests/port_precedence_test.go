package tests

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// Ported fixtures (metadata.provenance.class PORTED-FROM-26.2) never win
// default lookup over a genuine fixture that also matches the request. These
// tests run against the real vendored corpus and skip without it.

func loadCorpusResponses(t *testing.T) []*mockserver.MockResponse {
	t.Helper()
	walkVendoredFixtureCorpus(t) // skips when MOCK_RESPONSES_DIR is unset or the corpus is too small
	responses, err := mockserver.LoadResponsesFromDir(mockserver.ResolveResponsesDir("../testdata/mock-responses"))
	if err != nil {
		t.Fatalf("load corpus: %v", err)
	}
	return responses
}

func versionedRequest(method, target, version string) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	if version != "" {
		r.Header.Set("Accept", "application/json;version="+version)
	}
	return r
}

func TestPortPrecedence_PagedProfileSearchServesGenuine(t *testing.T) {
	responses := loadCorpusResponses(t)
	req := versionedRequest("GET", "/api/mdm/profiles/search?page=1&pagesize=500", "2")
	got := mockserver.NewRequestMatcher(responses).Match(req)
	if got == nil {
		t.Fatal("no fixture matched a paged v2 profile search")
	}
	if got.Metadata.IsPorted() {
		t.Errorf("paged v2 profile search served the port %s; a genuine search capture exists and must win", got.SourceFile)
	}
}

func TestPortPrecedence_SolePortStillServed(t *testing.T) {
	walkVendoredFixtureCorpus(t)
	port, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/search_v2.json"))
	if err != nil {
		t.Fatal(err)
	}
	detail, err := mockserver.LoadResponseFromFile(mockserver.ResolveResponsesDir("../testdata/mock-responses/profiles/get_android_testdouble.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !port.Metadata.IsPorted() {
		t.Fatalf("profiles/search_v2.json is expected to be a PORTED-FROM-26.2 fixture")
	}
	req := versionedRequest("GET", "/api/mdm/profiles/search?page=1&pagesize=500", "2")
	if got := mockserver.NewRequestMatcher([]*mockserver.MockResponse{port, detail}).Match(req); got != port {
		t.Errorf("isolated server: got %v, want the sole matching port search_v2.json", got)
	}
}

var endpointPlaceholder = regexp.MustCompile(`\{[^}]+\}`)

// TestPortPrecedence_SweepEveryPort builds a request from each port's own
// metadata. Wherever a genuine fixture also matches at least as specifically
// (not a genuine {id} template against the port's exact literal path), the
// genuine one must be served.
func TestPortPrecedence_SweepEveryPort(t *testing.T) {
	responses := loadCorpusResponses(t)
	var genuine []*mockserver.MockResponse
	for _, r := range responses {
		if !r.Metadata.IsPorted() {
			genuine = append(genuine, r)
		}
	}
	all := mockserver.NewRequestMatcher(responses)
	genuineOnly := mockserver.NewRequestMatcher(genuine)
	ports, shadowed := 0, 0
	for _, r := range responses {
		if !r.Metadata.IsPorted() {
			continue
		}
		ports++
		target := endpointPlaceholder.ReplaceAllString(r.Metadata.Endpoint, "12345")
		if len(r.Request.QueryParams) > 0 {
			q := make([]string, 0, len(r.Request.QueryParams))
			for k, v := range r.Request.QueryParams {
				q = append(q, k+"="+v)
			}
			sep := "?"
			for _, kv := range q {
				target += sep + kv
				sep = "&"
			}
		}
		req := versionedRequest(r.Metadata.Method, target, r.Metadata.Version)
		g := genuineOnly.Match(req)
		if g == nil {
			continue
		}
		// A port with an exact literal path is more specific than a genuine
		// {id} template and legitimately wins; skip that case.
		if r.Metadata.Endpoint == target && g.Metadata.Endpoint != target {
			continue
		}
		shadowed++
		if got := all.Match(req); got == nil || got.Metadata.IsPorted() {
			name := "<nil>"
			if got != nil {
				name = got.SourceFile
			}
			t.Errorf("%s %s (port %s): served %s, want a genuine fixture", r.Metadata.Method, target, r.SourceFile, name)
		}
	}
	if ports == 0 {
		t.Fatal("no PORTED-FROM-26.2 fixtures in the corpus; the sweep proved nothing")
	}
	t.Logf("%d ports swept, %d also matched by a genuine fixture", ports, shadowed)
}

// TestPortPrecedence_StatefulPreferenceReachesPort: stateful routing still
// serves the after-update port once a 2xx PUT has been observed.
func TestPortPrecedence_StatefulPreferenceReachesPort(t *testing.T) {
	walkVendoredFixtureCorpus(t)
	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	const path = "/api/mdm/smartgroups/12346"
	assertServes(t, ms, http.MethodPut, path, "1", "smartgroups/smartgroups_update.json")
	assertServes(t, ms, http.MethodGet, path, "1", "smartgroups/smartgroups_get_after_update.json")
}
