# Installation

This guide covers installing the Workspace ONE UEM Go SDK in your Go project.

## Prerequisites

- **Go 1.25 or later** — [Download Go](https://go.dev/dl/)
- **Git** — required by the Go module system for fetching dependencies
- **Workspace ONE UEM access** — a cloud or on-premises Workspace ONE UEM
  environment
- **API credentials** — either Basic Auth (username + password) or OAuth2
  client credentials (client ID + secret). See
  [Authentication](authentication.md) for how to obtain OAuth2 credentials
  from the Workspace ONE console.

### Verify your Go version

```bash
go version
# Should output: go version go1.25.x ...
```

## Installing the SDK

### Quick install

```bash
go get github.com/euc-oss/terraform-sdk-uem
```

This adds the latest SDK version to your `go.mod` and downloads the source.

### Pin to a specific version

For production use, pin a specific version to avoid unexpected changes while
the SDK is still pre-1.0:

```bash
go get github.com/euc-oss/terraform-sdk-uem@v0.1.0
```

Your `go.mod` will look like:

```text
module your-project-name

go 1.25

require (
    github.com/euc-oss/terraform-sdk-uem v0.1.0
)
```

### Initialize a new module

If your project does not yet have a `go.mod` file:

```bash
go mod init your-project-name
go get github.com/euc-oss/terraform-sdk-uem
```

## Verifying the installation

Create a minimal file to confirm the SDK imports and the client constructor
works:

```go
// verify_install.go
package main

import (
    "fmt"
    "log"

    wsone "github.com/euc-oss/terraform-sdk-uem"
)

func main() {
    // NewClient returns an error for invalid configuration; a missing
    // TenantCode will trigger it. This just confirms the package compiles
    // and links correctly.
    _, err := wsone.NewClient(wsone.Config{
        BaseURL:    "https://placeholder.example.com",
        TenantCode: "placeholder",
        Auth:       wsone.NewBasicAuth("user", "pass"),
    })
    if err != nil {
        log.Fatalf("unexpected error: %v", err)
    }
    fmt.Println("SDK installed and importable.")
}
```

Run it:

```bash
go run verify_install.go
```

Expected output:

```text
SDK installed and importable.
```

## Setting up credentials as environment variables

The SDK reads nothing from the environment automatically, but it is good
practice to keep credentials out of source code. A common pattern is to load
them from environment variables in your application:

```bash
# Workspace ONE Cloud (OAuth2)
export WSONE_API_URL="https://your-instance.workspaceone.com"
export WSONE_TENANT_CODE="your-tenant-code"
export WSONE_CLIENT_ID="your-client-id"
export WSONE_CLIENT_SECRET="your-client-secret"
export WSONE_TOKEN_URL="https://na.uemauth.workspaceone.com/connect/token"

# On-premises (Basic Auth example)
# export WSONE_API_URL="https://your-on-prem-host.example.com"
# export WSONE_TENANT_CODE="your-tenant-code"
# export WSONE_USERNAME="your-username"
# export WSONE_PASSWORD="your-password"
```

Then in your Go code:

```go
import (
    "os"
    wsone "github.com/euc-oss/terraform-sdk-uem"
)

auth, err := wsone.NewOAuth2Auth(wsone.OAuth2Config{
    ClientID:     os.Getenv("WSONE_CLIENT_ID"),
    ClientSecret: os.Getenv("WSONE_CLIENT_SECRET"),
    TokenURL:     os.Getenv("WSONE_TOKEN_URL"),
})
if err != nil {
    log.Fatal(err)
}

client, err := wsone.NewClient(wsone.Config{
    BaseURL:    os.Getenv("WSONE_API_URL"),
    TenantCode: os.Getenv("WSONE_TENANT_CODE"),
    Auth:       auth,
})
```

## IDE setup

### Visual Studio Code

1. Install the
   [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.go)
2. The extension detects the SDK automatically and provides code completion,
   go-to-definition, and inline documentation.

### GoLand / IntelliJ IDEA

1. Open your project; GoLand indexes the SDK automatically.
2. Use Ctrl+Q (Cmd+J on macOS) for quick documentation and Ctrl+B (Cmd+B) to
   navigate to source.

## Dependency management

### Update to the latest version

```bash
go get -u github.com/euc-oss/terraform-sdk-uem
go mod tidy
```

### Remove unused dependencies

```bash
go mod tidy
```

### List the installed SDK version

```bash
go list -m github.com/euc-oss/terraform-sdk-uem
```

## Troubleshooting

### "cannot find package"

Run `go mod download` to fetch all declared dependencies:

```bash
go mod download
```

### "go.mod file not found"

Initialize a Go module first:

```bash
go mod init your-project-name
```

### Version conflicts

Clean the module cache and rebuild:

```bash
go clean -modcache
go mod tidy
go mod download
```

### Import errors in the IDE

- **VS Code**: Reload window (Ctrl+Shift+P → "Developer: Reload Window")
- **GoLand**: File → Invalidate Caches / Restart

## Uninstalling

Remove the import statements from your source files, then run:

```bash
go mod tidy
```

The SDK will be dropped from `go.mod` once it is no longer referenced.

## Next steps

- [Authentication](authentication.md) — configure OAuth2 or Basic Auth
- [Quickstart](quickstart.md) — write your first integration in five minutes
- [Configuration](configuration.md) — timeouts, retry policy, custom HTTP
  transports
- [Error handling](error-handling.md) — working with `*wsone.APIError` and
  the retry budget
