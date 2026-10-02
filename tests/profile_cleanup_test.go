package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/client"
	"github.com/euc-oss/terraform-sdk-uem/resources"
)

func TestNewProfileDeleteCleanup_InvokesDeleteWithID(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, err := client.NewClient(&client.Config{
		InstanceURL: srv.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	})
	if err != nil {
		t.Fatalf("client.NewClient: %v", err)
	}
	profileService := resources.NewProfileService(c)

	cleanup := newProfileDeleteCleanup(profileService, 999, t.Logf)
	cleanup()

	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if !strings.Contains(gotPath, "999") {
		t.Errorf("path = %q, want it to contain profile id 999", gotPath)
	}
}

func TestNewProfileDeleteCleanup_LogsNotFailsOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	// This 500 is intentional (exercises log-don't-fail), and the shared
	// retry client (client/client.go) retries 500s 3x with 1-4s backoff by
	// design (workspaceOneRetryPolicy) before giving up -- ~7s of real,
	// harmless retry cost. client.Config.MaxRetries: 0 does NOT suppress
	// this: 0 is that struct's own "use the default (3)" sentinel
	// (client.go:97-98), not an explicit zero; there is no way to request
	// zero retries through the public Config today. Not fixed here --
	// changing that sentinel semantics is a client-package API decision
	// well outside this test's scope.
	c, err := client.NewClient(&client.Config{
		InstanceURL: srv.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	})
	if err != nil {
		t.Fatalf("client.NewClient: %v", err)
	}
	profileService := resources.NewProfileService(c)

	var loggedFormat string
	spyLogf := func(format string, args ...interface{}) { loggedFormat = format }

	cleanup := newProfileDeleteCleanup(profileService, 1, spyLogf)
	cleanup() // must not panic despite the 500

	if loggedFormat == "" {
		t.Error("expected the cleanup to log on Delete error, got no log call")
	}
}
