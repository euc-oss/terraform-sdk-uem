# Security Policy

The maintainers of this SDK take security seriously. This document describes
how to report security vulnerabilities and what to expect after you do.

## Supported versions

Until the SDK reaches v1.0, only the latest minor version receives security
fixes. After v1.0, supported versions will be listed in this section and on
the [GitHub releases page](../../releases).

## Reporting a vulnerability

**Please do not file public GitHub issues for security vulnerabilities.**

Use one of the following private channels:

- [GitHub Security Advisories](https://github.com/euc-oss/terraform-sdk-uem/security/advisories/new) — preferred. Allows the maintainers and you to coordinate a fix in private.
- Email the Omnissa product security team. <!-- TBD: confirm address; placeholder will be replaced before public launch. -->

The use of encrypted email is encouraged.

## What to include

A useful report includes:

- The affected SDK version
- A clear description of the vulnerability and its impact
- Steps to reproduce, including sample code if possible
- Your assessment of severity and the conditions under which exploitation is possible
- Any other projects or dependencies involved
- Your name and affiliation, if you would like attribution in the eventual advisory

## What is in scope

In scope for this policy:

- Vulnerabilities in the SDK code itself — authentication handling, request signing, retry / rate-limit logic, response deserialization, error handling, and the bundled mock server
- Vulnerabilities in the SDK's published documentation that materially mislead consumers about secure usage

Out of scope (route elsewhere):

- Vulnerabilities in the Workspace ONE UEM API server or in the Workspace ONE UEM product itself — please report these to Omnissa product security through the normal product channels
- Vulnerabilities in your own application that uses this SDK — these are outside the SDK's responsibility unless caused by an SDK defect
- Generic best-practice complaints unrelated to a specific exploitable issue

## What to expect

Once a report is received:

1. The maintainers acknowledge receipt within 3 business days.
2. The maintainers investigate and respond with a triage outcome — accepted, needs more information, or declined with reasoning.
3. If accepted, the maintainers work with you on a fix and a coordinated disclosure timeline. A draft GitHub Security Advisory is opened privately. CVSS scoring uses the [FIRST CVSS calculator](https://www.first.org/cvss/calculator/3.1).
4. Once a fix is ready, a patch release is prepared. The advisory and CHANGELOG entry are published when the fix ships.
5. If the report turns out not to be a vulnerability in this SDK, the maintainers explain the reasoning and, where appropriate, redirect to the right team (e.g., Omnissa product security for API-server issues).

## Public disclosure

The maintainers prefer coordinated disclosure. The default disclosure window
is 90 days from the initial report or until a fix ships, whichever comes
first; this can be extended by mutual agreement when a fix requires more
time.

## No bug bounty

This project does not offer a bug bounty. Reports are nevertheless
appreciated and acknowledged in the published advisory unless you request
otherwise.
