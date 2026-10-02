package tests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/v26/client"
	"github.com/euc-oss/terraform-sdk-uem/v26/models"
	"github.com/euc-oss/terraform-sdk-uem/v26/resources"
)

// recordRequest serves a tiny JSON body and records the last request's
// method, path and Accept header.
func recordRequest(t *testing.T, body string) (*client.Client, func() (string, string, string)) {
	t.Helper()
	var method, path, accept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, accept = r.Method, r.URL.Path, r.Header.Get("Accept")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	c, err := client.NewClient(&client.Config{InstanceURL: srv.URL, TenantCode: "t", AuthMethod: "basic", Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	return c, func() (string, string, string) { return method, path, accept }
}

const (
	acceptV2 = "application/json;version=2"
	acceptV4 = "application/json;version=4"
)

// TestResourcesProfileUpdate_VerbPerPlatform pins the HTTP verb, path and
// Accept header the hand-written resources.ProfileService.Update sends for
// every platform: only Windows 10 (WinRT) uses PUT; all others, Windows
// Rugged (QNX) included, use POST (api-doctrine quirk 4). Linux is written
// through API v4, every other platform through v2 (quirk 3). Expected verbs
// are canonical (release/26.2.0.0, canonical report on profile update verbs,
// 2026-09-24): the [HttpPost]/[HttpPut] attribute on each platform's update
// action in AirWatch API/AW.Mdm.Api/AW.Mdm.Api/Controllers/Profiles/V2/ (and
// ProfilesV4Controller for Linux).
func TestResourcesProfileUpdate_VerbPerPlatform(t *testing.T) {
	cases := []struct {
		platform, wantMethod, wantPath, wantAccept string
	}{
		// UpdateAndroidDeviceProfileAsync, ProfilesV2Controller.cs:626-628
		{models.PlatformAndroid, "POST", "/api/mdm/profiles/platforms/android/update", acceptV2},
		// UpdateAppleDeviceProfileAsync, ProfilesV2Controller.cs:685-687
		{models.PlatformAppleIOS, "POST", "/api/mdm/profiles/platforms/apple/update", acceptV2},
		// UpdateAppleOsXDeviceProfileAsync, ProfilesV2Controller.cs:741-743
		{models.PlatformAppleOSX, "POST", "/api/mdm/profiles/platforms/appleosx/update", acceptV2},
		// UpdateWindowsDesktopDeviceProfileAsync [HttpPut], ProfilesV2Controller.cs:899-901
		{models.PlatformWindows10, "PUT", "/api/mdm/profiles/platforms/winrt/update", acceptV2},
		// UpdateQnxDeviceProfileAsync [HttpPost], ProfilesV2Controller.cs:842-844
		{models.PlatformWindowsRugged, "POST", "/api/mdm/profiles/platforms/qnx/update", acceptV2},
		// ProfilesV4Controller.UpdateLinuxDeviceProfile [HttpPost], ProfilesV4Controller.Generated.cs:97-98
		{models.PlatformLinux, "POST", "/api/mdm/profiles/platforms/linux/update", acceptV4},
	}
	for _, tc := range cases {
		t.Run(tc.platform, func(t *testing.T) {
			c, last := recordRequest(t, `{}`)
			if _, err := resources.NewProfileService(c).Update(context.Background(), tc.platform, 4242, &models.ProfileUpdateRequest{}); err != nil {
				t.Fatalf("Update: %v", err)
			}
			method, path, accept := last()
			if method != tc.wantMethod || path != tc.wantPath || accept != tc.wantAccept {
				t.Errorf("%s: got %s %s Accept=%q, want %s %s Accept=%q", tc.platform, method, path, accept, tc.wantMethod, tc.wantPath, tc.wantAccept)
			}
		})
	}
}

// TestResourcesProfileCreate_AcceptPerPlatform pins the verb, path and
// Accept header of the hand-written resources.ProfileService.Create for
// every platform that has a create route: Linux through API v4, the rest
// through v2 (api-doctrine quirk 3). Windows Rugged (QNX) has no create
// route at all (quirk 4); Create refuses it without a request, pinned by
// TestResourcesProfileCreate_WindowsRuggedRefusedWithoutRequest.
func TestResourcesProfileCreate_AcceptPerPlatform(t *testing.T) {
	cases := []struct {
		platform, wantPath, wantAccept string
	}{
		{models.PlatformAndroid, "/api/mdm/profiles/platforms/android/create", acceptV2},
		{models.PlatformAppleIOS, "/api/mdm/profiles/platforms/apple/create", acceptV2},
		{models.PlatformAppleOSX, "/api/mdm/profiles/platforms/appleosx/create", acceptV2},
		{models.PlatformWindows10, "/api/mdm/profiles/platforms/winrt/create", acceptV2},
		{models.PlatformLinux, "/api/mdm/profiles/platforms/linux/create", acceptV4},
	}
	for _, tc := range cases {
		t.Run(tc.platform, func(t *testing.T) {
			c, last := recordRequest(t, `4242`)
			if _, err := resources.NewProfileService(c).Create(context.Background(), tc.platform, &models.ProfileCreateRequest{}); err != nil {
				t.Fatalf("Create: %v", err)
			}
			method, path, accept := last()
			if method != "POST" || path != tc.wantPath || accept != tc.wantAccept {
				t.Errorf("%s: got %s %s Accept=%q, want POST %s Accept=%q", tc.platform, method, path, accept, tc.wantPath, tc.wantAccept)
			}
		})
	}
}

// TestResourcesProfileCreate_WindowsRuggedRefusedWithoutRequest pins that
// Create refuses Windows Rugged (QNX), which has no create route on the
// server (api-doctrine quirk 4), with a typed *client.UnsupportedOperationError
// and without sending any request.
func TestResourcesProfileCreate_WindowsRuggedRefusedWithoutRequest(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`4242`))
	}))
	t.Cleanup(srv.Close)
	c, err := client.NewClient(&client.Config{InstanceURL: srv.URL, TenantCode: "t", AuthMethod: "basic", Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}

	got, err := resources.NewProfileService(c).Create(context.Background(), models.PlatformWindowsRugged, &models.ProfileCreateRequest{})
	var opErr *client.UnsupportedOperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("want *client.UnsupportedOperationError, got %T %v (profile %+v)", err, err, got)
	}
	if opErr.Operation != "create" || opErr.Platform != models.PlatformWindowsRugged {
		t.Errorf("error fields: got operation=%q platform=%q", opErr.Operation, opErr.Platform)
	}
	if !strings.Contains(err.Error(), "Windows Rugged (QNX) has no profile create endpoint (api-doctrine quirk 4)") {
		t.Errorf("error message %q does not explain the missing create endpoint", err.Error())
	}
	if n := atomic.LoadInt32(&hits); n != 0 {
		t.Errorf("Create(Windows_Rugged) sent %d request(s), want 0", n)
	}
}
