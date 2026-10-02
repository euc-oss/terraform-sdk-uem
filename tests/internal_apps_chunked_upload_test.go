package tests

import (
	"bytes"
	"context"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
	mamv1 "github.com/euc-oss/terraform-sdk-uem/internal/mam/v1"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// looksLikeUUIDForChunkedUploadTest reports whether s has the canonical
// 8-4-4-4-12 hex-with-hyphens UUID shape. Used instead of asserting a
// hardcoded literal prefix (e.g. "aaaaaaaa-") against the chunk-handler
// fixtures' Uuid field: the public package's sync scrubber rewrites any
// whole-UUID-shaped literal it finds anywhere in shipped content, including
// inside shipped fixture JSON (not just Go source), so a hardcoded expected
// prefix would break in the assembled public package even though the real
// invariant under test -- the SDK correctly deserializes and surfaces
// whatever Uuid value the fixture carries -- is unaffected by that rewrite.
func looksLikeUUIDForChunkedUploadTest(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range []byte(s) {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
			continue
		}
		isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
		if !isHex {
			return false
		}
	}
	return true
}

// chunkedUploadTestSetup spins up a fresh mock server with the chunk
// handler, an SDK client pointed at it, and a ChunkedUploader wired to
// that client. The mock server's ChunkHandler state is fresh per call.
//
// Each call to this helper returns a new MockServer + ChunkHandler. Tests
// that need session persistence across multiple Upload calls (e.g. resume
// tests) must call this helper ONCE and reuse the returned uploader, since
// the TransactionID minted by the first Upload only stays valid inside the
// ChunkHandler instance that minted it.
func chunkedUploadTestSetup(t *testing.T) (*sdk.ChunkedUploader, func()) {
	t.Helper()
	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	c := mockserver.NewMockClient(t, ms)
	uploader := sdk.NewChunkedUploader(c)
	return uploader, ms.Close
}

func TestChunkedUploadEndToEnd_BeginInstall(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	uploader, cleanup := chunkedUploadTestSetup(t)
	defer cleanup()

	const totalSize = 10 * 1024 // 10 KiB
	body := bytes.Repeat([]byte("Z"), totalSize)

	session, err := uploader.Upload(
		context.Background(),
		bytes.NewReader(body),
		int64(totalSize),
		&sdk.UploadOptions{ChunkSize: 4 * 1024},
	)
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if session.TransactionID == "" {
		t.Fatal("session.TransactionID is empty after Upload")
	}
	if session.LastSequence != 3 {
		t.Errorf("LastSequence: got %d, want 3 (10KiB / 4KiB rounded up)", session.LastSequence)
	}

	app, err := uploader.BeginInstall(
		context.Background(),
		session,
		&mamv1.InternalAppChunkTransactionV1{
			ApplicationName: "Forklift E2E Test",
			BundleID:        "com.example.forklift",
		},
	)
	if err != nil {
		t.Fatalf("BeginInstall returned error: %v", err)
	}
	if app == nil {
		t.Fatal("BeginInstall returned nil application")
		return
	}
	if app.UUID == "" {
		t.Error("Uuid missing; fixture must include server-hydrated value")
	}
	if app.OrganizationGroupUUID == "" {
		t.Error("OrganizationGroupUuid missing; fixture must include server-hydrated value")
	}
}

func TestChunkedUploadEndToEnd_CreateInternalApplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	uploader, cleanup := chunkedUploadTestSetup(t)
	defer cleanup()

	const totalSize = 5 * 1024
	body := bytes.Repeat([]byte("Y"), totalSize)

	session, err := uploader.Upload(
		context.Background(),
		bytes.NewReader(body),
		int64(totalSize),
		&sdk.UploadOptions{ChunkSize: 2 * 1024},
	)
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	app, err := uploader.CreateInternalApplication(
		context.Background(),
		session,
		&mamv1.InternalAppChunkTransactionV1Model{
			ApplicationName:   "Forklift V1Model E2E",
			BundleID:          "com.example.forklift.v1",
			ActualFileVersion: "1.0.0.42",
		},
	)
	if err != nil {
		t.Fatalf("CreateInternalApplication returned error: %v", err)
	}
	if !looksLikeUUIDForChunkedUploadTest(app.UUID) {
		t.Errorf("Uuid: got %q, want a canonical UUID-shaped value sourced from the fixture", app.UUID)
	}
}

// TestChunkedUploadEndToEnd_CreateInternalApplication_PushModeWireInt covers
// the finding that the live /api/mam/apps/internal/application create response
// returns PushMode as a JSON number (Auto=0, OnDemand=1) because canonical's
// InternalApplicationEntity.PushMode has no [JsonConverter(StringEnumConverter)]
// (unlike its ApplicationSource/EARAppUpdateMode siblings on the same class),
// while swagger (and therefore the generated Go field) declared it as a
// string enum. That mismatch broke ChunkedUploader.CreateInternalApplication's
// typed decode on every real successful create.
//
// This drives the request through the apk create fixture (internal_app_
// create_apk.json), which bakes in the real captured "PushMode": 0 body and
// is selected by ChunkHandler.selectInternalApplicationFixture when the
// request's file_name ends in .apk (see chunk_handler.go). Before the
// internal-source/mamv1.internalapps-chunk.overlay.yaml fix + regenerate,
// this fails with a JSON decode error on the typed InternalApplicationEntityV1
// struct; after the fix, PushMode decodes cleanly as the wire integer.
func TestChunkedUploadEndToEnd_CreateInternalApplication_PushModeWireInt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	uploader, cleanup := chunkedUploadTestSetup(t)
	defer cleanup()

	const totalSize = 4 * 1024
	body := bytes.Repeat([]byte("W"), totalSize)

	session, err := uploader.Upload(
		context.Background(),
		bytes.NewReader(body),
		int64(totalSize),
		&sdk.UploadOptions{ChunkSize: 2 * 1024},
	)
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	app, err := uploader.CreateInternalApplication(
		context.Background(),
		session,
		&mamv1.InternalAppChunkTransactionV1Model{
			ApplicationName: "JustSayHello",
			FileName:        "JustSayHello.apk",
		},
	)
	if err != nil {
		t.Fatalf("CreateInternalApplication returned error decoding the real apk create response (PushMode wire-type defect): %v", err)
	}
	if app == nil {
		t.Fatal("CreateInternalApplication returned nil application")
	}
	if !looksLikeUUIDForChunkedUploadTest(app.UUID) {
		t.Errorf("Uuid: got %q, want a canonical UUID-shaped value sourced from the fixture", app.UUID)
	}
	if app.PushMode == nil || *app.PushMode != 0 {
		t.Errorf("PushMode: got %v, want *0 (fixture's wire integer for Auto)", app.PushMode)
	}
}

func TestChunkedUploadEndToEnd_ResumeFromPartialSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	// CRITICAL: this test depends on the chunk handler's session map
	// persisting across both Upload calls below. chunkedUploadTestSetup
	// is called ONCE; both Upload calls share the same uploader and
	// therefore the same MockServer + ChunkHandler state. The
	// TransactionID minted by the first Upload's chunk 1 stays valid for
	// the resumed Upload's chunk 3. If you ever rewrite this test to
	// span multiple chunkedUploadTestSetup calls, the resume contract
	// breaks because each MockServer has its own empty session map.
	uploader, cleanup := chunkedUploadTestSetup(t)
	defer cleanup()

	const totalSize = int64(8 * 1024)
	const chunkSize = int64(2 * 1024)
	body := bytes.Repeat([]byte("R"), int(totalSize))

	// First Upload: stream the first 4 KiB (2 chunks). Simulate a failure
	// by reading only 4 KiB but claiming the full 8 KiB total — the mock
	// happily accepts each chunk; we stop the Upload manually by passing
	// a reader that yields only 4 KiB and accepting the resulting error.
	partial := bytes.NewReader(body[:4*1024])
	priorSession, err := uploader.Upload(
		context.Background(),
		partial,
		totalSize,
		&sdk.UploadOptions{ChunkSize: chunkSize},
	)
	if err == nil {
		t.Fatal("Upload should have errored on partial reader")
	}
	if priorSession == nil || priorSession.LastSequence != 2 {
		t.Fatalf("partial session: got %+v, want LastSequence=2", priorSession)
	}

	// Resume: caller reopens the file seeked to LastSequence*ChunkSize = 4 KiB,
	// supplies the tail (4 KiB), passes the priorSession back via ResumeFrom.
	tail := bytes.NewReader(body[4*1024:])
	finalSession, err := uploader.Upload(
		context.Background(),
		tail,
		totalSize,
		&sdk.UploadOptions{ChunkSize: chunkSize, ResumeFrom: priorSession},
	)
	if err != nil {
		t.Fatalf("resumed Upload returned error: %v", err)
	}
	if finalSession.TransactionID != priorSession.TransactionID {
		t.Errorf("TransactionID changed across resume: prior=%q, final=%q",
			priorSession.TransactionID, finalSession.TransactionID)
	}
	if finalSession.LastSequence != 4 {
		t.Errorf("final LastSequence: got %d, want 4", finalSession.LastSequence)
	}

	// Completion succeeds with the resumed session.
	app, err := uploader.BeginInstall(
		context.Background(),
		finalSession,
		&mamv1.InternalAppChunkTransactionV1{
			ApplicationName: "Forklift Resume Test",
		},
	)
	if err != nil {
		t.Fatalf("BeginInstall after resume failed: %v", err)
	}
	if app == nil {
		t.Fatal("BeginInstall returned nil app after resume")
	}
}
