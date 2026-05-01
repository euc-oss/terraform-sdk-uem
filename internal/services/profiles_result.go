// Code generated. DO NOT EDIT.
package services

import (
	mdmv2 "github.com/euc-oss/terraform-sdk-uem/internal/mdm/v2"
	mdmv4 "github.com/euc-oss/terraform-sdk-uem/internal/mdm/v4"
	"net/http"
)

// ProfileResult holds a typed response for a profiles resource.
// Exactly one platform field is non-nil. The Platform field contains
// the API value string (e.g., "Android", "Apple iOS", "To do").
type ProfileResult struct {
	Platform      string
	Headers       http.Header
	Android       *mdmv2.AndroidDeviceProfileV2Entity
	AppleOsX      *mdmv2.AppleOsXDeviceProfileEntityV2
	AppleiOS      *mdmv2.AppleDeviceProfileV2Entity
	Linux         *mdmv4.LinuxDeviceProfileEntity1V4
	Windows10     *mdmv2.WinRTDeviceProfileV2Entity
	WindowsRugged *mdmv2.QnxDeviceProfileEntityV2
}
