package tests

import (
	"context"

	"github.com/euc-oss/terraform-sdk-uem/resources"
)

// newProfileDeleteCleanup returns a closure that deletes the given profile,
// logging (never failing) on error. The logf is injected so callers can pass
// t.Logf directly, and unit tests can substitute a spy without a *testing.T.
func newProfileDeleteCleanup(profileService *resources.ProfileService, profileID int, logf func(format string, args ...interface{})) func() {
	return func() {
		if err := profileService.Delete(context.Background(), profileID); err != nil {
			logf("cleanup: profileService.Delete(%d) failed: %v", profileID, err)
		}
	}
}

// registerProfileDeleteCleanup registers newProfileDeleteCleanup's closure
// via t.Cleanup, called immediately after a profile's ID is known -- see
// tests/live_resource_lifecycle_test.go's at-creation registration pattern,
// which this generalizes for profile_crud_test.go.
func registerProfileDeleteCleanup(t interface {
	Cleanup(func())
	Logf(string, ...interface{})
}, profileService *resources.ProfileService, profileID int) {
	t.Cleanup(newProfileDeleteCleanup(profileService, profileID, t.Logf))
}
