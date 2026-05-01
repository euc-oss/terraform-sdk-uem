# Authentication Guide

This guide covers configuring authentication for the Workspace ONE UEM API using
the Go SDK.

## Authentication Methods

The SDK supports two authentication methods:

1. **Basic Authentication** — Username and password
2. **OAuth2** — Client credentials flow with automatic token refresh

## Basic Authentication

Basic Authentication uses a username and password to authenticate API requests.

```go
import wsone "github.com/euc-oss/terraform-sdk-uem"

auth := wsone.NewBasicAuth("admin-username", "admin-password")

c, err := wsone.NewClient(wsone.Config{
    BaseURL:    "https://your-instance.awmdm.com",
    TenantCode: "your-tenant-code",
    Auth:       auth,
})
```

Use Basic Authentication when you are developing locally or your environment
does not support OAuth2. For production deployments, prefer OAuth2.

## OAuth2 Authentication

OAuth2 uses client credentials to obtain access tokens. The SDK caches the token
and refreshes it automatically before it expires (with a 5-minute safety buffer).

```go
import wsone "github.com/euc-oss/terraform-sdk-uem"

auth, err := wsone.NewOAuth2Auth(wsone.OAuth2Config{
    ClientID:     "your-client-id",
    ClientSecret: "your-client-secret",
    TokenURL:     "https://na.uemauth.workspaceone.com/connect/token",
})
if err != nil {
    panic(err)
}

c, err := wsone.NewClient(wsone.Config{
    BaseURL:    "https://your-instance.awmdm.com",
    TenantCode: "your-tenant-code",
    Auth:       auth,
})
```

### Token Endpoint URLs

**Workspace ONE Cloud (SaaS)** — authenticate against a separate auth host:

| Region        | Token URL                                             |
| ------------- | ----------------------------------------------------- |
| North America | `https://na.uemauth.workspaceone.com/connect/token`   |
| Europe        | `https://eu.uemauth.workspaceone.com/connect/token`   |
| Asia Pacific  | `https://apac.uemauth.workspaceone.com/connect/token` |

**On-Premises deployments** — the token endpoint lives on the same host as the
API. The SDK appends `/oauth/token` automatically when `TokenURL` does not
already contain `/oauth/token` or `/connect/token`, so these two forms are
equivalent:

```go
TokenURL: "https://your-instance.awmdm.com/oauth/token" // explicit
TokenURL: "https://your-instance.awmdm.com"             // SDK appends /oauth/token
```

If your on-premises deployment uses the `/connect/token` path, supply the full
URL:

```go
TokenURL: "https://your-instance.awmdm.com/connect/token"
```

## Environment Variables

Avoid hardcoding credentials. Store them in environment variables or a secret
manager and read them at startup.

**Basic Auth `.env` example:**

```bash
WSONE_BASE_URL=https://your-instance.awmdm.com
WSONE_TENANT_CODE=your-tenant-code
WSONE_USERNAME=admin-username
WSONE_PASSWORD=admin-password
```

**OAuth2 `.env` example:**

```bash
WSONE_BASE_URL=https://your-instance.awmdm.com
WSONE_TENANT_CODE=your-tenant-code
WSONE_CLIENT_ID=your-client-id
WSONE_CLIENT_SECRET=your-client-secret
WSONE_TOKEN_URL=https://na.uemauth.workspaceone.com/connect/token
```

**Loading with `godotenv`:**

```go
import (
    "log"
    "os"

    "github.com/joho/godotenv"
    wsone "github.com/euc-oss/terraform-sdk-uem"
)

if err := godotenv.Load(); err != nil {
    log.Fatal("error loading .env file")
}

auth, err := wsone.NewOAuth2Auth(wsone.OAuth2Config{
    ClientID:     os.Getenv("WSONE_CLIENT_ID"),
    ClientSecret: os.Getenv("WSONE_CLIENT_SECRET"),
    TokenURL:     os.Getenv("WSONE_TOKEN_URL"),
})

c, err := wsone.NewClient(wsone.Config{
    BaseURL:    os.Getenv("WSONE_BASE_URL"),
    TenantCode: os.Getenv("WSONE_TENANT_CODE"),
    Auth:       auth,
})
```

Add `.env` and `*.env` to `.gitignore` so credentials are never committed to
version control.

## Security Best Practices

- Never commit credentials to version control.
- Use separate configuration for each environment (dev, staging, production).
- In production use a secret management service (AWS Secrets Manager,
  HashiCorp Vault, etc.) — never log credentials.
- Rotate OAuth2 client secrets periodically; revoke unused credentials
  immediately.
- Grant only the minimum required API permissions to the credential.

## Troubleshooting

| Symptom | Likely cause | Fix |
| ------- | ------------ | --- |
| 401 Unauthorized | Wrong username/password or invalid client ID/secret | Verify credentials and tenant code |
| 401 on every request (OAuth2) | Token endpoint unreachable | Check that `TokenURL` is correct and reachable from your network |
| 403 Forbidden | Insufficient permissions or wrong tenant code | Check API user permissions and tenant code |
| Token endpoint error | Wrong region or path in `TokenURL` | Match the URL to your deployment type (cloud vs. on-premises) |

## Next Steps

- [Configuration Reference](configuration.md) — Client options, retry policy, and timeouts
- [Quick Start Guide](../quickstart.md) — End-to-end example
- [Workspace ONE UEM API Documentation](https://developer.omnissa.com/ws1-uem-apis/)
- [OAuth2 Client Credentials Flow](https://oauth.net/2/grant-types/client-credentials/)
