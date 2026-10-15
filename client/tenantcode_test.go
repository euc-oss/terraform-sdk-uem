package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// tenantRecorder records, per path, whether each request carried the
// aw-tenant-code header and with what value.
type tenantRecorder struct {
	*httptest.Server
	mu   sync.Mutex
	seen map[string][]*string // path -> header value per request; nil = header absent
}

func newTenantRecorder(t *testing.T) *tenantRecorder {
	t.Helper()
	r := &tenantRecorder{seen: map[string][]*string{}}
	r.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var v *string
		if vals, ok := req.Header["Aw-Tenant-Code"]; ok && len(vals) > 0 {
			v = &vals[0]
		}
		r.mu.Lock()
		r.seen[req.URL.Path] = append(r.seen[req.URL.Path], v)
		r.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(req.URL.Path, "/oauth/token") {
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "token_type": "Bearer", "expires_in": 3600})
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(r.Close)
	return r
}

func (r *tenantRecorder) values(path string) []*string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*string(nil), r.seen[path]...)
}

func TestTenantCode_OAuth2WithoutTenantCodeOmitsHeader(t *testing.T) {
	r := newTenantRecorder(t)
	c, err := NewClient(&Config{
		InstanceURL:    r.URL,
		AuthMethod:     "oauth2",
		ClientID:       "id",
		ClientSecret:   "secret",
		OAuth2TokenURL: r.URL + "/oauth/token",
	})
	if err != nil {
		t.Fatalf("NewClient with OAuth2 and no tenant code: %v", err)
	}
	if _, err := c.DoRequest(context.Background(), "GET", "/api/system/info", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	got := r.values("/api/system/info")
	if len(got) != 1 {
		t.Fatalf("api requests = %d, want 1", len(got))
	}
	if got[0] != nil {
		t.Errorf("aw-tenant-code sent as %q; want the header absent", *got[0])
	}
}

func TestTenantCode_OAuth2ProviderWithoutTenantCodeOmitsHeader(t *testing.T) {
	r := newTenantRecorder(t)
	c, err := NewClient(&Config{
		InstanceURL: r.URL,
		Auth:        NewOAuth2Auth(r.URL+"/oauth/token", "id", "secret"),
	})
	if err != nil {
		t.Fatalf("NewClient with an OAuth2 provider and no tenant code: %v", err)
	}
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := r.values("/api/x"); len(got) != 1 || got[0] != nil {
		t.Errorf("aw-tenant-code on api call = %v; want exactly one request with the header absent", got)
	}
}

func TestTenantCode_SentWhenSet(t *testing.T) {
	r := newTenantRecorder(t)
	c, err := NewClient(&Config{
		InstanceURL:    r.URL,
		TenantCode:     "tenant-key",
		AuthMethod:     "oauth2",
		ClientID:       "id",
		ClientSecret:   "secret",
		OAuth2TokenURL: r.URL + "/oauth/token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := r.values("/api/x"); len(got) != 1 || got[0] == nil || *got[0] != "tenant-key" {
		t.Errorf("aw-tenant-code on api call = %v; want exactly one request with %q", got, "tenant-key")
	}
}

// wrappedAuth is a caller-defined provider that delegates to another one.
type wrappedAuth struct{ AuthProvider }

func TestTenantCode_BasicAuthRequiresTenantCode(t *testing.T) {
	for name, cfg := range map[string]*Config{
		"AuthMethod basic":   {InstanceURL: "https://x", AuthMethod: "basic", Username: "u", Password: "p"},
		"BasicAuth provider": {InstanceURL: "https://x", Auth: NewBasicAuth("u", "p")},
		// NewClient classifies any provider other than *OAuth2Auth as basic, so a
		// wrapper around *BasicAuth needs the tenant code too.
		"wrapped BasicAuth provider": {InstanceURL: "https://x", Auth: wrappedAuth{NewBasicAuth("u", "p")}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewClient(cfg)
			if err == nil || !strings.Contains(err.Error(), "TenantCode is required for basic authentication") {
				t.Errorf("NewClient error = %v; want the basic-auth tenant code error", err)
			}
		})
	}
}
