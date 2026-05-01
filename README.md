# Workspace ONE UEM Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/euc-oss/terraform-sdk-uem.svg)](https://pkg.go.dev/github.com/euc-oss/terraform-sdk-uem)
[![Go Report Card](https://goreportcard.com/badge/github.com/euc-oss/terraform-sdk-uem)](https://goreportcard.com/report/github.com/euc-oss/terraform-sdk-uem)
[![CI](https://github.com/euc-oss/terraform-sdk-uem/actions/workflows/ci.yml/badge.svg)](https://github.com/euc-oss/terraform-sdk-uem/actions/workflows/ci.yml)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

Typed Go client for the Workspace ONE UEM REST API.

> Badge URLs reference `github.com/euc-oss/terraform-sdk-uem` and `euc-oss/terraform-sdk-uem` placeholders that are
> substituted at sync time before this README ships in the public repository.

## What this is

This is a Go SDK that wraps the Workspace ONE UEM REST API with authentication,
retry, rate limiting, and platform-aware endpoint routing. It is designed to
be used directly from Go applications and as the underlying client for the
Workspace ONE UEM Terraform provider.

This is **not a Terraform provider**. The Terraform provider lives in a
separate repository and consumes this SDK.

This SDK is maintained by Omnissa.

## Status

The SDK is pre-1.0. The public API may change between minor versions until
v1.0 ships. See [CHANGELOG.md](CHANGELOG.md) for release history.

- **Go version:** 1.25 or later
- **Workspace ONE UEM:** cloud and on-premises deployments are both supported
- **API versions covered:** v1, v2, and v4 (depending on resource — see [docs/reference/platform-support.md](docs/reference/platform-support.md))

## Installation

```bash
go get github.com/euc-oss/terraform-sdk-uem
```

Add an import:

```go
import wsone "github.com/euc-oss/terraform-sdk-uem"
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

    wsone "github.com/euc-oss/terraform-sdk-uem"
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
auth, _ := wsone.NewOAuth2Auth(wsone.OAuth2Config{
    ClientID:     id,
    ClientSecret: secret,
    TokenURL:     "https://na.uemauth.workspaceone.com/connect/token",
})

client, _ := wsone.NewClient(wsone.Config{
    BaseURL:    apiURL,
    Auth:       auth,
    TenantCode: tenantCode, // ← tenant code goes here for both auth methods
})
```

**Basic auth:**

```go
auth := wsone.NewBasicAuth(user, pass)

client, _ := wsone.NewClient(wsone.Config{
    BaseURL:    apiURL,
    Auth:       auth,
    TenantCode: tenantCode,
})
```

See [docs/authentication.md](docs/authentication.md) for the full guide.

## Supported resources

### Current

| Resource     | Operations            | Platforms / notes                                          |
| ------------ | --------------------- | ---------------------------------------------------------- |
| Profiles     | CRUD                  | iOS, macOS, Android, Windows 10, Windows Rugged, Linux     |
| Smart Groups | Search                | All platforms                                              |
| Sensors      | Read                  | Read-only at v0                                            |
| Apps (MAM)   | Read                  | Internal apps; categories                                  |

### Roadmap

| Resource     | Planned work                |
| ------------ | --------------------------- |
| Smart Groups | Full CRUD (planned)         |
| Sensors      | Write support (planned)     |
| Org Groups   | Read (planned)              |

See issues for tracking planned work. The full coverage matrix lives at
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

API reference: [pkg.go.dev/github.com/euc-oss/terraform-sdk-uem](https://pkg.go.dev/github.com/euc-oss/terraform-sdk-uem).

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

This project follows [Semantic Versioning](https://semver.org/). While at
v0.x, breaking changes can occur in any minor release; pin a specific minor
version in your `go.mod` to avoid surprises. Once v1.0 ships, the public
API will be stable per semver guarantees.

**v0.0.2** is the next coordinated release and introduces a cleaner public API
surface: `wsone.Config{BaseURL, Auth, TenantCode, HTTPClient}`, `wsone.NewClient`,
`wsone.NewOAuth2Auth`, `wsone.ListProfiles`, and `apiErr.IsRetryable()`. v0.0.1
had a different, lower-level surface; v0.0.2 is a coordinated breaking change
while still pre-1.0.

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
