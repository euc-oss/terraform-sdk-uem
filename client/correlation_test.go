package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

var guidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestNewCorrelationID_IsAFreshV4GUID(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		id := newCorrelationID()
		if !guidRE.MatchString(id) {
			t.Fatalf("not a v4 GUID: %q", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}

func TestAPIError_ErrorWithAndWithoutCorrelationID(t *testing.T) {
	cases := []struct {
		name string
		err  APIError
		want string
	}{
		{"no id, no code", APIError{StatusCode: 404, Message: "nope"}, "API error 404: nope"},
		{"no id, code", APIError{StatusCode: 400, Message: "bad", ErrorCode: "1000"}, "API error 400 (1000): bad"},
		{"id, no code", APIError{StatusCode: 404, Message: "nope", CorrelationID: "abc-123"}, "API error 404: nope (correlation id: abc-123)"},
		{"sent id differs", APIError{StatusCode: 502, Message: "gw", CorrelationID: "srv-1", SentCorrelationID: "sent-2"}, "API error 502: gw (correlation id: srv-1) (sent correlation id: sent-2)"},
		{"sent id equal is not repeated", APIError{StatusCode: 502, Message: "gw", CorrelationID: "same", SentCorrelationID: "same"}, "API error 502: gw (correlation id: same)"},
		{"id and code", APIError{StatusCode: 500, Message: "boom", ErrorCode: "7", CorrelationID: "abc-123"}, "API error 500 (7): boom (correlation id: abc-123)"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.err.Error(); got != c.want {
				t.Errorf("Error() = %q, want %q", got, c.want)
			}
		})
	}
}

func basicClient(t *testing.T, url string) *Client {
	t.Helper()
	c, err := NewClient(&Config{InstanceURL: url, TenantCode: "t", AuthMethod: "basic", Username: "u", Password: "p", MaxRetries: 3})
	if err != nil {
		t.Fatal(err)
	}
	c.httpClient.RetryWaitMin, c.httpClient.RetryWaitMax = 1, 2
	return c
}

func TestCorrelation_SentOnEveryRequestAndEchoedIntoTheError(t *testing.T) {
	const echoed = "cf4c4732-fd3b-8f8a-55b6-0871950a2f22"
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent = r.Header.Get("X-Correlation-ID")
		w.Header().Set("x-correlation-id", echoed) // case differs from the canonical form
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad input","errorCode":1000}`))
	}))
	defer srv.Close()
	_, err := basicClient(t, srv.URL).DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want APIError, got %v", err)
	}
	if !guidRE.MatchString(sent) {
		t.Errorf("request carried X-Correlation-ID %q, want a v4 GUID", sent)
	}
	if apiErr.CorrelationID != echoed {
		t.Errorf("CorrelationID = %q, want the server's %q (authoritative over the sent %q)", apiErr.CorrelationID, echoed, sent)
	}
	if want := "API error 400 (1000): bad input (correlation id: " + echoed + ") (sent correlation id: " + sent + ")"; apiErr.Error() != want {
		t.Errorf("Error() = %q, want %q", apiErr.Error(), want)
	}
}

func TestCorrelation_FallsBackToTheSentIDWhenTheResponseHasNone(t *testing.T) {
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent = r.Header.Get("X-Correlation-ID")
		w.WriteHeader(http.StatusBadGateway) // a gateway page: no correlation header
		_, _ = w.Write([]byte(`<html>bad gateway</html>`))
	}))
	defer srv.Close()
	c := basicClient(t, srv.URL)
	c.httpClient.RetryMax = 0
	_, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want APIError, got %v", err)
	}
	if sent == "" || apiErr.CorrelationID != sent {
		t.Errorf("CorrelationID = %q, want the sent id %q", apiErr.CorrelationID, sent)
	}
}

func TestCorrelation_SuccessfulResponseHeaderIsReadableByTheCaller(t *testing.T) {
	const echoed = "feaccd25-ce86-7dc9-60f6-1b9477954856"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Correlation-ID", echoed)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	h, err := basicClient(t, srv.URL).DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := h.Get(CorrelationIDHeader); got != echoed {
		t.Errorf("returned header = %q, want %q", got, echoed)
	}
}

func TestCorrelation_OneIDPerLogicalCallReusedAcrossRetries(t *testing.T) {
	var mu sync.Mutex
	var ids []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		ids = append(ids, r.Header.Get("X-Correlation-ID"))
		n := len(ids)
		mu.Unlock()
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := basicClient(t, srv.URL)
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	first := append([]string(nil), ids...)
	mu.Unlock()
	if len(first) != 3 {
		t.Fatalf("server saw %d attempts, want 3", len(first))
	}
	if first[0] == "" || first[0] != first[1] || first[1] != first[2] {
		t.Errorf("attempts used different ids: %v", first)
	}
	// A second logical call gets its own id.
	mu.Lock()
	ids = nil
	mu.Unlock()
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(ids) == 0 || ids[0] == first[0] {
		t.Errorf("a second call reused the first call's id: %v vs %v", ids, first)
	}
}

func TestCorrelation_RetryExhaustedErrorCarriesTheID(t *testing.T) {
	const echoed = "89d07480-e9d0-9240-f9e7-d59d4913aa6a"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Correlation-ID", echoed)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"server fault"}`))
	}))
	defer srv.Close()
	c := basicClient(t, srv.URL)
	c.httpClient.RetryMax = 2
	_, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil)
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want APIError, got %v", err)
	}
	if apiErr.CorrelationID != echoed || apiErr.Attempts != 3 {
		t.Errorf("CorrelationID=%q Attempts=%d, want %q and 3", apiErr.CorrelationID, apiErr.Attempts, echoed)
	}
}

func TestCleanCorrelationID(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{" abc-123 ", "abc-123"},
		// A value that is not a clean token is absent, not repaired: a repaired id
		// would match nothing in the server's logs.
		{"abc\r\ndef", ""},
		{"a b", ""},
		{"abc\x00", ""},
		{"\xff\xfeabc", ""},
		{strings.Repeat("x", maxCorrelationIDLen), strings.Repeat("x", maxCorrelationIDLen)},
		{strings.Repeat("x", maxCorrelationIDLen+1), ""},
		{"café", ""},
	}
	for _, c := range cases {
		if got := cleanCorrelationID(c.in); got != c.want {
			t.Errorf("cleanCorrelationID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := resolveCorrelationID("", "sent-id"); got != "sent-id" {
		t.Errorf("empty response header must fall back to the sent id, got %q", got)
	}
	if got := resolveCorrelationID("srv-id", "sent-id"); got != "srv-id" {
		t.Errorf("response header must win, got %q", got)
	}
}

// withDebugOutput routes the SDK's debug output to a buffer for the test.
func withDebugOutput(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	orig := logWriter
	logWriter = &buf
	t.Cleanup(func() { logWriter = orig })
	return &buf
}

func TestDebugLine_OneCompactLinePerCall(t *testing.T) {
	var echoed string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		echoed = r.Header.Get("X-Correlation-ID") // UEM reuses a valid request GUID
		w.Header().Set("X-Correlation-ID", echoed)
		_, _ = w.Write([]byte(`{"secret":"BODYVALUE"}`))
	}))
	defer srv.Close()
	c := basicClient(t, srv.URL) // built with debug off: no verbose transport dump
	buf := withDebugOutput(t)
	if _, err := c.DoRequest(context.Background(), "GET", "/api/mdm/profiles/12?searchtext=QUERYVALUE", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	want := "[UEM-SDK] GET /api/mdm/profiles/12 -> 200 correlation-id=" + echoed + "\n"
	if out != want {
		t.Errorf("debug output = %q, want exactly %q", out, want)
	}
	for _, leak := range []string{"QUERYVALUE", "BODYVALUE", "Authorization", "Basic ", "aw-tenant-code", "searchtext"} {
		if strings.Contains(out, leak) {
			t.Errorf("debug output leaks %q: %q", leak, out)
		}
	}
}

func TestDebugLine_PathCannotBreakTheLine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) }))
	defer srv.Close()
	c := basicClient(t, srv.URL)
	buf := withDebugOutput(t)
	// A raw CR/LF or DEL in a caller-built path is dropped from the line.
	c.debugCall("GET", "/api/a\r\n[UEM-SDK] FORGED\x7f/b?q=1", 200, "id", "id")
	want := "[UEM-SDK] GET /api/a[UEM-SDK] FORGED/b -> 200 correlation-id=id\n"
	if buf.String() != want {
		t.Errorf("debug line = %q, want %q", buf.String(), want)
	}
	if strings.Count(buf.String(), "\n") != 1 {
		t.Errorf("the line was split: %q", buf.String())
	}
}

func TestDebugLine_SilentWhenDebugIsOff(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) }))
	defer srv.Close()
	orig := logWriter
	t.Cleanup(func() { logWriter = orig })
	logWriter = io.Discard
	var buf bytes.Buffer
	c := basicClient(t, srv.URL)
	if _, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil); err != nil {
		t.Fatal(err)
	}
	if isDebugEnabled() {
		t.Error("debug reads as enabled with io.Discard")
	}
	if buf.Len() != 0 {
		t.Errorf("wrote debug output with debug off: %q", buf.String())
	}
}

func TestDebugLine_FallsBackToTheSentIDWhenNoResponseArrives(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Error("no hijacker")
			return
		}
		conn, _, _ := hj.Hijack()
		_ = conn.Close() // reset: no response at all
	}))
	defer srv.Close()
	c := basicClient(t, srv.URL)
	c.httpClient.RetryMax = 0
	buf := withDebugOutput(t)
	_, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, nil)
	if err == nil {
		t.Fatal("want a transport error")
	}
	line := strings.TrimSpace(buf.String())
	re := regexp.MustCompile(`^\[UEM-SDK\] GET /api/x -> no response correlation-id=([0-9a-f-]{36})$`)
	m := re.FindStringSubmatch(line)
	if m == nil {
		t.Fatalf("debug line = %q", line)
	}
	// The transport error names the same id, the only handle on a call whose
	// response was lost.
	if !strings.HasPrefix(err.Error(), "request failed: ") || !strings.HasSuffix(err.Error(), " (correlation id: "+m[1]+")") {
		t.Errorf("error = %q, want it to end with the sent correlation id %s", err.Error(), m[1])
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Errorf("a lost response must not look like an APIError: %v", err)
	}
}

// ---- every returned error carries the id, with debug off ----

func requireOnlyOnce(t *testing.T, text, id string) {
	t.Helper()
	if strings.Count(text, "correlation id: "+id) != 1 {
		t.Errorf("error text %q must contain %q exactly once", text, "(correlation id: "+id+")")
	}
}

func doGet(c *Client, ctx context.Context) error {
	_, err := c.DoRequest(ctx, "GET", "/api/x", "application/json", "application/json", nil, nil)
	return err
}

// sentIDs records the correlation ids a server received, in order.
type sentIDs struct {
	mu  sync.Mutex
	ids []string
}

func (s *sentIDs) add(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids = append(s.ids, id)
}

func (s *sentIDs) first(t *testing.T) string {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.ids) == 0 || !guidRE.MatchString(s.ids[0]) {
		t.Fatalf("server did not receive a GUID correlation id: %v", s.ids)
	}
	return s.ids[0]
}

func TestCorrelatedError_TextUnwrapAndHelper(t *testing.T) {
	base := errors.New("boom")
	err := &CorrelatedError{ID: "abc", Err: base}
	if err.Error() != "boom (correlation id: abc)" {
		t.Errorf("Error() = %q", err.Error())
	}
	if !errors.Is(err, base) {
		t.Error("errors.Is must see through the wrapper")
	}
	wrapped := fmt.Errorf("outer: %w", err)
	if got := CorrelationID(wrapped); got != "abc" {
		t.Errorf("CorrelationID through further wrapping = %q", got)
	}
	if got := CorrelationID(&APIError{StatusCode: 500, Message: "m", CorrelationID: "api-id"}); got != "api-id" {
		t.Errorf("CorrelationID(APIError) = %q", got)
	}
	if got := CorrelationID(errors.New("plain")); got != "" {
		t.Errorf("CorrelationID(plain) = %q, want empty", got)
	}
	if got := CorrelationID(nil); got != "" {
		t.Errorf("CorrelationID(nil) = %q, want empty", got)
	}
	// An error that already carries an id is not wrapped again.
	if again := withCorrelation(err, "other"); again != error(err) {
		t.Errorf("withCorrelation wrapped an error that already has an id: %v", again)
	}
	apiErr := &APIError{StatusCode: 400, Message: "m", CorrelationID: "api-id"}
	if again := withCorrelation(apiErr, "other"); again != error(apiErr) {
		t.Errorf("withCorrelation wrapped an APIError: %v", again)
	}
	if withCorrelation(nil, "x") != nil {
		t.Error("withCorrelation(nil) must stay nil")
	}
}

func TestCorrelation_TimeoutErrorCarriesTheSentID(t *testing.T) {
	var seen sentIDs
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.add(r.Header.Get("X-Correlation-ID"))
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	c, err := NewClient(&Config{InstanceURL: srv.URL, TenantCode: "t", AuthMethod: "basic", Username: "u", Password: "p", MaxRetries: 1, Timeout: 50 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	c.httpClient.RetryWaitMin, c.httpClient.RetryWaitMax = 1, 2
	err = doGet(c, context.Background())
	if err == nil {
		t.Fatal("want a timeout")
	}
	id := seen.first(t)
	requireOnlyOnce(t, err.Error(), id)
	if got := CorrelationID(err); got != id {
		t.Errorf("CorrelationID(err) = %q, want the sent %q", got, id)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("errors.Is(err, context.DeadlineExceeded) must hold for a client timeout: %v", err)
	}
	var ne net.Error
	if !errors.As(err, &ne) || !ne.Timeout() {
		t.Errorf("errors.As(net.Error) with Timeout() must hold: %v", err)
	}
	var ce *CorrelatedError
	if !errors.As(err, &ce) {
		t.Errorf("want a *CorrelatedError, got %T", err)
	}
	// Every attempt carried the same id.
	seen.mu.Lock()
	defer seen.mu.Unlock()
	for _, other := range seen.ids {
		if other != id {
			t.Errorf("attempts used different ids: %v", seen.ids)
		}
	}
}

func TestCorrelation_ContextDeadlineAndCancelCarryTheSentID(t *testing.T) {
	var seen sentIDs
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.add(r.Header.Get("X-Correlation-ID"))
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)
	c := basicClient(t, srv.URL)

	t.Run("deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
		defer cancel()
		err := doGet(c, ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("want context.DeadlineExceeded, got %v", err)
		}
		id := CorrelationID(err)
		if !guidRE.MatchString(id) {
			t.Fatalf("no GUID on the error: %q", err.Error())
		}
		requireOnlyOnce(t, err.Error(), id)
	})
	t.Run("cancel", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() { time.Sleep(40 * time.Millisecond); cancel() }()
		err := doGet(c, ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
		id := CorrelationID(err)
		if !guidRE.MatchString(id) {
			t.Fatalf("no GUID on the error: %q", err.Error())
		}
		requireOnlyOnce(t, err.Error(), id)
	})
}

func TestCorrelation_ConnectionRefusedCarriesTheSentID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing listens any more
	c := basicClient(t, url)
	c.httpClient.RetryMax = 1
	err := doGet(c, context.Background())
	if err == nil {
		t.Fatal("want a connection error")
	}
	id := CorrelationID(err)
	if !guidRE.MatchString(id) {
		t.Fatalf("no GUID on the error: %q", err.Error())
	}
	requireOnlyOnce(t, err.Error(), id)
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Errorf("the original *net.OpError must stay reachable with errors.As: %v", err)
	}
}

func TestCorrelation_ResetConnectionWithRetriesExhaustedCarriesOneID(t *testing.T) {
	var seen sentIDs
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen.add(r.Header.Get("X-Correlation-ID"))
		conn, _, _ := w.(http.Hijacker).Hijack()
		_ = conn.Close()
	}))
	defer srv.Close()
	c := basicClient(t, srv.URL)
	c.httpClient.RetryMax = 2
	err := doGet(c, context.Background())
	if err == nil || !strings.Contains(err.Error(), "giving up after 3 attempt(s)") {
		t.Fatalf("want retry-exhausted, got %v", err)
	}
	id := seen.first(t)
	requireOnlyOnce(t, err.Error(), id)
	seen.mu.Lock()
	defer seen.mu.Unlock()
	if len(seen.ids) != 3 {
		t.Fatalf("server saw %d attempts, want 3", len(seen.ids))
	}
	for _, other := range seen.ids {
		if other != id {
			t.Errorf("attempts used different ids: %v", seen.ids)
		}
	}
}

func TestCorrelation_TLSFailureCarriesTheSentID(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	// The default client does not trust the test server's certificate.
	c := basicClient(t, srv.URL)
	c.httpClient.RetryMax = 0
	err := doGet(c, context.Background())
	if err == nil {
		t.Fatal("want a TLS verification error")
	}
	id := CorrelationID(err)
	if !guidRE.MatchString(id) {
		t.Fatalf("no GUID on the error: %q", err.Error())
	}
	requireOnlyOnce(t, err.Error(), id)
}

func TestCorrelation_TokenCallFailureCarriesTheIDAndSendsItOnTheTokenRequest(t *testing.T) {
	var tokenID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/oauth/token") {
			tokenID = r.Header.Get("X-Correlation-ID")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		t.Error("the API must not be called when the token request fails")
	}))
	defer srv.Close()
	c, err := NewClient(&Config{InstanceURL: srv.URL, TenantCode: "t", AuthMethod: "oauth2", ClientID: "id", ClientSecret: "s", OAuth2TokenURL: srv.URL + "/oauth/token"})
	if err != nil {
		t.Fatal(err)
	}
	err = doGet(c, context.Background())
	if err == nil || !strings.Contains(err.Error(), "token request failed with status 401") {
		t.Fatalf("want a token failure, got %v", err)
	}
	if !guidRE.MatchString(tokenID) {
		t.Fatalf("the token request carried no GUID correlation id: %q", tokenID)
	}
	requireOnlyOnce(t, err.Error(), tokenID)
	if got := CorrelationID(err); got != tokenID {
		t.Errorf("CorrelationID(err) = %q, want the id sent on the token request %q", got, tokenID)
	}
}

func TestCorrelation_ClientSideErrorsCarryTheIDToo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) }))
	defer srv.Close()
	c := basicClient(t, srv.URL)
	_, err := c.DoRequest(context.Background(), "POST", "/api/x", "application/json", "application/json", make(chan int), nil)
	if err == nil {
		t.Fatal("want a marshal error")
	}
	id := CorrelationID(err)
	if !guidRE.MatchString(id) {
		t.Fatalf("no GUID on the error: %q", err.Error())
	}
	requireOnlyOnce(t, err.Error(), id)
}

func TestCorrelation_APIErrorIsNotWrappedAndKeepsBothIDs(t *testing.T) {
	const echoed = "cf4c4732-fd3b-8f8a-55b6-0871950a2f22"
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent = r.Header.Get("X-Correlation-ID")
		w.Header().Set("X-Correlation-ID", echoed) // a gateway answers with its own id
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"bad"}`))
	}))
	defer srv.Close()
	err := doGet(basicClient(t, srv.URL), context.Background())
	if _, ok := err.(*APIError); !ok {
		t.Fatalf("an API error response must stay a bare *APIError, got %T", err)
	}
	var apiErr *APIError
	errors.As(err, &apiErr)
	if apiErr.CorrelationID != echoed || apiErr.SentCorrelationID != sent || sent == echoed {
		t.Errorf("CorrelationID=%q SentCorrelationID=%q (sent %q)", apiErr.CorrelationID, apiErr.SentCorrelationID, sent)
	}
	want := "API error 400: bad (correlation id: " + echoed + ") (sent correlation id: " + sent + ")"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
	if got := CorrelationID(err); got != echoed {
		t.Errorf("CorrelationID(err) = %q, want the server's", got)
	}
}

func TestDebugLine_ShowsBothIDsWhenTheServerAnswersWithAnotherOne(t *testing.T) {
	var sent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent = r.Header.Get("X-Correlation-ID")
		w.Header().Set("X-Correlation-ID", "gateway-id")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := basicClient(t, srv.URL)
	buf := withDebugOutput(t)
	if err := doGet(c, context.Background()); err != nil {
		t.Fatal(err)
	}
	want := "[UEM-SDK] GET /api/x -> 200 correlation-id=gateway-id sent-correlation-id=" + sent + "\n"
	if buf.String() != want {
		t.Errorf("debug output = %q, want %q", buf.String(), want)
	}
}

type failingAuth struct{ err error }

func (f failingAuth) GetToken() (string, error) { return "", f.err }
func (f failingAuth) RefreshToken() error       { return f.err }

func TestCorrelation_TokenFailureKeepsTheErrorChain(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the API must not be called when the token request fails")
	}))
	defer srv.Close()

	t.Run("a provider that is not OAuth2Auth", func(t *testing.T) {
		sentinel := errors.New("token store unavailable")
		c, err := NewClient(&Config{InstanceURL: srv.URL, TenantCode: "t", Auth: failingAuth{err: sentinel}})
		if err != nil {
			t.Fatal(err)
		}
		err = doGet(c, context.Background())
		if !errors.Is(err, sentinel) {
			t.Fatalf("errors.Is lost the token failure's cause: %v", err)
		}
		id := CorrelationID(err)
		if !guidRE.MatchString(id) {
			t.Fatalf("no GUID on the error: %q", err.Error())
		}
		requireOnlyOnce(t, err.Error(), id)
	})

	t.Run("OAuth2 token endpoint unreachable", func(t *testing.T) {
		dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		tokenURL := dead.URL + "/oauth/token"
		dead.Close()
		c, err := NewClient(&Config{InstanceURL: srv.URL, TenantCode: "t", AuthMethod: "oauth2", ClientID: "id", ClientSecret: "s", OAuth2TokenURL: tokenURL})
		if err != nil {
			t.Fatal(err)
		}
		err = doGet(c, context.Background())
		var opErr *net.OpError
		if !errors.As(err, &opErr) {
			t.Fatalf("errors.As lost the *net.OpError behind the token failure: %v", err)
		}
		if !guidRE.MatchString(CorrelationID(err)) {
			t.Fatalf("no GUID on the error: %q", err.Error())
		}
	})
}

type invalidBody struct{}

func (invalidBody) Validate() error { return errors.New("name is required") }

// debugLines runs one call with debug output captured and returns the lines.
func debugLines(t *testing.T, c *Client, ctx context.Context, body interface{}, out interface{}) (lines []string, err error) {
	t.Helper()
	buf := withDebugOutput(t)
	_, err = c.DoRequest(ctx, "POST", "/api/x?q=1", "application/json", "application/json", body, out)
	for _, l := range strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n") {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines, err
}

func TestDebugLine_EveryErrorThatReturnsBeforeAResponseGetsANoResponseLine(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) }))
	defer srv.Close()
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	failTokens, err := NewClient(&Config{InstanceURL: srv.URL, TenantCode: "t", Auth: failingAuth{err: errors.New("token store unavailable")}})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		c    *Client
		ctx  context.Context
		body interface{}
	}{
		{"rate limiter canceled", basicClient(t, srv.URL), canceled, nil},
		{"client-side validation", basicClient(t, srv.URL), context.Background(), invalidBody{}},
		{"marshal error", basicClient(t, srv.URL), context.Background(), make(chan int)},
		{"token failure", failTokens, context.Background(), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			lines, err := debugLines(t, tc.c, tc.ctx, tc.body, nil)
			if err == nil {
				t.Fatal("want an error")
			}
			id := CorrelationID(err)
			if !guidRE.MatchString(id) {
				t.Fatalf("no GUID on the error: %q", err.Error())
			}
			want := "[UEM-SDK] POST /api/x -> no response correlation-id=" + id
			if len(lines) != 1 || lines[0] != want {
				t.Errorf("debug lines = %q, want exactly [%q]", lines, want)
			}
		})
	}
}

func TestDebugLine_ExactlyOneLineWhenTheBodyFailsAfterTheResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Correlation-ID", r.Header.Get("X-Correlation-ID"))
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()
	var out map[string]any
	lines, err := debugLines(t, basicClient(t, srv.URL), context.Background(), nil, &out)
	if err == nil {
		t.Fatal("want an unmarshal error")
	}
	id := CorrelationID(err)
	if want := "[UEM-SDK] POST /api/x -> 200 correlation-id=" + id; len(lines) != 1 || lines[0] != want {
		t.Errorf("debug lines = %q, want exactly [%q] (a response arrived: no extra no-response line)", lines, want)
	}
}

func TestCorrelation_BodyFailuresAfterTheResponseCarryTheServersID(t *testing.T) {
	const serverID = "cf4c4732-fd3b-8f8a-55b6-0871950a2f22"
	var sent string
	truncated := func(w http.ResponseWriter, r *http.Request) {
		sent = r.Header.Get("X-Correlation-ID")
		w.Header().Set("X-Correlation-ID", serverID)
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte(`{"a":`)) // fewer bytes than announced: the read fails
	}
	notJSON := func(w http.ResponseWriter, r *http.Request) {
		sent = r.Header.Get("X-Correlation-ID")
		w.Header().Set("X-Correlation-ID", serverID)
		_, _ = w.Write([]byte(`not json`))
	}
	for name, handler := range map[string]http.HandlerFunc{"read failure": truncated, "unmarshal failure": notJSON} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(handler)
			defer srv.Close()
			c := basicClient(t, srv.URL)
			c.httpClient.RetryMax = 0
			var out map[string]any
			_, err := c.DoRequest(context.Background(), "GET", "/api/x", "application/json", "application/json", nil, &out)
			if err == nil {
				t.Fatal("want an error")
			}
			if got := CorrelationID(err); got != serverID {
				t.Errorf("CorrelationID = %q, want the server's %q (the sent one was %q)", got, serverID, sent)
			}
			requireOnlyOnce(t, err.Error(), serverID)
			if strings.Contains(err.Error(), sent) {
				t.Errorf("the error text carries the sent id %q as well: %q", sent, err.Error())
			}
		})
	}
}

func TestDebugLine_PathDropsUnicodeLineSeparators(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) }))
	defer srv.Close()
	c := basicClient(t, srv.URL)
	buf := withDebugOutput(t)
	// NEL, LINE SEPARATOR, PARAGRAPH SEPARATOR, a no-break space and a zero-width space
	// all go; printable non-ASCII letters stay.
	c.debugCall("GET", "/api/a\u0085b\u2028c\u2029d\u00a0e\u200bf/caf\u00e9", 200, "id", "id")
	want := "[UEM-SDK] GET /api/abcdef/caf\u00e9 -> 200 correlation-id=id\n"
	if buf.String() != want {
		t.Errorf("debug line = %q, want %q", buf.String(), want)
	}
}
