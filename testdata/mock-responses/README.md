# Mock API Responses

This directory contains mock API response files for testing the Workspace ONE
UEM SDK without requiring a live environment.

## Directory Structure

```text
mock-responses/
├── auth/                    # Authentication endpoints
│   ├── oauth2_token_success.json
│   └── oauth2_token_error.json
├── system/                  # System information endpoints
│   └── info_success.json
└── profiles/                # Profile management endpoints
    ├── search_android_success.json
    ├── search_ios_success.json
    ├── search_empty.json
    ├── get_profile_12345.json
    ├── get_profile_404.json
    ├── create_profile_success.json
    ├── update_profile_success.json
    └── delete_profile_success.json
```

## Response File Format

Each response file is a JSON document with three main sections:

### 1. Metadata

Describes the endpoint and its characteristics:

```json
{
  "metadata": {
    "endpoint": "/api/mdm/profiles/search",
    "method": "GET",
    "platform": "Android",
    "description": "Successful profile search returning Android profiles",
    "version": "2"
  }
}
```

### 2. Request Specification

Defines expected request characteristics (optional):

```json
{
  "request": {
    "headers": {
      "Accept": "application/json;version=2",
      "Content-Type": "application/json"
    },
    "query_params": {
      "platform": "Android"
    }
  }
}
```

### 3. Response Specification

Defines the response to return:

```json
{
  "response": {
    "status_code": 200,
    "headers": {
      "Content-Type": "application/json"
    },
    "body": {
      "Page": 0,
      "Total": 2,
      "Profiles": [...]
    }
  }
}
```

## Adding New Mock Responses

1. **Create a new JSON file** in the appropriate directory
2. **Follow the naming convention**: `{operation}_{platform}_{scenario}.json`
   - Examples: `search_android_success.json`, `get_profile_404.json`
3. **Include all three sections**: metadata, request, response
4. **Validate the JSON** structure matches actual API responses
5. **Test the mock response** by running tests with `TEST_MODE=mock`

## Using Mock Responses in Tests

Set the `TEST_MODE` environment variable to use mock responses:

```bash
# Use mock server (no live environment needed)
TEST_MODE=mock go test ./tests/...

# Use live Workspace ONE environment (requires .env file)
TEST_MODE=live go test ./tests/...
```

## Best Practices

1. **Capture Real Responses**: Base mock responses on actual API responses
2. **Include Edge Cases**: Create responses for error scenarios (404, 401, 500)
3. **Keep Responses Realistic**: Match actual API structure and data types
4. **Document Scenarios**: Use descriptive filenames and metadata descriptions
5. **Version Awareness**: Include API version in metadata when relevant

## Response Matching

The mock server matches requests based on:

1. **HTTP Method** (GET, POST, PUT, DELETE) - Required
2. **Path Pattern** (exact match or with parameters like
   `/api/mdm/profiles/{id}`) - Required
3. **Query Parameters** (optional, increases match score)
4. **Platform** (optional, for platform-specific responses)

The mock server selects the response with the highest match score.

## Examples

See the existing response files in this directory for examples of:

- Successful operations
- Error responses
- Paginated results
- Platform-specific responses
- Empty result sets
