# Platform Support Reference

Quick-lookup reference for the device platforms the Workspace ONE UEM Go SDK
supports. For end-to-end profile usage examples, see the
[Profile Management Guide](../guides/profiles.md).

## Supported Platforms

Six platforms are supported for profile CRUD. Each has its own URL segment,
API version, and update verb.

| Platform string    | URL segment | API version | Update verb |
|--------------------|-------------|-------------|-------------|
| `"Android"`        | `android`   | v2          | POST        |
| `"Apple iOS"`      | `apple`     | v2          | POST        |
| `"AppleOsX"`       | `appleosx`  | v2          | POST        |
| `"Windows 10"`     | `winrt`     | v2          | PUT         |
| `"Windows_Rugged"` | `qnx`       | v2          | PUT         |
| `"To do"` (Linux)  | `linux`     | v4          | POST        |

A few specifics that trip people up:

- **Apple iOS uses the URL segment `apple`, not `ios`.** This is the API's
  convention; the SDK matches it.
- **macOS is `"AppleOsX"`** with mixed case. Do not use `"Apple OS X"` or
  `"AppleMacOsX"`. The constant is `wsone.PlatformAppleOsX`.
- **Linux's platform string is the literal `"To do"`.** That is not a
  placeholder; that is what the UEM API returns. Always use
  `wsone.PlatformLinux` rather than hardcoding the string.
- **Linux uses API v4** at a separate base path (`/api/mdm/profiles/linux/`).
  All other platforms use v2 at `/api/mdm/profiles/`. This routing is handled
  internally; callers do not need to choose.
- **Windows 10 and Windows Rugged use PUT for updates.** Everything else uses
  POST. `wsone.UpdateProfile` picks the correct verb automatically.

## Platform Constants

Use the constants from the root `wsone` package for type safety:

```go
import wsone "github.com/euc-oss/terraform-sdk-uem"

profile, err := wsone.GetProfile(ctx, client, profileID, wsone.PlatformAndroid)
```

| Constant                      | API value          |
|-------------------------------|--------------------|
| `wsone.PlatformAndroid`       | `"Android"`        |
| `wsone.PlatformAppleiOS`      | `"Apple iOS"`      |
| `wsone.PlatformAppleOsX`      | `"AppleOsX"`       |
| `wsone.PlatformWindows10`     | `"Windows 10"`     |
| `wsone.PlatformWindowsRugged` | `"Windows_Rugged"` |
| `wsone.PlatformLinux`         | `"To do"`          |

## Resource and Platform Coverage

### Current (v0)

| Resource     | Operations | Notes                                                  |
|--------------|------------|--------------------------------------------------------|
| Profiles     | CRUD       | iOS, macOS, Android, Windows 10, Windows Rugged, Linux |
| Smart Groups | Search     | All platforms                                          |
| Sensors      | Read       | Read-only at v0                                        |
| Apps (MAM)   | Read       | Internal apps; categories                              |

### Roadmap

| Resource     | Planned work            |
|--------------|-------------------------|
| Smart Groups | Full CRUD (planned)     |
| Sensors      | Write support (planned) |
| Org Groups   | Read (planned)          |

## Profile Operation Support Matrix

| Platform               | Create | Get | Update    | Delete | Notes                                                                 |
|------------------------|--------|-----|-----------|--------|-----------------------------------------------------------------------|
| Android                | Yes    | Yes | Yes       | Yes    | See macOS/Android update note below                                   |
| Apple iOS              | Yes    | Yes | Yes       | Yes    |                                                                       |
| macOS (`AppleOsX`)     | Yes    | Yes | Yes       | Yes    | Security & Privacy profiles may return HTTP 400 on GET (server-side)  |
| Windows 10 (`winrt`)   | Yes    | Yes | Yes (PUT) | Yes    |                                                                       |
| Windows Rugged (`qnx`) | Yes    | Yes | Yes (PUT) | Yes    |                                                                       |
| Linux                  | No     | Yes | Yes       | Yes    | No create endpoint; profiles are seeded externally                    |

### macOS and Android update persistence

macOS and Android profile updates may return HTTP 200 but not persist the
changes server-side. This is a known server-side behavior in certain UEM
releases. iOS, Windows 10, Windows Rugged, and Linux updates do not have this
issue. The SDK side is correct.

### macOS Security and Privacy profiles

macOS profiles that carry GateKeeper sub-payloads may return HTTP 400 from the
GET endpoint. This is a server-side issue in the UEM API, not an SDK bug.
Other macOS profile types (Custom Settings, Disk Encryption, Network/Wi-Fi,
Restrictions) work normally.

## API Version Coverage

| API version | Resources using it                                          |
|-------------|-------------------------------------------------------------|
| v1          | Sensors (read), Smart Groups (search)                       |
| v2          | Profiles (Android, iOS, macOS, Windows 10, Windows Rugged)  |
| v4          | Profiles (Linux)                                            |

## See Also

- [Profile Management Guide](../guides/profiles.md)
- [Authentication guide](../authentication.md)
- [Error handling guide](../error-handling.md)
- [Workspace ONE UEM API documentation](https://developer.omnissa.com/ws1-uem-apis/)
