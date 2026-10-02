package tests

import (
	"os"
	"testing"
)

func TestDeriveLiveTenantAliases_MapsCanonicalNames(t *testing.T) {
	in := map[string]string{
		"HOST":           "tenant-a.example.com",
		"API_KEY":        "tenant123",
		"ADMIN_USERNAME": "admin",
		"ADMIN_PASSWORD": "secret",
		"OAUTH_ENDPOINT": "https://oauth.example.com/token",
		"CLIENT_ID":      "cid",
		"CLIENT_SECRET":  "csecret",
		"GROUP_ID":       "12345", // not aliased -- ORG_GROUP_ID is liveOrgGroupID's own separate convention
	}
	got := deriveLiveTenantAliases(in)
	want := map[string]string{
		"INSTANCE_URL":     "https://tenant-a.example.com",
		"TENANT_CODE":      "tenant123",
		"USERNAME":         "admin",
		"PASSWORD":         "secret",
		"OAUTH2_TOKEN_URL": "https://oauth.example.com/token",
		"CLIENT_ID":        "cid",
		"CLIENT_SECRET":    "csecret",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d keys %v, want %d keys %v", len(got), got, len(want), want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("key %q: got %q, want %q", k, got[k], v)
		}
	}
}

func TestDeriveLiveTenantAliases_EmptyInput(t *testing.T) {
	got := deriveLiveTenantAliases(map[string]string{})
	if len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

func TestDeriveLiveTenantAliases_MissingFieldsOmitted(t *testing.T) {
	got := deriveLiveTenantAliases(map[string]string{"HOST": "tenant-a.example.com"})
	if len(got) != 1 || got["INSTANCE_URL"] != "https://tenant-a.example.com" {
		t.Errorf("got %v, want only INSTANCE_URL set", got)
	}
}

func TestLoadLiveTenantEnv_DoesNotClobberExplicitlySet(t *testing.T) {
	t.Setenv("INSTANCE_URL", "https://operator-override.example.com")
	tenantEnv := map[string]string{"HOST": "tenant-a.example.com"}
	applyLiveTenantAliases(tenantEnv)
	if got := os.Getenv("INSTANCE_URL"); got != "https://operator-override.example.com" {
		t.Errorf("INSTANCE_URL was clobbered: got %q, want operator override preserved", got)
	}
}

func TestLoadLiveTenantEnv_SetsWhenUnset(t *testing.T) {
	t.Setenv("TENANT_CODE", "")
	_ = os.Unsetenv("TENANT_CODE")
	tenantEnv := map[string]string{"API_KEY": "tenant123"}
	applyLiveTenantAliases(tenantEnv)
	if got := os.Getenv("TENANT_CODE"); got != "tenant123" {
		t.Errorf("TENANT_CODE = %q, want tenant123", got)
	}
}
