package tests

import "os"

// liveTenantAliasMap maps this repo's canonical .env.<tenant>.local field
// names (documented in tests/README.md, used by every tenant file) to the harness names getLiveClientWithTimeout actually
// reads via os.Getenv (tests/integration_test.go:151-157). No aliasing
// existed anywhere before this file -- TEST_MODE=live never authenticated
// against a real tenant file, only ../.env under the harness's own names.
// Confirmed against a live tenant (Task 1.2.10 D6 probe):
// client.Config.TenantCode is documented as "the aw-tenant-code header
// value", i.e. API_KEY; InstanceURL needs an explicit "https://" prefix that
// HOST omits.
var liveTenantAliasMap = map[string]string{
	"HOST":           "INSTANCE_URL", // needs "https://" prefix -- handled in deriveLiveTenantAliases
	"API_KEY":        "TENANT_CODE",
	"ADMIN_USERNAME": "USERNAME",
	"ADMIN_PASSWORD": "PASSWORD",
	"OAUTH_ENDPOINT": "OAUTH2_TOKEN_URL",
}

// liveTenantPassthroughFields use identical names on both sides -- carried
// through unchanged rather than listed in liveTenantAliasMap (which maps
// old-name -> new-name pairs).
var liveTenantPassthroughFields = []string{"CLIENT_ID", "CLIENT_SECRET"}

// deriveLiveTenantAliases maps a canonical .env.<tenant>.local file's fields
// (as read by godotenv.Read) to the harness's expected env var names. Pure
// and unit-testable -- does not touch the process environment. GROUP_ID is
// deliberately NOT aliased here: ORG_GROUP_ID is liveOrgGroupID's own,
// already-established separate convention (live_resource_lifecycle_test.go).
func deriveLiveTenantAliases(tenantEnv map[string]string) map[string]string {
	out := make(map[string]string, len(liveTenantAliasMap)+len(liveTenantPassthroughFields))
	for canonical, harnessName := range liveTenantAliasMap {
		val, ok := tenantEnv[canonical]
		if !ok || val == "" {
			continue
		}
		if canonical == "HOST" {
			val = "https://" + val
		}
		out[harnessName] = val
	}
	for _, same := range liveTenantPassthroughFields {
		if val, ok := tenantEnv[same]; ok && val != "" {
			out[same] = val
		}
	}
	return out
}

// applyLiveTenantAliases sets the aliased harness env vars into the process
// environment, but only for vars not already explicitly set -- an operator's
// own INSTANCE_URL/TENANT_CODE/etc. always takes priority over a tenant
// file's alias.
func applyLiveTenantAliases(tenantEnv map[string]string) {
	for harnessName, val := range deriveLiveTenantAliases(tenantEnv) {
		if os.Getenv(harnessName) != "" {
			continue
		}
		os.Setenv(harnessName, val) //nolint:errcheck // os.Setenv only errs on NUL bytes in name/value, never realistic here
	}
}
