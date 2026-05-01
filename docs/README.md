# Workspace ONE UEM Go SDK — Documentation

This directory contains in-depth guides for using the Workspace ONE UEM Go SDK.
For installation and a quick overview, start with the top-level
[README](../README.md).

## Getting started

| Document                          | Description                                              |
|-----------------------------------|----------------------------------------------------------|
| [Installation](installation.md)   | Add the SDK to your Go project and verify the install    |
| [Quickstart](quickstart.md)       | Authenticate and list profiles in five minutes           |

## Using the SDK

| Document                              | Description                                                    |
|---------------------------------------|----------------------------------------------------------------|
| [Authentication](authentication.md)   | OAuth2 and Basic auth setup; cloud vs. on-premises token URLs  |
| [Configuration](configuration.md)     | HTTP client, retry policy, rate limiting, timeouts             |
| [Error handling](error-handling.md)   | `*wsone.APIError`, retry behavior, and error patterns          |

## Resource guides

| Document                          | Description                                                  |
|-----------------------------------|--------------------------------------------------------------|
| [Profiles](guides/profiles.md)    | Full CRUD for profiles across all six supported platforms    |

Other resources (Smart Groups, Sensors, Apps, Org Groups) are documented via
godoc on [pkg.go.dev/github.com/euc-oss/terraform-sdk-uem](https://pkg.go.dev/github.com/euc-oss/terraform-sdk-uem) and the runnable
[examples/](../examples/) directory for v0. Per-resource guides will be added
as the API surface stabilizes.

## Reference

| Document                                             | Description                                                              |
|------------------------------------------------------|--------------------------------------------------------------------------|
| [Platform support](reference/platform-support.md)   | Per-platform CRUD support matrix, URL segments, and API version coverage |

## Other

| Document                            | Description                                   |
|-------------------------------------|-----------------------------------------------|
| [Troubleshooting](troubleshooting.md) | Common errors and how to diagnose them      |

## External

- [pkg.go.dev/github.com/euc-oss/terraform-sdk-uem](https://pkg.go.dev/github.com/euc-oss/terraform-sdk-uem) — full Go API reference
