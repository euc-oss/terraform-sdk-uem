package tests

import (
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
)

func TestSDKExportsCompile(t *testing.T) {
	// Verify service constructors are accessible
	_ = sdk.NewDeviceSensorsV1Service
	_ = sdk.NewProfilesV2Service

	// Verify key model types are accessible
	var _ *sdk.DeviceSensorResponseV1Model
	var _ *sdk.ProfilesV2Service

	// Verify auth type re-exports are accessible without importing client package
	_ = sdk.NewBasicAuth
	_ = sdk.NewOAuth2Auth
	var _ sdk.AuthProvider
	var _ *sdk.BasicAuth
	var _ *sdk.OAuth2Auth
	var _ *sdk.APIError
}
