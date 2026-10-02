package tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	mdmv2 "github.com/euc-oss/terraform-sdk-uem/v26/internal/mdm/v2"
)

// UEM's Newtonsoft deserializer applies JSON keys in request order, and the
// macOS V2 Restrictions sub-objects gate child settings behind a parent flag:
// a child key serialized before its parent is saved as false (doctrine quirk
// 49). internal-source/mdmv2.profiles-macos-restrictions.overlay.yaml marks
// each parent with x-field-order-first, which the codegen IR honors for
// both Go and Python. These tests pin the marshaled key order.

// keyIndexProblems reports every gated child key that appears before the
// parent key in body, or is missing from it.
func keyIndexProblems(body, parent string, children []string) []string {
	quoted := func(k string) string { return `"` + k + `":` }
	p := strings.Index(body, quoted(parent))
	if p < 0 {
		return []string{fmt.Sprintf("parent %q missing from %s", parent, body)}
	}
	var bad []string
	for _, c := range children {
		i := strings.Index(body, quoted(c))
		switch {
		case i < 0:
			bad = append(bad, fmt.Sprintf("child %q missing", c))
		case i < p:
			bad = append(bad, fmt.Sprintf("child %q (index %d) precedes parent %q (index %d)", c, i, parent, p))
		}
	}
	return bad
}

func TestMacOSRestrictionsParentKeyMarshalsFirst(t *testing.T) {
	on := boolPtr(true)
	cases := []struct {
		name     string
		model    any
		parent   string
		children []string
	}{
		{
			// The 9 panes that sort before the parent alphabetically, plus
			// one that already sorted after it.
			name: "Preferences",
			model: mdmv2.AppleOsXRestrictionPreferencesPayloadEntityV2{
				EnabledPreferencePanes: on, Accessibility: on, AppStore: on, Bluetooth: on,
				CDsAndDVDs: on, DateAndTime: on, DesktopAndScreenSaver: on,
				DictationAndSpeech: on, Displays: on, Dock: on, EnergySaver: on,
			},
			parent: "EnabledPreferencePanes",
			children: []string{"Accessibility", "AppStore", "Bluetooth", "CDsAndDVDs", "DateAndTime",
				"DesktopAndScreenSaver", "DictationAndSpeech", "Displays", "Dock", "EnergySaver"},
		},
		{
			// The 8 services that sort before the parent, plus one after.
			name: "Sharing",
			model: mdmv2.AppleOsXRestrictionSharingPayloadEntityV2{
				RestrictWhichSharingServicesAreEnabled: on, AddtoAperture: on, AddtoReadingList: on,
				AddtoiPhoto: on, AirDrop: on, AutomaticallyEnableNewSharingServices: on,
				Facebook: on, Mail: on, Messages: on, VideoServices: on,
			},
			parent: "RestrictWhichSharingServicesAreEnabled",
			children: []string{"AddtoAperture", "AddtoReadingList", "AddtoiPhoto", "AirDrop",
				"AutomaticallyEnableNewSharingServices", "Facebook", "Mail", "Messages", "VideoServices"},
		},
	}
	for _, c := range cases {
		b, err := json.Marshal(c.model)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		body := string(b)
		if !strings.HasPrefix(body, `{"`+c.parent+`":`) {
			t.Errorf("%s: parent %q is not the first key: %s", c.name, c.parent, body)
		}
		for _, p := range keyIndexProblems(body, c.parent, c.children) {
			t.Errorf("%s: %s", c.name, p)
		}
	}
}

// TestMacOSRestrictionsParentKeyCheck_PlantedControl proves the check above
// would fail on the pre-fix alphabetical order: this local struct declares
// the Preferences parent in its alphabetical slot, as the generator did
// before x-field-order-first.
func TestMacOSRestrictionsParentKeyCheck_PlantedControl(t *testing.T) {
	type alphabeticalPreferences struct {
		Accessibility          *bool `json:"Accessibility,omitempty"`
		Dock                   *bool `json:"Dock,omitempty"`
		EnabledPreferencePanes *bool `json:"EnabledPreferencePanes,omitempty"`
		EnergySaver            *bool `json:"EnergySaver,omitempty"`
	}
	b, err := json.Marshal(alphabeticalPreferences{boolPtr(true), boolPtr(true), boolPtr(true), boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	bad := keyIndexProblems(string(b), "EnabledPreferencePanes", []string{"Accessibility", "Dock", "EnergySaver"})
	if len(bad) != 2 {
		t.Fatalf("planted alphabetical order: want exactly Accessibility and Dock flagged, got %v", bad)
	}
}
