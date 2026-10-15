package tests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/client"
)

// TestLayer2ProfileUpdateUnknownIDIsNotFound: an Update for an id that
// discovery cannot find returns *client.NotFoundError, as Get does, so
// callers can branch on client.IsNotFound for both.
func TestLayer2ProfileUpdateUnknownIDIsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent) // empty search: nothing to discover
	}))
	defer srv.Close()
	c, err := client.NewClient(&client.Config{InstanceURL: srv.URL, TenantCode: "t", AuthMethod: "basic", Username: "u", Password: "p"})
	if err != nil {
		t.Fatal(err)
	}
	svc, err := sdk.NewProfileService(context.Background(), c)
	if err != nil {
		t.Fatalf("NewProfileService: %v", err)
	}

	err = svc.Update(context.Background(), 999, &sdk.AndroidDeviceProfileV2Entity{})
	var nf *client.NotFoundError
	if !errors.As(err, &nf) || !client.IsNotFound(err) {
		t.Fatalf("Update(unknown id): want *client.NotFoundError, got %T %v", err, err)
	}
	if nf.Resource != "profiles" || nf.ID != "999" {
		t.Errorf("NotFoundError = {Resource: %q, ID: %q}, want {profiles, 999}", nf.Resource, nf.ID)
	}

	// Get reports the same kind of error for the same id.
	_, getErr := svc.Get(context.Background(), 999)
	var getNF *client.NotFoundError
	if !errors.As(getErr, &getNF) || getNF.Resource != nf.Resource || getNF.ID != nf.ID {
		t.Errorf("Get(unknown id) = %T %v, want the same *client.NotFoundError as Update", getErr, getErr)
	}
}
