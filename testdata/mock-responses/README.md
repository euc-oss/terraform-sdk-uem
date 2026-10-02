# Mock API Responses

HTTP response fixtures served by `internal/mockserver` during tests. Each
fixture is a JSON envelope carrying metadata, the request headers the test
should send, and the response the mock server returns when the matcher picks
this fixture.

Tests run against this server by default (`TEST_MODE=mock`) so no live tenant
is needed.

## Layout

```
mock-responses/
  <domain>/<resource>/
    [<platform>/<payload-type>/]
      <method>-<by-id|by-uuid>-v<N>.json    # GET / DELETE single
      <method>-v<N>.json                    # POST create, PUT update, etc.
      search-v<N>.json                      # list
```

Platform and payload-type subdirectories are added only when a resource has
platform- or payload-specific shapes (e.g.
`profiles/appleosx/disk-encryption/`).

## Filename standard

- **Kebab-case** for directories and filenames (no underscores, no camelCase).
- **Form**: `<method>-<path-param-form>-v<api-version>.json`
  - `<method>` — `get`, `create`, `update`, `delete`, `search`
  - `<path-param-form>` — `by-id` (int path param) or `by-uuid` (UUID path
    param). Omit when the endpoint has no path parameter.
  - `<api-version>` — matches the `Accept: application/json;version=N` header.
- **Examples**: `get-by-id-v2.json`, `create-v2.json`, `get-by-uuid-v1.json`.

Actual id/uuid values live only inside the JSON (`metadata.endpoint` +
response body), never in the filename. Filename tells you the call shape at a
glance; fixtures can be renumbered without renaming the file.

## Envelope shape

```json
{
  "metadata": {
    "endpoint": "/api/mdm/profiles/12346",
    "method": "GET",
    "platform": "AppleOsX",
    "version": "2",
    "description": "macOS disk-encryption profile — GET by int id (V2)",
    "api_validation": {
      "console_url": "https://cn0000.example.com",
      "uem_version": "25.9.1134.9",
      "api_version": "2",
      "accept_header": "application/json;version=2",
      "endpoint": "GET /api/mdm/profiles/12346",
      "org_group_id": 12345,
      "org_group_uuid": "00000000-0000-0000-0000-000000000000",
      "captured_at": "2026-04-23",
      "result": "accepted",
      "response_id_example": 68961,
      "ticket": "internal-ticket",
      "notes": "...",
      "request_templates": [
        "testdata/request-templates/profiles/appleosx/disk-encryption/disk-encryption-create-v2.json"
      ]
    }
  },
  "request": {
    "headers": {
      "Accept": "application/json;version=2",
      "Content-Type": "application/json"
    },
    "query_params": { "platform": "AppleOsX" }
  },
  "response": {
    "status_code": 200,
    "headers": { "Content-Type": "application/json; charset=utf-8" },
    "body": { "General": { "ProfileId": 12346, "...": "..." } }
  }
}
```

The `org_group_uuid` value above is a placeholder, not a real organization
group UUID; a real fixture records the UUID of the org group it was captured
in.

### Required metadata fields

| Field | Purpose |
|---|---|
| `endpoint` | Path the matcher compares against. Exact ids (e.g. `/api/mdm/profiles/12346`) beat template patterns (`/api/mdm/profiles/{id}`). |
| `method` | HTTP method. Must match exactly for the fixture to be considered. |
| `version` | `N` from `application/json;version=N`. Enables the version scoring bonus. |

### Optional but recommended

| Field | Purpose |
|---|---|
| `platform` | Matcher awards +50 when a `platform=` query param matches (used by search). |
| `description` | One-line human-readable summary. |
| `api_validation` | Provenance — how and when the fixture was captured. See below. |

### `api_validation` block

Documents *how* the fixture was captured. Not read by the mock server —
purely for developers reading the fixture to understand its lineage.

| Key | Purpose |
|---|---|
| `console_url` | UEM tenant the capture came from. |
| `uem_version` | `ProductVersion` from `GET /api/system/info`. |
| `api_version` / `accept_header` | Exact version header used. |
| `endpoint` | `<METHOD> <path>` hit on the live tenant. |
| `org_group_id` / `org_group_uuid` | Org group context for writes. |
| `captured_at` | ISO date (YYYY-MM-DD). |
| `result` | `accepted`, `rejected`, `error_400`, etc. |
| `response_id_example` | Real (pre-anonymization) resource id returned, for cross-referencing the UEM console UI. |
| `ticket` | Jira or similar tracking reference. |
| `notes` | Non-obvious behavior, quirks, validation-error trail. |
| `request_templates` | Repo-relative paths under `testdata/request-templates/` whose bodies produce a response of this shape. Shared endpoints may list multiple. |

## Matcher scoring

The mock server picks the highest-scoring fixture for each request. Scoring
(see `internal/mockserver/matcher.go`):

| Contributor | Score |
|---|---|
| HTTP method match | +100 (required; else fixture disqualified) |
| Exact path match | **+200** |
| Template path match (e.g. `/api/mdm/profiles/{id}` via regex) | +100 |
| Each matching query param | +10 each |
| `platform=` query param match | +50 |
| `Accept: version=N` matches fixture `metadata.version` | +75 |

Two fixtures should not tie on the same request. If you add a new GET that
shares a path template with an existing fixture, bake a unique anonymized id
into `metadata.endpoint` — the exact-match bonus wins.

## Capturing a new fixture

1. Use `bin/mock-capture` (see `tools/mock-capture/`). It handles OAuth,
   anonymization, and envelope writing. Pass `--accept-version=N` if your
   endpoint needs a non-default version.
2. Move the output into the right
   `<domain>/<resource>/<platform>/<payload>/` directory.
3. Rename to the kebab-case convention above.
4. If the fixture will share a path template with an existing one, pick an
   unused anonymized id and bake it into `metadata.endpoint` +
   `response.body.<id-field>`.
5. Fill in `metadata.api_validation` with provenance.
6. Link relevant request templates via
   `metadata.api_validation.request_templates`.
7. Add a test under `tests/` that exercises the fixture through the mock
   server.

## Migration state

Fixtures under `profiles/appleosx/` (2026-04-23, internal-ticket macOS CRUD
sweep) follow this standard. Flat legacy fixtures at the root of `profiles/`
(e.g. `get_profile_12345.json`, `get_android_68748.json`,
`create_appleosx_network_success.json`) predate the standard and will be
refactored in a separate pass — do not add new fixtures there, add them in
the structured layout.
