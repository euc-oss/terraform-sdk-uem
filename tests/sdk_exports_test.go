package tests

import (
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
)

func TestSDKExportsCompile(t *testing.T) {
	// Verify service constructors are accessible
	_ = sdk.NewDeviceSensorsV1Service
	_ = sdk.NewProfilesV2Service
	_ = sdk.NewScriptsV1Service
	_ = sdk.NewScriptAssignmentV1Service
	_ = sdk.NewBaselinesV1Service
	_ = sdk.NewCatalogsV1Service
	_ = sdk.NewOSVersionsV1Service
	_ = sdk.NewPlatformsV1Service
	_ = sdk.NewTemplatesV1Service
	_ = sdk.NewBaselineCreator

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
