package tests

import (
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem/v26"
	"github.com/euc-oss/terraform-sdk-uem/v26/internal/mockserver"
)

// TestGeneratedBlobsV2Lifecycle verifies the V2 blob upload/get/head/delete
// lifecycle, live-captured against UEM 26.2.0.0.
//
// Real-wire findings baked into these fixtures:
//   - Upload's hardcoded "application/octet-stream" Content-Type is CORRECT
//     (swagger declares it explicitly); the only real issue hit was a 400
//     "Invalid blob type provided" caused by using a disallowed file
//     extension (".txt" is not in the swagger-documented allow-list),
//     resolved by switching to ".xml". See upload_blob_v2.json.
//   - GetAsync (GET /blobs/{blobId}) previously hit a CONFIRMED SDK DEFECT:
//     the real endpoint returns a totally raw, unwrapped
//     octet-stream body (no JSON at all), but the generated method decoded
//     via client.DoRequest into `var response []byte`, and DoRequest
//     unconditionally called json.Unmarshal on the response body regardless
//     of content type -- so Get() could never succeed against real data.
//     Fixed by client.Client.handleResponse now copying raw
//     bytes into a *[]byte target instead of unmarshaling. This test now
//     asserts the FIXED behavior: Get() succeeds and returns the exact raw
//     wire bytes. The mock server always JSON-encodes a fixture's string
//     "body" value (internal/mockserver/server.go), so get_blob_v2.json's
//     "sample blob content" body arrives over the wire as the raw,
//     quote-wrapped bytes `"sample blob content"` -- asserted verbatim
//     below, not the unquoted Go string, since Get() no longer parses JSON
//     at all for this target type.
//   - Head and Delete both work cleanly as generated (no response body to
//     decode in either case).
//   - Post-delete, a follow-up GET/HEAD on the same tenant returned a
//     consistent 500 rather than the canonical doc's documented 403 --
//     see delete_blob_v2.json for the full note.
//
// Fixtures: testdata/mock-responses/blobs/{upload,get,head,delete}_blob_v2.json.
func TestGeneratedBlobsV2Lifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	c := mockserver.NewMockClient(t, ms)
	svc := sdk.NewBlobsV2Service(c)
	ctx := context.Background()

	// 1. Upload blob — POST /api/mam/blobs/uploadblob → 200
	body := []byte("sample blob content")
	_, uploaded, err := svc.UploadBlobAsync(ctx, body, &sdk.BlobsV2UploadBlobAsyncOptions{
		FileName:            "test_upload.xml",
		OrganizationGroupID: 12345,
	})
	if err != nil {
		t.Fatalf("UploadBlobAsync failed: %v", err)
	}
	if uploaded == nil {
		t.Fatal("expected non-nil upload response")
	}
	if uploaded.UUID == "" {
		t.Error("expected non-empty uuid in upload response (this is the id Get/Head/Delete expect)")
	}
	if uploaded.Value == nil {
		t.Error("expected non-nil Value in upload response")
	}

	blobID := "ed043f31-54c4-72fc-e8af-7c1a61cca7aa"

	// 2. Get blob — GET /api/mam/blobs/{blobId} → 200. Fixed:
	// the *[]byte target now receives the raw wire bytes directly, no
	// json.Unmarshal attempted. The mock server JSON-encodes the fixture's
	// string body, so the wire bytes are the quoted form.
	_, got, err := svc.Get(ctx, blobID)
	if err != nil {
		t.Fatalf("Get failed (raw-byte-response regression): %v", err)
	}
	// json.Encoder.Encode (used by the mock server) always appends a trailing
	// newline after the encoded value -- part of the real wire bytes here.
	wantRaw := "\"sample blob content\"\n"
	if string(got) != wantRaw {
		t.Errorf("Get() body = %q, want raw wire bytes %q", got, wantRaw)
	}

	// 3. Head blob — HEAD /api/mam/blobs/{blobId} → 200 (no body to decode,
	// so this works cleanly unlike Get).
	_, err = svc.Head(ctx, blobID)
	if err != nil {
		t.Fatalf("Head failed: %v", err)
	}

	// 4. Delete blob — DELETE /api/mam/blobs/{blobId} → 200 (teardown)
	_, err = svc.Delete(ctx, blobID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}
