# Troubleshooting Guide

Symptom-first reference for common issues using the Workspace ONE UEM SDK from
Go code or via the Terraform provider. For broader topics, see the
[Error Handling](error-handling.md) guide.

## Authentication

### `401 Unauthorized` on every request

**Cause.** Wrong credentials, wrong tenant code, wrong token URL, or a
revoked OAuth2 client.

**Fix.**

1. Verify Basic Auth credentials work directly:

   ```bash
   curl -s -o /dev/null -w "%{http_code}\n" \
     -H "aw-tenant-code: your-tenant-code" \
     -H "Authorization: Basic $(printf '%s' 'username:password' | base64)" \
     https://your-instance.awmdm.com/api/system/info
   ```

   A `200` proves the credentials and the tenant code; a `401` confirms
   the failure is at the credential layer, not in the SDK.

2. For OAuth2, double-check both the client credentials and the token URL.

   - **Cloud deployments:** The token URL is
     `https://na.uemauth.workspaceone.com/connect/token` (or the regional
     equivalent). Pass this full URL as `TokenURL` when constructing your
     `wsone.OAuth2Config`, then call `wsone.NewOAuth2Auth(cfg)`.
   - **On-premise deployments:** The token URL is typically
     `https://your-instance.awmdm.com/oauth/token`. The SDK auto-appends
     `/oauth/token` if you pass just the instance base URL (i.e. neither
     `/connect/token` nor `/oauth/token` is present in the URL).

3. Confirm the OAuth2 client hasn't been disabled or rotated in the UEM
   console: System → Advanced → API → REST API.

### `403 Forbidden` on a specific endpoint

**Cause.** Authenticated, but the user or OAuth2 client doesn't have
permission for that operation.

**Fix.** Check the role assignment in the UEM console. The OAuth2 client
needs explicit access to the API category you're hitting (Profiles, MAM,
Smart Groups, etc.).

## Connectivity

### `context deadline exceeded`

**Cause.** Network slow or unreachable, the request really did need
longer, or the timeout was set too low.

**Fix.**

- Confirm the instance is reachable:

  ```bash
  curl -I https://your-instance.awmdm.com
  ```

- For long-running operations, use a per-call context with a generous
  timeout:

  ```go
  ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
  defer cancel()
  profile, err := wsone.GetProfile(ctx, c, id, platform)
  ```

- Or raise the client-wide default by setting `wsone.Config.Timeout` when
  constructing the client.

### `dial tcp: lookup host: no such host`

**Cause.** DNS can't resolve the instance hostname.

**Fix.** Confirm the URL includes the scheme (`https://`), check that
`nslookup` resolves the host, and confirm there's no proxy interfering
(`HTTPS_PROXY`, `NO_PROXY`).

## Rate Limiting

### `429 Too Many Requests`

The SDK retries 429s automatically with exponential backoff; you only
see this surface as an error after the retry budget is exhausted
(default 3 retries).

**If you keep hitting it:**

- Lower the client's `wsone.Config.RateLimit` so your own ceiling is
  below whatever the tenant tolerates.
- Reduce parallelism in your calling code.
- Page through search results with smaller `PageSize` to spread load.
- If multiple clients share the same OAuth2 credentials, they share the
  tenant's rate budget. Split them onto separate clients.

## Profiles

### `404 Profile not found` (Read or Update)

**Cause.** The profile was deleted out from under your code, or the ID
is wrong.

**Fix.** In a Terraform Read function, treat 404 as "remove from state"
rather than an error. The
[Error Handling Guide](error-handling.md#404-detection-for-terraform)
shows the standard pattern:

```go
import (
    "errors"

    wsone "github.com/euc-oss/terraform-sdk-uem"
)

profile, err := wsone.GetProfile(ctx, c, profileID, platform)
if err != nil {
    var apiErr *wsone.APIError
    if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
        resp.State.RemoveResource(ctx)
        return
    }
    // ... other error handling
}
_ = profile
```

### `400 ProfileId is required` on Update

**Cause.** The Update body is missing `General.ProfileId`. The UEM API
uses that field to route the update.

**Fix.** Always include `ProfileId` (and almost always `CreateNewVersion: true`)
in the `General` map you pass to `wsone.UpdateProfile`. Because
`models.ProfileUpdateRequest` is a `map[string]interface{}`, use map-key
syntax to set values:

```go
import wsone "github.com/euc-oss/terraform-sdk-uem"

request := models.ProfileUpdateRequest{}
general := map[string]interface{}{
    "ProfileId":        profileID,
    "CreateNewVersion": true,
}
request["General"] = general

updatedProfile, err := wsone.UpdateProfile(ctx, c, platform, profileID, &request)
if err != nil {
    // handle error
}
_ = updatedProfile // use the returned profile as needed
```

If you retrieved the profile with `wsone.GetProfile` first, you can copy its
existing `General` map into the request and add `"CreateNewVersion": true`
without overwriting `"ProfileId"`.

### `200 OK` on Update but nothing actually changed

**Known API bug.** The macOS and Android Update endpoints return HTTP 200
but persist nothing because the server does not apply the update. There
is no SDK-side workaround. Check the GitHub issues for current status.

### `400 Invalid Payload Key` on macOS profile GET

**Known API bug.** Specific to macOS profiles carrying GateKeeper
sub-payloads (the Security & Privacy profile type). The API's
payload-key dictionary is missing several GateKeeper keys.

Other macOS profile types (Custom Settings, Disk Encryption, Network
Wi-Fi, Restrictions) work normally.

## Installation

### `cannot find package "github.com/euc-oss/terraform-sdk-uem"`

**Fix.**

```bash
go mod download
go list -m "github.com/euc-oss/terraform-sdk-uem"
```

If the second command shows nothing, add the dependency:

```bash
go get "github.com/euc-oss/terraform-sdk-uem"
```

## Datetime Fields

### `UEMTime: cannot parse "..."`

The SDK accepts twelve datetime layouts (RFC3339 with or without
nanoseconds, no-timezone variants, date-only, US slashes, epoch
integers) and emits an error with a typo-suggesting hint when none
match:

```
UEMTime: cannot parse "2026/04/27": MM/DD/YYYY is supported (e.g. "04/27/2026"); DD/MM/YYYY is not
UEMTime: cannot parse "2026-04-27T16:00": did you mean "2026-04-27T16:00:00"? (seconds are required, e.g. T16:00:00)
```

If the API sent a layout that isn't listed, that's a bug — open an
issue with the raw value so the SDK can be updated. See
[Error Handling: Datetime Parse Errors](error-handling.md#datetime-parse-errors)
for how to handle this error type in consuming code.

## Environment

### Environment variables not loading

**Fix.** If you're using a `.env` file in tests, load it with
`godotenv` at the start of `TestMain`:

```go
import "github.com/joho/godotenv"

func TestMain(m *testing.M) {
    _ = godotenv.Load()
    os.Exit(m.Run())
}
```

For production use, set the variables in your runtime environment
directly. The SDK and the Terraform provider read standard names
(`UEM_INSTANCE_URL`, `UEM_TENANT_CODE`, `UEM_CLIENT_ID`, etc.).

## Debugging

### Enable request and response logging

Set an environment variable; no code change required:

```bash
UEM_DEBUG=true terraform apply
# or
TF_LOG=DEBUG terraform apply
TF_LOG=TRACE terraform apply
```

Both `UEM_DEBUG=true` and `UEM_DEBUG=1` are accepted. Output lands on
stderr and shows the request line, headers (with `Authorization`,
`Cookie`, `X-Api-Key`, and `aw-tenant-code` always redacted), the
response status, and response headers. Query parameters in URLs are
redacted too.

### Inspect a specific API error

```go
import (
    "errors"
    "log"

    wsone "github.com/euc-oss/terraform-sdk-uem"
)

profile, err := wsone.GetProfile(ctx, c, profileID, platform)
if err != nil {
    var apiErr *wsone.APIError
    if errors.As(err, &apiErr) {
        log.Printf("status=%d errorCode=%q message=%q",
            apiErr.StatusCode, apiErr.ErrorCode, apiErr.Message)
    }
}
_ = profile
```

`*wsone.APIError` has exactly three fields. The `ErrorCode` is the
machine-readable code from the API response body (e.g.,
`"PROFILE_NOT_FOUND"`); `Message` is the human-readable text.

## Getting Help

If you can't resolve an issue:

1. Check the [Error Handling](error-handling.md) guide for patterns on
   inspecting and handling `*wsone.APIError`.
2. Search existing GitHub issues for the module.
3. Open a new issue with:
   - SDK version (or commit SHA)
   - Go version
   - The full error message
   - The HTTP request URL and method
   - The `apiErr.ErrorCode` from any `*wsone.APIError`
   - Steps to reproduce

## Additional Resources

- [Error Handling](error-handling.md)
