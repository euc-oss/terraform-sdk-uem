# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

Derived mechanically from the exported-symbol diff of the assembled public
module (the 26.2.0 internal baseline against this release).

- `client.NotFoundError` (`Resource`, `ID`; `Error()`) and `client.IsNotFound(err)`,
  which recognises the not-found responses of the profile, smart group and internal app
  endpoints as well as a plain 404.
- `client.UnsupportedPlatformError` (`Resource`, `ID`, `RawPlatform`; `Error()`), returned
  when a profile search reports a platform with no mapping to a detail-GET platform.
- `client.UnsupportedOperationError` (`Resource`, `Operation`, `Platform`, `Reason`;
  `Error()`), returned when an operation does not exist for a platform.
- `client.APIError.Attempts`: the number of requests made when the retry policy gave up.
- `sdk.ProfileServiceOption` and `sdk.WithProfileServiceOrganizationGroupID`, to scope a
  `ProfileService`'s discovery to one organization group.
- `sdk.OrganizationGroupsService` and `sdk.NewOrganizationGroupsService`, with
  `GetAsync`, `GetChildLocationGroups`, `GetParents` and `LocationGroupSearch`, plus
  `sdk.OrganizationGroupsLocationGroupSearchOptions` (`Groupid`, `Name`, `OrderBy`, `Page`,
  `PageSize`, `SortOrder`, `Type`) and the models `sdk.LocationGroupV1` (`ID`, `UUID`,
  `Name`, `GroupID`, `LocationGroupType`, `ParentLocationGroup`, `LgLevel`, `Locale`,
  `Country`, `CreatedOn`, `AddDefaultLocation`, `IsCustomerTypeExistInParentHierarchy`,
  `Admins`, `Devices`, `Users`), `sdk.LocationGroupSearchResultV1` (`LocationGroups`,
  `Page`, `PageSize`, `Total`), `sdk.OrganizationGroupCollectionV1Model` (`Items`) and
  `sdk.EntityReferenceV1` (`ID`, `Name`, `UUID`).
- `sdk.ProfilesV1Service.DeleteDeviceProfileAsync`.
- `sdk.BaselineDevicePoliciesV1Model` (`Results`, `Total`), `sdk.BaselineDevicePolicyV1Model`
  (`UUID`, `Name`, `ClassName`, `Path`, `Status`, `Compliance`),
  `sdk.BaselineDevicePolicyComplianceV1Model` (`Status`, `ErrorCode`) and
  `sdk.BaselinesV1GetBaselineDevicePoliciesAsyncOptions` (`Offset`, `Limit`,
  `ComplianceLevel`, `SortBy`, `SortOrder`, `Language`).

### Breaking

- `ProfilesV2Service.DeleteProfileAsync` is removed. UEM registers
  `DELETE /api/mdm/profiles/{id}` only at API v1, so the v2 method never had a v2
  route behind it; the server quietly served every call at v1. Use
  `ProfilesV1Service.DeleteDeviceProfileAsync`, which has the same arguments and sends
  `Accept: application/json;version=1`. `ProfileService.Delete` is unchanged for callers.
- `BaselinesV1Service.GetBaselineDevicePoliciesAsync` now returns
  `BaselineDevicePoliciesV1Model` (`Total` plus `Results` of per-policy items) instead of
  a slice of `BaselineDeviceComplianceV1Model`, and takes an options argument for
  `offset`, `limit`, `compliance_level`, `sort_by`, `sort_order` and `language`.

### Changed

- **The module path is now `github.com/euc-oss/terraform-sdk-uem/v26`** (it ends in `/v26`), and releases are tagged
  `v26.<minor>.<patch>[-beta.N]`, starting with `v26.2.0-beta.1`. Go requires the
  major-version suffix from v2 on, and the major version is the UEM major version
  (UEM 27 will be `/v27`). Earlier `v0.x` versions were published without a suffix:
  add `/v26` to the module path in your import statements and in the `require` line of
  your `go.mod`, for example `go get github.com/euc-oss/terraform-sdk-uem/v26@v26.2.0-beta.1`.
- When the client's retry policy gives up on a retryable status (for example a
  persistent HTTP 500), the error is now a `*client.APIError` carrying the final
  status, `errorCode` and message, with `Attempts` set to the number of requests
  made. Previously it was a plain "giving up after N attempt(s)" error and the
  response was dropped. No type changed; `APIError` gains the `Attempts` field.
  Network errors and canceled contexts still return a plain error.
- `resources.ProfileService.Delete` now sends `Accept: application/json;version=1`,
  the only version UEM registers for `DELETE /api/mdm/profiles/{id}`.
- `sdk.NewProfileService` and `sdk.NewProfileServiceWithoutDiscovery` gain a variadic
  `opts ...ProfileServiceOption` parameter. Existing calls compile unchanged; code that
  stores either function in a variable of the old function type must update that type.

### Removed

- `testdata/mock-responses/profiles/update_profile_success.json` is removed. It stood for a
  `PUT /api/mdm/profiles/{id}` route that no SDK method calls (profile updates use the
  per-platform update routes), and no test consumed it.

### Known limitations

- `client.IsNotFound` does not recognise the response to a repeat
  `DELETE /api/system/groups/{uuid}` (API v2) of an already-deleted organization group:
  HTTP 400, errorCode 400, "Resource's organization group: <id> not found or user does not
  have access to this organization group." It surfaces as a plain `*client.APIError`.

### Fixed

- macOS profile Restrictions: `Preferences` and `Sharing` now serialize their parent
  key first (`EnabledPreferencePanes`, `RestrictWhichSharingServicesAreEnabled`). UEM
  applies request keys in order and ignores a pane or sharing-service setting that
  arrives before its parent is true, so the settings that sorted before the parent
  alphabetically were saved as false: the 9 panes `Accessibility` to `Dock`, and the
  7 sharing services `AddtoAperture`, `AddtoiPhoto`, `AddtoReadingList`, `AirDrop`,
  `Facebook`, `Mail`, `Messages`. A further sharing key,
  `AutomaticallyEnableNewSharingServices`, also sorts before the parent but is not
  order-dependent, so it is not counted.
- `resources.ProfileService.Update` now uses POST for Windows Rugged (QNX). Only
  Windows 10 (WinRT) profile updates use PUT; the method previously sent PUT for
  Windows Rugged as well.
- `resources.ProfileService.Create` and `Update` now send
  `Accept: application/json;version=4` for Linux, as `Get` already did. Linux profile create
  and update go through API v4; with the previous version=2 header the server rejected the
  Linux body with 422 (errorCode 1012).
- `resources.ProfileService.Create` (and `wsone.CreateProfile`) now refuses Windows Rugged
  (QNX) immediately with a `*client.UnsupportedOperationError`, without sending a request.
  The server has no profile create endpoint for Windows Rugged, so the request previously
  failed with a 404.
