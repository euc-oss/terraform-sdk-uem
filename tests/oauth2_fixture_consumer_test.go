package tests

import (
	"testing"

	"github.com/euc-oss/terraform-sdk-uem/client"
	"github.com/euc-oss/terraform-sdk-uem/internal/mockserver"
)

// TestOAuth2GetToken_ConsumesVendoredFixture genuinely exercises the SDK's
// OAuth2 client-credentials token exchange against the mock server, loaded
// from the normal scanned corpus (respecting MOCK_RESPONSES_DIR) rather than
// the internal/mockserver/testdata/synthetic-placeholders bypass path every
// other OAuth2-mock test in this package uses. Deletion control: removing
// auth/oauth2_token_success.json from the corpus makes the mock server 404
// the token exchange, so client.OAuth2Auth.GetToken() returns a non-nil
// error and this test fails loudly -- unlike TestIntegration_OAuth2_SystemInfo,
// which loads that fixture's byte-identical twin via a hardcoded bypass path
// immune to corpus deletion.
func TestOAuth2GetToken_ConsumesVendoredFixture(t *testing.T) {
	ms := mockserver.LoadMockResponses(t, "../testdata/mock-responses")
	t.Cleanup(ms.Close)

	auth := client.NewOAuth2Auth(ms.URL()+"/oauth/token", "test-client-id", "test-client-secret")

	token, err := auth.GetToken()
	if err != nil {
		t.Fatalf("GetToken failed (mock server could not match a POST /oauth/token response -- is auth/oauth2_token_success.json present in the corpus?): %v", err)
	}

	const wantToken = "synthetic-placeholder-access-token-not-a-real-secret"
	if token != wantToken {
		t.Errorf("expected access token %q (the fixture's disclosed placeholder value), got %q", wantToken, token)
	}
}
