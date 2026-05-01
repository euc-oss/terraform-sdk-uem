# Integration Tests

This directory contains integration tests for the go-wsone-sdk that test against
a real Workspace ONE UEM environment.

## Prerequisites

1. **Environment Variables**: Create a `.env` file in the project root with your
   Workspace ONE credentials:

   ```bash
   # Workspace ONE UEM Environment Variables
   FRIENDLY_NAME=your-environment-name
   CLIENT_ID=your-oauth2-client-id
   CLIENT_SECRET=your-oauth2-client-secret
   API_KEY=your-tenant-code
   HOST=your-instance.awmdm.com
   ADMIN_USERNAME=your-username
   ADMIN_PASSWORD=your-password
   GROUP_ID=your-group-id
   GROUP_ID_NAME=your-group-name
   ```

2. **Workspace ONE Access**: Ensure you have access to a Workspace ONE UEM
   environment (sandbox or production).

## Running Integration Tests

### Run All Integration Tests

```bash
# From project root
go test ./tests/... -v

# Or using make
make test-integration
```

### Run Specific Test

```bash
go test ./tests/... -v -run TestIntegration_BasicAuth_SystemInfo
```

### Skip Integration Tests (Run Only Unit Tests)

```bash
go test -short ./...
```

## Test Coverage

The integration tests cover:

1. **Authentication**
   - Basic Authentication with username/password
   - OAuth2 Authentication with client credentials
   - Token refresh and caching

2. **API Requests**
   - System info endpoint (`/api/system/info`)
   - Profile search endpoint (`/api/mdm/profiles/search`)
   - Proper header injection (aw-tenant-code, Authorization)

3. **Rate Limiting**
   - Token bucket rate limiter
   - Request throttling

4. **Error Handling**
   - 404 errors for invalid endpoints
   - APIError type parsing
   - Context cancellation

5. **Retry Logic**
   - Automatic retry on transient errors
   - Exponential backoff

## Test Structure

Each integration test:

- Checks for required environment variables
- Skips if variables are not set (allows CI/CD without credentials)
- Uses `testing.Short()` to allow skipping with `-short` flag
- Provides detailed logging of API responses
- Validates expected behavior

## Safety

All integration tests are **read-only** operations:

- ✅ GET `/api/system/info` - System information
- ✅ GET `/api/mdm/profiles/search` - Profile listing
- ❌ No CREATE, UPDATE, or DELETE operations

This ensures tests can run safely against production environments without
modifying data.

## Troubleshooting

### Tests Skip with "environment variable not set"

Ensure your `.env` file is in the project root and contains all required
variables.

### Authentication Failures

1. Verify credentials are correct
2. Check that the tenant code (API_KEY) matches your environment
3. Ensure the user has API access permissions

### Rate Limiting Warnings

If rate limiting tests show warnings, this is informational only. The rate
limiter may be working correctly but with timing variance.

## Adding New Integration Tests

When adding new integration tests:

1. Use the `getTestClient()` helper function
2. Add `testing.Short()` check to allow skipping
3. Use read-only operations when possible
4. Provide detailed logging with `t.Logf()`
5. Handle missing environment variables gracefully
