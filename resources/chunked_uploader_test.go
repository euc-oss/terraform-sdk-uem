package resources

// Tests for ChunkedUploader live here. See chunked_uploader.go for the
// types under test. Tests use a stub internalAppsChunkClient rather than
// a real HTTP client.
//
// Pointer-field reminder: codegen emits InternalAppChunkV1's
// ChunkSequenceNumber/ChunkSize/TotalApplicationSize as *int / *int64 and
// AppChunkTranscationResponseV1's UploadSuccess as *bool. Tests construct
// these via addressable locals or the test-local intPtr/int64Ptr helpers
// at the bottom of this file. Plan's test stubs assumed bare int/bool but
// the emit-vs-plan translation lives in chunked_uploader.go's top-of-file
// docblock.

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"

	mamv1 "github.com/euc-oss/terraform-sdk-uem/v26/internal/mam/v1"
)

// stubChunkClient is a configurable mock of internalAppsChunkClient. Each
// method records its call arguments and returns from a queue of canned
// responses. The queue empties as calls land; if a test consumes more calls
// than queued, the stub returns a default success response (allows tests
// to leave tail-of-queue unspecified).
type stubChunkClient struct {
	chunkCalls []*mamv1.InternalAppChunkV1
	chunkResps []*mamv1.AppChunkTranscationResponseV1
	chunkErrs  []error

	beginCalls []*mamv1.InternalAppChunkTransactionV1
	beginResp  *mamv1.InternalApplicationEntityV1
	beginErr   error

	appCalls []*mamv1.InternalAppChunkTransactionV1Model
	appResp  *mamv1.InternalApplicationEntityV1
	appErr   error
}

func (s *stubChunkClient) InsertInternalApplicationChunkAsync(
	ctx context.Context,
	chunk *mamv1.InternalAppChunkV1,
) (http.Header, *mamv1.AppChunkTranscationResponseV1, error) {
	s.chunkCalls = append(s.chunkCalls, chunk)
	idx := len(s.chunkCalls) - 1
	var resp *mamv1.AppChunkTranscationResponseV1
	var err error
	if idx < len(s.chunkResps) {
		resp = s.chunkResps[idx]
	} else {
		txID := "tx-stub-1"
		if chunk.TransactionID != "" {
			txID = chunk.TransactionID
		}
		uploadOK := true
		resp = &mamv1.AppChunkTranscationResponseV1{
			TranscationID:       txID,
			ChunkSequenceNumber: chunk.ChunkSequenceNumber,
			UploadSuccess:       &uploadOK,
		}
	}
	if idx < len(s.chunkErrs) {
		err = s.chunkErrs[idx]
	}
	return nil, resp, err
}

func (s *stubChunkClient) CreateInternalAppFromBlobAsync(
	ctx context.Context,
	body *mamv1.InternalAppChunkTransactionV1,
) (http.Header, *mamv1.InternalApplicationEntityV1, error) {
	s.beginCalls = append(s.beginCalls, body)
	return nil, s.beginResp, s.beginErr
}

func (s *stubChunkClient) CreateInternalApplicationFromBlob(
	ctx context.Context,
	body *mamv1.InternalAppChunkTransactionV1Model,
) (http.Header, *mamv1.InternalApplicationEntityV1, error) {
	s.appCalls = append(s.appCalls, body)
	return nil, s.appResp, s.appErr
}

func intPtr(i int) *int    { return &i }
func boolPtr(b bool) *bool { return &b }

func TestUpload_SingleChunk_PostsOnceAndReturnsSession(t *testing.T) {
	const totalSize = int64(1024) // 1 KiB, fits in one default-size chunk
	body := bytes.Repeat([]byte("A"), int(totalSize))

	stub := &stubChunkClient{
		chunkResps: []*mamv1.AppChunkTranscationResponseV1{
			{
				TranscationID:       "tx-abc",
				ChunkSequenceNumber: intPtr(1),
				UploadSuccess:       boolPtr(true),
			},
		},
	}
	u := &ChunkedUploader{apps: stub}

	session, err := u.Upload(context.Background(), bytes.NewReader(body), totalSize, nil)
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if session == nil {
		t.Fatal("Upload returned nil session")
		return
	}
	if session.TransactionID != "tx-abc" {
		t.Errorf("TransactionID: got %q, want %q", session.TransactionID, "tx-abc")
	}
	if session.TotalSize != totalSize {
		t.Errorf("TotalSize: got %d, want %d", session.TotalSize, totalSize)
	}
	if session.LastSequence != 1 {
		t.Errorf("LastSequence: got %d, want 1", session.LastSequence)
	}

	if len(stub.chunkCalls) != 1 {
		t.Fatalf("expected 1 chunk call, got %d", len(stub.chunkCalls))
	}
	call := stub.chunkCalls[0]
	if call.TransactionID != "" {
		t.Errorf("first chunk TransactionID: got %q, want empty (server mints)", call.TransactionID)
	}
	if call.ChunkSequenceNumber == nil || *call.ChunkSequenceNumber != 1 {
		got := 0
		if call.ChunkSequenceNumber != nil {
			got = *call.ChunkSequenceNumber
		}
		t.Errorf("ChunkSequenceNumber: got %d, want 1", got)
	}
	if call.TotalApplicationSize == nil || *call.TotalApplicationSize != totalSize {
		got := int64(0)
		if call.TotalApplicationSize != nil {
			got = *call.TotalApplicationSize
		}
		t.Errorf("TotalApplicationSize: got %d, want %d", got, totalSize)
	}
	if call.ChunkSize == nil || *call.ChunkSize != totalSize {
		got := int64(0)
		if call.ChunkSize != nil {
			got = *call.ChunkSize
		}
		t.Errorf("ChunkSize: got %d, want %d", got, totalSize)
	}

	// ChunkData renders as Go string (Outcome A per Task 1 Step 4b): the
	// implementation base64-encodes the raw chunk bytes.
	wantData := base64.StdEncoding.EncodeToString(body)
	if call.ChunkData != wantData {
		t.Errorf("ChunkData: got %q (%d bytes), want %q (%d bytes)",
			truncate(call.ChunkData), len(call.ChunkData),
			truncate(wantData), len(wantData))
	}
}

func truncate(s string) string {
	if len(s) > 40 {
		return s[:40] + "..."
	}
	return s
}

func TestUpload_TwoChunks_ThreadsTransactionId(t *testing.T) {
	// 3 KiB payload with a 2 KiB chunk size -> 2 chunks (2 KiB + 1 KiB).
	const totalSize = int64(3 * 1024)
	const chunkSize = int64(2 * 1024)
	body := bytes.Repeat([]byte("B"), int(totalSize))

	stub := &stubChunkClient{
		chunkResps: []*mamv1.AppChunkTranscationResponseV1{
			{TranscationID: "tx-xyz", ChunkSequenceNumber: intPtr(1), UploadSuccess: boolPtr(true)},
			{TranscationID: "tx-xyz", ChunkSequenceNumber: intPtr(2), UploadSuccess: boolPtr(true)},
		},
	}
	u := &ChunkedUploader{apps: stub}

	session, err := u.Upload(
		context.Background(),
		bytes.NewReader(body),
		totalSize,
		&UploadOptions{ChunkSize: chunkSize},
	)
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if session.TransactionID != "tx-xyz" {
		t.Errorf("TransactionID: got %q, want %q", session.TransactionID, "tx-xyz")
	}
	if session.LastSequence != 2 {
		t.Errorf("LastSequence: got %d, want 2", session.LastSequence)
	}

	if len(stub.chunkCalls) != 2 {
		t.Fatalf("expected 2 chunk calls, got %d", len(stub.chunkCalls))
	}
	if stub.chunkCalls[0].TransactionID != "" {
		t.Errorf("first chunk TransactionID: got %q, want empty", stub.chunkCalls[0].TransactionID)
	}
	if got := derefInt(stub.chunkCalls[0].ChunkSequenceNumber); got != 1 {
		t.Errorf("first chunk sequence: got %d, want 1", got)
	}
	if got := derefInt64(stub.chunkCalls[0].ChunkSize); got != chunkSize {
		t.Errorf("first chunk size: got %d, want %d", got, chunkSize)
	}
	if stub.chunkCalls[1].TransactionID != "tx-xyz" {
		t.Errorf("second chunk TransactionID: got %q, want %q", stub.chunkCalls[1].TransactionID, "tx-xyz")
	}
	if got := derefInt(stub.chunkCalls[1].ChunkSequenceNumber); got != 2 {
		t.Errorf("second chunk sequence: got %d, want 2", got)
	}
	if got := derefInt64(stub.chunkCalls[1].ChunkSize); got != totalSize-chunkSize {
		t.Errorf("second chunk size: got %d, want %d", got, totalSize-chunkSize)
	}
}

func TestUpload_ZeroTotalSize_Errors(t *testing.T) {
	// totalSize=0 must be rejected up front: with no chunks issued the
	// session would carry an empty TransactionID and any subsequent
	// completion call would fail server-side with a cryptic 400
	// (cycle-1 finding #2).
	stub := &stubChunkClient{}
	u := &ChunkedUploader{apps: stub}

	session, err := u.Upload(context.Background(), bytes.NewReader(nil), 0, nil)
	if err == nil {
		t.Fatal("Upload should reject totalSize=0")
	}
	if !strings.Contains(err.Error(), "totalSize must be positive") {
		t.Errorf("error should mention 'totalSize must be positive', got: %v", err)
	}
	if session != nil {
		t.Errorf("expected nil session on contract violation, got %+v", session)
	}
	if len(stub.chunkCalls) != 0 {
		t.Errorf("expected 0 chunk calls on validation error, got %d", len(stub.chunkCalls))
	}
}

func TestUpload_ShortReader_ExhaustsBeforeFirstChunk(t *testing.T) {
	// totalSize=2048, body=1024 -> first io.ReadFull tries 2048 bytes, gets
	// 1024, fails. Zero chunk POSTs issued.
	const totalSize = int64(2048)
	body := bytes.Repeat([]byte("C"), 1024)

	stub := &stubChunkClient{}
	u := &ChunkedUploader{apps: stub}

	session, err := u.Upload(context.Background(), bytes.NewReader(body), totalSize, nil)
	if err == nil {
		t.Fatal("Upload should have returned an error for short reader")
	}
	if !strings.Contains(err.Error(), "reader exhausted") {
		t.Errorf("error should wrap 'reader exhausted', got: %v", err)
	}
	if session == nil {
		t.Fatal("Upload should still return the (partial) session on error")
	}
	if len(stub.chunkCalls) != 0 {
		t.Errorf("expected 0 chunk calls on short read (single-chunk path), got %d", len(stub.chunkCalls))
	}
}

func TestUpload_ShortReader_ExhaustsAfterFirstChunk(t *testing.T) {
	// totalSize=3072, chunkSize=1024, body=1024 -> first chunk (1024 bytes)
	// posts successfully, second io.ReadFull tries 1024 from an exhausted
	// reader and fails. Exactly 1 chunk POST issued.
	const totalSize = int64(3072)
	const chunkSize = int64(1024)
	body := bytes.Repeat([]byte("C"), 1024)

	stub := &stubChunkClient{}
	u := &ChunkedUploader{apps: stub}

	session, err := u.Upload(
		context.Background(),
		bytes.NewReader(body),
		totalSize,
		&UploadOptions{ChunkSize: chunkSize},
	)
	if err == nil {
		t.Fatal("Upload should have returned an error after partial progress")
	}
	if !strings.Contains(err.Error(), "reader exhausted") {
		t.Errorf("error should wrap 'reader exhausted', got: %v", err)
	}
	if session == nil {
		t.Fatal("Upload should still return the partial session on error")
		return
	}
	if len(stub.chunkCalls) != 1 {
		t.Errorf("expected 1 chunk call (one success, then read fails on chunk 2), got %d", len(stub.chunkCalls))
	}
	if session.LastSequence != 1 {
		t.Errorf("session.LastSequence should reflect the one successful chunk: got %d, want 1", session.LastSequence)
	}
}

func TestBeginInstall_ThreadsTransactionIDIntoMetadata(t *testing.T) {
	session := &ChunkSession{
		TransactionID: "tx-install-1",
		TotalSize:     2048,
		ChunkSize:     1024,
		LastSequence:  2,
	}
	metadata := &mamv1.InternalAppChunkTransactionV1{
		ApplicationName: "Forklift.ipa",
		// TransactionID deliberately not set; BeginInstall fills it in.
	}
	stub := &stubChunkClient{
		beginResp: &mamv1.InternalApplicationEntityV1{},
	}
	u := &ChunkedUploader{apps: stub}

	resp, err := u.BeginInstall(context.Background(), session, metadata)
	if err != nil {
		t.Fatalf("BeginInstall returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("BeginInstall returned nil response")
	}
	if len(stub.beginCalls) != 1 {
		t.Fatalf("expected 1 BeginInstall call, got %d", len(stub.beginCalls))
	}
	if stub.beginCalls[0].TransactionID != "tx-install-1" {
		t.Errorf("metadata.TransactionID: got %q, want %q", stub.beginCalls[0].TransactionID, "tx-install-1")
	}
	if stub.beginCalls[0].ApplicationName != "Forklift.ipa" {
		t.Errorf("metadata.ApplicationName lost: got %q, want %q", stub.beginCalls[0].ApplicationName, "Forklift.ipa")
	}
}

func TestBeginInstall_DoesNotMutateCallerMetadata(t *testing.T) {
	// Caller pre-populates TransactionID (e.g., for diagnostics) and a
	// few other fields, calls BeginInstall, then inspects their own
	// struct. None of their fields should have changed — the helper
	// shallow-copies before threading the session's TransactionID
	// (cycle-1 finding #3).
	session := &ChunkSession{TransactionID: "tx-session-1"}
	metadata := &mamv1.InternalAppChunkTransactionV1{
		ApplicationName: "Forklift.ipa",
		BundleID:        "com.example.forklift",
		TransactionID:   "caller-diagnostic-id",
	}
	original := *metadata // value copy for after-the-fact comparison

	stub := &stubChunkClient{beginResp: &mamv1.InternalApplicationEntityV1{}}
	u := &ChunkedUploader{apps: stub}

	if _, err := u.BeginInstall(context.Background(), session, metadata); err != nil {
		t.Fatalf("BeginInstall returned error: %v", err)
	}

	if metadata.TransactionID != original.TransactionID {
		t.Errorf("caller's metadata.TransactionID mutated: got %q, want %q",
			metadata.TransactionID, original.TransactionID)
	}
	if metadata.ApplicationName != original.ApplicationName || metadata.BundleID != original.BundleID {
		t.Errorf("caller's other fields mutated: got %+v, want %+v", *metadata, original)
	}

	// The Layer-1 call should have received the session's TransactionID,
	// not the caller's diagnostic id.
	if len(stub.beginCalls) != 1 {
		t.Fatalf("expected 1 BeginInstall call, got %d", len(stub.beginCalls))
	}
	if stub.beginCalls[0].TransactionID != "tx-session-1" {
		t.Errorf("wire TransactionID: got %q, want %q",
			stub.beginCalls[0].TransactionID, "tx-session-1")
	}
}

func TestCreateInternalApplication_DoesNotMutateCallerMetadata(t *testing.T) {
	session := &ChunkSession{TransactionID: "tx-session-2"}
	metadata := &mamv1.InternalAppChunkTransactionV1Model{
		ApplicationName: "Forklift V1Model.ipa",
		BundleID:        "com.example.forklift.v1",
		TransactionID:   "caller-diagnostic-id-v1model",
	}
	original := *metadata

	stub := &stubChunkClient{appResp: &mamv1.InternalApplicationEntityV1{}}
	u := &ChunkedUploader{apps: stub}

	if _, err := u.CreateInternalApplication(context.Background(), session, metadata); err != nil {
		t.Fatalf("CreateInternalApplication returned error: %v", err)
	}

	if metadata.TransactionID != original.TransactionID {
		t.Errorf("caller's metadata.TransactionID mutated: got %q, want %q",
			metadata.TransactionID, original.TransactionID)
	}
	if metadata.ApplicationName != original.ApplicationName || metadata.BundleID != original.BundleID {
		t.Errorf("caller's other fields mutated: got %+v, want %+v", *metadata, original)
	}

	if len(stub.appCalls) != 1 {
		t.Fatalf("expected 1 CreateInternalApplication call, got %d", len(stub.appCalls))
	}
	if stub.appCalls[0].TransactionID != "tx-session-2" {
		t.Errorf("wire TransactionID: got %q, want %q",
			stub.appCalls[0].TransactionID, "tx-session-2")
	}
}

func TestBeginInstall_NilSession_Errors(t *testing.T) {
	u := &ChunkedUploader{apps: &stubChunkClient{}}
	_, err := u.BeginInstall(context.Background(), nil, &mamv1.InternalAppChunkTransactionV1{})
	if err == nil {
		t.Fatal("BeginInstall should reject a nil session")
	}
}

func TestBeginInstall_NilMetadata_Errors(t *testing.T) {
	u := &ChunkedUploader{apps: &stubChunkClient{}}
	_, err := u.BeginInstall(context.Background(), &ChunkSession{TransactionID: "tx-x"}, nil)
	if err == nil {
		t.Fatal("BeginInstall should reject nil metadata")
	}
}

func TestCreateInternalApplication_ThreadsTransactionIDIntoV1Model(t *testing.T) {
	session := &ChunkSession{TransactionID: "tx-app-1"}
	metadata := &mamv1.InternalAppChunkTransactionV1Model{
		ApplicationName: "Forklift.ipa",
	}
	stub := &stubChunkClient{
		appResp: &mamv1.InternalApplicationEntityV1{},
	}
	u := &ChunkedUploader{apps: stub}

	resp, err := u.CreateInternalApplication(context.Background(), session, metadata)
	if err != nil {
		t.Fatalf("CreateInternalApplication returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("CreateInternalApplication returned nil response")
	}
	if len(stub.appCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(stub.appCalls))
	}
	if stub.appCalls[0].TransactionID != "tx-app-1" {
		t.Errorf("metadata.TransactionID: got %q, want %q", stub.appCalls[0].TransactionID, "tx-app-1")
	}
}

func TestCreateInternalApplication_NilSession_Errors(t *testing.T) {
	u := &ChunkedUploader{apps: &stubChunkClient{}}
	_, err := u.CreateInternalApplication(context.Background(), nil, &mamv1.InternalAppChunkTransactionV1Model{})
	if err == nil {
		t.Fatal("CreateInternalApplication should reject a nil session")
	}
}

func TestCreateInternalApplication_NilMetadata_Errors(t *testing.T) {
	u := &ChunkedUploader{apps: &stubChunkClient{}}
	_, err := u.CreateInternalApplication(context.Background(), &ChunkSession{TransactionID: "tx-x"}, nil)
	if err == nil {
		t.Fatal("CreateInternalApplication should reject nil metadata")
	}
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func derefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func TestUpload_ResumeFrom_SkipsCompletedChunks(t *testing.T) {
	const totalSize = int64(3 * 1024)
	const chunkSize = int64(1024)

	// Simulate: caller already uploaded chunks 1 and 2. They have a
	// session with LastSequence=2 and they re-open the file seeked to
	// offset 2*1024 = 2048. Upload should send only chunk 3 with
	// TransactionID already populated.
	priorSession := &ChunkSession{
		TransactionID: "tx-resume",
		TotalSize:     totalSize,
		ChunkSize:     chunkSize,
		LastSequence:  2,
	}
	tail := bytes.Repeat([]byte("D"), int(chunkSize)) // chunk 3 = 1 KiB

	stub := &stubChunkClient{
		chunkResps: []*mamv1.AppChunkTranscationResponseV1{
			{TranscationID: "tx-resume", ChunkSequenceNumber: intPtr(3), UploadSuccess: boolPtr(true)},
		},
	}
	u := &ChunkedUploader{apps: stub}

	session, err := u.Upload(
		context.Background(),
		bytes.NewReader(tail),
		totalSize,
		&UploadOptions{ChunkSize: chunkSize, ResumeFrom: priorSession},
	)
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	if len(stub.chunkCalls) != 1 {
		t.Fatalf("expected 1 chunk call on resume, got %d", len(stub.chunkCalls))
	}
	if stub.chunkCalls[0].TransactionID != "tx-resume" {
		t.Errorf("resumed chunk TransactionID: got %q, want %q", stub.chunkCalls[0].TransactionID, "tx-resume")
	}
	if got := derefInt(stub.chunkCalls[0].ChunkSequenceNumber); got != 3 {
		t.Errorf("resumed chunk sequence: got %d, want 3", got)
	}
	if session.LastSequence != 3 {
		t.Errorf("LastSequence after resume: got %d, want 3", session.LastSequence)
	}
}

func TestUpload_Progress_CalledAfterEachChunk(t *testing.T) {
	const totalSize = int64(3 * 1024)
	const chunkSize = int64(1024)
	body := bytes.Repeat([]byte("F"), int(totalSize))

	stub := &stubChunkClient{}
	u := &ChunkedUploader{apps: stub}

	type tick struct{ uploaded, total int64 }
	var ticks []tick
	progress := func(uploaded, total int64) {
		ticks = append(ticks, tick{uploaded, total})
	}

	_, err := u.Upload(
		context.Background(),
		bytes.NewReader(body),
		totalSize,
		&UploadOptions{ChunkSize: chunkSize, Progress: progress},
	)
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}

	wantTicks := []tick{
		{1024, 3072},
		{2048, 3072},
		{3072, 3072},
	}
	if len(ticks) != len(wantTicks) {
		t.Fatalf("expected %d progress ticks, got %d (%+v)", len(wantTicks), len(ticks), ticks)
	}
	for i, want := range wantTicks {
		if ticks[i] != want {
			t.Errorf("tick %d: got %+v, want %+v", i, ticks[i], want)
		}
	}
}

func TestUpload_ResumeFrom_ChunkSizeMismatch_Errors(t *testing.T) {
	stub := &stubChunkClient{}
	u := &ChunkedUploader{apps: stub}

	priorSession := &ChunkSession{
		TransactionID: "tx-mismatch",
		TotalSize:     2048,
		ChunkSize:     1024, // the resumed-from ChunkSession used 1 KiB chunks
		LastSequence:  1,
	}
	_, err := u.Upload(
		context.Background(),
		bytes.NewReader(bytes.Repeat([]byte("E"), 1024)),
		2048,
		&UploadOptions{ChunkSize: 512, ResumeFrom: priorSession}, // mismatched
	)
	if err == nil {
		t.Fatal("Upload should reject ResumeFrom with a different ChunkSize")
	}
	if len(stub.chunkCalls) != 0 {
		t.Errorf("expected 0 chunk calls on resume mismatch, got %d", len(stub.chunkCalls))
	}
}
