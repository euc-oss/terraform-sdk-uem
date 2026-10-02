package tests

import (
	"encoding/json"
	"testing"

	mdmv2 "github.com/euc-oss/terraform-sdk-uem/internal/mdm/v2"
	mdmv4 "github.com/euc-oss/terraform-sdk-uem/internal/mdm/v4"
)

// TestGeneralPayloadV2Entity_AfwOemType_WireInt and its v4 sibling below
// cover the AfwOemType finding that
// GeneralPayloadV2Entity.AfwOemType / GeneralPayloadV4Entity.AfwOemType are
// bare integers on the wire on BOTH request and response, not the
// swagger-declared string+enum ("Zebra"/"Samsung").
//
// Canonical evidence: GeneralPayloadV2Entity.AfwOemType (AirWatch.
// ServiceModel/Profiles/V2/Resources/GeneralPayloadV2Entity.cs:59-61) is a
// nullable ENUM directly (AfwProfileOemType?), attributes [XmlElement]
// [DataMember] only -- no [JsonConverter]/[JsonIgnore]/[IgnoreDataMember]
// and no companion ...Str property. This is NOT the quirk-12/13
// hidden-int-behind-a-string-property pattern. Backing enum
// AfwProfileOemType { Samsung = 0, Zebra = 1 } (Framework/.../
// Enumeration.cs:844-855), no None sentinel, map complete.
// ProfilesV2Controller.cs:321 and :648 bind the same GeneralPayloadV2Entity
// on BOTH create and update; the host request deserializer
// (JsonNetFormatter.cs:117-149) builds its JsonSerializer with only an
// IsoDateTimeConverter -- no StringEnumConverter anywhere -- so
// Newtonsoft's default enum handling (integer on both read and write)
// applies in both directions. Per doctrine quirk 14, a field's schema can
// be duplicated as an independent, separately-patchable entry across API
// surfaces (there: v1/v2; here: v2/v4 profile General payloads) -- fixing
// the v2 copy did not fix the v4 copy, hence both overlay actions
// (internal-source/mdmv2.profiles-general.overlay.yaml and
// internal-source/mdmv4.overlay.yaml) and both cases below.
//
// Honesty caveat (do not overstate): "int is safe on the request too"
// partly rests on documented Newtonsoft default-enum-reader behavior (its
// default reader also tolerates a string enum-name token on read, which is
// asymmetric with int-only write) -- third-party-library reasoning
// (Newtonsoft is a NuGet dependency, not itself in the canonical
// checkout), not a directly canonical-observed fact for the request side.
// Belt-and-suspenders: AndroidDeviceProfileV2Entity.Validate()
// (AndroidDeviceProfileV2Entity.cs:1220-1231) overwrites General.AfwOemType
// from the Zebra/Samsung sub-payloads regardless of the submitted value on
// Android create/update anyway.
//
// These are plain unit tests of Go's encoding/json behavior against the
// regenerated *int fields -- not live-capture claims. No fixture exists
// yet under testdata/mock-responses/ that exercises this exact field.
func TestGeneralPayloadV2Entity_AfwOemType_WireInt(t *testing.T) {
	tests := []struct {
		name string
		json string
		want int
	}{
		{name: "Samsung (canonical 0)", json: `{"AfwOemType": 0}`, want: 0},
		{name: "Zebra (canonical 1)", json: `{"AfwOemType": 1}`, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var general mdmv2.GeneralPayloadV2Entity
			if err := json.Unmarshal([]byte(tc.json), &general); err != nil {
				t.Fatalf("json.Unmarshal(%s) into GeneralPayloadV2Entity: %v", tc.json, err)
			}
			if general.AfwOemType == nil {
				t.Fatalf("AfwOemType: got nil, want *%d", tc.want)
			}
			if *general.AfwOemType != tc.want {
				t.Errorf("AfwOemType: got %d, want %d", *general.AfwOemType, tc.want)
			}
		})
	}
}

func TestGeneralPayloadV4Entity_AfwOemType_WireInt(t *testing.T) {
	tests := []struct {
		name string
		json string
		want int
	}{
		{name: "Samsung (canonical 0)", json: `{"AfwOemType": 0}`, want: 0},
		{name: "Zebra (canonical 1)", json: `{"AfwOemType": 1}`, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var general mdmv4.GeneralPayloadV4Entity
			if err := json.Unmarshal([]byte(tc.json), &general); err != nil {
				t.Fatalf("json.Unmarshal(%s) into GeneralPayloadV4Entity: %v", tc.json, err)
			}
			if general.AfwOemType == nil {
				t.Fatalf("AfwOemType: got nil, want *%d", tc.want)
			}
			if *general.AfwOemType != tc.want {
				t.Errorf("AfwOemType: got %d, want %d", *general.AfwOemType, tc.want)
			}
		})
	}
}
