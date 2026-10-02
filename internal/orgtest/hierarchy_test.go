package orgtest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/v26/client"
)

// fakeT is a standalone HierarchyT double -- deliberately NOT built on a
// real *testing.T, since Fatalf's runtime.Goexit() must unwind only the
// goroutine running the code under test, not a goroutine testing.T itself
// owns (calling Goexit inside a plain function invoked from a *testing.T's
// own test goroutine, without going through t.FailNow's bookkeeping, trips
// "test executed panic(nil) or runtime.Goexit"). Cleanup funcs are recorded
// and run manually via runCleanups so tests control LIFO ordering directly.
type fakeT struct {
	cleanups []func()
	errs     []string
	fatal    string
	fataled  bool
}

func (f *fakeT) Helper()             {}
func (f *fakeT) Logf(string, ...any) {}

func (f *fakeT) Cleanup(fn func()) {
	f.cleanups = append(f.cleanups, fn)
}

func (f *fakeT) Errorf(format string, args ...any) {
	f.errs = append(f.errs, fmt.Sprintf(format, args...))
}

func (f *fakeT) Fatalf(format string, args ...any) {
	f.fataled = true
	f.fatal = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

// runCleanups fires f's registered Cleanup funcs in LIFO order, mirroring
// testing.T's own registration-order contract.
func runCleanups(f *fakeT) {
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i]()
	}
}

// callHierarchy runs NewDisposableHierarchy on its own goroutine so a
// Fatalf-triggered runtime.Goexit unwinds only that goroutine (running its
// deferred close(done)) rather than the test's own goroutine.
func callHierarchy(ft *fakeT, c *client.Client, parentID int, opts ...HierarchyOption) *Hierarchy { //nolint:unparam // parentID stays a parameter so each call site states the parent it targets
	var h *Hierarchy
	done := make(chan struct{})
	go func() {
		defer close(done)
		h = NewDisposableHierarchy(ft, c, parentID, opts...)
	}()
	<-done
	return h
}

func newTestClient(t *testing.T, url string) *client.Client {
	t.Helper()
	c, err := client.NewClient(&client.Config{
		InstanceURL: url, TenantCode: "test-tenant",
		AuthMethod: "basic", Username: "test-user", Password: "test-pass",
	})
	if err != nil {
		t.Fatalf("client.NewClient: %v", err)
	}
	return c
}

type reqLog struct {
	method string
	path   string
}

// deleted tracks which delete paths a test's handler has already seen, so
// the second hit of the same path can return the doctrine-quirk-20 400.
// Reset (fresh map assignment) at the top of every test that uses it.
var deleted = map[string]bool{}

func TestNewDisposableHierarchy_CreatesRootAndChildrenInOrder(t *testing.T) {
	deleted = map[string]bool{}
	var log []reqLog
	nextID := 5000

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log = append(log, reqLog{r.Method, r.URL.Path})
		switch r.Method {
		case http.MethodGet:
			// DiscoverSdkTestRunNames: parent GET then tree GET.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if r.URL.Path == "/api/system/groups/1000" {
				_, _ = w.Write([]byte(`{"Id": 1000, "Uuid": "parent-uuid"}`))
			} else {
				_, _ = w.Write([]byte(`{"children": []}`))
			}
		case http.MethodPost:
			nextID++
			id := nextID
			uuid := fmt.Sprintf("uuid-%d", id)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = fmt.Fprintf(w, `{"Id": %d, "Uuid": %q}`, id, uuid)
		case http.MethodDelete:
			path := r.URL.Path
			if deleted[path] {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"errorCode": 5005, "message": "not found"}`))
				return
			}
			deleted[path] = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)

	c := newTestClient(t, srv.URL)

	ft := &fakeT{}
	h := callHierarchy(ft, c, 1000, WithChildren(2))
	if ft.fataled {
		t.Fatalf("unexpected Fatalf: %s", ft.fatal)
	}
	if len(ft.errs) != 0 {
		t.Fatalf("unexpected Errorf calls: %v", ft.errs)
	}

	if h.Root.ID == 0 || h.Root.UUID == "" || h.Root.Name == "" {
		t.Fatalf("root not populated: %+v", h.Root)
	}
	if len(h.Children) != 2 {
		t.Fatalf("got %d children, want 2", len(h.Children))
	}
	for i, child := range h.Children {
		wantName := h.Root.Name + "c" + fmt.Sprint(i+1)
		if child.Name != wantName {
			t.Errorf("child[%d].Name = %q, want %q", i, child.Name, wantName)
		}
	}

	// Expected creation sequence so far: discover (GET parent, GET tree),
	// POST root, POST child1, POST child2.
	wantCreateMethods := []string{"GET", "GET", "POST", "POST", "POST"}
	if len(log) < len(wantCreateMethods) {
		t.Fatalf("got %d requests before cleanup, want at least %d: %+v", len(log), len(wantCreateMethods), log)
	}
	for i, want := range wantCreateMethods {
		if log[i].method != want {
			t.Errorf("request[%d] method = %q, want %q (full log: %+v)", i, log[i].method, want, log)
		}
	}

	runCleanups(ft)
	if len(ft.errs) != 0 {
		t.Fatalf("unexpected Errorf calls during cleanup: %v", ft.errs)
	}

	// Cleanup (LIFO): DELETE child2 x2, DELETE child1 x2, DELETE root x2.
	wantDeletes := []reqLog{
		{"DELETE", "/api/system/groups/" + h.Children[1].UUID},
		{"DELETE", "/api/system/groups/" + h.Children[1].UUID},
		{"DELETE", "/api/system/groups/" + h.Children[0].UUID},
		{"DELETE", "/api/system/groups/" + h.Children[0].UUID},
		{"DELETE", "/api/system/groups/" + h.Root.UUID},
		{"DELETE", "/api/system/groups/" + h.Root.UUID},
	}
	got := log[len(wantCreateMethods):]
	if len(got) != len(wantDeletes) {
		t.Fatalf("got %d delete requests, want %d: %+v", len(got), len(wantDeletes), got)
	}
	for i, want := range wantDeletes {
		if got[i] != want {
			t.Errorf("delete[%d] = %+v, want %+v", i, got[i], want)
		}
	}
}

func TestNewDisposableHierarchy_SecondDeleteWrongStatus_Errorf(t *testing.T) {
	deleted = map[string]bool{}
	secondStatus := http.StatusOK // planted negative control: should be 400

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			if r.URL.Path == "/api/system/groups/1000" {
				_, _ = w.Write([]byte(`{"Id": 1000, "Uuid": "parent-uuid"}`))
			} else {
				_, _ = w.Write([]byte(`{"children": []}`))
			}
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"Id": 6001, "Uuid": "uuid-6001"}`))
		case http.MethodDelete:
			if deleted[r.URL.Path] {
				w.WriteHeader(secondStatus)
				return
			}
			deleted[r.URL.Path] = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)

	ft := &fakeT{}
	callHierarchy(ft, c, 1000)
	runCleanups(ft)
	if len(ft.errs) == 0 {
		t.Error("expected Errorf on wrong second-delete status (planted 200), got none")
	}
}

func TestNewDisposableHierarchy_SecondDeleteServerError_Errorf(t *testing.T) {
	deleted = map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			if r.URL.Path == "/api/system/groups/1000" {
				_, _ = w.Write([]byte(`{"Id": 1000, "Uuid": "parent-uuid"}`))
			} else {
				_, _ = w.Write([]byte(`{"children": []}`))
			}
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"Id": 6002, "Uuid": "uuid-6002"}`))
		case http.MethodDelete:
			if deleted[r.URL.Path] {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"errorCode": 1000, "message": "Internal Server Error"}`))
				return
			}
			deleted[r.URL.Path] = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)

	ft := &fakeT{}
	callHierarchy(ft, c, 1000)
	runCleanups(ft)
	if len(ft.errs) == 0 {
		t.Error("expected Errorf on planted 500 second-delete status, got none")
	}
}

func TestNewDisposableHierarchy_SecondDeleteCorrectStatus_NoErrorf(t *testing.T) {
	deleted = map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			if r.URL.Path == "/api/system/groups/1000" {
				_, _ = w.Write([]byte(`{"Id": 1000, "Uuid": "parent-uuid"}`))
			} else {
				_, _ = w.Write([]byte(`{"children": []}`))
			}
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"Id": 6003, "Uuid": "uuid-6003"}`))
		case http.MethodDelete:
			if deleted[r.URL.Path] {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"errorCode": 5005, "message": "not found"}`))
				return
			}
			deleted[r.URL.Path] = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)

	ft := &fakeT{}
	callHierarchy(ft, c, 1000)
	runCleanups(ft)
	if len(ft.errs) != 0 {
		t.Errorf("unexpected Errorf calls on correct 400 second-delete: %v", ft.errs)
	}
}

func TestNewDisposableHierarchy_FirstDeleteFails_Errorf(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			if r.URL.Path == "/api/system/groups/1000" {
				_, _ = w.Write([]byte(`{"Id": 1000, "Uuid": "parent-uuid"}`))
			} else {
				_, _ = w.Write([]byte(`{"children": []}`))
			}
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"Id": 6004, "Uuid": "uuid-6004"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errorCode": 1000, "message": "Internal Server Error"}`))
		}
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)

	ft := &fakeT{}
	callHierarchy(ft, c, 1000)
	runCleanups(ft)
	if len(ft.errs) == 0 {
		t.Error("expected Errorf when the first delete itself fails, got none")
	}
}

func TestNewDisposableHierarchy_ChildCreateFails_Fatalf(t *testing.T) {
	deleted = map[string]bool{}
	childCreateAttempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			if r.URL.Path == "/api/system/groups/1000" {
				_, _ = w.Write([]byte(`{"Id": 1000, "Uuid": "parent-uuid"}`))
			} else {
				_, _ = w.Write([]byte(`{"children": []}`))
			}
		case http.MethodPost:
			if r.URL.Path == "/api/system/groups/1000" {
				// root create succeeds
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"Id": 7000, "Uuid": "uuid-7000"}`))
				return
			}
			// any child create fails
			childCreateAttempts++
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errorCode": 1000, "message": "Internal Server Error"}`))
		case http.MethodDelete:
			if deleted[r.URL.Path] {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			deleted[r.URL.Path] = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)

	ft := &fakeT{}
	h := callHierarchy(ft, c, 1000, WithChildren(1))
	if !ft.fataled {
		t.Fatal("expected Fatalf on child create failure, got none")
	}
	if childCreateAttempts == 0 {
		t.Error("expected at least one child create attempt")
	}
	if h != nil {
		t.Errorf("expected NewDisposableHierarchy to return nil after Fatalf-triggered Goexit, got %+v", h)
	}

	// Root's own delete cleanup must still have been registered (before the
	// child create was attempted) even though the call never returned.
	if len(ft.cleanups) == 0 {
		t.Fatal("expected root cleanup to be registered before the child create failure")
	}
	runCleanups(ft)
	if !deleted["/api/system/groups/uuid-7000"] {
		t.Error("expected root cleanup to still run after child create failure")
	}
}

func TestRegisterDeleteWithProof_StatusMatch_NoErrorf(t *testing.T) {
	ft := &fakeT{}
	RegisterDeleteWithProof(ft, "desc",
		func() error { return nil },
		func() (int, error) { return 400, fmt.Errorf("not found") },
		400,
	)
	runCleanups(ft)
	if len(ft.errs) != 0 {
		t.Errorf("unexpected Errorf: %v", ft.errs)
	}
}

func TestRegisterDeleteWithProof_StatusMismatch_Errorf(t *testing.T) {
	ft := &fakeT{}
	RegisterDeleteWithProof(ft, "desc",
		func() error { return nil },
		func() (int, error) { return 404, fmt.Errorf("not found") },
		400,
	)
	runCleanups(ft)
	if len(ft.errs) == 0 {
		t.Error("expected Errorf on status mismatch (404 != 400)")
	}
}

func TestRegisterDeleteWithProof_SecondCallSucceeds_Errorf(t *testing.T) {
	ft := &fakeT{}
	RegisterDeleteWithProof(ft, "desc",
		func() error { return nil },
		func() (int, error) { return 200, nil },
		400,
	)
	runCleanups(ft)
	if len(ft.errs) == 0 {
		t.Error("expected Errorf when the second delete unexpectedly succeeds")
	}
}

func TestRegisterDeleteWithProof_FirstDeleteErrors_Errorf(t *testing.T) {
	ft := &fakeT{}
	RegisterDeleteWithProof(ft, "desc",
		func() error { return fmt.Errorf("boom") },
		func() (int, error) { return 400, fmt.Errorf("not found") },
		400,
	)
	runCleanups(ft)
	if len(ft.errs) == 0 {
		t.Error("expected Errorf when the first delete itself errors")
	}
}

func TestSnapshotOrgGroup_DiffEqual(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		switch r.URL.Path {
		case "/api/system/groups/42/children":
			_, _ = w.Write([]byte(`[{"Id": {"Value": 2}}, {"Id": {"Value": 1}}]`))
		default:
			_, _ = w.Write([]byte(`{"Id": {"Value": 42}, "Uuid": "u42", "Name": "n", "GroupId": "g", "LocationGroupType": "Container"}`))
		}
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)

	a, err := SnapshotOrgGroup(context.Background(), c, 42)
	if err != nil {
		t.Fatalf("SnapshotOrgGroup: %v", err)
	}
	b, err := SnapshotOrgGroup(context.Background(), c, 42)
	if err != nil {
		t.Fatalf("SnapshotOrgGroup: %v", err)
	}
	if diff := a.Diff(b); diff != "" {
		t.Errorf("expected no diff between identical snapshots, got: %s", diff)
	}
	if len(a.SubtreeIDs) != 2 || a.SubtreeIDs[0] != 1 || a.SubtreeIDs[1] != 2 {
		t.Errorf("SubtreeIDs = %v, want sorted [1 2]", a.SubtreeIDs)
	}
}

func TestSnapshotOrgGroup_DiffDetectsChange(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/api/system/groups/42/children" {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		callCount++
		name := "before"
		if callCount > 1 {
			name = "after"
		}
		_, _ = fmt.Fprintf(w, `{"Id": {"Value": 42}, "Uuid": "u42", "Name": %q, "GroupId": "g", "LocationGroupType": "Container"}`, name)
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, srv.URL)

	before, err := SnapshotOrgGroup(context.Background(), c, 42)
	if err != nil {
		t.Fatalf("SnapshotOrgGroup: %v", err)
	}
	after, err := SnapshotOrgGroup(context.Background(), c, 42)
	if err != nil {
		t.Fatalf("SnapshotOrgGroup: %v", err)
	}
	diff := before.Diff(after)
	if diff == "" {
		t.Error("expected a diff between before/after snapshots with different Name, got none")
	}
}
