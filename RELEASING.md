# Releasing

This document describes the manual release process for the Workspace ONE UEM
Go SDK.

## Versions and tags

- The Go module path is `github.com/euc-oss/terraform-sdk-uem/v26`: the major-version suffix is the UEM major
  version (`/v26` for UEM 26.x; UEM 27 will publish `/v27`). Go requires the
  suffix from v2 on, so every import and every `go get` carries it. The sync derives
  it from the release version, so nothing is edited by hand.
- Tags are `v26.<minor>.<patch>`, with `-beta.N` on a beta
  (`v26.2.0-beta.1` -> `v26.2.0` -> `v26.2.1`), and are created on the release branch of
  the UEM version they target. Three parts only: the internal four-part build number is
  never a tag.
- A tag is never moved or reused; a bad release is superseded by the next patch.

## When to release

A release is cut when the internal source pipeline produces a sync commit
that the maintainers determine warrants a new public version. There is no
fixed cadence.

## Release checklist

1. Confirm `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md) describes all
   user-visible changes since the prior release.
2. Choose the next version number (`v26.<minor>.<patch>[-beta.N]`):
   - **Patch** (`v26.2.0` -> `v26.2.1`): bug fixes only, no API changes.
   - **Minor** (`v26.2.0` -> `v26.3.0`): new functionality and additive API
     changes; a breaking change to the Go API also ships in a minor.
   - **Beta** (`v26.2.0-beta.1` -> `v26.2.0-beta.2`): a pre-release of the next
     version; its API may still change before the final tag.
   A breaking change is recorded under "Breaking" in the CHANGELOG.
3. The `Version` constant in `version.go` needs no manual bump: the sync renders it
   from its `--version` flag (the 3-part form, e.g. `26.2.0`) each time it assembles
   the tree, so it carries the internal release version of the build, which is not
   necessarily the `vX.Y.Z` tag chosen in step 2.
4. Move CHANGELOG `[Unreleased]` content into a new `[vX.Y.Z] - YYYY-MM-DD`
   section. Leave a fresh empty `[Unreleased]` above it.
5. Open a release PR titled `Release vX.Y.Z`. The PR body should be a copy
   of the new CHANGELOG section.
6. Once merged, tag the merge commit:

   ```bash
   git tag -s vX.Y.Z -m "Release vX.Y.Z"
   git push origin vX.Y.Z
   ```

7. Publish a GitHub release for the tag. The release body should be a copy
   of the CHANGELOG section.
8. Verify pkg.go.dev picks up the new version (allow up to 30 minutes):

   ```bash
   curl -s "https://proxy.golang.org/github.com/euc-oss/terraform-sdk-uem/v26/@v/vX.Y.Z.info"
   ```

   and that a consumer resolves it: `go get github.com/euc-oss/terraform-sdk-uem/v26@vX.Y.Z`.

## Out-of-band patch releases

For security fixes, the same checklist applies, but the release PR can be
expedited and the public CHANGELOG entry should reference the [SECURITY
advisory](SECURITY.md) without disclosing exploitation details.
