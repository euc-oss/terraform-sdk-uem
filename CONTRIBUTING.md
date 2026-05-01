# Contributing

Thank you for your interest in this Go SDK for Workspace ONE UEM. This
document describes how to participate.

## Scope

This SDK is currently maintained by the internal Omnissa team. The Go source
in this repository is generated from internal swagger sources and assembled
by an internal release pipeline. Day-to-day development happens upstream in
that pipeline; this public repository is the published output.

## Reporting bugs and requesting features

GitHub issues are very welcome. For production support questions, please
use [Omnissa Customer Connect](https://customerconnect.omnissa.com/home).

When filing a bug, the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md)
will ask you for:

- SDK version (the `Version` constant exported by the SDK, or your `go.mod` entry)
- Go version (`go version`)
- Operating system
- Workspace ONE UEM deployment type (cloud or on-prem)
- Steps to reproduce
- Expected vs. actual behavior
- Logs or error output, with credentials and tenant identifiers redacted

## Security issues

Please **do not** file security issues as public GitHub issues. See
[SECURITY.md](SECURITY.md) for the disclosure process.

## Pull requests

We are not actively soliciting external pull requests at this time. If you
do submit one, we will evaluate it; if accepted, the change is re-applied
in our internal source repository and the next public release includes it
with `Co-authored-by:` attribution back to you. Your name will also be added
to [CONTRIBUTORS.md](CONTRIBUTORS.md).

All contributions, whether internal or external, must be signed off under
the [Developer Certificate of Origin](Developer%20Certificate%20of%20Origin.md)
using `git commit -s`. The sign-off line must match the email address of the
git author.

## Code of conduct

Participation in this project is governed by the [Code of Conduct](CODE_OF_CONDUCT.md).
