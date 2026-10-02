package orgtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/client"
)

func TestNextSdkTestRunName_NoExisting(t *testing.T) {
	got := NextSdkTestRunName(nil)
	if got != "sdktestrun1" {
		t.Errorf("NextSdkTestRunName(nil) = %q, want sdktestrun1", got)
	}
}

func TestNextSdkTestRunName_DiscoversMax(t *testing.T) {
	existing := []string{"sdktestrun1", "sdktestrun3", "sdktestrun2", "unrelated-og-name"}
	got := NextSdkTestRunName(existing)
	if got != "sdktestrun4" {
		t.Errorf("NextSdkTestRunName(%v) = %q, want sdktestrun4", existing, got)
	}
}

func TestAddOrgGroupRequest_MarshalsExpectedFields(t *testing.T) {
	req := AddOrgGroupRequest{
		Name:                  "sdktestrun4",
		GroupCode:             "sdktestrun4",
		Country:               "United States",
		OrganizationGroupType: "Container",
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	for _, key := range []string{"name", "group_code", "country", "organization_group_type"} {
		if _, ok := got[key]; !ok {
			t.Errorf("marshaled request missing key %q, got %v", key, got)
		}
	}
}

func TestOrgGroupResponse_DecodesIdAndUuid(t *testing.T) {
	body := []byte(`{"Id": 42, "Uuid": "db8055e0-e030-7d5a-016b-ec4dc338d698"}`)
	var resp OrgGroupResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if resp.ID != 42 || resp.UUID != "db8055e0-e030-7d5a-016b-ec4dc338d698" {
		t.Errorf("got %+v, want ID=42 UUID=db8055e0-e030-7d5a-016b-ec4dc338d698", resp)
	}
}

func TestCreateDisposableOrgGroup_PostsAndRegistersCleanup(t *testing.T) {
	var createAccept string
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			createAccept = r.Header.Get("Accept")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"Id": 4001, "Uuid": "11e594f4-8195-8c10-e301-5d0bf0447a22"}`))
		case http.MethodDelete:
			deleteCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	// t.Cleanup, not defer: a test function's own defers run BEFORE its
	// t.Cleanup callbacks, so a plain "defer srv.Close()" would close the
	// server before CreateDisposableOrgGroup's own t.Cleanup(delete) fires
	// below -- registering srv.Close via t.Cleanup here (before that call)
	// means LIFO ordering runs the delete first, server-close second.
	t.Cleanup(srv.Close)

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

	id, uuid, err := CreateDisposableOrgGroup(t, c, 1000, "sdktestrun1")
	if err != nil {
		t.Fatalf("CreateDisposableOrgGroup: %v", err)
	}
	if id != 4001 || uuid != "11e594f4-8195-8c10-e301-5d0bf0447a22" {
		t.Errorf("got id=%d uuid=%q, want id=4001 uuid=11e594f4-8195-8c10-e301-5d0bf0447a22", id, uuid)
	}
	if createAccept != "application/json;version=3" {
		t.Errorf("create Accept header = %q, want application/json;version=3 (doctrine quirk 1)", createAccept)
	}
	_ = deleteCalled // cleanup-fires is verified separately, see the second test below
}

func TestCreateDisposableOrgGroup_CleanupDeletesOnExit(t *testing.T) {
	var deleteCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"Id": 4002, "Uuid": "e79acd97-ac88-0866-65d8-5a762f43d533"}`))
		case http.MethodDelete:
			deleteCalled = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	c, _ := client.NewClient(&client.Config{
		InstanceURL: srv.URL, TenantCode: "test-tenant",
		AuthMethod: "basic", Username: "test-user", Password: "test-pass",
	})

	t.Run("inner", func(t *testing.T) {
		if _, _, err := CreateDisposableOrgGroup(t, c, 1000, "sdktestrun2"); err != nil {
			t.Fatalf("CreateDisposableOrgGroup: %v", err)
		}
	})
	if !deleteCalled {
		t.Error("expected DELETE to fire on t.Cleanup after inner subtest exit")
	}
}

func TestParseSdkTestRunNamesFromTree_FiltersToConvention(t *testing.T) {
	// Shape confirmed against a live tenant (Task 1.2.10 D6 probe): GET
	// /api/system/groups/{uuid}/tree returns LOWERCASE keys (children/name),
	// unlike the PascalCase single-group GET (Id/Uuid/Name) -- two different
	// response shapes for the same resource type.
	tree := map[string]interface{}{
		"uuid": "parent-uuid",
		"name": "parent",
		"children": []interface{}{
			map[string]interface{}{"name": "sdktestrun1", "uuid": "c1"},
			map[string]interface{}{"name": "sdktestrun3", "uuid": "c3"},
			map[string]interface{}{"name": "unrelated-og", "uuid": "c4"},
		},
	}
	got := ParseSdkTestRunNamesFromTree(tree)
	want := []string{"sdktestrun1", "sdktestrun3"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("ParseSdkTestRunNamesFromTree(...) = %v, want %v", got, want)
	}
}

func TestParseSdkTestRunNamesFromTree_TolerantOfPascalCase(t *testing.T) {
	tree := map[string]interface{}{
		"Children": []interface{}{
			map[string]interface{}{"Name": "sdktestrun2"},
		},
	}
	got := ParseSdkTestRunNamesFromTree(tree)
	if len(got) != 1 || got[0] != "sdktestrun2" {
		t.Errorf("ParseSdkTestRunNamesFromTree(PascalCase) = %v, want [sdktestrun2]", got)
	}
}

func TestParseSdkTestRunNamesFromTree_NoChildrenKey(t *testing.T) {
	got := ParseSdkTestRunNamesFromTree(map[string]interface{}{"uuid": "parent-uuid", "name": "parent"})
	if len(got) != 0 {
		t.Errorf("ParseSdkTestRunNamesFromTree(no children) = %v, want empty", got)
	}
}

func TestNewDisposableOrgGroup_FallsBackOn403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errorCode": 6310, "message": "Admin does not have permission to create Partner type Organization Group"}`))
	}))
	defer srv.Close()

	c, _ := client.NewClient(&client.Config{
		InstanceURL: srv.URL, TenantCode: "test-tenant",
		AuthMethod: "basic", Username: "test-user", Password: "test-pass",
	})

	got := NewDisposableOrgGroup(t, c, 1000, nil)
	if got != 1000 {
		t.Errorf("NewDisposableOrgGroup fallback = %d, want parent id 1000 (shared-fixed fallback, D5)", got)
	}
}

// fatalfSpy is a minimal Fatalf-capturing double for testing
// NewDisposableOrgGroupOrFatal without actually failing the enclosing test
// (a real t.Fatalf would abort TestNewDisposableOrgGroupOrFatal_FailsLoudOnFallback
// itself, which is testing FOR that fatal call, not triggering it for real).
type fatalfSpy struct {
	*testing.T
	fataled bool
	msg     string
}

func (f *fatalfSpy) Fatalf(format string, args ...interface{}) {
	f.fataled = true
	f.msg = fmt.Sprintf(format, args...)
	// Deliberately does NOT call f.T.Fatalf / panic: a real mutate-tier
	// caller relies on Fatalf aborting the test via runtime.Goexit, but this
	// spy only needs to RECORD that Fatalf was called with the right
	// message -- calling the real T.Fatalf here would abort this very test.
}

func TestNewDisposableOrgGroupOrFatal_FailsLoudOnFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errorCode": 6310, "message": "Admin does not have permission to create Partner type Organization Group"}`))
	}))
	defer srv.Close()

	c, _ := client.NewClient(&client.Config{
		InstanceURL: srv.URL, TenantCode: "test-tenant",
		AuthMethod: "basic", Username: "test-user", Password: "test-pass",
	})

	spy := &fatalfSpy{T: t}
	got := NewDisposableOrgGroupOrFatal(spy, c, 1000, nil)
	if !spy.fataled {
		t.Fatal("expected NewDisposableOrgGroupOrFatal to call Fatalf when the disposable-OG create falls back to the shared OG, but it did not")
	}
	if !strings.Contains(spy.msg, "1000") {
		t.Errorf("Fatalf message should name the shared OG id it refused to use, got: %q", spy.msg)
	}
	if got != 1000 {
		t.Errorf("NewDisposableOrgGroupOrFatal returned %d, want the fallback id 1000 returned alongside the Fatalf call", got)
	}
}

func TestNewDisposableOrgGroupOrFatal_FailsLoudOnServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errorCode": 1000, "message": "Internal Server Error"}`))
	}))
	defer srv.Close()

	c, _ := client.NewClient(&client.Config{
		InstanceURL: srv.URL, TenantCode: "test-tenant",
		AuthMethod: "basic", Username: "test-user", Password: "test-pass",
	})

	spy := &fatalfSpy{T: t}
	got := NewDisposableOrgGroupOrFatal(spy, c, 1000, nil)
	if !spy.fataled {
		t.Fatal("expected NewDisposableOrgGroupOrFatal to call Fatalf when the disposable-OG create fails with a 500, but it did not")
	}
	if !strings.Contains(spy.msg, "1000") {
		t.Errorf("Fatalf message should name the shared OG id it refused to use, got: %q", spy.msg)
	}
	if got != 1000 {
		t.Errorf("NewDisposableOrgGroupOrFatal returned %d, want the fallback id 1000 returned alongside the Fatalf call", got)
	}
}

func TestNewDisposableOrgGroupOrFatal_NoFatalOnRealDisposableOG(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"Id": 5001, "Uuid": "8cca3400-b26b-951f-6157-d39a58db3ed8"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)

	c, _ := client.NewClient(&client.Config{
		InstanceURL: srv.URL, TenantCode: "test-tenant",
		AuthMethod: "basic", Username: "test-user", Password: "test-pass",
	})

	spy := &fatalfSpy{T: t}
	got := NewDisposableOrgGroupOrFatal(spy, c, 1000, nil)
	if spy.fataled {
		t.Errorf("unexpected Fatalf call %q -- disposable-OG create succeeded (id 5001), should not have fallen back", spy.msg)
	}
	if got != 5001 {
		t.Errorf("NewDisposableOrgGroupOrFatal = %d, want the real disposable OG id 5001", got)
	}
}
