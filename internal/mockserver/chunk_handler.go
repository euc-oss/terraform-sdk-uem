package mockserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// ChunkHandler is the stateful HTTP handler for the three internal-app
// chunk routes: /uploadchunk, /begininstall, /internal/application.
//
// It mints TransactionId values on first /uploadchunk POST, tracks them in
// an in-memory map keyed by TransactionId, validates subsequent chunk
// POSTs against that map, and validates completion-endpoint POSTs against
// the same map.
//
// State lifecycle is per-ChunkHandler instance, which is per-MockServer
// instance. Tests get isolation through normal MockServer setup/teardown.
type ChunkHandler struct {
	mu       sync.Mutex
	sessions map[string]*chunkSessionState
	nextID   int // monotonic counter for mint
}

type chunkSessionState struct {
	transactionID    string
	lastSequence     int
	totalSize        int64
	chunkSize        int64
	chunksReceived   int
	completionCalled bool
}

// NewChunkHandler creates a handler with empty state.
func NewChunkHandler() *ChunkHandler {
	return &ChunkHandler{sessions: make(map[string]*chunkSessionState)}
}

// Handle returns true if the request is one of the chunk routes and was
// served. Returns false to let the caller's outer dispatch fall through.
func (h *ChunkHandler) Handle(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodPost {
		return false
	}
	switch {
	case strings.HasSuffix(r.URL.Path, "/apps/internal/uploadchunk"):
		h.handleUploadChunk(w, r)
		return true
	case strings.HasSuffix(r.URL.Path, "/apps/internal/begininstall"):
		h.handleBeginInstall(w, r)
		return true
	case strings.HasSuffix(r.URL.Path, "/apps/internal/application"):
		h.handleInternalApplication(w, r)
		return true
	}
	return false
}

// Reset clears the state (useful between tests inside the same handler).
func (h *ChunkHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sessions = make(map[string]*chunkSessionState)
	h.nextID = 0
}

func (h *ChunkHandler) handleUploadChunk(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TransactionId        string `json:"TransactionId"`
		ChunkData            string `json:"ChunkData"`
		ChunkSequenceNumber  int    `json:"ChunkSequenceNumber"`
		TotalApplicationSize int64  `json:"TotalApplicationSize"`
		ChunkSize            int64  `json:"ChunkSize"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeChunkJSON(w, http.StatusBadRequest, map[string]string{"errorCode": "400", "message": "invalid request body"})
		return
	}
	if req.ChunkSequenceNumber == 0 {
		writeChunkJSON(w, http.StatusBadRequest, map[string]string{"errorCode": "400", "message": "Chunk sequence number cannot be zero."})
		return
	}
	if req.ChunkData == "" {
		writeChunkJSON(w, http.StatusBadRequest, map[string]string{"errorCode": "400", "message": "Chunk data cannot be empty."})
		return
	}

	// Mutate session state under the lock; capture the response values
	// into locals; release the lock BEFORE writing the HTTP response so a
	// slow or blocked client cannot serialize concurrent /uploadchunk
	// POSTs behind the lock.
	h.mu.Lock()
	var sess *chunkSessionState
	if req.TransactionId == "" {
		// Mint a new session.
		h.nextID++
		tx := fmt.Sprintf("mock-tx-%d", h.nextID)
		sess = &chunkSessionState{
			transactionID: tx,
			totalSize:     req.TotalApplicationSize,
			chunkSize:     req.ChunkSize,
		}
		h.sessions[tx] = sess
	} else {
		sess = h.sessions[req.TransactionId]
		if sess == nil {
			h.mu.Unlock()
			writeChunkJSON(w, http.StatusBadRequest, map[string]string{
				"errorCode": "400",
				"message":   "Unknown TransactionId.",
			})
			return
		}
	}
	sess.lastSequence = req.ChunkSequenceNumber
	sess.chunksReceived++
	txToReturn := sess.transactionID
	h.mu.Unlock()

	writeChunkJSON(w, http.StatusOK, map[string]interface{}{
		"TranscationId":       txToReturn, // typo preserved on wire per Q7-ii
		"ChunkSequenceNumber": req.ChunkSequenceNumber,
		"UploadSuccess":       true,
	})
}

func (h *ChunkHandler) handleBeginInstall(w http.ResponseWriter, r *http.Request) {
	h.handleCompletion(w, r, "TransactionId", func(_ map[string]interface{}) string {
		return chunkFixturePath("begininstall-response.json")
	})
}

func (h *ChunkHandler) handleInternalApplication(w http.ResponseWriter, r *http.Request) {
	h.handleCompletion(w, r, "transaction_id", selectInternalApplicationFixture)
}

// selectInternalApplicationFixture picks a platform-specific create fixture
// based on the request's file_name extension.
// Every real create call sets file_name (InternalAppChunkTransactionV1Model.
// FileName -> "file_name" on the wire); falls back to the pre-existing
// shared fixture when file_name is absent or its extension isn't recognized,
// so callers that don't care about platform-specific response shape are
// unaffected.
func selectInternalApplicationFixture(raw map[string]interface{}) string {
	fileName, _ := raw["file_name"].(string)
	switch {
	case strings.HasSuffix(strings.ToLower(fileName), ".apk"):
		return chunkFixturePath(filepath.Join("apk", "internal_app_create_apk.json"))
	case strings.HasSuffix(strings.ToLower(fileName), ".msi"):
		return chunkFixturePath(filepath.Join("msi", "internal_app_create_msi.json"))
	case strings.HasSuffix(strings.ToLower(fileName), ".ipa"):
		// Live-captured against UEM 26.2.1614.12 -- see
		// testdata/mock-responses/internal-apps-v1/ipa/internal_app_create_ipa.json.
		return chunkFixturePath(filepath.Join("ipa", "internal_app_create_ipa.json"))
	case strings.HasSuffix(strings.ToLower(fileName), ".pkg"):
		// Live-captured (UEM 26.2.1614.12) for the plain InternalAppsV1
		// chunked-upload path -- distinct from the full-SLM
		// MacOsAppsV1Service .pkg/.dmg coverage in
		// generated_macosappsv1_lifecycle_test.go. See
		// testdata/mock-responses/internal-apps-v1/pkg/
		// internal_app_create_pkg.json's own description for the full
		// capture writeup.
		return chunkFixturePath(filepath.Join("pkg", "internal_app_create_pkg.json"))
	case strings.HasSuffix(strings.ToLower(fileName), ".zip"):
		// SYNTHETIC / hand-authored PLACEHOLDER (NOT live-captured). Live
		// .zip capture is explicitly HELD pending org-group Software
		// Distribution feature-flag confirmation (see vendored-test-apps/
		// windows/SPEC.md section 9) -- do not treat this fixture as a
		// verified real-wire shape. See testdata/mock-responses/
		// internal-apps-v1/zip/internal_app_create_zip.json's own
		// description for the full label.
		return chunkFixturePath(filepath.Join("zip", "internal_app_create_zip.json"))
	default:
		return chunkFixturePath("internal-application-response.json")
	}
}

// handleCompletion is the shared body for the begininstall and
// internal/application routes. The txField parameter names the JSON key
// the wire uses for the transaction identifier on each endpoint
// (PascalCase vs snake_case); fixtureFor picks the absolute path to the
// fixture file to serve on success, given the decoded request body (a fixed
// path for begininstall, content-dependent for internal/application).
func (h *ChunkHandler) handleCompletion(w http.ResponseWriter, r *http.Request, txField string, fixtureFor func(raw map[string]interface{}) string) {
	var raw map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeChunkJSON(w, http.StatusBadRequest, map[string]string{"errorCode": "400", "message": "invalid request body"})
		return
	}
	tx, _ := raw[txField].(string)
	if tx == "" {
		writeChunkJSON(w, http.StatusBadRequest, map[string]string{"errorCode": "400", "message": "TransactionId required"})
		return
	}

	h.mu.Lock()
	sess, ok := h.sessions[tx]
	if ok {
		sess.completionCalled = true
	}
	h.mu.Unlock()
	if !ok {
		writeChunkJSON(w, http.StatusBadRequest, map[string]string{"errorCode": "400", "message": "Unknown TransactionId."})
		return
	}

	fixture := fixtureFor(raw)
	data, err := os.ReadFile(fixture)
	if err != nil {
		writeChunkJSON(w, http.StatusInternalServerError, map[string]string{"errorCode": "500", "message": fmt.Sprintf("fixture load error: %v", err)})
		return
	}
	var fx struct {
		Response struct {
			StatusCode int                    `json:"status_code"`
			Headers    map[string]string      `json:"headers"`
			Body       map[string]interface{} `json:"body"`
		} `json:"response"`
	}
	if err := json.Unmarshal(data, &fx); err != nil {
		writeChunkJSON(w, http.StatusInternalServerError, map[string]string{"errorCode": "500", "message": fmt.Sprintf("fixture parse error: %v", err)})
		return
	}
	for k, v := range fx.Response.Headers {
		w.Header().Set(k, v)
	}
	if fx.Response.StatusCode == 0 {
		fx.Response.StatusCode = http.StatusOK
	}
	w.WriteHeader(fx.Response.StatusCode)
	_ = json.NewEncoder(w).Encode(fx.Response.Body)
}

// chunkFixtureSyntheticPlaceholderOverrides maps a chunkFixturePath name
// argument to its path (relative to internal/mockserver/testdata/
// synthetic-placeholders/) when that specific fixture must ALWAYS resolve
// from the synthetic-placeholders location, unconditionally -- regardless of
// whether MOCK_RESPONSES_DIR is set -- instead of chunkFixturePath's normal
// resolution below.
//
// Why this exists: chunkFixturePath's normal resolution (see below) is
// MOCK_RESPONSES_DIR-aware, so a name backed by a genuinely-verified fixture
// that DOES exist in a vendored/staged fixture tree (the apk/msi/ipa/pkg
// create fixtures) resolves correctly whether or not that env var is set.
// But three names are backed by SYNTHETIC PLACEHOLDERS that were never
// migrated to (and, for .zip, never will be, and for begininstall/
// internal-application-response, are demoted-fabricated originals) any
// vendored fixture tree at all:
//
//   - zip/internal_app_create_zip.json (see selectInternalApplicationFixture's
//     own comment on the .zip case)
//   - begininstall-response.json
//   - internal-application-response.json (the shared default/unrecognized-
//     extension fallback)
//
// For these three, a MOCK_RESPONSES_DIR-aware fallback is useless -- the name
// simply isn't present under any vendored tree to be found -- so they're
// copied to internal/mockserver/testdata/synthetic-placeholders/internal-apps-v1/
// and unconditionally redirected here instead, specifically so they survive a
// Phase-3 `git rm testdata/` (see internal/mockserver/testdata/
// synthetic-placeholders/README.md and internal/mockserver/
// bypass_governance_test.go's population 2/3 section). Without this override,
// chunkFixturePath kept resolving these three from testdata/mock-responses/...,
// which defeated the point of that migration once testdata/ is actually
// removed -- confirmed by moving testdata/ aside (with MOCK_RESPONSES_DIR
// pointed at a vendored/staged fixture tree that has none of these three
// names) and running TestChunkHandler_InternalApplication_
// ZipFileName_ServesZipFixture, TestChunkHandler_BeginInstall_
// KnownTransactionId_..., and TestChunkHandler_InternalApplication_
// {KnownTransactionId,NoFileName,UnrecognizedExtension}_...
// (internal/mockserver/chunk_handler_test.go) -- all failed with a
// mock-server 500 (fixture not found) before this override covered them, and
// pass with testdata/ absent after it. NOTE: the generated
// tests/generated_internalappsv1_lifecycle_test.go's
// TestGeneratedInternalAppsV1Lifecycle_ZIP does NOT exercise the .zip
// call site above -- its CreateInternalApplication call never sets FileName,
// so its create step always hits the internal-application-response.json
// default fallback instead (a separate, pre-existing, out-of-scope gap
// shared with the apk/msi/ipa/pkg lifecycle tests; see the 2026-08-06
// chunkFixturePath synthetic-placeholder fix report for the full
// testdata-absent failure inventory).
//
// The .zip GET/DELETE fixtures do NOT need an entry here: they're loaded
// directly via the bypass mechanism in
// tests/generated_internalappsv1_lifecycle_test.go (newMockServerFromBypassFixturesOnly),
// never through ChunkHandler/chunkFixturePath at all -- ChunkHandler.Handle
// only serves POST requests to /uploadchunk, /begininstall, and
// /internal/application, so GET/DELETE requests never reach it.
var chunkFixtureSyntheticPlaceholderOverrides = map[string]string{
	filepath.Join("zip", "internal_app_create_zip.json"): filepath.Join("internal-apps-v1", "zip", "internal_app_create_zip.json"),
	"begininstall-response.json":                         filepath.Join("internal-apps-v1", "begininstall-response.json"),
	"internal-application-response.json":                 filepath.Join("internal-apps-v1", "internal-application-response.json"),
}

// chunkFixturePath resolves a fixture path relative to this source file so
// tests don't depend on the working directory.
//
// Resolution order:
//
//  1. chunkFixtureSyntheticPlaceholderOverrides (see its own doc comment) --
//     checked first, unconditionally, for names backed by a synthetic
//     placeholder that no vendored fixture tree will ever contain.
//  2. Otherwise, ResolveResponsesDir (internal/mockserver/loader.go) against
//     this package's normal testdata/mock-responses/internal-apps-v1/
//     directory -- reused directly rather than duplicated here, so this call
//     site gets the exact same MOCK_RESPONSES_DIR override semantics
//     (env var, "subdirectory preserved" behavior) LoadResponsesFromDir's
//     callers already get. When MOCK_RESPONSES_DIR is unset, ResolveResponsesDir
//     is a no-op and this resolves exactly as before. When it is set to a
//     directory containing an internal-apps-v1/<name> fixture (true today for
//     the apk/msi/ipa/pkg create fixtures, which ARE present in the vendored
//     fixture set), that fixture is found there instead of under repo-root
//     testdata/.
//
// Limitation: runtime.Caller(0) returns the path the source file was
// compiled from. For `go test` this points at a path that exists at test
// time. If a CI pipeline builds the test binary in one place and runs it
// elsewhere (pre-built test binary, container artifact), the path may not
// exist at runtime. Project CI uses `go test ./...` directly, so this is
// safe today; revisit if test binaries get shipped separately.
func chunkFixturePath(name string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	thisDir := filepath.Dir(thisFile)
	if rel, ok := chunkFixtureSyntheticPlaceholderOverrides[name]; ok {
		return filepath.Join(thisDir, "testdata", "synthetic-placeholders", rel)
	}
	dir := ResolveResponsesDir(filepath.Join(thisDir, "..", "..", "testdata", "mock-responses", "internal-apps-v1"))
	return filepath.Join(dir, name)
}

// writeChunkJSON is a small helper used by the per-route handlers.
func writeChunkJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if body != nil {
		_ = json.NewEncoder(w).Encode(body)
	}
}
