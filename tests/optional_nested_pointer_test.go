package tests

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"

	sdk "github.com/euc-oss/terraform-sdk-uem"
)

// emptyBlockPattern matches any JSON object key whose value is an empty
// object literal `{}`. The codegen bug that this test pins caused optional
// nested-struct fields to serialize as `"Foo":{}` because encoding/json
// silently ignores `omitempty` on non-pointer struct fields. After the fix,
// such fields are emitted as `*T` and omit cleanly when nil.
var emptyBlockPattern = regexp.MustCompile(`"[A-Za-z0-9_]+"\s*:\s*\{\s*\}`)

// assertNoEmptySubBlocks fails the test with a diagnostic if the marshaled
// entity contains any `"Field":{}` sub-blocks.
func assertNoEmptySubBlocks(t *testing.T, label string, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("%s: marshal: %v", label, err)
	}
	if matches := emptyBlockPattern.FindAllString(string(raw), -1); len(matches) > 0 {
		t.Errorf("%s: serialized entity contains %d empty sub-block(s) %v — UEM rejects these with HTTP 500.\nFull payload: %s",
			label, len(matches), matches, raw)
	}
}

// TestOptionalNestedStructsOmitWhenNil is the regression for internal-ticket.
// An untouched entity with only scalar fields populated must not serialize
// empty `{}` objects for its optional nested payload fields.
func TestOptionalNestedStructsOmitWhenNil(t *testing.T) {
	t.Run("AppleOsXDeviceProfileEntityV2", func(t *testing.T) {
		ent := sdk.AppleOsXDeviceProfileEntityV2{
			General: &sdk.GeneralPayloadV2Entity{Name: "macOS Profile"},
		}
		assertNoEmptySubBlocks(t, "AppleOsXDeviceProfileEntityV2", ent)

		// Also confirm only the populated fields appear in the payload.
		raw, _ := json.Marshal(ent)
		if !strings.Contains(string(raw), `"General"`) {
			t.Errorf("expected General sub-block in payload, got %s", raw)
		}
		for _, forbidden := range []string{`"DiskEncryption"`, `"GateKeeper"`, `"Restrictions"`, `"Passcode"`, `"AssociatedDomains"`} {
			if strings.Contains(string(raw), forbidden) {
				t.Errorf("untouched field %s leaked into payload: %s", forbidden, raw)
			}
		}
	})

	// Cross-platform generality: a second Apple device profile variant.
	t.Run("AppleDeviceProfileV2Entity", func(t *testing.T) {
		ent := sdk.AppleDeviceProfileV2Entity{
			General: &sdk.GeneralPayloadV2Entity{Name: "iOS Profile"},
		}
		assertNoEmptySubBlocks(t, "AppleDeviceProfileV2Entity", ent)
	})

	// Cross-package generality: MAM v1 entity (different generated package).
	t.Run("InternalAppModelV1", func(t *testing.T) {
		ent := sdk.InternalAppModelV1{
			ApplicationName: "TestApp",
		}
		assertNoEmptySubBlocks(t, "InternalAppModelV1", ent)
	})
}

// TestOptionalNestedStructsIncludedWhenSet confirms the symmetric case:
// when a caller explicitly populates a nested pointer, it IS serialized.
func TestOptionalNestedStructsIncludedWhenSet(t *testing.T) {
	ent := sdk.AppleOsXDeviceProfileEntityV2{
		General:        &sdk.GeneralPayloadV2Entity{Name: "macOS Profile"},
		DiskEncryption: &sdk.AppleOsXDiskEncryptionPayloadEntityV2{},
	}
	raw, err := json.Marshal(ent)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"DiskEncryption"`) {
		t.Errorf("explicitly-set DiskEncryption must appear in payload, got %s", raw)
	}
}
