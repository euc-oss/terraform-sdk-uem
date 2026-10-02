package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/euc-oss/terraform-sdk-uem/client"
	mdmv1 "github.com/euc-oss/terraform-sdk-uem/internal/mdm/v1"
	"github.com/euc-oss/terraform-sdk-uem/internal/orgtest"
	"github.com/euc-oss/terraform-sdk-uem/models"
	"github.com/euc-oss/terraform-sdk-uem/resources"
)

// recordingTransport keeps the status and X-Api-Version of the last
// response to a request whose path matches pathRE.
type recordingTransport struct {
	next   http.RoundTripper
	pathRE *regexp.Regexp

	mu         sync.Mutex
	status     int
	apiVersion string
}

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.next.RoundTrip(req)
	if err == nil && r.pathRE.MatchString(req.URL.Path) {
		r.mu.Lock()
		r.status, r.apiVersion = resp.StatusCode, resp.Header.Get("X-Api-Version")
		r.mu.Unlock()
	}
	return resp, err
}

func (r *recordingTransport) last() (int, string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status, r.apiVersion
}

// liveBasicClientWithTransport builds the same basic-auth live client as
// getLiveClientWithTimeout, with a caller-supplied inner transport.
func liveBasicClientWithTransport(t *testing.T, rt http.RoundTripper) *client.Client {
	t.Helper()
	for _, k := range []string{"INSTANCE_URL", "TENANT_CODE", "USERNAME", "PASSWORD"} {
		if os.Getenv(k) == "" {
			t.Skipf("%s not set, skipping live test", k)
		}
	}
	c, err := client.NewClient(&client.Config{
		InstanceURL: os.Getenv("INSTANCE_URL"), TenantCode: os.Getenv("TENANT_CODE"),
		AuthMethod: "basic", Username: os.Getenv("USERNAME"), Password: os.Getenv("PASSWORD"),
		MaxRetries: 3, RateLimit: 1000, Timeout: 5 * time.Minute,
		HTTPClient: &http.Client{Transport: rt},
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	return c
}

// snapshotSharedParent snapshots the shared ORG_GROUP_ID parent now and
// fails the test if it differs after every other cleanup has run.
func snapshotSharedParent(t *testing.T, c *client.Client, parent int) {
	t.Helper()
	before, err := orgtest.SnapshotOrgGroup(context.Background(), c, parent)
	if err != nil {
		t.Fatalf("snapshot parent %d before: %v", parent, err)
	}
	t.Cleanup(func() {
		after, err := orgtest.SnapshotOrgGroup(context.Background(), c, parent)
		if err != nil {
			t.Errorf("snapshot parent %d after: %v", parent, err)
			return
		}
		if diff := before.Diff(after); diff != "" {
			t.Errorf("shared parent %d changed: %s", parent, diff)
		}
	})
}

// TestLive_ResourcesProfileDelete_ServedByV1 creates one AppleOsX and one
// Linux profile in a disposable org group, deletes each through the
// hand-written resources.ProfileService.Delete, and asserts the delete is
// served by API v1 with a 200, then that a second Delete is a 400
// recognized by client.IsNotFound (api-doctrine quirks 1 and 20).
func TestLive_ResourcesProfileDelete_ServedByV1(t *testing.T) {
	skipUnlessLiveMutate(t)
	rec := &recordingTransport{next: http.DefaultTransport, pathRE: regexp.MustCompile(`^/api/mdm/profiles/\d+$`)}
	c := liveBasicClientWithTransport(t, rec)
	ctx := context.Background()
	parent := liveOrgGroupID(t)
	snapshotSharedParent(t, c, parent)
	h := orgtest.NewDisposableHierarchy(t, c, parent)
	profiles := resources.NewProfileService(c)

	cases := []struct {
		name, template, createPath, accept string
	}{
		{"AppleOsX", "../testdata/request-templates/profiles/appleosx/network-wifi/network-wifi-create-v2.json", "/api/mdm/profiles/platforms/appleosx/create", "application/json;version=2"},
		{"Linux", "../testdata/request-templates/profiles/linux/wifi/wifi-create-v4.json", "/api/mdm/profiles/platforms/linux/create", "application/json;version=4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile(tc.template)
			if err != nil {
				t.Fatal(err)
			}
			var body map[string]any
			if err := json.Unmarshal(data, &body); err != nil {
				t.Fatal(err)
			}
			delete(body, "api_validation")
			gen := body["General"].(map[string]any)
			gen["Name"] = fmt.Sprintf("sdkv1del-%s-%d", tc.name, time.Now().UnixNano())
			gen["ManagedLocationGroupID"] = h.Root.ID
			var id int
			if _, err := c.DoRequest(ctx, "POST", tc.createPath, tc.accept, "application/json", body, &id); err != nil {
				t.Fatalf("create %s profile: %v", tc.name, err)
			}
			if err := profiles.Delete(ctx, id); err != nil {
				t.Fatalf("Delete(%d): %v", id, err)
			}
			if status, ver := rec.last(); status != http.StatusOK || ver != "v1" {
				t.Errorf("Delete(%d): status=%d X-Api-Version=%q, want 200 served by v1", id, status, ver)
			}
			err = profiles.Delete(ctx, id)
			var apiErr *client.APIError
			if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest || !client.IsNotFound(err) {
				t.Errorf("second Delete(%d): got %v, want a 400 recognized by IsNotFound (proof the profile is gone)", id, err)
			}
			if status, ver := rec.last(); ver != "v1" {
				t.Errorf("second Delete(%d): status=%d X-Api-Version=%q, want served by v1", id, status, ver)
			}
		})
	}
}

// TestLive_DeletedScriptGet_IsAPIError500 creates a script in a disposable
// org group, deletes it, and asserts the follow-up GET surfaces as an
// *client.APIError 500 with errorCode 1000 after the retry policy gives up,
// and that IsNotFound does NOT treat it as not-found (api-doctrine quirk 39).
func TestLive_DeletedScriptGet_IsAPIError500(t *testing.T) {
	skipUnlessLiveMutate(t)
	c := getLiveClientWithTimeout(t, "basic", 5*time.Minute)
	ctx := context.Background()
	parent := liveOrgGroupID(t)
	snapshotSharedParent(t, c, parent)
	h := orgtest.NewDisposableHierarchy(t, c, parent)
	scripts := mdmv1.NewScriptsV1Service(c)

	allowed := false
	hdr, err := scripts.CreateScriptAsync(ctx, h.Root.UUID, &mdmv1.CreateScriptV1{
		Name: fmt.Sprintf("sdkdelscript-%d", time.Now().UnixNano()), Platform: "APPLE_OSX",
		ScriptType: "BASH", ExecutionContext: "SYSTEM", ScriptData: "echo sdk", AllowedInCatalog: allowed,
	})
	if err != nil {
		t.Fatalf("create script: %v", err)
	}
	loc := hdr.Get("Location")
	uuid := regexp.MustCompile(`[0-9a-fA-F-]{36}$`).FindString(loc)
	if uuid == "" {
		t.Fatalf("no script uuid in Location %q", loc)
	}
	if _, res, err := scripts.ScriptBulkDeleteAsync(ctx, h.Root.UUID, &[]string{uuid}); err != nil {
		t.Fatalf("delete script %s: %v", uuid, err)
	} else if res.ScriptsDeleted == nil || *res.ScriptsDeleted != 1 {
		t.Fatalf("delete script %s: scripts_deleted=%v, want 1", uuid, res.ScriptsDeleted)
	}

	_, _, err = scripts.GetScriptAsync(ctx, uuid)
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("GET deleted script: want *client.APIError, got %T %v", err, err)
	}
	if apiErr.StatusCode != http.StatusInternalServerError || apiErr.ErrorCode != "1000" {
		t.Errorf("GET deleted script: status=%d errorCode=%q, want 500/1000", apiErr.StatusCode, apiErr.ErrorCode)
	}
	if apiErr.Attempts < 2 {
		t.Errorf("GET deleted script: Attempts=%d, want the retry policy to have retried", apiErr.Attempts)
	}
	if client.IsNotFound(err) {
		t.Error("IsNotFound must be false for 500/errorCode 1000 (quirk 39)")
	}
	t.Logf("GET deleted script %s -> HTTP %d errorCode %s after %d attempts", uuid, apiErr.StatusCode, apiErr.ErrorCode, apiErr.Attempts)
}

// TestLive_ResourcesProfileLinux_CreateUpdateV4 creates, updates and
// deletes a Linux profile through the hand-written resources.ProfileService
// in a disposable org group, asserting that create and update are served by
// API v4 with a 200 (api-doctrine quirk 3; the v2 routes of the same name
// reject the Linux body with 422/1012), then proves the delete with a
// second Delete that must return 400.
func TestLive_ResourcesProfileLinux_CreateUpdateV4(t *testing.T) {
	skipUnlessLiveMutate(t)
	rec := &recordingTransport{next: http.DefaultTransport, pathRE: regexp.MustCompile(`^/api/mdm/profiles/platforms/linux/(create|update)$`)}
	c := liveBasicClientWithTransport(t, rec)
	ctx := context.Background()
	parent := liveOrgGroupID(t)
	snapshotSharedParent(t, c, parent)
	h := orgtest.NewDisposableHierarchy(t, c, parent)
	profiles := resources.NewProfileService(c)

	load := func(path string) map[string]any {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var body map[string]any
		if err := json.Unmarshal(data, &body); err != nil {
			t.Fatal(err)
		}
		delete(body, "api_validation")
		return body
	}
	name := fmt.Sprintf("sdklinuxv4-%d", time.Now().UnixNano())
	create := load("../testdata/request-templates/profiles/linux/wifi/wifi-create-v4.json")
	cg := create["General"].(map[string]any)
	cg["Name"], cg["ManagedLocationGroupID"] = name, h.Root.ID
	createReq := models.ProfileCreateRequest(create)
	created, err := profiles.Create(ctx, models.PlatformLinux, &createReq)
	if err != nil {
		t.Fatalf("Create(Linux): %v", err)
	}
	id := created.GetProfileID()
	if status, ver := rec.last(); status != http.StatusOK || ver != "v4" {
		t.Errorf("Create(Linux): status=%d X-Api-Version=%q, want 200 served by v4", status, ver)
	}
	orgtest.RegisterDeleteWithProof(t, fmt.Sprintf("Linux profile %d", id),
		func() error { return profiles.Delete(context.Background(), id) },
		func() (int, error) {
			err := profiles.Delete(context.Background(), id)
			var e *client.APIError
			if errors.As(err, &e) {
				return e.StatusCode, err
			}
			return 0, err
		}, http.StatusBadRequest)

	update := load("../testdata/request-templates/profiles/linux/wifi/wifi-update-v4.json")
	ug := update["General"].(map[string]any)
	ug["ProfileID"], ug["Name"], ug["ManagedLocationGroupID"] = id, name, h.Root.ID
	updateReq := models.ProfileUpdateRequest(update)
	if _, err := profiles.Update(ctx, models.PlatformLinux, id, &updateReq); err != nil {
		t.Fatalf("Update(Linux %d): %v", id, err)
	}
	if status, ver := rec.last(); status != http.StatusOK || ver != "v4" {
		t.Errorf("Update(Linux %d): status=%d X-Api-Version=%q, want 200 served by v4", id, status, ver)
	}
}
