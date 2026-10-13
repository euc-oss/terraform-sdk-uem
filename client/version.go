package client

import "strings"

// Version is the SDK release version that appears in the default User-Agent
// ("uem-sdk-go/<Version>"). The publishing sync replaces the placeholder with
// the three-part release version (for example 26.2.0), the same value it renders
// into the root package's Version constant. In the private repository the
// placeholder is still in place and the version reads as "dev".
const Version = "26.2.0"

// sdkVersion returns Version, or "dev" while the placeholder has not been replaced.
func sdkVersion() string {
	if strings.HasPrefix(Version, "$") {
		return "dev"
	}
	return Version
}
