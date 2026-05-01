# Releasing

This document describes the manual release process for the Workspace ONE UEM
Go SDK. Automated release tooling is planned for v1.0.

## When to release

A release is cut when the internal source pipeline produces a sync commit
that the maintainers determine warrants a new public version. There is no
fixed cadence at v0.

## Release checklist

1. Confirm `[Unreleased]` in [CHANGELOG.md](CHANGELOG.md) describes all
   user-visible changes since the prior release.
2. Choose the next version number per [semver](https://semver.org/):
   - **Patch** (`v0.x.Y` → `v0.x.Y+1`): bug fixes only, no API changes.
   - **Minor** (`v0.X.0` → `v0.X+1.0`): new functionality, additive API
     changes. While at v0, minor versions MAY include breaking changes.
   - **Major** (`v0.X.Y` → `v1.0.0`): API stability commitment.
3. Bump the `Version` constant in `version.go` to match.
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
   curl -s "https://proxy.golang.org/github.com/euc-oss/terraform-sdk-uem/@v/vX.Y.Z.info"
   ```

## Out-of-band patch releases

For security fixes, the same checklist applies, but the release PR can be
expedited and the public CHANGELOG entry should reference the [SECURITY
advisory](SECURITY.md) without disclosing exploitation details.
