# Quickstart

This guide walks you from zero to a running Go program that lists Workspace ONE
UEM profiles in about five minutes. It assumes you have Go 1.25 or later
installed and access to a Workspace ONE UEM environment with OAuth2 credentials.

## 1. Install the SDK

```bash
go mod init myapp
go get github.com/euc-oss/terraform-sdk-uem
```

## 2. Set environment variables

```bash
export WSONE_API_URL="https://your-instance.workspaceone.com"
export WSONE_TENANT_CODE="your-tenant-code"
export WSONE_CLIENT_ID="your-client-id"
export WSONE_CLIENT_SECRET="your-client-secret"
export WSONE_TOKEN_URL="https://na.uemauth.workspaceone.com/connect/token"
```

For Workspace ONE Cloud, the token URL is always
`https://na.uemauth.workspaceone.com/connect/token`. For on-premises
deployments, it is typically `https://<your-host>/oauth/token`. If you only
have the base URL, the SDK appends `/oauth/token` automatically.

## 3. Write the program

Create `main.go`:

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
    // Build an OAuth2 auth provider. Tokens are fetched and refreshed
    // automatically — no manual token management needed.
    auth, err := wsone.NewOAuth2Auth(wsone.OAuth2Config{
        ClientID:     os.Getenv("WSONE_CLIENT_ID"),
        ClientSecret: os.Getenv("WSONE_CLIENT_SECRET"),
        TokenURL:     os.Getenv("WSONE_TOKEN_URL"),
    })
    if err != nil {
        log.Fatalf("auth: %v", err)
    }

    // TenantCode is the aw-tenant-code header required on every request.
    // It lives on wsone.Config, not on the auth provider.
    client, err := wsone.NewClient(wsone.Config{
        BaseURL:    os.Getenv("WSONE_API_URL"),
        TenantCode: os.Getenv("WSONE_TENANT_CODE"),
        Auth:       auth,
    })
    if err != nil {
        log.Fatalf("client: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // ListProfiles returns a flat []*models.Profile slice (page 0, up to 500
    // results by default). Pass *wsone.ListOptions to paginate.
    profiles, err := wsone.ListProfiles(ctx, client, nil)
    if err != nil {
        log.Fatalf("list profiles: %v", err)
    }

    fmt.Printf("Found %d profiles\n\n", len(profiles))
    for _, p := range profiles {
        fmt.Printf("  ID: %-6d  Platform: %-12s  Name: %s\n",
            p.GetProfileID(), p.Platform, p.GetName())
    }
}
```

## 4. Run it

```bash
go run main.go
```

Expected output (values will differ):

```text
Found 3 profiles

  ID: 42      Platform: Android       Name: Corporate Android Policy
  ID: 107     Platform: Apple iOS     Name: iOS Restrictions
  ID: 215     Platform: AppleOsX      Name: macOS Security Baseline
```

---

## Going further: get a specific profile by ID

Once you have a profile ID and platform, call `wsone.GetProfile` directly:

```go
profile, err := wsone.GetProfile(ctx, client, profileID, platform)
if err != nil {
    log.Fatalf("get profile %d: %v", profileID, err)
}
fmt.Printf("Profile %d — %s (%s)\n",
    profile.GetProfileID(), profile.GetName(), profile.GetPlatform())
```

The `platform` argument must be one of the `wsone.Platform*` constants:

| Constant                    | API value       |
| --------------------------- | --------------- |
| `wsone.PlatformAndroid`     | `"Android"`     |
| `wsone.PlatformAppleiOS`    | `"Apple iOS"`   |
| `wsone.PlatformAppleOsX`    | `"AppleOsX"`    |
| `wsone.PlatformWindows10`   | `"Windows 10"`  |
| `wsone.PlatformWindowsRugged` | `"Windows_Rugged"` |
| `wsone.PlatformLinux`       | `"To do"`       |

> Note: The Linux platform value is the literal string `"To do"` — this is what
> the UEM API returns, not a placeholder. Always use `wsone.PlatformLinux`
> rather than a hardcoded string.

---

## Error handling

API errors come back as `*wsone.APIError`. Use `errors.As` to inspect them:

```go
import "errors"

profiles, err := wsone.ListProfiles(ctx, client, nil)
if err != nil {
    var apiErr *wsone.APIError
    if errors.As(err, &apiErr) {
        // The SDK has already exhausted its automatic retry budget for
        // transient failures (429, 500, 502, 503, 504).
        // IsRetryable() reports whether this error code would have been retried.
        if apiErr.IsRetryable() {
            log.Printf("transient failure after retries — status %d", apiErr.StatusCode)
        } else {
            log.Printf("permanent error %d: %s (code: %s)",
                apiErr.StatusCode, apiErr.Message, apiErr.ErrorCode)
        }
    }
    return
}
```

---

## Using Basic Auth instead of OAuth2

For on-premises deployments that do not support OAuth2, substitute
`wsone.NewBasicAuth`:

```go
auth := wsone.NewBasicAuth(
    os.Getenv("WSONE_USERNAME"),
    os.Getenv("WSONE_PASSWORD"),
)

client, err := wsone.NewClient(wsone.Config{
    BaseURL:    os.Getenv("WSONE_API_URL"),
    TenantCode: os.Getenv("WSONE_TENANT_CODE"),
    Auth:       auth,
})
```

`NewBasicAuth` never returns an error (credentials are only encoded, not
validated at construction time), so no error check is needed on that call.

---

## Paginating results

By default, `wsone.ListProfiles` returns page 0 with up to 500 results. To
walk through all profiles in batches:

```go
const pageSize = 100
for page := 0; ; page++ {
    batch, err := wsone.ListProfiles(ctx, client, &wsone.ListOptions{
        Page:     page,
        PageSize: pageSize,
    })
    if err != nil {
        log.Fatal(err)
    }
    if len(batch) == 0 {
        break
    }
    for _, p := range batch {
        fmt.Printf("%d  %s\n", p.GetProfileID(), p.GetName())
    }
    if len(batch) < pageSize {
        break // last partial page
    }
}
```

---

## Next steps

- [Authentication](authentication.md) — full OAuth2 setup, on-prem vs. cloud
  token endpoints, and Basic Auth details
- [Configuration](configuration.md) — custom HTTP client, retry budget,
  rate limiter, request timeouts
- [Error handling](error-handling.md) — full error type reference and retry
  behavior
- [Profiles guide](guides/profiles.md) — create, update, and delete profiles
  across all platforms
- [pkg.go.dev reference](https://pkg.go.dev/github.com/euc-oss/terraform-sdk-uem) — full API reference
