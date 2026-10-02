package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/v26/client"
	"github.com/euc-oss/terraform-sdk-uem/v26/resources"
)

// TestResourcesProfileDelete_SendsVersion1 pins the Accept header of the
// hand-written resources.ProfileService.Delete. DELETE by int id is
// registered only at API v1 (api-doctrine quirk 1), so a v2 header would
// only work through the server's above-the-highest-version fallback.
func TestResourcesProfileDelete_SendsVersion1(t *testing.T) {
	var gotMethod, gotPath, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotAccept = r.Method, r.URL.Path, r.Header.Get("Accept")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	c, err := client.NewClient(&client.Config{InstanceURL: srv.URL, TenantCode: "t", AuthMethod: "basic", Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if err := resources.NewProfileService(c).Delete(context.Background(), 4242); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if gotMethod != "DELETE" || gotPath != "/api/mdm/profiles/4242" || gotAccept != "application/json;version=1" {
		t.Errorf("got %s %s Accept=%q, want DELETE /api/mdm/profiles/4242 Accept=application/json;version=1", gotMethod, gotPath, gotAccept)
	}
}
