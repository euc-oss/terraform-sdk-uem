package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/client"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestMamV2InternalAppGetByUuid verifies the generated InternalAppsV2.GetInternalAppByUuid
// against a fixture-backed mock server.
// Fixture: testdata/mock-responses/mam-apps/apps_get_internal.json
func TestMamV2InternalAppGetByUuid(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewInternalAppsV2Service(c)

	appUUID := "f3be5b88-fb76-176d-845b-0be80ea3f3ad"
	_, resp, err := svc.GetInternalAppByUuid(context.Background(), appUUID)
	if err != nil {
		t.Fatalf("GetInternalAppByUuid failed: %v", err)
	}
	if resp == nil {
		t.Fatal("GetInternalAppByUuid returned nil response")
	}
	if resp.ApplicationName != "Test Lovesmell" {
		t.Errorf("ApplicationName: got %q, want Test Lovesmell", resp.ApplicationName)
	}
	if resp.Platform != "WinRT" {
		t.Errorf("Platform: got %q, want WinRT", resp.Platform)
	}
}

// TestMamV2BlobsUploadBlobRequestShape captures the raw HTTP request issued by
// the generated BlobsV2.UploadBlobAsync method (raw-capture style — no fixture).
func TestMamV2BlobsUploadBlobRequestShape(t *testing.T) {
	var captured struct {
		method      string
		path        string
		rawQuery    string
		accept      string
		contentType string
		body        []byte
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured.method = r.Method
		captured.path = r.URL.Path
		captured.rawQuery = r.URL.RawQuery
		captured.accept = r.Header.Get("Accept")
		captured.contentType = r.Header.Get("Content-Type")
		captured.body = body
		w.Header().Set("Content-Type", "application/json;version=2")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Uuid":"7ac1b8d7-010b-b6cd-3a3e-84e7f90136b8","Value":42}`))
	}))
	t.Cleanup(srv.Close)

	c, err := client.NewClient(&client.Config{
		InstanceURL: srv.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	svc := sdk.NewBlobsV2Service(c)
	payload := []byte("binary-blob-payload")
	_, _, err = svc.UploadBlobAsync(
		context.Background(),
		payload,
		&sdk.BlobsV2UploadBlobAsyncOptions{
			FileName:            "installer.msi",
			OrganizationGroupID: 570,
		},
	)
	if err != nil {
		t.Fatalf("UploadBlobAsync failed: %v", err)
	}

	if captured.method != http.MethodPost {
		t.Errorf("method: got %q, want POST", captured.method)
	}
	if captured.path != "/api/mam/blobs/uploadblob" {
		t.Errorf("path: got %q, want /api/mam/blobs/uploadblob", captured.path)
	}
	if captured.accept != "application/json;version=2" {
		t.Errorf("Accept: got %q, want application/json;version=2 (no trailing semicolon)", captured.accept)
	}
	if !bytes.Contains([]byte(captured.contentType), []byte("application/octet-stream")) {
		t.Errorf("Content-Type: got %q, want application/octet-stream", captured.contentType)
	}
	if !bytes.Equal(captured.body, payload) {
		t.Errorf("body: got %q, want %q", captured.body, payload)
	}
	// Required query params must be present.
	if !bytes.Contains([]byte(captured.rawQuery), []byte("fileName=installer.msi")) {
		t.Errorf("query missing fileName: got %q", captured.rawQuery)
	}
	if !bytes.Contains([]byte(captured.rawQuery), []byte("organizationGroupId=570")) {
		t.Errorf("query missing organizationGroupId: got %q", captured.rawQuery)
	}
}

// TestMamV2AssignmentRulesPostRequestShape asserts the generated
// AppsV2.GetListOfDevices request shape (POST /apps/{applicationUuid}/assignment-rules).
// This is one of the gap-fill routes added via internal-source/mamv2.apps.overlay.yaml.
func TestMamV2AssignmentRulesPostRequestShape(t *testing.T) {
	var captured struct {
		method      string
		path        string
		rawQuery    string
		accept      string
		contentType string
		body        []byte
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured.method = r.Method
		captured.path = r.URL.Path
		captured.rawQuery = r.URL.RawQuery
		captured.accept = r.Header.Get("Accept")
		captured.contentType = r.Header.Get("Content-Type")
		captured.body = body
		w.Header().Set("Content-Type", "application/json;version=2")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"devices":[]}`))
	}))
	t.Cleanup(srv.Close)

	c, err := client.NewClient(&client.Config{
		InstanceURL: srv.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	appUUID := "f3be5b88-fb76-176d-845b-0be80ea3f3ad"
	svc := sdk.NewAppsV2Service(c)
	req := &sdk.AppAssignmentRuleV2Model{
		UUID: appUUID,
	}
	_, _, err = svc.GetListOfDevices(
		context.Background(),
		appUUID,
		req,
		&sdk.AppsV2GetListOfDevicesOptions{
			Action: "PREVIEW_DEVICES",
		},
	)
	if err != nil {
		t.Fatalf("GetListOfDevices failed: %v", err)
	}

	if captured.method != http.MethodPost {
		t.Errorf("method: got %q, want POST", captured.method)
	}
	wantPath := "/api/mam/apps/" + appUUID + "/assignment-rules"
	if captured.path != wantPath {
		t.Errorf("path: got %q, want %q", captured.path, wantPath)
	}
	if captured.accept != "application/json;version=2" {
		t.Errorf("Accept: got %q, want application/json;version=2 (no trailing semicolon)", captured.accept)
	}
	if !bytes.Contains([]byte(captured.contentType), []byte("application/json")) {
		t.Errorf("Content-Type: got %q, want application/json", captured.contentType)
	}
	if !bytes.Contains([]byte(captured.rawQuery), []byte("action=PREVIEW_DEVICES")) {
		t.Errorf("query missing action=PREVIEW_DEVICES: got %q", captured.rawQuery)
	}
	var decoded map[string]any
	if err := json.Unmarshal(captured.body, &decoded); err != nil {
		t.Fatalf("request body is not JSON: %v (body=%s)", err, captured.body)
	}
	if decoded["uuid"] != appUUID {
		t.Errorf("body uuid: got %v, want %s", decoded["uuid"], appUUID)
	}
}

// TestMamV2PurchasedInstallRequestShape asserts the generated
// PurchasedAppsV2.InstallVppAppForDeviceAsync request shape
// (POST /apps/purchased/{applicationUuid}/install).
func TestMamV2PurchasedInstallRequestShape(t *testing.T) {
	var captured struct {
		method      string
		path        string
		accept      string
		contentType string
		body        []byte
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		captured.method = r.Method
		captured.path = r.URL.Path
		captured.accept = r.Header.Get("Accept")
		captured.contentType = r.Header.Get("Content-Type")
		captured.body = body
		w.Header().Set("Content-Type", "application/json;version=2")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c, err := client.NewClient(&client.Config{
		InstanceURL: srv.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	})
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	appUUID := "9af645a8-fef3-3e6d-3408-5cc69e0937d4"
	svc := sdk.NewPurchasedAppsV2Service(c)
	req := &sdk.DeviceInformationV2Model{
		DeviceUUID: "666ff6cc-aa5b-3c07-feaa-3a95d3a4bd2c",
	}
	_, err = svc.InstallVppAppForDeviceAsync(context.Background(), appUUID, req)
	if err != nil {
		t.Fatalf("InstallVppAppForDeviceAsync failed: %v", err)
	}

	if captured.method != http.MethodPost {
		t.Errorf("method: got %q, want POST", captured.method)
	}
	wantPath := "/api/mam/apps/purchased/" + appUUID + "/install"
	if captured.path != wantPath {
		t.Errorf("path: got %q, want %q", captured.path, wantPath)
	}
	if captured.accept != "application/json;version=2" {
		t.Errorf("Accept: got %q, want application/json;version=2 (no trailing semicolon)", captured.accept)
	}
	if !bytes.Contains([]byte(captured.contentType), []byte("application/json")) {
		t.Errorf("Content-Type: got %q, want application/json", captured.contentType)
	}
	var decoded map[string]any
	if err := json.Unmarshal(captured.body, &decoded); err != nil {
		t.Fatalf("request body is not JSON: %v (body=%s)", err, captured.body)
	}
	if decoded["device_uuid"] != "666ff6cc-aa5b-3c07-feaa-3a95d3a4bd2c" {
		t.Errorf("body device_uuid: got %v, want 666ff6cc-aa5b-3c07-feaa-3a95d3a4bd2c", decoded["device_uuid"])
	}
}

// TestMamV1InternalAppRenewalDateNoTimezone regresses internal-ticket:
// InternalAppsV1_GetInternalAppByIdAsync used to fail with
//
//	parsing time "2026-04-27T16:00:00.000" as "2006-01-02T15:04:05Z07:00":
//	cannot parse "" as "Z07:00"
//
// because the UEM API returns RenewalDate without a timezone suffix and
// Go's default time.Time JSON unmarshaler requires RFC3339. The fix wraps
// every generated date-time field in client.UEMTime, which accepts both
// timezone-bearing and bare layouts.
func TestMamV1InternalAppRenewalDateNoTimezone(t *testing.T) {
	body := []byte(`{
		"ApplicationName": "Test App",
		"RenewalDate": "2026-04-27T16:00:00.000"
	}`)

	var resp sdk.InternalAppModelV1
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("UEM-format RenewalDate must unmarshal cleanly, got: %v", err)
	}
	if resp.RenewalDate.IsZero() {
		t.Fatal("RenewalDate parsed to zero time")
	}
	if resp.RenewalDate.Year() != 2026 {
		t.Errorf("RenewalDate year: got %d want 2026", resp.RenewalDate.Year())
	}
}

// TestMamV1InternalAppGetByIdRenewalDateFixture exercises the full SDK path —
// generated V1 service → mock server → fixture — and asserts that an iOS
// internal app whose RenewalDate uses UEM's no-timezone format (2024-01-15
// T00:00:00.000) parses without error. Companion to the inline-JSON regression
// above; this one proves the fix at the wire level via the InternalAppsV1
// service.
//
// Fixture: testdata/mock-responses/mam-apps/apps_get_internal_ios_renewaldate.json
func TestMamV1InternalAppGetByIdRenewalDateFixture(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewInternalAppsV1Service(c)

	_, resp, err := svc.GetInternalAppByIdAsync(context.Background(), 12345)
	if err != nil {
		t.Fatalf("GetInternalAppByIdAsync failed: %v", err)
	}
	if resp == nil {
		t.Fatal("GetInternalAppByIdAsync returned nil response")
	}
	if resp.RenewalDate.IsZero() {
		t.Fatal("RenewalDate parsed to zero — UEM no-timezone format was not handled")
	}
	if resp.RenewalDate.Year() != 2024 {
		t.Errorf("RenewalDate year: got %d want 2024", resp.RenewalDate.Year())
	}
	if len(resp.Assignments) > 0 && resp.Assignments[0].EffectiveDate.IsZero() {
		t.Error("Assignments[0].EffectiveDate parsed to zero — same UEM format")
	}
}

// TestMamV2AppsSearchFixtureUnmarshal is a regression for internal-task.
// It verifies that the apps_get_search.json fixture (real wire format with
// string platform values like "WIN_RT", "APPLE") unmarshals cleanly into the
// generated ApplicationSearchV2Model, which now declares Platform as string.
func TestMamV2AppsSearchFixtureUnmarshal(t *testing.T) {
	// Load the fixture captured from a live Workspace ONE instance.
	raw, err := os.ReadFile("../testdata/mock-responses/mam-apps/apps_get_search.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	// The fixture is a mock-capture envelope; unwrap to get the response body.
	var envelope struct {
		Response struct {
			Body json.RawMessage `json:"body"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	var result sdk.ApplicationSearchV2Model
	if err := json.Unmarshal(envelope.Response.Body, &result); err != nil {
		t.Fatalf("unmarshal ApplicationSearchV2Model: %v — platform field may still be integer type", err)
	}

	if len(result.Applications) == 0 {
		t.Fatal("fixture contained zero applications — check fixture path")
	}

	// Every application in the fixture must have a non-empty string Platform.
	for i, app := range result.Applications {
		if app.Platform == "" {
			t.Errorf("applications[%d]: Platform is empty string (want a string enum like WIN_RT, APPLE, etc.)", i)
		}
	}
}
