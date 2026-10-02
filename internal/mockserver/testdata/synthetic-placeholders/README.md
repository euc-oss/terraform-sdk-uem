# Synthetic placeholder fixtures (Phase-3-surviving location)

This directory holds honestly-labeled synthetic/unverified mock-response
fixtures for endpoints that have no live-tenant capture to draw on (fabricated
originals demoted by the 2026-08 integrity audit, endpoints permanently or
temporarily unreachable, or fixtures that were never anything but hand-typed
stubs). Each fixture's own `metadata.description` says so explicitly, and the
corresponding `tests/verified-endpoints.json` `unverified[]` entry carries the
same disclosure.

## Why here and NOT under the repo-root `testdata/` tree

The repo-root `testdata/mock-responses/` directory is expected to be removed
wholesale (`git rm testdata/`) once Phase 3 of the deterministic-capture
migration makes the vendored orphan branch (see
`docs/guides/vendored-swagger-sourcing.md`) the sole source of mock-response
fixtures. Any file placed under repo-root `testdata/` — however honestly
labeled — would be silently deleted in that sweep, taking its dependent
test's only fixture with it.

`internal/mockserver/testdata/synthetic-placeholders/` is a **different**
directory from the repo-root `testdata/` this note warns about. It is:

- Co-located with the `internal/mockserver` package that actually loads these
  fixtures (via `mockserver.LoadResponseFromFile`, the same override-blind
  bypass path documented in `internal/mockserver/bypass_governance_test.go`
  and `tests/bypass_fixture_helper_test.go`), not a repo-root `testdata/`
  directory at all -- so a repo-root `git rm testdata/` does not touch it.
- Explicitly un-ignored by `.gitignore`'s `/internal/*` blanket-ignore rule:
  `.gitignore` carves out `!/internal/mockserver/` and
  `!/internal/mockserver/**` specifically because `internal/mockserver` is
  hand-written (everything else under `internal/` is generated code). This
  directory inherits that carve-out, so it is a normal, committed, tracked
  part of the repo -- not gitignored, not treated as ephemeral build output.
- Not read by anything that respects `MOCK_RESPONSES_DIR` /
  `mockserver.ResolveResponsesDir` (see `internal/mockserver/loader.go`):
  fixtures here are only ever loaded via a literal path passed straight to
  `mockserver.LoadResponseFromFile`, so setting `MOCK_RESPONSES_DIR` to point
  at a vendored/staged fixture tree can never redirect or hide them. That is
  the whole point -- these fixtures exist precisely for endpoints the
  vendored tree does not (yet, or ever) have.

## Layout

Sub-paths mirror the repo-root `testdata/mock-responses/<service>/<file>.json`
layout the fixtures were originally modeled on or copied from, purely for
readability/traceability -- there is no code that depends on the sub-path
shape itself, only on the literal path each bypass call site hard-codes.

## Governance

Most fixtures here have a corresponding `unverified[]` entry in
`tests/verified-endpoints.json` explaining why they are synthetic and what,
if anything, would be needed to replace them with a genuine live capture. A
subset, however, were later recaptured for real and now back a `verified[]`
row instead, each with an explicit per-field disclosure reason (e.g. a
hand-anonymized id alongside otherwise-genuine wire content) -- see
`tests/verified-endpoints.json` for which row governs which fixture.
`internal/mockserver/bypass_governance_test.go` guards that these fixtures'
bypass call sites stay in sync with a committed manifest
(`tools/internal/capturetools/testdata/bypass-loaded-fixtures.json`) and that
each cited fixture's ledger entry is well-formed.
