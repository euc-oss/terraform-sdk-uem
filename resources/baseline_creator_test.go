package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/v26/client"
	mdmv1 "github.com/euc-oss/terraform-sdk-uem/v26/internal/mdm/v1"
)

// TestBaselineCreator_Create_NoCustomFile verifies that BaselineCreator
// emits a multipart body with the "baseline" JSON part and no file part
// when customFile is nil, and that the Content-Type header carries a
// multipart boundary.
func TestBaselineCreator_Create_NoCustomFile(t *testing.T) {
	var capturedContentType string
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		capturedBody = body

		w.Header().Set("Content-Type", "application/json;version=1")
		w.WriteHeader(http.StatusOK)
		// Echo a minimal BaselineV1Model back so the helper's response
		// unmarshal succeeds.
		_, _ = w.Write([]byte(`{"baselineUUID":"7ac1b8d7-010b-b6cd-3a3e-84e7f90136b8","name":"Test baseline"}`))
	}))
	t.Cleanup(server.Close)

	c, err := client.NewClient(&client.Config{
		InstanceURL: server.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	})
	if err != nil {
		t.Fatalf("client.NewClient: %v", err)
	}

	creator := NewBaselineCreator(c)
	body := &mdmv1.CreateBaselineRequestV1Model{
		Name:        "Test baseline",
		Description: "Created in unit test",
	}

	_, result, err := creator.Create(context.Background(), "16fea16d-024d-1e8b-e41d-b5981759f00d", body, nil)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if result == nil || result.Name != "Test baseline" {
		t.Fatalf("unexpected response: %+v", result)
	}

	mediaType, params, err := mime.ParseMediaType(capturedContentType)
	if err != nil {
		t.Fatalf("parse Content-Type %q: %v", capturedContentType, err)
	}
	if mediaType != "multipart/form-data" {
		t.Errorf("Content-Type media-type: got %q, want multipart/form-data", mediaType)
	}
	if params["boundary"] == "" {
		t.Errorf("Content-Type missing boundary parameter; got %q", capturedContentType)
	}

	parts := parseMultipartParts(t, capturedBody, params["boundary"])
	if len(parts) != 1 {
		t.Fatalf("expected exactly 1 multipart field (baseline) with no customFile, got %d", len(parts))
	}
	baselinePart, ok := parts["baseline"]
	if !ok {
		t.Fatal("expected a 'baseline' form field, not found")
	}
	var roundTrip mdmv1.CreateBaselineRequestV1Model
	if err := json.Unmarshal(baselinePart, &roundTrip); err != nil {
		t.Fatalf("unmarshal baseline JSON part: %v", err)
	}
	if roundTrip.Name != "Test baseline" || roundTrip.Description != "Created in unit test" {
		t.Errorf("baseline JSON round-trip mismatch: %+v", roundTrip)
	}
}

// TestBaselineCreator_Create_WithCustomFile verifies that the customFile
// form part is added with the correct content and form-file framing.
func TestBaselineCreator_Create_WithCustomFile(t *testing.T) {
	var capturedContentType string
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		capturedBody = body
		w.Header().Set("Content-Type", "application/json;version=1")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"baselineUUID":"90d5b0a2-6955-772e-bda6-1cd86cef90ec","name":"With file"}`))
	}))
	t.Cleanup(server.Close)

	c, err := client.NewClient(&client.Config{
		InstanceURL: server.URL,
		TenantCode:  "test-tenant",
		AuthMethod:  "basic",
		Username:    "test-user",
		Password:    "test-pass",
	})
	if err != nil {
		t.Fatalf("client.NewClient: %v", err)
	}

	creator := NewBaselineCreator(c)
	body := &mdmv1.CreateBaselineRequestV1Model{Name: "With file"}
	fileContent := []byte("PK\x03\x04 fake zip contents")

	_, _, err = creator.Create(
		context.Background(),
		"16fea16d-024d-1e8b-e41d-b5981759f00d",
		body,
		bytes.NewReader(fileContent),
	)
	if err != nil {
		t.Fatalf("Create with file failed: %v", err)
	}

	_, params, err := mime.ParseMediaType(capturedContentType)
	if err != nil {
		t.Fatalf("parse Content-Type: %v", err)
	}
	parts := parseMultipartParts(t, capturedBody, params["boundary"])
	if _, ok := parts["baseline"]; !ok {
		t.Error("expected 'baseline' form field")
	}
	got, ok := parts["customFile"]
	if !ok {
		t.Fatal("expected 'customFile' form field, not found")
	}
	if !bytes.Equal(got, fileContent) {
		t.Errorf("customFile content mismatch:\n  got:  %q\n  want: %q", got, fileContent)
	}
}

// parseMultipartParts reads the captured body as a multipart envelope
// and returns the raw bytes of each form field, keyed by field name.
// File fields are returned by their form-name, not filename.
func parseMultipartParts(t *testing.T, body []byte, boundary string) map[string][]byte {
	t.Helper()
	if boundary == "" {
		t.Fatal("empty boundary; cannot parse multipart")
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	out := map[string][]byte{}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("multipart NextPart: %v", err)
		}
		name := part.FormName()
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, part); err != nil {
			t.Fatalf("read part %q: %v", name, err)
		}
		out[name] = buf.Bytes()
	}
	return out
}
