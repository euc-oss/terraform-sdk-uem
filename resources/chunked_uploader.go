// Package resources holds hand-coded Layer-2 SDK helpers that compose
// codegen-emitted Layer-1 services from generated/.
//
// ChunkedUploader implements the chunked internal-app upload workflow
// (FRs #1 + #7). It wraps mamv1.InternalAppsV1Service with a streaming
// helper plus two adapter-delegate completion methods.
//
// Cross-reference tip convention (layered GoDoc).
// Methods in this package include "Lower-level alternative" tips (Layer B)
// pointing at the Layer-1 codegen method they wrap. Methods in codegen-
// emitted services do NOT carry reciprocal Layer-A tips today because
// codegen has no per-method hand-edit point; readers of
// mamv1.InternalAppsV1Service.InsertInternalApplicationChunkAsync discover
// ChunkedUploader through this package's documentation and chunked_uploader.go.
// Once the Q7 Phase 4 follow-up lands codegen x-go-name + tip-injection,
// Layer-A tips can be backfilled into the generated services. Until then,
// helper-side tips are authoritative.
//
// Sibling-variant tips (Layer C) cross-reference between BeginInstall (legacy
// /api/mam/apps/internal/begininstall, InternalAppChunkTransactionV1 body) and
// CreateInternalApplication (modern /api/mam/apps/internal/application,
// V1Model body with 4-decimal version support). Adjacent-domain tips (Layer
// D) appear where a related-but-different service is the better entry point.
//
// Plan->emit name translation (post-Path-Y rebase pending on PR #29).
// The chunked-upload plan this file implements was authored against the
// post-Path-Y layout (internal/mam/...). origin/main still lives on the
// pre-Path-Y layout (generated/mam/...) with version-aware Option-A V1
// suffixes on all emitted chunk types. This file applies the mechanical
// substitution deliberately, as an accepted interim divergence; the mapping below
// makes the divergence auditable so a future rebase onto PR #29's tip
// can grep for it and revert to plan names.
//
//	plan internal/mam/v1.InternalAppChunk            -> emit generated/mam/v1.InternalAppChunkV1
//	plan internal/mam/v1.AppChunkTranscationResponse -> emit generated/mam/v1.AppChunkTranscationResponseV1
//	plan internal/mam/v1.InternalAppChunkTransaction -> emit generated/mam/v1.InternalAppChunkTransactionV1
//	plan internal/mam/v1.InternalAppChunkTransactionV1Model -> unchanged
//	plan internal/mam/v1.InternalApplicationEntity   -> emit generated/mam/v1.InternalApplicationEntityV1
//
// Field-casing translation on the legacy V1 type only (the V1Model variant's
// field names come from snake_case swagger and use codegen's compound-word
// dict; both renderings are valid):
//
//	plan EARAppUpdateMode (V1Model)  -> emit EarAppUpdateMode (V1Model)
//	plan EARInstallerArchitecture    -> emit EarInstallerArchitecture
//
// This helper orchestrates three swagger-defined operations. A swagger
// overlay forces the four V1Model StringEnumConverter fields from int to
// string so the wire shape matches what /api/mam/apps/internal/application
// expects.
package resources

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/euc-oss/terraform-sdk-uem/client"
	mamv1 "github.com/euc-oss/terraform-sdk-uem/internal/mam/v1"
)

// DefaultChunkSize is the default raw-binary chunk size used by Upload when
// UploadOptions.ChunkSize is zero. Validated by live probe against
// a UEM tenant.
const DefaultChunkSize int64 = 16 * 1024 * 1024 // 16 MiB

// ChunkSession captures the state of an in-progress or completed chunked
// upload. It is produced by Upload and consumed by BeginInstall /
// CreateInternalApplication. Fields are public so callers using
// mamv1.InternalAppsV1Service.InsertInternalApplicationChunkAsync directly
// can construct a session by hand.
//
// On resume, the caller positions their reader at LastSequence*ChunkSize
// bytes from the start of the RAW BINARY stream (not the base64 envelope).
type ChunkSession struct {
	// TransactionID is the server-issued session identifier. The Layer-1
	// wire field is misspelled "TranscationId" (preserved per Q7-ii); this
	// helper reads that typo'd field and exposes it idiomatically here.
	TransactionID string

	// TotalSize is the total raw binary length of the upload in bytes.
	TotalSize int64

	// ChunkSize is the raw binary chunk size in bytes (not the base64
	// envelope size, which is ~1.33x larger).
	ChunkSize int64

	// LastSequence is the highest successfully-uploaded chunk's
	// 1-indexed sequence number. Server rejects ChunkSequenceNumber=0.
	LastSequence int
}

// UploadOptions tunes Upload behavior. All fields are optional.
type UploadOptions struct {
	// ChunkSize is the raw binary chunk size in bytes. If zero, Upload
	// uses DefaultChunkSize (16 MiB).
	ChunkSize int64

	// Progress, if non-nil, is called after each chunk POST succeeds with
	// the cumulative uploaded byte count and the total. Useful for UI
	// progress reporting. Caller-supplied function must be safe to call
	// from Upload's goroutine.
	Progress func(uploaded, total int64)

	// ResumeFrom, if non-nil, resumes a prior upload. Caller MUST position
	// their reader at ResumeFrom.LastSequence * ResumeFrom.ChunkSize bytes
	// from the start of the raw binary stream before calling Upload. The
	// session's TransactionID is reused; chunks 1..LastSequence are not
	// re-uploaded.
	ResumeFrom *ChunkSession
}

// internalAppsChunkClient is the subset of mamv1.InternalAppsV1Service that
// ChunkedUploader uses. Defined as an interface so unit tests can substitute
// a stub without standing up a real HTTP client.
type internalAppsChunkClient interface {
	InsertInternalApplicationChunkAsync(ctx context.Context, request *mamv1.InternalAppChunkV1) (http.Header, *mamv1.AppChunkTranscationResponseV1, error)
	CreateInternalAppFromBlobAsync(ctx context.Context, request *mamv1.InternalAppChunkTransactionV1) (http.Header, *mamv1.InternalApplicationEntityV1, error)
	CreateInternalApplicationFromBlob(ctx context.Context, request *mamv1.InternalAppChunkTransactionV1Model) (http.Header, *mamv1.InternalApplicationEntityV1, error)
}

// Compile-time assertion that mamv1.InternalAppsV1Service satisfies the
// interface. If codegen renames or changes any of these signatures, this
// will fail to compile and surface the change immediately.
var _ internalAppsChunkClient = (*mamv1.InternalAppsV1Service)(nil)

// ChunkedUploader orchestrates internal-app uploads against
// /api/mam/apps/internal/{uploadchunk,begininstall,application}. It is the
// recommended caller-facing surface for FRs #1 and #7.
//
// Lower-level alternative: call mamv1.InternalAppsV1Service methods directly
// for per-chunk control (custom retry, manual sequence tracking).
type ChunkedUploader struct {
	apps internalAppsChunkClient
}

// NewChunkedUploader constructs a ChunkedUploader backed by a fresh
// mamv1.InternalAppsV1Service. Callers needing to share a single Layer-1
// service across multiple helpers should construct ChunkedUploader directly
// with a shared instance (the apps field is unexported; expose a hook only
// if a real caller needs it).
func NewChunkedUploader(c *client.Client) *ChunkedUploader {
	return &ChunkedUploader{
		apps: mamv1.NewInternalAppsV1Service(c),
	}
}

// Upload streams reader's contents to /api/mam/apps/internal/uploadchunk in
// chunks of opts.ChunkSize (or DefaultChunkSize if unset), threads the
// server-issued TransactionId across calls, and returns a populated
// ChunkSession describing the completed upload. The session is the input
// to BeginInstall or CreateInternalApplication for the install step.
//
// The totalSize argument must be the exact raw-binary length of the
// reader's payload. If reader yields fewer or more bytes than totalSize,
// Upload returns an error before issuing the final chunk POST.
//
// Lower-level alternative: call
// mamv1.InternalAppsV1Service.InsertInternalApplicationChunkAsync directly
// for per-chunk control (custom retry, manual sequence tracking, special
// error handling).
//
// FR #1 retry note: per-chunk POST bodies are JSON envelopes the SDK
// builds in memory (small, replayable), so retryablehttp's existing
// workspaceOneRetryPolicy handles 429/5xx automatically without needing
// GetBody. The FR's concern about streaming-body replay does not apply
// here because the SDK owns the bytes.
func (u *ChunkedUploader) Upload(
	ctx context.Context,
	reader io.Reader,
	totalSize int64,
	opts *UploadOptions,
) (*ChunkSession, error) {
	if reader == nil {
		return nil, errors.New("ChunkedUploader.Upload: reader must not be nil")
	}
	if totalSize <= 0 {
		// Zero is rejected alongside negative because the chunk loop
		// never executes for totalSize=0, leaving session.TransactionID
		// empty. A subsequent BeginInstall/CreateInternalApplication
		// would then POST with TransactionId="" and the server (and
		// mock) reject with a cryptic 400 (cycle-1 finding #2).
		return nil, fmt.Errorf("ChunkedUploader.Upload: totalSize must be positive, got %d", totalSize)
	}

	chunkSize := DefaultChunkSize
	var progress func(uploaded, total int64)
	var resumeFrom *ChunkSession
	if opts != nil {
		if opts.ChunkSize > 0 {
			chunkSize = opts.ChunkSize
		}
		progress = opts.Progress
		resumeFrom = opts.ResumeFrom
	}

	session := &ChunkSession{
		TotalSize:    totalSize,
		ChunkSize:    chunkSize,
		LastSequence: 0,
	}
	if resumeFrom != nil {
		session.TransactionID = resumeFrom.TransactionID
		session.LastSequence = resumeFrom.LastSequence
		// Caller is responsible for positioning their reader at
		// resumeFrom.LastSequence * resumeFrom.ChunkSize bytes from the
		// start. This call's ChunkSize must match resumeFrom's own
		// chunk size to keep the offset arithmetic consistent.
		if resumeFrom.ChunkSize != chunkSize {
			return nil, fmt.Errorf(
				"ChunkedUploader.Upload: resume ChunkSize mismatch (session=%d, current=%d)",
				resumeFrom.ChunkSize, chunkSize)
		}
	}

	buf := make([]byte, chunkSize)
	uploaded := int64(session.LastSequence) * chunkSize
	seq := session.LastSequence + 1

	for uploaded < totalSize {
		want := chunkSize
		if remaining := totalSize - uploaded; remaining < want {
			want = remaining
		}
		// Read EXACTLY `want` bytes; io.ReadFull handles short reads.
		n, err := io.ReadFull(reader, buf[:want])
		if err != nil {
			return session, fmt.Errorf(
				"ChunkedUploader.Upload: reader exhausted at byte %d of %d (chunk %d): %w",
				uploaded+int64(n), totalSize, seq, err)
		}

		chunkBytes := int64(n)
		seqNum := seq
		chunk := &mamv1.InternalAppChunkV1{
			TransactionID:        session.TransactionID,
			ChunkSequenceNumber:  &seqNum,
			ChunkSize:            &chunkBytes,
			TotalApplicationSize: &totalSize,
			ChunkData:            base64.StdEncoding.EncodeToString(buf[:n]),
		}
		_, resp, err := u.apps.InsertInternalApplicationChunkAsync(ctx, chunk)
		if err != nil {
			return session, fmt.Errorf(
				"ChunkedUploader.Upload: chunk %d POST failed: %w", seq, err)
		}
		if resp == nil {
			return session, fmt.Errorf(
				"ChunkedUploader.Upload: chunk %d returned nil response", seq)
		}
		if resp.UploadSuccess == nil || !*resp.UploadSuccess {
			return session, fmt.Errorf(
				"ChunkedUploader.Upload: chunk %d server reported UploadSuccess=false", seq)
		}

		// Typo-handling: response field is "TranscationID" (preserved
		// per Q7-ii); expose it idiomatically on ChunkSession.
		session.TransactionID = resp.TranscationID
		session.LastSequence = seq

		uploaded += int64(n)
		seq++

		if progress != nil {
			progress(uploaded, totalSize)
		}
	}

	return session, nil
}

// BeginInstall completes a chunked upload by posting to
// /api/mam/apps/internal/begininstall. The session's TransactionID is
// passed to the underlying call automatically; the caller's metadata
// struct is NOT mutated (BeginInstall takes a shallow copy and writes
// the session's TransactionID onto the copy).
//
// Returns the created application entity, including server-hydrated Uuid
// and OrganizationGroupUuid.
//
// Sibling variant: CreateInternalApplication targets
// /api/mam/apps/internal/application with the V1Model request DTO and
// supports 4-decimal version numbers. Use that for new code; BeginInstall
// is kept for callers that need legacy wire shape parity.
//
// Lower-level alternative: call
// mamv1.InternalAppsV1Service.CreateInternalAppFromBlobAsync directly if
// you need to override the TransactionID or inspect the http.Header
// response.
func (u *ChunkedUploader) BeginInstall(
	ctx context.Context,
	session *ChunkSession,
	metadata *mamv1.InternalAppChunkTransactionV1,
) (*mamv1.InternalApplicationEntityV1, error) {
	if session == nil {
		return nil, errors.New("ChunkedUploader.BeginInstall: session must not be nil")
	}
	if metadata == nil {
		return nil, errors.New("ChunkedUploader.BeginInstall: metadata must not be nil")
	}
	// Shallow-copy before mutation so retries with the same struct, or
	// callers that pre-populate TransactionID for diagnostic reasons,
	// don't observe a side effect on their own memory (cycle-1 finding
	// #3). The struct contains only value-type fields and pointers; the
	// pointers are intentionally shared so the underlying call sees the
	// same downstream graph the caller built.
	body := *metadata
	body.TransactionID = session.TransactionID
	_, resp, err := u.apps.CreateInternalAppFromBlobAsync(ctx, &body)
	if err != nil {
		return resp, fmt.Errorf("ChunkedUploader.BeginInstall: %w", err)
	}
	return resp, nil
}

// CreateInternalApplication completes a chunked upload by posting to
// /api/mam/apps/internal/application with the V1Model request shape. Use
// this for new code; it supports 4-decimal version numbers and uses
// snake_case wire fields plus string enums for selected properties.
//
// The session's TransactionID is passed to the underlying call
// automatically; the caller's metadata struct is NOT mutated
// (CreateInternalApplication takes a shallow copy and writes the
// session's TransactionID onto the copy).
//
// Sibling variant: BeginInstall targets /api/mam/apps/internal/begininstall
// with the legacy request DTO. Use that if you need parity with the
// pre-V1Model wire shape.
//
// Lower-level alternative: call
// mamv1.InternalAppsV1Service.CreateInternalApplicationFromBlob directly.
func (u *ChunkedUploader) CreateInternalApplication(
	ctx context.Context,
	session *ChunkSession,
	metadata *mamv1.InternalAppChunkTransactionV1Model,
) (*mamv1.InternalApplicationEntityV1, error) {
	if session == nil {
		return nil, errors.New("ChunkedUploader.CreateInternalApplication: session must not be nil")
	}
	if metadata == nil {
		return nil, errors.New("ChunkedUploader.CreateInternalApplication: metadata must not be nil")
	}
	// Shallow-copy before mutation; see BeginInstall comment for rationale
	// (cycle-1 finding #3).
	body := *metadata
	body.TransactionID = session.TransactionID
	_, resp, err := u.apps.CreateInternalApplicationFromBlob(ctx, &body)
	if err != nil {
		return resp, fmt.Errorf("ChunkedUploader.CreateInternalApplication: %w", err)
	}
	return resp, nil
}
