# Workspace ONE UEM Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/euc-oss/terraform-sdk-uem/v26.svg)](https://pkg.go.dev/github.com/euc-oss/terraform-sdk-uem/v26)
[![License: Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

Typed Go client for the Workspace ONE UEM REST API.

This is a Go SDK that wraps the Workspace ONE UEM REST API with authentication,
retry, rate limiting, and platform-aware endpoint routing. It is designed to
be used directly from Go applications and as the underlying client for the
Workspace ONE UEM Terraform provider.

This is **not a Terraform provider**. The Terraform provider lives in a
separate repository and consumes this SDK.

This SDK is maintained by Omnissa.

## Where the code is

This `main` branch is the project's landing page. It holds no code. Each
Workspace ONE UEM version has its own release branch, which holds the source,
documentation, examples and changelog for that version.

| UEM version | Module path                                | Latest release                                                                                  | Branch                                                                                       |
| ----------- | ------------------------------------------ | ----------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| 26.2        | `github.com/euc-oss/terraform-sdk-uem/v26` | [v26.2.0-beta.1](https://github.com/euc-oss/terraform-sdk-uem/releases/tag/v26.2.0-beta.1) (beta) | [release/26.2.0-beta1](https://github.com/euc-oss/terraform-sdk-uem/tree/release/26.2.0-beta1) |

All releases are listed on the [Releases](https://github.com/euc-oss/terraform-sdk-uem/releases) page.

## Installation

```text
go get github.com/euc-oss/terraform-sdk-uem/v26@v26.2.0-beta.1
```

```text
import wsone "github.com/euc-oss/terraform-sdk-uem/v26"
```

Requires Go 1.25 or later. Start with the
[README](https://github.com/euc-oss/terraform-sdk-uem/blob/release/26.2.0-beta1/README.md)
and [docs](https://github.com/euc-oss/terraform-sdk-uem/tree/release/26.2.0-beta1/docs)
on the release branch.

## Versioning

Releases are tagged `v<UEM major>.<minor>.<patch>`, with `-beta.N` on a beta
(`v26.2.0-beta.1`, then `v26.2.0`), on the release branch of the UEM version
they target. Go's rule for major versions applies: from v2 on, the module path
ends in `/vN`. So the UEM 26 line is `github.com/euc-oss/terraform-sdk-uem/v26`,
and a UEM 27 line will be a new module path ending in `/v27`. Both can be
required side by side.

The `v0.0.x` tags are early releases on the un-suffixed module path
`github.com/euc-oss/terraform-sdk-uem`. They are superseded by the `/v26` line.

## Supported device platforms

- Android
- Apple iOS / iPadOS
- Apple macOS
- Windows 10 / 11
- Windows Rugged
- Linux

## Supported API areas (UEM 26.2)

### Mobile Device Management (MDM)

- Device profiles: create, read, update and delete on all six platforms (Windows Rugged: no create), with smart-group assignment in the profile payload
- Smart groups
- Device sensors and sensor assignments
- Scripts and script assignments
- Baselines, including baseline templates, catalogs and per-device policy compliance
- OS updates and OS version information
- Platform information

### Mobile Application Management (MAM)

- Internal (enterprise) applications, including chunked upload of large app packages
- Purchased (public store) applications
- macOS applications
- App bookmarks
- Enterprise App Repository
- App removal protection logs
- Blob storage (file upload and management)

### System

- Organization groups: lookup, search, parent and child hierarchy

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
