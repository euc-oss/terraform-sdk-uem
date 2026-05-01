package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	"github.com/euc-oss/terraform-sdk-uem/client"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestGeneratedUploadCertificateV1 verifies UploadCertificate produces the
// correct HTTP request against a fixture-backed mock server.
// Fixture: testdata/mock-responses/profiles/upload_certificate_success.json
func TestGeneratedUploadCertificateV1(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewProfilesV1Service(c)

	ctx := context.Background()
	req := &sdk.CertificateV1{
		CertificatePayload: "TUlJS0dnSUJBekNDQ2RZR0NT", // base64 PFX stub
		Password:           "hunter2",
	}
	_, resp, err := svc.UploadCertificate(ctx, req)
	if err != nil {
		t.Fatalf("UploadCertificate failed: %v", err)
	}
	if resp == nil {
		t.Fatal("UploadCertificate returned nil response")
	}
	if resp.Value == nil || *resp.Value != 12345 {
		t.Errorf("response Value: got %v, want *int64 = 12345", resp.Value)
	}
}

// TestUploadCertificateV1RequestShape captures the raw HTTP request issued by
// the generated UploadCertificate method and asserts method, path, Accept
// header (no trailing semicolon), Content-Type, and request body shape.
func TestUploadCertificateV1RequestShape(t *testing.T) {
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
		w.Header().Set("Content-Type", "application/json;version=1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"Value":67890}`))
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

	svc := sdk.NewProfilesV1Service(c)
	req := &sdk.CertificateV1{
		CertificatePayload: "cGF5bG9hZA==",
		Password:           "secret",
	}
	_, resp, err := svc.UploadCertificate(context.Background(), req)
	if err != nil {
		t.Fatalf("UploadCertificate failed: %v", err)
	}
	if resp == nil || resp.Value == nil || *resp.Value != 67890 {
		t.Errorf("response Value: got %+v, want EntityIdV1{Value: *int64 = 67890}", resp)
	}

	if captured.method != http.MethodPost {
		t.Errorf("method: got %q, want POST", captured.method)
	}
	if captured.path != "/api/mdm/profiles/uploadcertificate" {
		t.Errorf("path: got %q, want /api/mdm/profiles/uploadcertificate", captured.path)
	}
	if captured.accept != "application/json;version=1" {
		t.Errorf("Accept: got %q, want application/json;version=1 (no trailing semicolon)", captured.accept)
	}
	if !bytes.Contains([]byte(captured.contentType), []byte("application/json")) {
		t.Errorf("Content-Type: got %q, want application/json", captured.contentType)
	}

	var decoded map[string]any
	if err := json.Unmarshal(captured.body, &decoded); err != nil {
		t.Fatalf("request body is not JSON: %v (body=%s)", err, captured.body)
	}
	if decoded["CertificatePayload"] != "cGF5bG9hZA==" {
		t.Errorf("CertificatePayload: got %v, want cGF5bG9hZA==", decoded["CertificatePayload"])
	}
	if decoded["Password"] != "secret" {
		t.Errorf("Password: got %v, want secret", decoded["Password"])
	}
}
