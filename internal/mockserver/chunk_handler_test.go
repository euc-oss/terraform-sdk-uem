package mockserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func postUploadChunk(t *testing.T, h *ChunkHandler, body string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/mam/apps/internal/uploadchunk",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	if !h.Handle(w, req) {
		t.Fatal("expected ChunkHandler.Handle to return true for /uploadchunk")
	}
	var decoded map[string]interface{}
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("response body not valid JSON: %v (body=%q)", err, w.Body.String())
		}
	}
	return w.Code, decoded
}

func TestChunkHandler_UploadChunk_FirstCallMintsTransactionId(t *testing.T) {
	h := NewChunkHandler()
	body := `{"TransactionId":"","ChunkData":"YQ==","ChunkSequenceNumber":1,"TotalApplicationSize":1,"ChunkSize":1}`
	status, resp := postUploadChunk(t, h, body)
	if status != http.StatusOK {
		t.Fatalf("status: got %d, want 200", status)
	}
	tx, _ := resp["TranscationId"].(string) // typo preserved on the wire
	if tx == "" {
		t.Error("response should mint a TransactionId; got empty")
	}
	if ok, _ := resp["UploadSuccess"].(bool); !ok {
		t.Error("UploadSuccess: got false, want true")
	}
}

func TestChunkHandler_UploadChunk_SubsequentCallValidatesTransactionId(t *testing.T) {
	h := NewChunkHandler()
	// First call mints a TransactionId.
	_, first := postUploadChunk(t, h,
		`{"TransactionId":"","ChunkData":"YQ==","ChunkSequenceNumber":1,"TotalApplicationSize":2,"ChunkSize":1}`)
	tx, _ := first["TranscationId"].(string)
	if tx == "" {
		t.Fatal("first response did not mint a TransactionId")
	}

	// Second call uses the minted TransactionId.
	body2 := `{"TransactionId":"` + tx + `","ChunkData":"Yg==","ChunkSequenceNumber":2,"TotalApplicationSize":2,"ChunkSize":1}`
	status2, resp2 := postUploadChunk(t, h, body2)
	if status2 != http.StatusOK {
		t.Fatalf("second status: got %d, want 200", status2)
	}
	if got, _ := resp2["TranscationId"].(string); got != tx {
		t.Errorf("second response TransactionId: got %q, want %q", got, tx)
	}
	if got, _ := resp2["ChunkSequenceNumber"].(float64); int(got) != 2 {
		t.Errorf("ChunkSequenceNumber echo: got %v, want 2", resp2["ChunkSequenceNumber"])
	}
}

func TestChunkHandler_UploadChunk_UnknownTransactionId_Returns400(t *testing.T) {
	h := NewChunkHandler()
	body := `{"TransactionId":"never-issued","ChunkData":"YQ==","ChunkSequenceNumber":1,"TotalApplicationSize":1,"ChunkSize":1}`
	status, _ := postUploadChunk(t, h, body)
	if status != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", status)
	}
}

func TestChunkHandler_UploadChunk_ZeroSequenceNumber_Returns400(t *testing.T) {
	h := NewChunkHandler()
	body := `{"TransactionId":"","ChunkData":"YQ==","ChunkSequenceNumber":0,"TotalApplicationSize":1,"ChunkSize":1}`
	status, _ := postUploadChunk(t, h, body)
	if status != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", status)
	}
}

func TestChunkHandler_UploadChunk_EmptyChunkData_Returns400(t *testing.T) {
	h := NewChunkHandler()
	body := `{"TransactionId":"","ChunkData":"","ChunkSequenceNumber":1,"TotalApplicationSize":1,"ChunkSize":1}`
	status, _ := postUploadChunk(t, h, body)
	if status != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", status)
	}
}

func postCompletion(t *testing.T, h *ChunkHandler, path string, body string) (int, map[string]interface{}) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	if !h.Handle(w, req) {
		t.Fatalf("ChunkHandler.Handle returned false for %s", path)
	}
	var decoded map[string]interface{}
	if w.Body.Len() > 0 {
		_ = json.Unmarshal(w.Body.Bytes(), &decoded)
	}
	return w.Code, decoded
}

func TestChunkHandler_BeginInstall_KnownTransactionId_Returns200WithHydratedEntity(t *testing.T) {
	h := NewChunkHandler()
	_, first := postUploadChunk(t, h,
		`{"TransactionId":"","ChunkData":"YQ==","ChunkSequenceNumber":1,"TotalApplicationSize":1,"ChunkSize":1}`)
	tx, _ := first["TranscationId"].(string)
	if tx == "" {
		t.Fatal("setup: upload-chunk did not mint a TransactionId")
	}

	body := `{"TransactionId":"` + tx + `","ApplicationName":"Forklift Test App","BundleId":"com.example.forklift"}`
	status, resp := postCompletion(t, h, "/api/mam/apps/internal/begininstall", body)
	if status != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (response: %v)", status, resp)
	}
	if resp["Uuid"] == nil {
		t.Error("response missing server-hydrated Uuid")
	}
	if resp["OrganizationGroupUuid"] == nil {
		t.Error("response missing server-hydrated OrganizationGroupUuid")
	}
}

func TestChunkHandler_BeginInstall_UnknownTransactionId_Returns400(t *testing.T) {
	h := NewChunkHandler()
	body := `{"TransactionId":"never-issued","ApplicationName":"x"}`
	status, _ := postCompletion(t, h, "/api/mam/apps/internal/begininstall", body)
	if status != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", status)
	}
}

// blockingResponseWriter signals on `startedWrite` the first time Write
// is called, then blocks until `release` is closed. Used to verify that
// ChunkHandler does NOT hold its mutex across the HTTP write — if it
// did, two concurrent /uploadchunk POSTs would serialize at the lock and
// only one goroutine would reach Write at a time.
type blockingResponseWriter struct {
	header       http.Header
	startedWrite chan struct{}
	release      chan struct{}
	once         sync.Once
}

func newBlockingResponseWriter() *blockingResponseWriter {
	return &blockingResponseWriter{
		header:       make(http.Header),
		startedWrite: make(chan struct{}),
		release:      make(chan struct{}),
	}
}

func (b *blockingResponseWriter) Header() http.Header { return b.header }
func (b *blockingResponseWriter) WriteHeader(int)     {}
func (b *blockingResponseWriter) Write(p []byte) (int, error) {
	b.once.Do(func() { close(b.startedWrite) })
	<-b.release
	return len(p), nil
}

func TestChunkHandler_UploadChunk_DoesNotHoldLockDuringWrite(t *testing.T) {
	h := NewChunkHandler()

	body := func() string {
		return `{"TransactionId":"","ChunkData":"YQ==","ChunkSequenceNumber":1,"TotalApplicationSize":1,"ChunkSize":1}`
	}

	// Goroutine A: starts a chunk POST, will block inside Write.
	wA := newBlockingResponseWriter()
	rA := httptest.NewRequest(http.MethodPost,
		"/api/mam/apps/internal/uploadchunk",
		strings.NewReader(body()))
	doneA := make(chan struct{})
	go func() {
		h.Handle(wA, rA)
		close(doneA)
	}()

	// Wait for goroutine A to reach Write.
	select {
	case <-wA.startedWrite:
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine A never reached Write within 2s")
	}

	// At this point goroutine A is blocked inside writeChunkJSON. If the
	// handler still holds h.mu, goroutine B's Handle call will block at
	// h.mu.Lock(). Goroutine B uses a non-blocking recorder so we can
	// verify it completes promptly.
	wB := httptest.NewRecorder()
	rB := httptest.NewRequest(http.MethodPost,
		"/api/mam/apps/internal/uploadchunk",
		strings.NewReader(body()))
	doneB := make(chan struct{})
	go func() {
		h.Handle(wB, rB)
		close(doneB)
	}()

	select {
	case <-doneB:
		// goroutine B completed while A is still inside Write -- proves
		// the lock was released before the HTTP write happened.
	case <-time.After(2 * time.Second):
		t.Fatal("goroutine B blocked while A holds the response write; handler held mutex across HTTP I/O (cycle-1 finding #1 regression)")
	}

	// Drain goroutine A.
	close(wA.release)
	<-doneA
}

// chunkFixtureUuidForTest reads chunkFixturePath(name)'s own "Uuid" field at
// test-run time, rather than a test source asserting a hardcoded literal
// value. The public package's sync scrubber rewrites any whole-UUID-shaped
// literal it finds anywhere in shipped content -- including inside shipped
// fixture JSON (not just Go source) -- so a hardcoded expected UUID would
// break in the assembled public package even though the handler's actual
// behavior under test (return whatever the fixture contains) is unaffected
// by that rewrite. Deriving the expectation from the same fixture the
// handler reads keeps the assertion meaningful regardless of the literal
// value on disk.
func chunkFixtureUuidForTest(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(chunkFixturePath(name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", chunkFixturePath(name), err)
	}
	var decoded struct {
		Response struct {
			Body struct {
				Uuid string
			} `json:"body"`
		} `json:"response"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("parse fixture %s: %v", chunkFixturePath(name), err)
	}
	if decoded.Response.Body.Uuid == "" {
		t.Fatalf("fixture %s has an empty response.body.Uuid field -- cannot derive an expected value", chunkFixturePath(name))
	}
	return decoded.Response.Body.Uuid
}

func TestChunkHandler_InternalApplication_KnownTransactionId_Returns200WithDistinctUuid(t *testing.T) {
	h := NewChunkHandler()
	_, first := postUploadChunk(t, h,
		`{"TransactionId":"","ChunkData":"YQ==","ChunkSequenceNumber":1,"TotalApplicationSize":1,"ChunkSize":1}`)
	tx, _ := first["TranscationId"].(string)
	if tx == "" {
		t.Fatal("setup: upload-chunk did not mint a TransactionId")
	}

	body := `{"transaction_id":"` + tx + `","application_name":"Forklift V1Model Test"}`
	status, resp := postCompletion(t, h, "/api/mam/apps/internal/application", body)
	if status != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (response: %v)", status, resp)
	}
	wantUuid := chunkFixtureUuidForTest(t, "internal-application-response.json")
	uuid, _ := resp["Uuid"].(string)
	if uuid != wantUuid {
		t.Errorf("Uuid: got %q, want the shared default fixture's own Uuid %q", uuid, wantUuid)
	}
}

// handleInternalApplication used to hardcode a single
// shared fixture for every create, so a captured platform-specific fixture
// (e.g. apk vs msi, which differ in real response shape) could never
// actually be served by a mock-driven test. Selecting by the request's
// file_name extension (present on every real create call, per the SDK's
// InternalAppChunkTransactionV1Model.FileName -> "file_name" on the wire)
// lets a platform-specific capture be wired in without breaking callers
// that don't set it.
func newInternalAppSession(t *testing.T, h *ChunkHandler) string {
	t.Helper()
	_, first := postUploadChunk(t, h,
		`{"TransactionId":"","ChunkData":"YQ==","ChunkSequenceNumber":1,"TotalApplicationSize":1,"ChunkSize":1}`)
	tx, _ := first["TranscationId"].(string)
	if tx == "" {
		t.Fatal("setup: upload-chunk did not mint a TransactionId")
	}
	return tx
}

func TestChunkHandler_InternalApplication_ApkFileName_ServesApkFixture(t *testing.T) {
	h := NewChunkHandler()
	tx := newInternalAppSession(t, h)

	body := `{"transaction_id":"` + tx + `","file_name":"JustSayHello.apk"}`
	status, resp := postCompletion(t, h, "/api/mam/apps/internal/application", body)
	if status != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (response: %v)", status, resp)
	}
	if name, _ := resp["ApplicationName"].(string); name != "JustSayHello" {
		t.Errorf("ApplicationName: got %q, want the apk fixture's %q", name, "JustSayHello")
	}
}

func TestChunkHandler_InternalApplication_MsiFileName_ServesMsiFixture(t *testing.T) {
	h := NewChunkHandler()
	tx := newInternalAppSession(t, h)

	body := `{"transaction_id":"` + tx + `","file_name":"7z2409-x64.msi"}`
	status, resp := postCompletion(t, h, "/api/mam/apps/internal/application", body)
	if status != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (response: %v)", status, resp)
	}
	if name, _ := resp["ApplicationName"].(string); name != "7z2409-x64-Test" {
		t.Errorf("ApplicationName: got %q, want the msi fixture's %q", name, "7z2409-x64-Test")
	}
}

func TestChunkHandler_InternalApplication_IpaFileName_ServesIpaFixture(t *testing.T) {
	h := NewChunkHandler()
	tx := newInternalAppSession(t, h)

	body := `{"transaction_id":"` + tx + `","file_name":"HelloMDMTest.ipa"}`
	status, resp := postCompletion(t, h, "/api/mam/apps/internal/application", body)
	if status != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (response: %v)", status, resp)
	}
	if name, _ := resp["ApplicationName"].(string); name != "Test Rabbitthrow" {
		t.Errorf("ApplicationName: got %q, want the ipa fixture's %q", name, "Test Rabbitthrow")
	}
}

func TestChunkHandler_InternalApplication_PkgFileName_ServesPkgFixture(t *testing.T) {
	h := NewChunkHandler()
	tx := newInternalAppSession(t, h)

	body := `{"transaction_id":"` + tx + `","file_name":"MacOSPkgTestApp-1.0.pkg"}`
	status, resp := postCompletion(t, h, "/api/mam/apps/internal/application", body)
	if status != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (response: %v)", status, resp)
	}
	// "Test Rabbitthrow" coincidentally matches the ipa fixture's own
	// anonymized ApplicationName (see TestChunkHandler_InternalApplication_IpaFileName_ServesIpaFixture
	// above) -- both are independent anonymizer outputs for unrelated real
	// captures, not a routing bug; this test still verifies file_name ->
	// pkg-specific-fixture routing independently of what the fake name is.
	if name, _ := resp["ApplicationName"].(string); name != "Test Rabbitthrow" {
		t.Errorf("ApplicationName: got %q, want the pkg fixture's %q", name, "Test Rabbitthrow")
	}
}

func TestChunkHandler_InternalApplication_ZipFileName_ServesZipFixture(t *testing.T) {
	h := NewChunkHandler()
	tx := newInternalAppSession(t, h)

	body := `{"transaction_id":"` + tx + `","file_name":"WindowsZipTestApp-1.0.zip"}`
	status, resp := postCompletion(t, h, "/api/mam/apps/internal/application", body)
	if status != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (response: %v)", status, resp)
	}
	if name, _ := resp["ApplicationName"].(string); name != "SDK Synthetic Zip Test" {
		t.Errorf("ApplicationName: got %q, want the synthetic zip fixture's %q", name, "SDK Synthetic Zip Test")
	}
}

func TestChunkHandler_InternalApplication_NoFileName_ServesSharedDefaultFixture(t *testing.T) {
	h := NewChunkHandler()
	tx := newInternalAppSession(t, h)

	// No file_name at all -- must fall back to the pre-existing shared
	// fixture, exactly as every caller before this change relied on.
	body := `{"transaction_id":"` + tx + `","application_name":"Forklift V1Model Test"}`
	status, resp := postCompletion(t, h, "/api/mam/apps/internal/application", body)
	if status != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (response: %v)", status, resp)
	}
	if name, _ := resp["ApplicationName"].(string); name != "Forklift V1Model Test App" {
		t.Errorf("ApplicationName: got %q, want the shared default fixture's %q", name, "Forklift V1Model Test App")
	}
}

func TestChunkHandler_InternalApplication_UnrecognizedExtension_ServesSharedDefaultFixture(t *testing.T) {
	h := NewChunkHandler()
	tx := newInternalAppSession(t, h)

	body := `{"transaction_id":"` + tx + `","file_name":"app.exe"}`
	status, resp := postCompletion(t, h, "/api/mam/apps/internal/application", body)
	if status != http.StatusOK {
		t.Fatalf("status: got %d, want 200 (response: %v)", status, resp)
	}
	if name, _ := resp["ApplicationName"].(string); name != "Forklift V1Model Test App" {
		t.Errorf("ApplicationName: got %q, want the shared default fixture's %q", name, "Forklift V1Model Test App")
	}
}
