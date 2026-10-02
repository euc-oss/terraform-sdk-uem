# Request Templates

Request bodies used to exercise the live UEM API via `bin/mock-capture`, and
preserved here as the canonical "known-accepted" inputs for each resource /
platform / payload-type combination.

These files are **not** served by the mock server. They are inputs to the
capture tool — posted to the real API, the response is what becomes a
fixture under `testdata/mock-responses/`. The loop:

```
request-templates/.../foo-create-v2.json   <-- this tree
                         │
                         ▼  POST via bin/mock-capture
                 live UEM (your-instance.awmdm.com)
                         │
                         ▼  anonymized response
mock-responses/.../create-v2.json          (fixture served by mock server)
```

## Layout

Mirrors the mock-responses tree, so a developer who wants to understand the
"create-then-get-then-update" contract for a resource can navigate the same
path in both trees:

```
request-templates/
  <domain>/<resource>/
    [<platform>/<payload-type>/]
      <payload-type>-<method>-v<N>.json
```

Example:

```
request-templates/profiles/appleosx/disk-encryption/disk-encryption-create-v2.json
mock-responses/profiles/appleosx/disk-encryption/get-by-id-v2.json
```

## Filename standard

- **Kebab-case** for directories and filenames.
- **Form**: `<payload-type>-<method>-v<api-version>.json`
  - `<payload-type>` — matches the parent directory name, e.g.
    `disk-encryption`, `custom-settings`, `restrictions`.
  - `<method>` — `create`, `update`, `delete`, `patch`.
  - `<api-version>` — matches the Accept header version the request uses.
- **Examples**: `disk-encryption-create-v2.json`,
  `custom-settings-update-v2.json`.

The payload-type prefix is redundant with the directory but keeps the file
self-identifying when opened in isolation (e.g. in an editor tab strip).

## File shape

```json
{
  "api_validation": {
    "console_url": "https://your-instance.awmdm.com",
    "uem_version": "25.9.1134.9",
    "api_version": "2",
    "accept_header": "application/json;version=2",
    "endpoint": "POST /api/mdm/profiles/platforms/appleosx/create",
    "org_group_id": 12345,
    "org_group_uuid": "00000000-0000-0000-0000-000000000000",
    "captured_at": "2026-04-23",
    "result": "accepted",
    "response_id_example": 68961,
    "ticket": "internal-ticket",
    "notes": "..."
  },

  "General": {
    "Name": "mocktest-disk-encryption",
    "DeviceType": "AppleOsX",
    "...": "..."
  },
  "DiskEncryption": {
    "DiskEncryptionFileVault2": { "...": "..." }
  }
}
```

The `org_group_uuid` value above is a placeholder, not a real organization
group UUID; a real template records the UUID of the org group it was
validated against.

Fields outside `api_validation` are the **actual request body** — sent
verbatim by `bin/mock-capture`. The UEM server is lenient about unknown keys
at the root (verified), so `api_validation` is ignored by the API and does
not affect the server's response.

## `api_validation` block

Same as the one in `mock-responses/` fixtures. Documents how the template
was validated: the tenant, UEM version, the endpoint, the result, any
non-obvious behavior discovered while iterating on the body (e.g. required-
field trial and error, enum values rejected by the server, interactions
between fields).

## Adding a new template

1. Draft the body minimally from canonical docs (see
   `docs/guides/profiles/canonical-models/`).
2. Run `bin/mock-capture --body path/to/template.json` against a `mocktest*`
   org group on the live tenant.
3. Iterate: the server validation errors (1012, "invalid X", etc.) tell you
   what's required or mutually exclusive.
4. Once the server accepts the body, fill in `api_validation` at the top
   with what you learned (validation hoops, enum surprises, etc.).
5. Commit the template + the captured `mock-responses/` fixture together.

## Migration state

Templates under `profiles/appleosx/` (2026-04-23, internal-ticket macOS CRUD
sweep) follow this layout. As with mock-responses, legacy flat templates (if
any) will be refactored in a separate pass — put new templates here in the
structured layout.
