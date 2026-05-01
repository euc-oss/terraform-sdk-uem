# Error Handling

This guide covers how to handle errors from the Workspace ONE UEM SDK in a
Terraform provider or any Go consumer.

## The `APIError` Type

When the UEM API returns a non-2xx status code, the SDK returns an `*wsone.APIError`.
The struct has exactly three fields:

```go
type APIError struct {
    StatusCode int    // HTTP status code (e.g. 404, 429, 500)
    Message    string // Human-readable error message from the API response body
    ErrorCode  string // Machine-readable error code string from the API response body
}
```

`APIError` implements the `error` interface. Its `Error()` string includes the
status code, the `ErrorCode` when present, and the `Message`:

```
API error 404 (PROFILE_NOT_FOUND): The requested profile does not exist.
API error 500: Internal Server Error
```

`StatusCode` is never serialized from JSON — it is set by the SDK from the HTTP
response. `Message` and `ErrorCode` are deserialized from the JSON response body
when present.

## Type Assertion with `errors.As`

Use `errors.As` to check whether an error is an `*wsone.APIError`. This is the
recommended approach because it unwraps any wrapping done by the SDK or the
standard library.

```go
import (
    "errors"
    "fmt"

    wsone "github.com/euc-oss/terraform-sdk-uem"
)

profile, err := wsone.GetProfile(ctx, c, id, wsone.PlatformAndroid)
if err != nil {
    var apiErr *wsone.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
    }
    return err
}
_ = profile
```

All SDK operations surface API failures as the same `*wsone.APIError` type.
The handling pattern above works for any SDK call.

## Datetime Parse Errors

When the SDK deserializes a datetime field and the server sends an
unrecognized format, the unmarshal fails with a typo-suggesting message:

```
UEMTime: cannot parse "2026/04/27": MM/DD/YYYY is supported (e.g. "04/27/2026"); DD/MM/YYYY is not
UEMTime: cannot parse "2026-04-27T16:00": did you mean "2026-04-27T16:00:00"? (seconds are required, e.g. T16:00:00)
UEMTime: cannot parse "2026-04-27T16-00-00": did you mean "2026-04-27T16:00:00"? (time portion uses ':' between hours/minutes/seconds)
UEMTime: cannot parse "2026-04-27 16:00:00 UTC": strip the trailing "UTC" — try "2026-04-27 16:00:00" (or use "Z")
UEMTime: cannot parse "20260427": did you mean "2026-04-27"? (insert hyphens: YYYY-MM-DD)
```

These come back as plain `fmt.Errorf("UEMTime: ...")` errors, not as
`json.UnmarshalTypeError` wrappers. Don't `errors.As` against
`json.UnmarshalTypeError` for datetime parse failures; it won't match.
Depending on where in the unmarshal chain the error fires, you may see
additional field context wrapped around it by the outer JSON decoder, but
the leaf is always the `UEMTime: ...` message above. There's no separate
sentinel error for parse failures; the message text is the contract, and
it's stable enough to log directly to the user.

If you see one of these in the wild, the API returned a datetime layout
the SDK hasn't seen before. Open an issue so the SDK can be updated to
support the new layout; do not work around it in the calling code.

## 404 Detection for Terraform

Terraform resource `Read` functions must remove the resource from state when it
no longer exists in the remote system. Check for a 404 using a direct type
assertion:

```go
import (
    "context"

    wsone "github.com/euc-oss/terraform-sdk-uem"
    "github.com/hashicorp/terraform-plugin-framework/resource"
)

func (r *profileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    // ... retrieve id and platform from state ...

    profile, err := wsone.GetProfile(ctx, c, id, platform)
    if err != nil {
        if apiErr, ok := err.(*wsone.APIError); ok && apiErr.StatusCode == 404 {
            resp.State.RemoveResource(ctx)
            return
        }
        resp.Diagnostics.AddError("Failed to read profile", err.Error())
        return
    }

    // ... map profile to state ...
    _ = profile
}
```

This pattern is intentionally direct — when you already know you're handling
one specific status code, the type assertion plus nil check is clear and concise.
Reserve `errors.As` for situations where the error may be wrapped.

## Retry Behavior

The SDK retries automatically on transient failures using exponential backoff.
No caller action is needed — retries are transparent.

**Status codes that trigger a retry:**

| Status | Meaning               |
|--------|-----------------------|
| 429    | Too Many Requests     |
| 500    | Internal Server Error |
| 502    | Bad Gateway           |
| 503    | Service Unavailable   |
| 504    | Gateway Timeout       |

Network errors (connection reset, timeout before a response is received) are
also retried unconditionally.

**Status codes that are NOT retried:**

- 4xx client errors other than 429 (400, 401, 403, 404, 422, etc.)
- Any 2xx or 3xx response

**Retry limits and back-off:**

- Default maximum: **3 retries** (set `Config.MaxRetries` to override)
- Minimum wait between retries: 1 second
- Maximum wait between retries: 30 seconds
- Back-off algorithm: exponential

`MaxRetries = N` means up to N retries, not N total attempts. The default of
3 retries allows up to 4 total attempts. If `apiErr.IsRetryable()` returns
`true` on a returned error, the SDK already retried and exhausted its budget —
do not retry again in your code.

If the context is cancelled, retries stop immediately and the context error is
returned.

After all retries are exhausted, the final error is returned as an
`*wsone.APIError` (for API-level failures) or a wrapped network error.

## Rate Limiting

The SDK includes a token-bucket rate limiter. All requests flow through it
before being dispatched.

Configure the limit when creating the client:

```go
import wsone "github.com/euc-oss/terraform-sdk-uem"

c, err := wsone.NewClient(wsone.Config{
    // ...
    RateLimit: 600, // maximum requests per minute (default: 1000)
})
```

When the token bucket is exhausted, the SDK blocks until a token becomes
available (or the context is cancelled). The caller does not need to implement
its own throttling.

If `Config.RateLimit` is 0 or unset, it defaults to **1000 requests per
minute**.

## Debugging

Enable request and response logging by setting an environment variable before
running Terraform or your test binary. No code changes are required.

```bash
# UEM-specific debug flag
UEM_DEBUG=true terraform apply

# Terraform's own log level also enables SDK debug output
TF_LOG=DEBUG terraform apply
TF_LOG=TRACE terraform apply
```

Both `UEM_DEBUG=true` and `UEM_DEBUG=1` are accepted.

Debug output is written to **stderr** and looks like:

```
[UEM-SDK] >>> POST /api/mdm/profiles/android/
[UEM-SDK] Request Headers:
  Accept: application/json;version=2
  Content-Type: application/json
  Authorization: ***
  Aw-Tenant-Code: ***

[UEM-SDK] <<< 200 OK /api/mdm/profiles/android/
[UEM-SDK] Response Headers:
  Content-Type: application/json; charset=utf-8
  ...
```

**Sensitive headers are always redacted**, regardless of log level:

- `Authorization`
- `Cookie` / `Set-Cookie`
- `X-Api-Key`
- `aw-tenant-code`

Query parameters in URLs are also redacted (replaced with `?<redacted>`) to
avoid leaking filter values or tokens that appear in query strings.
