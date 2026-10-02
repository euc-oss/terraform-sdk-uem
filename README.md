# Workspace ONE UEM Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/euc-oss/terraform-sdk-uem/v26.svg)](https://pkg.go.dev/github.com/euc-oss/terraform-sdk-uem/v26)
[![Go Report Card](https://goreportcard.com/badge/github.com/euc-oss/terraform-sdk-uem)](https://goreportcard.com/report/github.com/euc-oss/terraform-sdk-uem)
[![CI](https://github.com/euc-oss/terraform-sdk-uem/actions/workflows/ci.yml/badge.svg)](https://github.com/euc-oss/terraform-sdk-uem/actions/workflows/ci.yml)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

Typed Go client for the Workspace ONE UEM REST API.

## What this is

This is a Go SDK that wraps the Workspace ONE UEM REST API with authentication,
retry, rate limiting, and platform-aware endpoint routing. It is designed to
be used directly from Go applications and as the underlying client for the
Workspace ONE UEM Terraform provider.

This is **not a Terraform provider**. The Terraform provider lives in a
separate repository and consumes this SDK.

This SDK is maintained by Omnissa.

## Status

The module follows the Workspace ONE UEM release line it targets: version
`v26.2.0-beta.1` is the first beta of the UEM 26.2 line, and its module path ends in
`/v26`. See [Versioning](#versioning) and [CHANGELOG.md](CHANGELOG.md).

- **Go version:** 1.25 or later
- **Workspace ONE UEM:** cloud and on-premises deployments are both supported
- **API versions covered:** v1, v2, and v4 (depending on resource — see [docs/reference/platform-support.md](docs/reference/platform-support.md))

## Installation

```bash
go get github.com/euc-oss/terraform-sdk-uem/v26@v26.2.0-beta.1
```

Add an import:

```go
import wsone "github.com/euc-oss/terraform-sdk-uem/v26"
```

## Quickstart

The example below authenticates with OAuth2, creates a client, and lists all
profiles. `wsone.ListProfiles` returns a flat `[]*models.Profile` slice; use
`wsone.ListOptions` to paginate.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    wsone "github.com/euc-oss/terraform-sdk-uem/v26"
)

func main() {
    auth, err := wsone.NewOAuth2Auth(wsone.OAuth2Config{
        ClientID:     os.Getenv("WSONE_CLIENT_ID"),
        ClientSecret: os.Getenv("WSONE_CLIENT_SECRET"),
        TokenURL:     os.Getenv("WSONE_TOKEN_URL"),
    })
    if err != nil {
        log.Fatalf("auth: %v", err)
    }

    client, err := wsone.NewClient(wsone.Config{
        BaseURL:    os.Getenv("WSONE_API_URL"),
        Auth:       auth,
        TenantCode: os.Getenv("WSONE_TENANT_CODE"),
    })
    if err != nil {
        log.Fatalf("client: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    profiles, err := wsone.ListProfiles(ctx, client, nil)
    if err != nil {
        log.Fatalf("list profiles: %v", err)
    }

    for _, p := range profiles {
        fmt.Printf("%d  %s  (%s)\n", p.GetProfileID(), p.GetName(), p.Platform)
    }
}
```

A runnable version of this snippet is in [examples/quickstart](examples/quickstart/main.go).

## Authentication

The SDK supports two authentication providers:

- **OAuth2 (recommended)** — works for both cloud and on-premises deployments. Cloud deployments use a separate auth host (`https://na.uemauth.workspaceone.com/connect/token`); on-prem deployments typically use the deployment host (`https://<host>/oauth/token`). The SDK auto-detects when given a base URL alone, but you can pass an explicit `TokenURL` for fully-qualified setups.
- **Basic auth** — username + password. Available for legacy deployments where OAuth2 is not configured.

The tenant code (`aw-tenant-code` header) is always wired through `wsone.Config.TenantCode`,
not through the auth constructor. This applies to both OAuth2 and Basic auth.

**OAuth2:**

```go
auth, err := wsone.NewOAuth2Auth(wsone.OAuth2Config{
    ClientID:     id,
    ClientSecret: secret,
    TokenURL:     "https://na.uemauth.workspaceone.com/connect/token",
})
if err != nil {
    return err
}

client, err := wsone.NewClient(wsone.Config{
    BaseURL:    apiURL,
    Auth:       auth,
    TenantCode: tenantCode, // ← tenant code goes here for both auth methods
})
if err != nil {
    return err
}
```

**Basic auth:**

```go
auth := wsone.NewBasicAuth(user, pass)

client, err := wsone.NewClient(wsone.Config{
    BaseURL:    apiURL,
    Auth:       auth,
    TenantCode: tenantCode,
})
if err != nil {
    return err
}
```

See [docs/authentication.md](docs/authentication.md) for the full guide.

## Supported resources

Every service below is a type of the root package (`wsone.<Service>` for the name in
the first column), built on the same client as the rest of the SDK. The Python package
exposes the same services under the same names without the `Service` suffix (all but
`ProfileService`, the platform-aware convenience client). A service
named for an API version (`V1`, `V2`, `V4`) wraps the routes of that version; the
version is chosen per operation, so you call the method, not the version.

### Device management (MDM)

| Service | What it covers |
| ------- | -------------- |
| `ProfileService` | Platform-aware profile client: get, create (Android, Apple iOS, macOS, Windows 10), update (all six platforms) and delete, discovering each profile's platform |
| `ProfilesV2Service` | Get profile details and search; create and update for Android, Apple iOS, macOS and Windows 10; Windows Rugged: update only (UEM has no create route for it) |
| `ProfilesV4Service` | Create and update Linux profiles |
| `ProfilesV1Service` | Delete a profile (the delete route exists only at API v1), search, upload a certificate |
| `SmartGroupsService` | Create, load, update, delete and search smart groups |
| `DeviceSensorsV1Service` | Get, list, create, update and bulk delete sensors; assign a sensor |
| `DeviceSensorsV2Service` | Get, list, create and update sensors; sensor assignments (get, list, add, update, delete, bulk re-rank) |
| `DeviceSensorsService` | The sensors of one device |
| `ScriptsV1Service` | Create, get, list by organization group, replace the definition, bulk delete; script samples |
| `ScriptAssignmentV1Service` | Get, list, add and bulk update script assignments |
| `BaselinesV1Service` | Get, list, clone, update, delete and assign baselines; assignments, status, devices and per-device policy compliance |
| `TemplatesV1Service` | Baseline templates: list, search, get a policy |
| `CatalogsV1Service` | Baseline policy catalogs: get the catalog, list all policies, get a policy |
| `UpdatesV1Service` | OS update deployments (create, update, delete, bulk update) and device update status, readiness and details |
| `OSVersionsV1Service` | Active OS versions, all platforms or one platform |
| `PlatformsV1Service` | Active platforms |

Windows Rugged profiles have no create operation anywhere in the SDK; every other
platform (Android, Apple iOS, macOS, Windows 10, Linux) can be created, read, updated
and deleted. Linux profiles use API v4.

### Application management (MAM)

| Service | What it covers |
| ------- | -------------- |
| `InternalAppsV1Service` | Create an internal app from a blob, upload it in chunks, get, delete, and manage its assignments |
| `InternalAppsV2Service` | Get by UUID, list, branch cache statistics, renew a provisioning profile |
| `AppsV2Service` | Search apps, assignment rules, categories, Android custom tracks, filter values, configuration template, Office 365 policy |
| `MacOsAppsV1Service` | Create a macOS application |
| `PurchasedAppsV1Service` | Purchased (VPP) app search |
| `PurchasedAppsV2Service` | Get a purchased app with its assignments; install it on and remove it from a device |
| `AppBookmarksV2Service` | List kiosk bookmarks |
| `EnterpriseAppRepositoryV2Service` | Search, bulk search, package details, import a package |
| `AppRemovalProtectionLogsV2Service` | Removal protection logs, device reports, the removal threshold |
| `BlobsV1Service` | Upload and delete a blob (delete takes the numeric id) |
| `BlobsV2Service` | Upload, get, head and delete a blob (by UUID) |

### System

| Service | What it covers |
| ------- | -------------- |
| `OrganizationGroupsService` | Read organization groups: get one, list its children and parents, search (read-only) |

The full list of operations per service is in the package documentation on
[pkg.go.dev](https://pkg.go.dev/github.com/euc-oss/terraform-sdk-uem/v26); platform behaviour is in
[docs/reference/platform-support.md](docs/reference/platform-support.md).

## Configuration

The client supports several configuration knobs. See [docs/configuration.md](docs/configuration.md)
for the complete list. Common ones:

- **Custom HTTP client** — pass your own `*http.Client` to inject proxies, custom TLS, or mock transports
- **Retry policy** — built on `hashicorp/go-retryablehttp`; configurable backoff and retry budget
- **Rate limiting** — token-bucket limiter (`golang.org/x/time/rate`) configurable per-client
- **Context cancellation** — every API call accepts `context.Context`; pass timeouts and cancellation through normally

## Error handling

API errors are returned as `*wsone.APIError`, which exposes the HTTP status
code, an error message parsed from the response, and a structured error code
where available:

```go
profiles, err := wsone.ListProfiles(ctx, client, nil)
if err != nil {
    var apiErr *wsone.APIError
    if errors.As(err, &apiErr) {
        if apiErr.IsRetryable() {
            // The SDK already retried automatically; reaching here means
            // the retry budget was exhausted. Retryable codes are:
            // 429 (rate limit), 500, 502, 503, 504.
        }
        log.Printf("API error %d: %s", apiErr.StatusCode, apiErr.Message)
    }
    return err
}
log.Printf("found %d profiles", len(profiles))
```

See [docs/error-handling.md](docs/error-handling.md) for full details.

## Documentation

- [Installation](docs/installation.md)
- [Quickstart](docs/quickstart.md)
- [Authentication](docs/authentication.md)
- [Configuration](docs/configuration.md)
- [Error handling](docs/error-handling.md)
- [Profiles guide](docs/guides/profiles.md)
- [Platform support reference](docs/reference/platform-support.md)
- [Troubleshooting](docs/troubleshooting.md)

API reference: [pkg.go.dev/github.com/euc-oss/terraform-sdk-uem/v26](https://pkg.go.dev/github.com/euc-oss/terraform-sdk-uem/v26).

## Examples

Each directory in [examples/](examples/) is a runnable program demonstrating
one feature:

- [quickstart](examples/quickstart) — minimum path: OAuth2 → list profiles
- [auth-oauth2](examples/auth-oauth2) — cloud and on-prem OAuth2
- [auth-basic](examples/auth-basic) — basic auth
- [profile-crud](examples/profile-crud) — full create / read / update / delete
- [pagination](examples/pagination) — iterating through paginated lists
- [smart-groups](examples/smart-groups) — assignment management
- [error-handling](examples/error-handling) — type-asserting and reacting to errors
- [custom-http-client](examples/custom-http-client) — proxies, custom transports

Run any example:

```bash
cd examples/quickstart
WSONE_CLIENT_ID=... WSONE_CLIENT_SECRET=... WSONE_TOKEN_URL=... WSONE_API_URL=... go run .
```

## Testing your integration

The SDK ships with an in-process mock server (`internal/mockserver`) and a
collection of JSON fixtures in `testdata/`. These exist to support the SDK's
own test suite and the Example godoc functions.

> **Note:** The bundled mock server lives under `internal/mockserver/` and is
> not directly importable from consumer packages (Go's `internal` rule). It
> exists for the SDK's own tests and Example godoc functions. Consumers who
> want a similar testing setup should use Go's `httptest` package with their
> own response fixtures; see [examples/error-handling](examples/error-handling/)
> for a starting point on wiring a custom HTTP transport.

The JSON fixtures under `testdata/mock-responses/` are a useful reference for
the exact request and response shapes the API expects. The mock server itself
is stateful for profile CRUD — it accepts creates, tracks IDs in memory, and
returns 404 on reads after a delete — which mirrors real API behavior without
network access.

## Versioning

Releases are tagged `v26.<minor>.<patch>`, with `-beta.N` on a beta
(`v26.2.0-beta.1`, then `v26.2.0`), on the release branch of the UEM version they
target. The first number is the UEM major version, and Go's rule for major versions
applies: from v2 on, the module path ends in `/vN`. This module is
`github.com/euc-oss/terraform-sdk-uem/v26`, so every import and every `go get` carries the `/v26`:

```text
import wsone "github.com/euc-oss/terraform-sdk-uem/v26"
import "github.com/euc-oss/terraform-sdk-uem/v26/client"
```

A line for UEM 27 will be a new module path ending in `/v27`; both can be required
side by side. Within a line, a patch release changes no API and a minor release adds API; a
breaking change to the Go API ships in a minor release and is listed under "Breaking"
in [CHANGELOG.md](CHANGELOG.md), so pin the version you have tested.

See [CHANGELOG.md](CHANGELOG.md) for release history.

## Reporting issues

Two channels:

- **GitHub issues** — bugs, feature requests, and questions about the SDK itself. See the [issue templates](.github/ISSUE_TEMPLATE/).
- **[Omnissa Customer Connect](https://customerconnect.omnissa.com/home)** — production support questions about Workspace ONE UEM or about using this SDK in production.

## Security

Please **do not** file public GitHub issues for security vulnerabilities.
See [SECURITY.md](SECURITY.md) for the disclosure process.

## Contributing

This SDK is currently maintained by the internal Omnissa team. GitHub issues
are very welcome. We are not actively soliciting external pull requests at
this time, but we evaluate any that arrive — see [CONTRIBUTING.md](CONTRIBUTING.md)
for the attribution model and DCO sign-off requirement.

## License

This SDK is licensed under the [Apache License, Version 2.0](LICENSE).
For Omnissa licensing terms generally, see the [Omnissa Legal Center](https://www.omnissa.com/legal-center/)
and the [Open Source Notices](https://www.omnissa.com/open-source-notices/) page.

## Copyright and trademarks

Copyright © Omnissa, LLC. All rights reserved. This product is protected
by copyright and intellectual property laws in the United States and other
countries as well as by international treaties. Omnissa products are
covered by one or more patents listed at:
<https://www.omnissa.com/omnissa-patent-information/>. Omnissa products
are also covered by general and offering-specific legal terms, as well as
the privacy and open-source software notices hosted on the Omnissa Legal
Center at: <https://www.omnissa.com/legal-center/>.

Omnissa, the Omnissa Logo, and Workspace ONE are registered trademarks or
trademarks of Omnissa in the United States and other jurisdictions. All
other marks and names mentioned herein may be trademarks of their respective
companies. "Omnissa" refers to Omnissa, LLC, Omnissa International Unlimited
Company, and/or their subsidiaries.

This SDK is provided by Omnissa. It is not affiliated with HashiCorp.
