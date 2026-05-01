# Configuration Reference

This document describes the configuration options available when constructing a
Workspace ONE UEM SDK client.

## Overview

`wsone.Config` is the entry point for all client configuration. Pass it to
`wsone.NewClient` to obtain a `*wsone.Client` that handles authentication,
request routing, rate limiting, and retries.

```go
import wsone "github.com/euc-oss/terraform-sdk-uem"

c, err := wsone.NewClient(wsone.Config{
    BaseURL:    "https://your-instance.awmdm.com",
    TenantCode: "your-tenant-code",
    Auth:       auth, // wsone.AuthProvider
})
```

## Required Fields

| Field        | Type                | Description                                          |
| ------------ | ------------------- | ---------------------------------------------------- |
| `BaseURL`    | `string`            | Base URL of the Workspace ONE UEM instance           |
| `TenantCode` | `string`            | Value sent as the `aw-tenant-code` header on every request |
| `Auth`       | `wsone.AuthProvider`| Authentication provider (OAuth2 or Basic)            |

`NewClient` returns an error if any required field is empty.

### BaseURL

The base URL must include the scheme and host, without a trailing slash:

```go
BaseURL: "https://your-instance.awmdm.com"
```

### TenantCode

The tenant code (sometimes called API key) is sent as the `aw-tenant-code`
header on every request. It is required by the Workspace ONE API on all
endpoints. Omitting it produces a 401 from the server.

### Auth

Construct an `AuthProvider` with `wsone.NewOAuth2Auth` or `wsone.NewBasicAuth`
and assign it to this field. The SDK calls `GetToken()` before every request;
OAuth2 tokens are cached and refreshed automatically. See
[authentication.md](authentication.md) for configuration examples and token
endpoint selection.

## Optional Tuning Fields

`wsone.Config` exposes three optional fields for tuning retry, rate-limit, and
timeout behavior. All three default to zero, which tells the SDK to apply its
built-in defaults. You only need to set these when the defaults do not suit your
deployment.

| Field        | Type           | SDK default              |
| ------------ | -------------- | ------------------------ |
| `MaxRetries` | `int`          | 3 retries                |
| `RateLimit`  | `int`          | 1000 requests per minute |
| `Timeout`    | `time.Duration`| 30 seconds               |

Example — override all three:

```go
import (
    "time"
    wsone "github.com/euc-oss/terraform-sdk-uem"
)

c, err := wsone.NewClient(wsone.Config{
    BaseURL:    "https://your-instance.awmdm.com",
    TenantCode: "your-tenant-code",
    Auth:       auth,
    MaxRetries: 5,
    RateLimit:  500,
    Timeout:    60 * time.Second,
})
```

## Custom HTTP Client

Inject a custom `*http.Client` when you need a proxy, custom TLS configuration,
or a non-default dialer:

```go
import (
    "crypto/tls"
    "net/http"
    "net/url"
    wsone "github.com/euc-oss/terraform-sdk-uem"
)

proxy, _ := url.Parse("http://proxy.corp.example.com:8080")

transport := &http.Transport{
    Proxy: http.ProxyURL(proxy),
    TLSClientConfig: &tls.Config{
        // Custom TLS settings here.
    },
}

c, err := wsone.NewClient(wsone.Config{
    BaseURL:    "https://your-instance.awmdm.com",
    TenantCode: "your-tenant-code",
    Auth:       auth,
    HTTPClient: &http.Client{Transport: transport},
})
```

**Important:** The SDK always applies its own request timeout to the supplied
`HTTPClient`, overriding any `Timeout` you set on it directly. Set
`wsone.Config.Timeout` to control the per-request timeout (default: 30 seconds).

## Retry Policy

The SDK uses `hashicorp/go-retryablehttp` to retry failed requests
automatically. The default behavior:

- **Retried status codes:** 429 (Too Many Requests), 500, 502, 503, 504
- **Network errors:** retried unconditionally
- **Max retries:** 3 (so up to 4 total attempts: 1 initial + 3 retries)
- **Backoff:** exponential, between 1 second and 30 seconds per attempt
- **Context cancellation:** if the request context is cancelled, retries stop
  immediately

`MaxRetries` is the number of *retries*, not total attempts. With
`MaxRetries = N`, the client makes at most `N + 1` total attempts.

Override the default with `wsone.Config.MaxRetries`. Zero preserves the default:

```go
c, err := wsone.NewClient(wsone.Config{
    BaseURL:    "https://your-instance.awmdm.com",
    TenantCode: "your-tenant-code",
    Auth:       auth,
    MaxRetries: 5, // up to 6 total attempts
})
```

## Rate Limiting

The SDK uses a token-bucket rate limiter (`golang.org/x/time/rate`). The
default limit is **1000 requests per minute** (~16.7 req/s). The limiter is
applied before every outbound request; if the bucket is exhausted the call
blocks until a token is available (or the context is cancelled).

Override the default with `wsone.Config.RateLimit`. Zero preserves the default:

```go
c, err := wsone.NewClient(wsone.Config{
    BaseURL:    "https://your-instance.awmdm.com",
    TenantCode: "your-tenant-code",
    Auth:       auth,
    RateLimit:  200, // 200 requests per minute
})
```

## Timeouts

The default per-request HTTP timeout is **30 seconds**. Override it with
`wsone.Config.Timeout`. Zero preserves the default:

```go
import "time"

c, err := wsone.NewClient(wsone.Config{
    BaseURL:    "https://your-instance.awmdm.com",
    TenantCode: "your-tenant-code",
    Auth:       auth,
    Timeout:    60 * time.Second,
})
```

The timeout applies per attempt. A request that is retried three times can
consume up to `(MaxRetries + 1) * Timeout` wall-clock time before the SDK
gives up, in addition to the backoff delays between attempts.

For per-operation deadlines independent of the per-request timeout, use a
`context.Context` with a deadline (see [Context Cancellation](#context-cancellation)).

## Context Cancellation

Every SDK method accepts a `context.Context` as its first argument. Use
contexts to enforce per-operation timeouts and to cancel in-flight requests.

**Per-operation timeout:**

```go
import (
    "context"
    "time"
    wsone "github.com/euc-oss/terraform-sdk-uem"
)

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

profiles, err := wsone.ListProfiles(ctx, c, nil)
```

**Cancellation:**

```go
ctx, cancel := context.WithCancel(context.Background())

go func() {
    // Cancel after some external event.
    time.Sleep(5 * time.Second)
    cancel()
}()

profiles, err := wsone.ListProfiles(ctx, c, nil)
if err != nil {
    // err wraps context.Canceled when the context was cancelled.
}
```

Context cancellation propagates through the retry loop: an in-progress retry
sequence stops as soon as the context is cancelled.

## User-Agent

The SDK does not set a custom `User-Agent` header. Requests are sent with the
default Go `net/http` user-agent string (`Go-http-client/1.1` or
`Go-http-client/2.0` depending on the protocol).

If you need a custom user-agent, wrap the transport:

```go
type uaTransport struct {
    base      http.RoundTripper
    userAgent string
}

func (t *uaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    clone := req.Clone(req.Context())
    clone.Header.Set("User-Agent", t.userAgent)
    return t.base.RoundTrip(clone)
}

c, err := wsone.NewClient(wsone.Config{
    BaseURL:    "https://your-instance.awmdm.com",
    TenantCode: "your-tenant-code",
    Auth:       auth,
    HTTPClient: &http.Client{
        Transport: &uaTransport{
            base:      http.DefaultTransport,
            userAgent: "my-app/1.0",
        },
    },
})
```

## Debug Logging

The SDK does not log by default. Detailed request and response logging is
activated by setting one of these environment variables before the process
starts:

| Variable  | Value                     | Effect                  |
| --------- | ------------------------- | ----------------------- |
| `UEM_DEBUG` | `true` or `1`           | Enable SDK debug output |
| `TF_LOG`  | `TRACE` or `DEBUG`        | Enable SDK debug output |

When enabled, the SDK logs every outgoing request and incoming response to
`stderr`, including headers. Sensitive headers (`Authorization`,
`aw-tenant-code`, cookies) are automatically redacted to `***`.

```bash
UEM_DEBUG=true go run ./cmd/myapp
```

To implement your own logging, inject a custom `http.RoundTripper` via
`HTTPClient`:

```go
type loggingTransport struct {
    base http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    log.Printf("--> %s %s", req.Method, req.URL.Path)
    resp, err := t.base.RoundTrip(req)
    if err == nil {
        log.Printf("<-- %s %s", resp.Status, req.URL.Path)
    }
    return resp, err
}

c, err := wsone.NewClient(wsone.Config{
    BaseURL:    "https://your-instance.awmdm.com",
    TenantCode: "your-tenant-code",
    Auth:       auth,
    HTTPClient: &http.Client{Transport: &loggingTransport{base: http.DefaultTransport}},
})
```

**Note:** If `UEM_DEBUG` is set, the SDK wraps your custom transport with its
own debug logger, so both run. Disable one or the other to avoid duplicate
output.

## Further Reading

- [Authentication Guide](authentication.md) — OAuth2 and Basic Auth setup
- [Quick Start Guide](../quickstart.md) — End-to-end example
- [Workspace ONE UEM API Documentation](https://developer.omnissa.com/ws1-uem-apis/)
