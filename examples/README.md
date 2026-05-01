# Examples

Each subdirectory is a runnable Go program demonstrating one feature.
Compile any example with `go run .` from its directory; all examples take
configuration from environment variables — no hardcoded credentials.

## Required environment variables

| Variable                | Description                                                | Used by                                               |
| ----------------------- | ---------------------------------------------------------- | ----------------------------------------------------- |
| `WSONE_API_URL`         | Base URL of the Workspace ONE UEM API                      | all examples                                          |
| `WSONE_TENANT_CODE`     | UEM tenant code (`aw-tenant-code` header value)            | all examples                                          |
| `WSONE_CLIENT_ID`       | OAuth2 client ID                                           | OAuth2 examples                                       |
| `WSONE_CLIENT_SECRET`   | OAuth2 client secret                                       | OAuth2 examples                                       |
| `WSONE_TOKEN_URL`       | OAuth2 token endpoint URL                                  | OAuth2 examples (cloud uses `na.uemauth.workspaceone.com/connect/token`) |
| `WSONE_USERNAME`        | UEM admin username                                         | Basic-auth examples                                   |
| `WSONE_PASSWORD`        | UEM admin password                                         | Basic-auth examples                                   |

## Examples

| Example                                       | What it shows                                          |
| --------------------------------------------- | ------------------------------------------------------ |
| [quickstart/](quickstart)                     | Smallest path: OAuth2 → list profiles                  |
| [auth-oauth2/](auth-oauth2)                   | OAuth2 cloud and on-prem token endpoints               |
| [auth-basic/](auth-basic)                     | Basic auth flow                                        |
| [profile-crud/](profile-crud)                 | Create / read / update / delete a macOS profile        |
| [pagination/](pagination)                     | Iterating through paginated list responses             |
| [smart-groups/](smart-groups)                 | List smart groups; assign a profile                    |
| [error-handling/](error-handling)             | Type-asserting errors; retryable detection             |
| [custom-http-client/](custom-http-client)     | Inject your own `*http.Client` (proxy, custom TLS)     |

## Running

```bash
cd quickstart
export WSONE_API_URL=...
export WSONE_TENANT_CODE=...
export WSONE_CLIENT_ID=...
export WSONE_CLIENT_SECRET=...
export WSONE_TOKEN_URL=...
go run .
```

If any required environment variable is missing, the example exits with a
clear error message and a non-zero status.

## Notes

- **No hardcoded credentials.** Every example reads from the environment. Do not paste secrets into the source.
- **Tenant code lives on `wsone.Config`,** not on the auth constructor — see the Authentication doc for details.
- **All examples use a `context.Context` with an explicit timeout.** Adjust the timeout if your tenant has a slow API.
- **Subdirectories not yet present** in this repo (`smart-groups/`, `error-handling/`, `custom-http-client/`) will be added in subsequent releases. The full set is listed above for completeness.
