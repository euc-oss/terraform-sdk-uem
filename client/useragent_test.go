package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDefaultUserAgentCarriesTheSDKVersion(t *testing.T) {
	want := "uem-sdk-go/" + sdkVersion()
	if got := DefaultUserAgent(); got != want {
		t.Errorf("DefaultUserAgent() = %q, want %q", got, want)
	}
	// In the private tree the placeholder is unreplaced and reads "dev"; in a
	// published tree it is the three-part release version.
	if strings.Contains(DefaultUserAgent(), "$") || strings.HasSuffix(DefaultUserAgent(), "/") {
		t.Errorf("default user agent is malformed: %q", DefaultUserAgent())
	}
}

func TestSanitizeUserAgentPrefix(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"plain", "my-app/1.0", "my-app/1.0"},
		{"several tokens", "my-app/1.2.0 my-framework/3.4.0", "my-app/1.2.0 my-framework/3.4.0"},
		{"trims", "  my-app/1.0  ", "my-app/1.0"},
		{"collapses spaces", "my-app/1.0     terraform/1.9.0", "my-app/1.0 terraform/1.9.0"},
		{"collapses tabs and mixed white space", "a \t \t b", "a b"},
		{"CR LF becomes a separator", "my-app/1.0\r\nX-Injected: yes", "my-app/1.0 X-Injected: yes"},
		{"lone CR", "a\rb", "a b"},
		{"lone LF", "a\nb", "a b"},
		{"NUL and other controls", "a\x00b\x07c\x1bd", "a b c d"},
		{"DEL", "a\x7fb", "a b"},
		{"only control characters", "\r\n\x00", ""},
		{"empty", "", ""},
		{"only spaces", "   ", ""},
		{"non-ASCII dropped, token not split", "café-app/1", "caf-app/1"},
		{"unicode space is a separator", "a b c", "a b c"},
		{"leading control", "\r\nmy-app/1", "my-app/1"},
		{"trailing control", "my-app/1\r\n", "my-app/1"},
		{"capped at a token boundary", strings.Repeat("abcdefghi ", 40), strings.TrimRight(strings.Repeat("abcdefghi ", 40)[:250], " ")},
		{"capped inside one huge token", strings.Repeat("x", 1000), strings.Repeat("x", MaxUserAgentPrefixLen)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SanitizeUserAgentPrefix(c.in)
			if got != c.want {
				t.Errorf("SanitizeUserAgentPrefix(%q) = %q, want %q", c.in, got, c.want)
			}
			if strings.Contains(got, "  ") || strings.HasPrefix(got, " ") || strings.HasSuffix(got, " ") {
				t.Errorf("result %q has a double, leading or trailing space", got)
			}
			if strings.ContainsAny(got, "\r\n\x00") {
				t.Errorf("result %q still contains a control character", got)
			}
		})
	}
}

func TestBuildUserAgent(t *testing.T) {
	def := DefaultUserAgent()
	cases := []struct{ name, in, want string }{
		{"no caller token", "", def},
		{"caller token is prepended", "my-app/1.2.0 my-framework/3.4.0", "my-app/1.2.0 my-framework/3.4.0 " + def},
		{"sanitized to empty sends only the default", "\r\n \t", def},
		{"sanitized before joining", "  my-app/1\r\n  x/2 ", "my-app/1 x/2 " + def},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := BuildUserAgent(c.in)
			if got != c.want {
				t.Errorf("BuildUserAgent(%q) = %q, want %q", c.in, got, c.want)
			}
			if strings.Contains(got, "  ") {
				t.Errorf("user agent %q has a double space", got)
			}
		})
	}
}

// uaRecorder is a server that records the User-Agent of every request it gets
// and answers the token endpoint and any API path.
type uaRecorder struct {
	*httptest.Server
	mu     sync.Mutex
	agents map[string][]string // path -> user agents, in arrival order
}

func newUARecorder(t *testing.T) *uaRecorder {
	t.Helper()
	r := &uaRecorder{agents: map[string][]string{}}
	r.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.agents[req.URL.Path] = append(r.agents[req.URL.Path], req.Header.Get("User-Agent"))
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

func (r *uaRecorder) seen(path string) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.agents[path]...)
}

func oauthConfig(r *uaRecorder, ua string) *Config {
	return &Config{
		InstanceURL:    r.URL,
		TenantCode:     "t",
		AuthMethod:     "oauth2",
		ClientID:       "id",
		ClientSecret:   "secret",
		OAuth2TokenURL: r.URL + "/oauth/token",
		UserAgent:      ua,
	}
}

func TestUserAgent_DefaultOnApiAndTokenCalls(t *testing.T) {
	r := newUARecorder(t)
	c, err := NewClient(oauthConfig(r, ""))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	want := DefaultUserAgent()
	if got := r.seen("/oauth/token"); len(got) != 1 || got[0] != want {
		t.Errorf("token call user agents = %q, want [%q]", got, want)
	}
	if got := r.seen("/api/x"); len(got) != 1 || got[0] != want {
		t.Errorf("api call user agents = %q, want [%q]", got, want)
	}
	if strings.HasPrefix(want, "Go-http-client") {
		t.Error("the default is Go's own user agent")
	}
}

func TestUserAgent_CallerTokenIsPrependedOnApiAndTokenCalls(t *testing.T) {
	r := newUARecorder(t)
	const caller = "my-app/1.2.0 my-framework/3.4.0"
	c, err := NewClient(oauthConfig(r, caller))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	want := caller + " " + DefaultUserAgent()
	for _, path := range []string{"/oauth/token", "/api/x"} {
		if got := r.seen(path); len(got) != 1 || got[0] != want {
			t.Errorf("%s user agents = %q, want [%q]", path, got, want)
		}
	}
}

func TestUserAgent_NotSetByTheSDKMeansNoPrefixAtAll(t *testing.T) {
	// The SDK ships no caller prefix of its own.
	if ua := BuildUserAgent(""); ua != DefaultUserAgent() || strings.Contains(ua, " ") {
		t.Errorf("without a caller token the user agent must be the SDK token alone, got %q", ua)
	}
}

func TestUserAgent_HostileCallerTokenCannotSplitTheHeaderOrDoubleSpace(t *testing.T) {
	r := newUARecorder(t)
	c, err := NewClient(oauthConfig(r, "  app/1\r\nX-Evil: 1   tail "))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	want := "app/1 X-Evil: 1 tail " + DefaultUserAgent()
	for _, path := range []string{"/oauth/token", "/api/x"} {
		got := r.seen(path)
		if len(got) != 1 || got[0] != want {
			t.Errorf("%s user agents = %q, want [%q]", path, got, want)
		}
	}
}

func TestUserAgent_EmptyAfterSanitizingSendsOnlyTheDefault(t *testing.T) {
	r := newUARecorder(t)
	c, err := NewClient(oauthConfig(r, " \r\n\t "))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/oauth/token", "/api/x"} {
		if got := r.seen(path); len(got) != 1 || got[0] != DefaultUserAgent() {
			t.Errorf("%s user agents = %q, want [%q]", path, got, DefaultUserAgent())
		}
	}
}

func TestUserAgent_EveryRetryAttemptCarriesIt(t *testing.T) {
	var mu sync.Mutex
	var agents []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		agents = append(agents, req.Header.Get("User-Agent"))
		n := len(agents)
		mu.Unlock()
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c, err := NewClient(&Config{InstanceURL: srv.URL, TenantCode: "t", AuthMethod: "basic", Username: "u", Password: "p", UserAgent: "app/1", MaxRetries: 3})
	if err != nil {
		t.Fatal(err)
	}
	// Do not wait out the production backoff.
	c.httpClient.RetryWaitMin, c.httpClient.RetryWaitMax = 1, 2
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(agents) != 3 {
		t.Fatalf("server saw %d attempts, want 3", len(agents))
	}
	for i, a := range agents {
		if a != "app/1 "+DefaultUserAgent() {
			t.Errorf("attempt %d user agent = %q", i+1, a)
		}
	}
}

func TestUserAgent_ProviderValueThatSanitizesToNothingKeepsInheriting(t *testing.T) {
	r := newUARecorder(t)
	auth := NewOAuth2Auth(r.URL+"/oauth/token", "id", "secret")
	auth.SetUserAgent("  \r\n ")
	c, err := NewClient(&Config{InstanceURL: r.URL, TenantCode: "t", Auth: auth, UserAgent: "three/3"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	want := "three/3 " + DefaultUserAgent()
	for _, path := range []string{"/oauth/token", "/api/x"} {
		if got := r.seen(path); len(got) != 1 || got[0] != want {
			t.Errorf("%s user agents = %q, want [%q]", path, got, want)
		}
	}
}

func TestUserAgent_ProviderSharedByTwoClientsTakesTheLastOnesValue(t *testing.T) {
	r := newUARecorder(t)
	auth := NewOAuth2Auth(r.URL+"/oauth/token", "id", "secret")
	for _, ua := range []string{"one/1", "two/2"} {
		if _, err := NewClient(&Config{InstanceURL: r.URL, TenantCode: "t", Auth: auth, UserAgent: ua}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := auth.GetToken(); err != nil {
		t.Fatal(err)
	}
	if got := r.seen("/oauth/token"); len(got) != 1 || got[0] != "two/2 "+DefaultUserAgent() {
		t.Errorf("token call user agents = %q, want the last client's value", got)
	}
}

func TestUserAgent_ProviderWithItsOwnTokenUserAgentKeepsIt(t *testing.T) {
	r := newUARecorder(t)
	auth := NewOAuth2Auth(r.URL+"/oauth/token", "id", "secret")
	auth.SetUserAgent("token-app/9")
	c, err := NewClient(&Config{InstanceURL: r.URL, TenantCode: "t", Auth: auth, UserAgent: "api-app/1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := r.seen("/oauth/token"); len(got) != 1 || got[0] != "token-app/9 "+DefaultUserAgent() {
		t.Errorf("token call user agents = %q", got)
	}
	if got := r.seen("/api/x"); len(got) != 1 || got[0] != "api-app/1 "+DefaultUserAgent() {
		t.Errorf("api call user agents = %q", got)
	}
}

func TestUserAgent_InjectedProviderInheritsTheClientUserAgent(t *testing.T) {
	r := newUARecorder(t)
	auth := NewOAuth2Auth(r.URL+"/oauth/token", "id", "secret")
	c, err := NewClient(&Config{InstanceURL: r.URL, TenantCode: "t", Auth: auth, UserAgent: "app/1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := r.seen("/oauth/token"); len(got) != 1 || got[0] != "app/1 "+DefaultUserAgent() {
		t.Errorf("token call user agents = %q", got)
	}
}

func TestUserAgent_StandaloneProviderStillSendsTheSDKToken(t *testing.T) {
	r := newUARecorder(t)
	auth := NewOAuth2Auth(r.URL+"/oauth/token", "id", "secret")
	if _, err := auth.GetToken(); err != nil {
		t.Fatal(err)
	}
	if got := r.seen("/oauth/token"); len(got) != 1 || got[0] != DefaultUserAgent() {
		t.Errorf("token call user agents = %q, want [%q]", got, DefaultUserAgent())
	}
}

func TestUserAgent_ClearingTheProviderValueResumesInheriting(t *testing.T) {
	r := newUARecorder(t)
	auth := NewOAuth2Auth(r.URL+"/oauth/token", "id", "secret")
	auth.SetUserAgent("token-app/9")
	c, err := NewClient(&Config{InstanceURL: r.URL, TenantCode: "t", Auth: auth, UserAgent: "api-app/1"})
	if err != nil {
		t.Fatal(err)
	}
	auth.SetUserAgent(" \r\n ") // empty after cleaning counts as not given: back to the client's value
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	want := "api-app/1 " + DefaultUserAgent()
	if got := r.seen("/oauth/token"); len(got) != 1 || got[0] != want {
		t.Errorf("token call user agents = %q, want [%q]", got, want)
	}
	// and an explicit value can be set again afterwards
	auth.SetUserAgent("again/2")
	auth.mu.Lock()
	auth.token, auth.expiryTime = "", time.Time{}
	auth.mu.Unlock()
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := r.seen("/oauth/token"); len(got) != 2 || got[1] != "again/2 "+DefaultUserAgent() {
		t.Errorf("token call user agents = %q, want the second to be the new explicit value", got)
	}
}
