package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/client"
)

// discoverPagingServer serves GET /api/mdm/profiles/search with the
// 1-indexed paging the live endpoint uses: page=0 is rejected outright
// (so any page-0 request fails the test loudly), pages past the end return
// an empty 204, and each non-empty page carries totalResults (unless
// omitTotal is set). It records every page number requested.
type discoverPagingServer struct {
	t         *testing.T
	ids       []int
	pageSize  int
	omitTotal bool
	// reportedTotal overrides the TotalResults value when non-nil, to
	// simulate a server whose total disagrees with the pages it serves.
	reportedTotal *int

	mu        sync.Mutex
	requested []int
}

func (s *discoverPagingServer) handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/mdm/profiles/search" {
		http.NotFound(w, r)
		return
	}
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		s.t.Errorf("search request without a numeric page param: %q", r.URL.RawQuery)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	s.requested = append(s.requested, page)
	s.mu.Unlock()
	if page == 0 {
		s.t.Errorf("Discover requested page=0; the search endpoint is 1-indexed and page=0 returns an empty 204")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	start := (page - 1) * s.pageSize
	if start >= len(s.ids) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	end := start + s.pageSize
	if end > len(s.ids) {
		end = len(s.ids)
	}
	items := make([]map[string]any, 0, end-start)
	for _, id := range s.ids[start:end] {
		items = append(items, map[string]any{"ProfileId": id, "Platform": "Android", "ProfileType": "Device"})
	}
	body := map[string]any{"ProfileList": items}
	if !s.omitTotal {
		total := len(s.ids)
		if s.reportedTotal != nil {
			total = *s.reportedTotal
		}
		body["TotalResults"] = total
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func newDiscoverPagingClient(t *testing.T, s *discoverPagingServer) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(s.handler))
	t.Cleanup(srv.Close)
	c, err := client.NewClient(&client.Config{
		InstanceURL: srv.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
		MaxRetries:  0,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func sequentialIDs(n int) []int {
	ids := make([]int, n)
	for i := range ids {
		ids[i] = 1000 + i
	}
	return ids
}

// Discover's page size is 500; 501 profiles forces a second page.
func TestDiscover_StartsAtPageOneAndCollectsEveryPage(t *testing.T) {
	s := &discoverPagingServer{t: t, ids: sequentialIDs(501), pageSize: 500}
	svc, err := sdk.NewProfileService(context.Background(), newDiscoverPagingClient(t, s))
	if err != nil {
		t.Fatalf("NewProfileService: %v", err)
	}
	if len(s.requested) == 0 || s.requested[0] != 1 {
		t.Fatalf("first requested page = %v, want 1", s.requested)
	}
	for _, p := range s.requested {
		if p == 0 {
			t.Fatalf("page=0 was requested: %v", s.requested)
		}
	}
	// A registry hit makes Get skip rediscovery, so a present id triggers no
	// further search request (its detail GET 404s against this stub, which
	// is fine -- only the absence of a re-search matters here).
	before := len(s.requested)
	for _, id := range []int{1000, 1499, 1500} {
		_, _ = svc.Get(context.Background(), id)
	}
	if after := len(s.requested); after != before {
		t.Errorf("Get on discovered ids re-ran discovery (%d -> %d search requests), so they were missing from the registry", before, after)
	}
}

// An organization group with zero profiles returns an empty 204 on page 1.
// That is a legitimate empty result, not a failure.
func TestDiscover_EmptyOrgGroupYieldsEmptyRegistryNoError(t *testing.T) {
	s := &discoverPagingServer{t: t, ids: nil, pageSize: 500}
	svc, err := sdk.NewProfileService(context.Background(), newDiscoverPagingClient(t, s))
	if err != nil {
		t.Fatalf("NewProfileService on an empty org group: %v", err)
	}
	if len(s.requested) != 1 || s.requested[0] != 1 {
		t.Errorf("requested pages = %v, want exactly [1]", s.requested)
	}
	if _, err := svc.Get(context.Background(), 1000); !client.IsNotFound(err) {
		t.Errorf("Get on an empty registry: err = %v, want a not-found error", err)
	}
}

// When the response carries a total, a collected count that disagrees
// with it means pages were missed; Discover must fail loud.
func TestDiscover_FailsLoudWhenCollectedCountDisagreesWithTotal(t *testing.T) {
	claimed := 900
	s := &discoverPagingServer{t: t, ids: sequentialIDs(501), pageSize: 500, reportedTotal: &claimed}
	_, err := sdk.NewProfileService(context.Background(), newDiscoverPagingClient(t, s))
	if err == nil {
		t.Fatal("NewProfileService succeeded despite collected (501) != reported total (900)")
	}
}

// Without a total, a short page (fewer items than the page size) is the stop.
func TestDiscover_StopsOnShortPageWhenNoTotal(t *testing.T) {
	s := &discoverPagingServer{t: t, ids: sequentialIDs(3), pageSize: 500, omitTotal: true}
	if _, err := sdk.NewProfileService(context.Background(), newDiscoverPagingClient(t, s)); err != nil {
		t.Fatalf("NewProfileService: %v", err)
	}
	if len(s.requested) != 1 || s.requested[0] != 1 {
		t.Errorf("requested pages = %v, want exactly [1]", s.requested)
	}
}

// Get's registry miss (after a discovery refresh) returns the typed
// not-found error, not a plain formatted error.
func TestProfileServiceGet_RegistryMissIsTypedNotFound(t *testing.T) {
	s := &discoverPagingServer{t: t, ids: sequentialIDs(2), pageSize: 500}
	svc, err := sdk.NewProfileService(context.Background(), newDiscoverPagingClient(t, s))
	if err != nil {
		t.Fatalf("NewProfileService: %v", err)
	}
	_, err = svc.Get(context.Background(), 424242)
	if !client.IsNotFound(err) {
		t.Fatalf("Get on an unknown id: err = %v, want IsNotFound", err)
	}
	var nf *client.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("Get on an unknown id: err = %T, want *client.NotFoundError", err)
	}
}

// The search endpoint reports platform as the canonical DeviceType enum
// member name, which differs from the detail-GET switch key for some
// platforms (live capture + canonical): "Apple" vs "Apple iOS",
// "WinRT" vs "Windows 10", "Linux" vs "To do", "Qnx" vs "Windows_Rugged"
// (canonical only). Discover must normalize those so Get reaches the detail
// endpoint. Any other value -- another DeviceType name, a numeric string,
// or an empty string -- must make Get fail with a typed
// *client.UnsupportedPlatformError naming the raw value.
func TestDiscover_NormalizesSearchPlatformStrings(t *testing.T) {
	mapped := map[int]string{1: "Apple", 2: "WinRT", 3: "Linux", 4: "AppleOsX", 5: "Android", 6: "Qnx"}
	unmapped := map[int]string{7: "AppleTv", 8: "3", 9: ""}
	platforms := map[int]string{}
	for k, v := range mapped {
		platforms[k] = v
	}
	for k, v := range unmapped {
		platforms[k] = v
	}
	var mu sync.Mutex
	var detailHits []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/mdm/profiles/search" {
			if r.URL.Query().Get("page") != "1" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			var items []map[string]any
			for id := 1; id <= len(platforms); id++ {
				items = append(items, map[string]any{"ProfileId": id, "Platform": platforms[id]})
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"ProfileList": items, "TotalResults": len(items)})
			return
		}
		mu.Lock()
		detailHits = append(detailHits, r.URL.Path)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	c, err := client.NewClient(&client.Config{InstanceURL: srv.URL, TenantCode: "t", AuthMethod: "basic", Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := sdk.NewProfileService(context.Background(), c)
	if err != nil {
		t.Fatalf("NewProfileService: %v", err)
	}
	for id, raw := range mapped {
		if _, err := svc.Get(context.Background(), id); err != nil {
			t.Errorf("Get(%d) with mapped search platform %q: %v", id, raw, err)
		}
	}
	for id, raw := range unmapped {
		_, err := svc.Get(context.Background(), id)
		var upe *client.UnsupportedPlatformError
		if !errors.As(err, &upe) {
			t.Errorf("Get(%d) with unmapped search platform %q: err = %v, want *client.UnsupportedPlatformError", id, raw, err)
			continue
		}
		if upe.RawPlatform != raw {
			t.Errorf("Get(%d): RawPlatform = %q, want %q", id, upe.RawPlatform, raw)
		}
		if raw == "" && !strings.Contains(err.Error(), "EMPTY") {
			t.Errorf("Get(%d) with an empty platform: error %q should call out the empty value explicitly", id, err)
		}
		if raw != "" && !strings.Contains(err.Error(), fmt.Sprintf("%q", raw)) {
			t.Errorf("Get(%d): error %q does not name the raw value %q", id, err, raw)
		}
	}
	if len(detailHits) != len(mapped) {
		t.Errorf("detail GETs = %d (%v), want %d (one per mapped platform, none for unmapped ones)", len(detailHits), detailHits, len(mapped))
	}
}

// WithProfileServiceOrganizationGroupID scopes every Discover search page
// to the given org group; without it, no organizationgroupid param is sent
// at all.
func TestDiscover_OrganizationGroupIDOption(t *testing.T) {
	const wantOGID = 149104

	newOrgGroupServer := func(t *testing.T, requireOGID bool) (*httptest.Server, *[]string) {
		t.Helper()
		var mu sync.Mutex
		var ogParams []string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/mdm/profiles/search" {
				http.NotFound(w, r)
				return
			}
			ogID := r.URL.Query().Get("organizationgroupid")
			mu.Lock()
			ogParams = append(ogParams, ogID)
			mu.Unlock()
			if requireOGID && ogID != strconv.Itoa(wantOGID) {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if !requireOGID && ogID != "" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if r.URL.Query().Get("page") != "1" {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"ProfileList":  []map[string]any{{"ProfileId": 1, "Platform": "Android", "ProfileType": "Device"}},
				"TotalResults": 1,
			})
		}))
		t.Cleanup(srv.Close)
		return srv, &ogParams
	}

	newClientFor := func(t *testing.T, srv *httptest.Server) *client.Client {
		t.Helper()
		c, err := client.NewClient(&client.Config{
			InstanceURL: srv.URL,
			TenantCode:  "test-tenant",
			AuthMethod:  "basic",
			Username:    "test-user",
			Password:    "test-pass",
			MaxRetries:  0,
		})
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		return c
	}

	t.Run("with option, every page request carries organizationgroupid", func(t *testing.T) {
		srv, ogParams := newOrgGroupServer(t, true)
		c := newClientFor(t, srv)
		svc := sdk.NewProfileServiceWithoutDiscovery(c, sdk.WithProfileServiceOrganizationGroupID(wantOGID))
		if err := svc.Discover(context.Background()); err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if len(*ogParams) == 0 {
			t.Fatal("no search requests were recorded")
		}
		for _, got := range *ogParams {
			if got != strconv.Itoa(wantOGID) {
				t.Errorf("organizationgroupid = %q, want %d", got, wantOGID)
			}
		}
	})

	t.Run("without option, no organizationgroupid is ever sent", func(t *testing.T) {
		srv, ogParams := newOrgGroupServer(t, false)
		c := newClientFor(t, srv)
		svc := sdk.NewProfileServiceWithoutDiscovery(c)
		if err := svc.Discover(context.Background()); err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if len(*ogParams) == 0 {
			t.Fatal("no search requests were recorded")
		}
		for _, got := range *ogParams {
			if got != "" {
				t.Errorf("organizationgroupid = %q, want empty (no scope configured)", got)
			}
		}
	})
}
