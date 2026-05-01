# Profile Management Guide

This guide covers profile operations in the Workspace ONE UEM Go SDK — listing,
retrieving, creating, updating, and deleting profiles across all supported
platforms.

## Quick Start

```go
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
        TenantCode: os.Getenv("WSONE_TENANT_CODE"),
        Auth:       auth,
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

## CRUD Operations

### List profiles

`wsone.ListProfiles` returns a flat `[]*models.Profile` slice. Pass nil for
default pagination (page 0, up to 500 results), or supply `*wsone.ListOptions`
to paginate:

```go
// Default: page 0, up to 500 results
profiles, err := wsone.ListProfiles(ctx, client, nil)

// Explicit page and page size
batch, err := wsone.ListProfiles(ctx, client, &wsone.ListOptions{
    Page:     2,
    PageSize: 100,
})
```

Each `*models.Profile` in the result carries at minimum:

- `p.GetProfileID() int` — normalized profile ID (checked in three locations
  across API response shapes; see [Profile ID normalization](#profile-id-normalization))
- `p.GetName() string` — normalized profile name
- `p.Platform string` — platform string as the API returns it (e.g.,
  `"Android"`, `"Apple iOS"`, `"AppleOsX"`)

### Get a profile by ID

`wsone.GetProfile` fetches a single profile by its integer ID. Because the
underlying API routes to a platform-specific endpoint, you must also supply the
platform string:

```go
profile, err := wsone.GetProfile(ctx, client, profileID, wsone.PlatformAndroid)
if err != nil {
    log.Fatalf("get profile: %v", err)
}
fmt.Printf("Profile %d — %s (%s)\n",
    profile.GetProfileID(), profile.GetName(), profile.Platform)
```

Use the `wsone.Platform*` constants (see [Platform constants](#platform-constants))
rather than raw strings to avoid silent mis-routing.

### Create a profile

`wsone.CreateProfile` accepts a platform string and a `*models.ProfileCreateRequest`,
which is a `map[string]interface{}`. The map shape is platform-specific; the
`"General"` key carries common fields, and additional keys carry payload
sections:

```go
req := &models.ProfileCreateRequest{
    "General": map[string]interface{}{
        "Name":                   "Corporate Email",
        "AssignmentType":         "Auto",
        "IsActive":               true,
        "ManagedLocationGroupId": 14165,
    },
    "Passcode": map[string]interface{}{
        "MinimumPasscodeLength": 6,
        "RequireAlphanumeric":   false,
    },
}

profile, err := wsone.CreateProfile(ctx, client, wsone.PlatformAndroid, req)
if err != nil {
    log.Fatalf("create profile: %v", err)
}
fmt.Printf("Created profile ID: %d\n", profile.GetProfileID())
```

The API returns only the new profile ID on a successful create. The SDK wraps
that value into a `*models.Profile` with `GetProfileID()` populated.

> **Linux has no create endpoint.** The Workspace ONE UEM API does not expose
> a create endpoint for Linux profiles; they are seeded outside the API.
> `wsone.CreateProfile` called with `wsone.PlatformLinux` will return an API
> error.

### Update a profile

`wsone.UpdateProfile` accepts a platform string, the existing profile ID, and a
`*models.ProfileUpdateRequest` (also a `map[string]interface{}`). Include the
profile ID inside the `"General"` map to satisfy the API:

```go
req := &models.ProfileUpdateRequest{
    "General": map[string]interface{}{
        "ProfileId":        profileID,
        "Name":             "Corporate Email v2",
        "AssignmentType":   "Auto",
        "IsActive":         true,
        "CreateNewVersion": true,
    },
}

updated, err := wsone.UpdateProfile(ctx, client, wsone.PlatformAndroid, profileID, req)
if err != nil {
    log.Fatalf("update profile: %v", err)
}
fmt.Printf("Updated profile id=%d\n", updated.GetProfileID())
```

The SDK picks the correct HTTP verb for the platform automatically. Windows 10
and Windows Rugged updates use HTTP PUT; all other platforms use POST. You do
not need to know this at the call site.

### Delete a profile

`wsone.DeleteProfile` requires only the profile ID. The platform is not needed
because the delete endpoint is not platform-specific:

```go
if err := wsone.DeleteProfile(ctx, client, profileID); err != nil {
    log.Fatalf("delete profile: %v", err)
}
```

## Platform Constants

Use the `wsone.Platform*` constants instead of raw strings. The API returns
several non-obvious values:

| Constant                      | API value          | URL segment | API version | Update verb |
|-------------------------------|--------------------|-------------|-------------|-------------|
| `wsone.PlatformAndroid`       | `"Android"`        | `android`   | v2          | POST        |
| `wsone.PlatformAppleiOS`      | `"Apple iOS"`      | `apple`     | v2          | POST        |
| `wsone.PlatformAppleOsX`      | `"AppleOsX"`       | `appleosx`  | v2          | POST        |
| `wsone.PlatformWindows10`     | `"Windows 10"`     | `winrt`     | v2          | PUT         |
| `wsone.PlatformWindowsRugged` | `"Windows_Rugged"` | `qnx`       | v2          | PUT         |
| `wsone.PlatformLinux`         | `"To do"`          | `linux`     | v4          | POST        |

Several of these are surprising:

- **Apple iOS uses URL segment `apple`, not `ios`.** This is the API's
  convention; the SDK matches it.
- **macOS is `"AppleOsX"`** with mixed case. Use `wsone.PlatformAppleOsX`.
  Do not use `"Apple OS X"` or `"AppleMacOsX"`.
- **Linux's platform string is the literal `"To do"`.** This is not a
  placeholder — it is the exact string the Workspace ONE UEM API returns for
  Linux. Always use `wsone.PlatformLinux` rather than hardcoding the string.
- **Linux uses API v4** at a different base path (`/api/mdm/profiles/linux/`).
  All other platforms use v2 at `/api/mdm/profiles/`. This routing is handled
  internally; callers do not need to choose.
- **Windows 10 and Windows Rugged** platform strings are `"Windows 10"` and
  `"Windows_Rugged"` respectively. The URL segments they map to are `winrt` and
  `qnx`. This naming mismatch is an API convention, not an SDK choice.
- **Windows uses PUT for updates.** Windows 10 (`winrt`) and Windows Rugged
  (`qnx`) use HTTP PUT for profile updates; all other platforms use POST.
  `wsone.UpdateProfile` picks the correct verb based on the platform argument.

## Profile ID Normalization

The Workspace ONE UEM API returns the profile ID in different locations
depending on the endpoint:

- **Search response:** `Id.Value` (nested object)
- **Get response:** `ProfileId` (direct integer field)
- **General section:** `General.ProfileId` (inside the general map)

`(*models.Profile).GetProfileID()` checks all three locations in order and
returns the first non-zero value. Always call `GetProfileID()` rather than
accessing `ProfileID` directly to handle all response shapes correctly:

```go
profile, err := wsone.GetProfile(ctx, client, 42, wsone.PlatformAppleiOS)
// Use GetProfileID(), not profile.ProfileID
fmt.Println(profile.GetProfileID())
```

Similarly, `GetName()` normalizes the name across `ProfileName` and
`General.Name`, and `GetPlatform()` normalizes the platform field.

## Error Handling

All SDK functions return `(result, error)`. API errors come back as
`*wsone.APIError`:

```go
import "errors"

profile, err := wsone.GetProfile(ctx, client, profileID, wsone.PlatformAndroid)
if err != nil {
    var apiErr *wsone.APIError
    if errors.As(err, &apiErr) {
        switch apiErr.StatusCode {
        case 404:
            fmt.Println("profile not found")
        case 400:
            fmt.Printf("bad request: %s\n", apiErr.Message)
        default:
            if apiErr.IsRetryable() {
                // The SDK already exhausted its retry budget
                // (transient codes: 429, 500, 502, 503, 504).
                fmt.Printf("transient failure after retries — status %d\n",
                    apiErr.StatusCode)
            } else {
                fmt.Printf("permanent error %d: %s (code: %s)\n",
                    apiErr.StatusCode, apiErr.Message, apiErr.ErrorCode)
            }
        }
    }
    return err
}
```

The SDK retries transient errors automatically with exponential backoff before
surfacing them to the caller. `IsRetryable()` reports whether the error code is
in the retryable set.

### Accept header versioning

The API silently falls back to v1 responses if the `Accept` header contains a
trailing semicolon or the wrong version. The SDK always sends correct
`Accept: application/json;version=N` headers. This is handled internally; no
caller action is needed.

## Pagination

By default, `ListProfiles` returns page 0 with up to 500 results per page. To
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

## Known API Behaviors

### macOS Security and Privacy profiles

macOS profiles that carry GateKeeper sub-payloads may return HTTP 400 from the
GET endpoint. This is a server-side issue in the UEM API. Other macOS profile
types (Custom Settings, Disk Encryption, Network/Wi-Fi, Restrictions) work
normally.

### macOS and Android update persistence

macOS and Android profile updates may return HTTP 200 but not persist the
changes server-side. This is a known server-side behavior in certain UEM
releases. iOS, Windows 10, Windows Rugged, and Linux updates do not have this
issue.

## See Also

- [Platform support reference](../reference/platform-support.md) — per-platform
  operation support matrix and API version coverage
- [Error handling guide](../error-handling.md) — comprehensive error patterns
- [Authentication guide](../authentication.md) — OAuth2 setup, on-prem vs. cloud
- [pkg.go.dev reference](https://pkg.go.dev/github.com/euc-oss/terraform-sdk-uem) — full API reference
