// Code generated. DO NOT EDIT.

package mamv2

import "github.com/euc-oss/terraform-sdk-uem/client"

// AppAssignmentBspV1ModelV2 represents BSP app assignments with smart groups applicable for online and offline licenses.

type AppAssignmentBspV1ModelV2 struct {
	// Collection of smart group uuids applicable for offline licenses.
	SmartGroupsOfflineLicenses []string `json:"smart_groups_offline_licenses,omitempty"`
	// Collection of smart group uuids applicable for online licenses.
	SmartGroupsOnlineLicenses []string `json:"smart_groups_online_licenses,omitempty"`
}

// AppAssignmentDistributionV2Model represents Distribution Model for assignments of an application.

type AppAssignmentDistributionV2Model struct {
	// App delivery method.
	AppDeliveryMethod string `json:"app_delivery_method,omitempty"`
	// App track id for Android For Work Apps.
	AppTrackID string `json:"app_track_id,omitempty"`
	// Collection of application transforms uuids applicable only for Windows SFD apps.
	ApplicationTransforms []string `json:"application_transforms,omitempty"`
	// Auto update devices with previous versions is applicable for Android, iOS and Windows internal apps.
	AutoUpdateDevicesWithPreviousVersions *bool `json:"auto_update_devices_with_previous_versions,omitempty"`
	// Auto update priority. This is supposed to be only for Android and will default to null.
	AutoUpdatePriority *int `json:"auto_update_priority,omitempty"`
	// BSP app assignments with smart groups applicable for online and offline licenses.
	BspAssignments *AppAssignmentBspV1ModelV2 `json:"bsp_assignments,omitempty"`
	// Gets or sets the deferral message content for uem application deferral.
	DeferralMessageContent string `json:"deferral_message_content,omitempty"`
	// Gets or sets the deferral message headline for uem application deferral.
	DeferralMessageHeadline string `json:"deferral_message_headline,omitempty"`
	// Gets or sets an application installation deferral notification type.
	DeferralNotificationType *int `json:"deferral_notification_type,omitempty"`
	// Description of the assignment group.
	Description string `json:"description,omitempty"`
	// Display in App Catalog flag is applicable for macOS and Windows SFD internal apps.
	DisplayInAppCatalog *bool `json:"display_in_app_catalog,omitempty"`
	// The effective datetime for the application in Admin's timezone. Applicable for internal application only. If effective datetime is null or not provided then current admin's datetime will be considered.
	EffectiveDate client.UEMTime `json:"effective_date,omitempty"`
	// Hide notifications flag is applicable only for Windows SFD apps.
	HideNotifications *bool `json:"hide_notifications,omitempty"`
	// Gets or sets whether installer deferral is allowed.
	InstallerDeferralAllowed *bool `json:"installer_deferral_allowed,omitempty"`
	// Gets or sets the installer deferral exit code.
	InstallerDeferralExitCode string `json:"installer_deferral_exit_code,omitempty"`
	// Gets or sets the number of hours of installer deferral interval.
	InstallerDeferralInterval *int `json:"installer_deferral_interval,omitempty"`
	// Flag to check if the assignment is default.
	IsDefaultAssignment *bool `json:"is_default_assignment,omitempty"`
	// Keep app updated automatically (To push the application to the eligible devices). This is supposed to be only for macOS.
	KeepAppUpdatedAutomatically *bool `json:"keep_app_updated_automatically,omitempty"`
	// Gets or sets the maximum duration until the execution can be deferred (in days).
	MaxDeferralDurationInDays *int `json:"max_deferral_duration_in_days,omitempty"`
	// Gets or sets the maximum number of times installation can be deferred for UEM Deferral.
	MaxDeferrals *int `json:"max_deferrals,omitempty"`
	// Assignment level msi deployment override options
	MsiDeploymentOverrideParams *MsiDeploymentOptionsV1ModelV2 `json:"msi_deployment_override_params,omitempty"`
	// Name of the assignment group.
	Name string `json:"name"`
	// App pre release version applicable for Android For Work apps.
	PreReleaseVersion *int `json:"pre_release_version,omitempty"`
	// Flag to check the reboot override option.
	RebootOverride *bool `json:"reboot_override,omitempty"`
	// Requires approval flag is applicable only for Windows SFD apps.
	RequiresApproval *bool `json:"requires_approval,omitempty"`
	// Collection of smart group uuids.
	SmartGroups []string `json:"smart_groups,omitempty"`
	// Purchased application assignments with VPP licenses
	VppAppDetails *AppAssignmentVppV1ModelV2 `json:"vpp_app_details,omitempty"`
}

// AppAssignmentRestrictionV1ModelV2 represents Restriction Model for assignments of an application.

type AppAssignmentRestrictionV1ModelV2 struct {
	// Desired state management flag and applicable for macOS SFD apps &amp; Windows Desktop SFD apps.
	DesiredStateManagement *bool `json:"desired_state_management,omitempty"`
	// This flag allows admin to take over management of user installed application so that additional actions like Removals, Application configuration can be applied to the end user device. This flag is applicable for iOS and Windows SFD apps.
	MakeAppMdmManaged *bool `json:"make_app_mdm_managed,omitempty"`
	// If Managed access is enabled, by default Require Management will be enabled
	ManagedAccess *bool `json:"managed_access,omitempty"`
	// Prevent backing up application data to iCloud(iOS only)
	PreventApplicationBackup *bool `json:"prevent_application_backup,omitempty"`
	// If prevent removal is enabled, then it will prevent an app from being removed from devices after installation or conversion to management.
	PreventRemoval *bool `json:"prevent_removal,omitempty"`
	// Removes the application from end user device when the device unenrolls and applicable for Apple, macOS and Android apps.
	RemoveOnUnenroll *bool `json:"remove_on_unenroll,omitempty"`
}

// AppAssignmentRuleDeviceV1ModelV2 represents Model contains the URL to fetch the devices and corresponding assignment status

type AppAssignmentRuleDeviceV1ModelV2 struct {
	// Gets a list of JSON HAL links.
	Links map[string]interface{} `json:"_links,omitempty"`
	// Assignment status of the application for a device.
	AssignmentStatus string `json:"assignment_status,omitempty"`
}

// AppAssignmentRuleDevicesV1ModelListV2 represents Devices list for which the given application assignment has been added, removed or unchanged.

type AppAssignmentRuleDevicesV1ModelListV2 struct {
	// Model containing assignment status and link to fetch the device details for list of devices
	DevicesAssignmentStatus []AppAssignmentRuleDeviceV1ModelV2 `json:"devices_assignment_status,omitempty"`
	// Total number of record count
	TotalCount *int `json:"total_count,omitempty"`
}

// AppAssignmentRuleV2Model represents List of assignments and exclusions assigned for an application.

type AppAssignmentRuleV2Model struct {
	// Assignment level msi deployment override options
	ApplicationMsiDeploymentParams *MsiDeploymentOptionsV1ModelV2 `json:"application_msi_deployment_params,omitempty"`
	// Application Assignment list
	Assignments []AppAssignmentV2Model `json:"assignments,omitempty"`
	// Collection of SmartGroup UUIDs
	ExcludedSmartGroups []string `json:"excluded_smart_groups,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// AppAssignmentTunnelV1ModelV2 represents Tunnel Model for assignments of an application.

type AppAssignmentTunnelV1ModelV2 struct {
	// AFW Tunnel profile UUID Accepted formats **uuid** E.g. f6caf44b-5fc7-1880-f09a-4a5bb560d5d8
	AfwPerAppVpnProfileUUID string `json:"afw_per_app_vpn_profile_uuid,omitempty"`
	// AMAPI Tunnel profile UUID Accepted formats **uuid** E.g. f6caf44b-5fc7-1880-f09a-4a5bb560d5d8
	AmapiPerAppVpnProfileUUID string `json:"amapi_per_app_vpn_profile_uuid,omitempty"`
	// Tunnel profile UUID Accepted formats **uuid** E.g. f6caf44b-5fc7-1880-f09a-4a5bb560d5d8
	PerAppVpnProfileUUID string `json:"per_app_vpn_profile_uuid,omitempty"`
}

// AppAssignmentV2Model represents Model for assignments of an application.

type AppAssignmentV2Model struct {
	// Gets or sets a SDK profile, modular SDK profile mappings with the assignment.
	AppProfilesMapping []ApplicationDeploymentProfileMapV1ModelV2 `json:"app_profiles_mapping,omitempty"`
	// List of application attributes.
	ApplicationAttributes []AppConfigurationV1ModelV2 `json:"application_attributes,omitempty"`
	// List of application configurations.
	ApplicationConfiguration []AppConfigurationV1ModelV2 `json:"application_configuration,omitempty"`
	// Default Application Policy Response Model.
	ApplicationPolicy *ApplicationPolicyV1ModelV2 `json:"application_policy,omitempty"`
	// Distribution Model for assignments of an application.
	Distribution AppAssignmentDistributionV2Model `json:"distribution"`
	// Gets or sets a value indicating whether the configuration was saved from managed configuration enterprise template.
	IsAndroidEnterpriseConfigTemplate *bool `json:"is_android_enterprise_config_template,omitempty"`
	// Gets or sets a value indicating whether an Apple education assignment is configured or not. Edu assignment does not allow deletion on assignment page, this is used to group edu assignment separately with other assignment.
	IsAppleEducationAssignment *bool `json:"is_apple_education_assignment,omitempty"`
	// Flag to check if the assignment configuration is being saved through DDUI.
	IsDynamicTemplateSaved *bool `json:"is_dynamic_template_saved,omitempty"`
	// Priority of an assignment policy with 0 being the highest priority.
	Priority *int `json:"priority"`
	// Restriction Model for assignments of an application.
	Restriction *AppAssignmentRestrictionV1ModelV2 `json:"restriction,omitempty"`
	// Tunnel Model for assignments of an application.
	Tunnel *AppAssignmentTunnelV1ModelV2 `json:"tunnel,omitempty"`
}

// AppAssignmentVppV1ModelV2 represents Purchased application assignments with VPP licenses

type AppAssignmentVppV1ModelV2 struct {
	// Purchased Application's VPP assignments.
	LicenseUsage []AssignmentLicenseUsageV1ModelV2 `json:"license_usage,omitempty"`
}

// AppConfigTemplateV2Model represents Application configuration model

type AppConfigTemplateV2Model struct {
	// The key of the application configuration
	Key                  string                     `json:"key,omitempty"`
	NestedConfigurations []AppConfigTemplateV2Model `json:"nested_configurations,omitempty"`
	// Application configuration type
	Type *int `json:"type,omitempty"`
}

// AppConfigurationV1ModelV2 represents Application configuration model

type AppConfigurationV1ModelV2 struct {
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// The key of the application configuration
	Key string `json:"key,omitempty"`
	// Nested Configurations for app configs
	NestedConfigurations []AppConfigurationV1ModelV2 `json:"nested_configurations,omitempty"`
	// Application configuration type
	Type *int `json:"type,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
	// Value of the application configuration
	Value string `json:"value,omitempty"`
}

// AppCriteriaApiModelV2 represents Model for application criteria modal dialog.

type AppCriteriaApiModelV2 struct {
	// Gets or sets the identifier of the application.
	ApplicationIdentifier string `json:"ApplicationIdentifier,omitempty"`
	// Gets or sets the build version of the application.
	BuildNumber *int `json:"BuildNumber,omitempty"`
	// Gets or sets the major version of the application.
	MajorVersion *int `json:"MajorVersion,omitempty"`
	// Gets or sets the minor version of the application.
	MinorVersion *int `json:"MinorVersion,omitempty"`
	// Gets or sets the fix version of the application.
	RevisionNumber *int `json:"RevisionNumber,omitempty"`
	// Gets or sets the version condition. Supported values- Any, EqualTo, NotEqualTo, GreaterThan, GreaterThanEqualTo, LessThan, LessThanEqualTo.
	VersionCondition string `json:"VersionCondition,omitempty"`
}

// AppDependencyModelV2 represents Application dependency model.

type AppDependencyModelV2 struct {
	// Gets or sets the application dependency id.
	ApplicationDependencyID *int `json:"ApplicationDependencyId,omitempty"`
	// Gets or sets the name of the dependency.
	Name string `json:"Name,omitempty"`
}

// AppDeploymentOptionsModelV2 represents A model class for deployment options.

type AppDeploymentOptionsModelV2 struct {
	// A model class for how to install options.
	HowToInstall *HowToInstallApiModelV2 `json:"HowToInstall,omitempty"`
	// A model class for how to install options.
	WhenToCallInstallComplete *WhenToCallInstallCompleteApiModelV2 `json:"WhenToCallInstallComplete,omitempty"`
	// A model class for when to install options.
	WhenToInstall *WhenToInstallApiModelV2 `json:"WhenToInstall,omitempty"`
}

// AppFilesOptionsModelV2 represents A model class for files options.

type AppFilesOptionsModelV2 struct {
	// Gets or sets list of application dependency Ids.
	AppDependenciesList []AppDependencyModelV2 `json:"AppDependenciesList,omitempty"`
	// Gets or sets list of uploaded patch files.
	AppPatchesList []AppPatchModelV2 `json:"AppPatchesList,omitempty"`
	// Gets or sets list of uploaded transform files.
	AppTransformsList []AppTransformModelV2 `json:"AppTransformsList,omitempty"`
	// Model class for the application uninstall process section applicable for .exe/.msi application(s).
	ApplicationUnInstallProcess *AppUnInstallProcessModelV2 `json:"ApplicationUnInstallProcess,omitempty"`
}

// AppListUsingProvisioningProfileModelV2 represents A list of applications using the same provisioning profile.

type AppListUsingProvisioningProfileModelV2 struct {
	// Application UUIDs using this provisioning profile.
	AppUUIDs []ApplicationUuidV2 `json:"appUuids,omitempty"`
	// Date on which the provisioning profile was created.
	CreationDate string `json:"creationDate,omitempty"`
	// Type of the device (e.g. Apple).
	DeviceType string `json:"deviceType,omitempty"`
	// Date on which the provisioning profile will expire.
	ExpirationDate string `json:"expirationDate,omitempty"`
	UUID           string `json:"uuid,omitempty"`
}

// AppPatchModelV2 represents Patch for a MSI file uploaded.

type AppPatchModelV2 struct {
	// Gets or sets the uploaded patch file blob id.
	PatchBlobID *int `json:"PatchBlobId,omitempty"`
	// Gets or sets the uploaded patch file name.
	PatchFileName string `json:"PatchFileName,omitempty"`
	// Gets or sets the id of the uploaded patch.
	PatchID *int `json:"PatchId,omitempty"`
	// Gets or sets the type of the uploaded patch. Supported values : Additive, Cumulative.
	PatchType string `json:"PatchType,omitempty"`
}

// AppRemovalDeviceResponseV2Model represents Response model that returns a list of Impacted Devices for App removal command based on the search criteria.

type AppRemovalDeviceResponseV2Model struct {
	// List of AppRemovalDeviceV2Model.
	Events []AppRemovalDeviceV2Model `json:"events,omitempty"`
	// The result set page index.
	Page *int `json:"page,omitempty"`
	// Maximum records per page.
	PageSize *int `json:"page_size,omitempty"`
	// Total number of results.
	Total *int `json:"total,omitempty"`
}

// AppRemovalDeviceV2Model represents Represents the app removal impacted device model.

type AppRemovalDeviceV2Model struct {
	// Bundle Identifier of the app for which removal command was queued.
	BundleIdentifier string `json:"bundle_identifier,omitempty"`
	// Represent the device unique identifier.
	DeviceUUID string `json:"device_uuid,omitempty"`
	// Device friendly name.
	FriendlyName string `json:"friendly_name,omitempty"`
	// Date on which the App Removal command was last modified.
	ModifiedOn client.UEMTime `json:"modified_on,omitempty"`
	// Represent the unique identifier of the organization group.
	OrganizationGroupUUID string `json:"organization_group_uuid,omitempty"`
	// Represents the value of the threshold Status.
	ThresholdStatus *int `json:"threshold_status,omitempty"`
	// Represent the unique identifier of threshold.
	ThresholdUUID string `json:"threshold_uuid,omitempty"`
}

// AppRemovalDevicesReportRequestModelV2 represents Request model that encapsulates the data for ARP impacted devices log report.

type AppRemovalDevicesReportRequestModelV2 struct {
	// Represents report's export format.
	ExportFormat *int `json:"export_format,omitempty"`
	// The name of the column the results should be ordered by.
	OrderBy *int `json:"order_by,omitempty"`
	// The unique identifier of the organization group.
	OrganizationGroupUUID string `json:"organization_group_uuid,omitempty"`
	// The search text for which the results will be filtered by.
	SearchText string `json:"search_text,omitempty"`
	// The sort order by direction.
	SortDirection *int `json:"sort_direction,omitempty"`
	// The Threshold Status for which the results will be filtered by.
	ThresholdStatus *int `json:"threshold_status,omitempty"`
	// Represents the threshold uuid for which command is queued.
	ThresholdUUID string `json:"threshold_uuid,omitempty"`
}

// AppRemovalLogReportRequestModelV2 represents Request model that encapsulates the data for ARP log report.

type AppRemovalLogReportRequestModelV2 struct {
	// List of bundle identifiers.
	BundleIdentifiers []string `json:"bundle_identifiers,omitempty"`
	// The end of the application removal command date range in ISO 8601 format.
	EndDate client.UEMTime `json:"end_date,omitempty"`
	// Represents report's export format..
	ExportFormat *int `json:"export_format,omitempty"`
	// The name of the column the results should be ordered by.
	OrderBy *int `json:"order_by,omitempty"`
	// The unique identifier of the organization group.
	OrganizationGroupUUID string `json:"organization_group_uuid,omitempty"`
	// The search text for which the results will be filtered by.
	SearchText string `json:"search_text,omitempty"`
	// The sort order by direction.
	SortDirection *int `json:"sort_direction,omitempty"`
	// The start of the application removal command date range in ISO 8601 format.
	StartDate client.UEMTime `json:"start_date,omitempty"`
	// The Threshold Status for which the results will be filtered by.
	ThresholdStatus []*int `json:"threshold_status,omitempty"`
}

// AppRemovalLogRequestModelV2 represents Request model that encapsulates the data for ARP log load and search.

type AppRemovalLogRequestModelV2 struct {
	// The application name for which the results will be filtered by.
	ApplicationName string `json:"application_name,omitempty"`
	// List of bundle identifiers.
	BundleIdentifier []string `json:"bundle_identifier,omitempty"`
	// The end of the application removal command date range in UTC by which the results will be filtered. Format of the date is YYYY-MM-DD.
	EndDate client.UEMTime `json:"end_date,omitempty"`
	// The name of the column the results should be ordered by.
	OrderBy *int `json:"order_by,omitempty"`
	// The unique identifier of the organization group.
	OrganizationGroupUUID string `json:"organization_group_uuid,omitempty"`
	// The result set page index.
	Page *int `json:"page,omitempty"`
	// Maximum records per page.
	PageSize *int `json:"page_size,omitempty"`
	// The search text for which the results will be filtered by.
	SearchText string `json:"search_text,omitempty"`
	// The sort order by direction.
	SortDirection *int `json:"sort_direction,omitempty"`
	// The start of the application removal command date range in UTC by which the results will be filtered. Format of the date is YYYY-MM-DD.
	StartDate client.UEMTime `json:"start_date,omitempty"`
	// The Threshold Status for which the results will be filtered by.
	ThresholdStatus []*int `json:"threshold_status,omitempty"`
}

// AppRemovalProtectionLogResponseV2Model represents Response model that returns a list of App removal protection logs based on the search criteria.

type AppRemovalProtectionLogResponseV2Model struct {
	// List of AppRemovalProtectionLogV2Model.
	Events []AppRemovalProtectionLogV2Model `json:"events,omitempty"`
	// The result set page index.
	Page *int `json:"page,omitempty"`
	// Maximum records per page.
	PageSize *int `json:"page_size,omitempty"`
	// Total number of results.
	Total *int `json:"total,omitempty"`
}

// AppRemovalProtectionLogV2Model represents Represents the app removal protection log model.

type AppRemovalProtectionLogV2Model struct {
	// Represent the name of the application.
	ApplicationName string `json:"application_name,omitempty"`
	// Bundle Id of the app for which removal command was queued.
	BundleIdentifier string `json:"bundle_identifier,omitempty"`
	// Date on which the App Removal command was created.
	CreatedOn client.UEMTime `json:"created_on,omitempty"`
	// Date on which the App Removal command was last modified.
	ModifiedOn client.UEMTime `json:"modified_on,omitempty"`
	// Represents the source of command.
	Source *int `json:"source,omitempty"`
	// Count of devices for which command is queued.
	ThresholdCount *int `json:"threshold_count,omitempty"`
	// Indicate whether the threshold is created via the DSM flow for the ARP.
	ThresholdCreatedViaDsmFlow *bool `json:"threshold_created_via_dsm_flow,omitempty"`
	// Represents the value of the threshold Status.
	ThresholdStatus *int `json:"threshold_status,omitempty"`
	// Represents the the threshold uuid for which command is queued.
	ThresholdUUID string `json:"threshold_uuid,omitempty"`
}

// AppRemovalThresholdDetailV2Model represents Represents the app removal Threshold details model.

type AppRemovalThresholdDetailV2Model struct {
	// Action to be performed
	Action *int `json:"action,omitempty"`
	// The unique identifier of the organization group.
	OrganizationGroupUUID string `json:"organization_group_uuid,omitempty"`
	// Represents the the threshold uuid for which command is queued.
	ThresholdUUID string `json:"threshold_uuid,omitempty"`
}

// AppTrackV2 is a generated model type.

type AppTrackV2 struct {
	TrackAlias string `json:"trackAlias,omitempty"`
	TrackID    string `json:"trackId,omitempty"`
}

// AppTransformModelV2 represents Transform for a MSI file uploaded.

type AppTransformModelV2 struct {
	// Gets or sets the blob id for the uploaded transform file.
	TransformBlobID *int `json:"TransformBlobId,omitempty"`
	// Gets or sets the uploaded transform file name.
	TransformFileName string `json:"TransformFileName,omitempty"`
	// Gets or sets the application transform id.
	TransformID *int `json:"TransformId,omitempty"`
	// Gets or sets Application Transform Uuid.
	TransformUUID string `json:"TransformUuid,omitempty"`
}

// AppUnInstallProcessModelV2 represents Model class for the application uninstall process section applicable for .exe/.msi application(s).

type AppUnInstallProcessModelV2 struct {
	// Model class for the application uninstall process section applicable for .exe/.msi application(s).
	CustomScript *CustomScriptApiModelV2 `json:"CustomScript,omitempty"`
	// Gets or sets a value indicating whether value indicating whether custom script is used or not.
	UseCustomScript *bool `json:"UseCustomScript,omitempty"`
}

// ApplicationAssignmentModelV2 represents The application assignment model.

type ApplicationAssignmentModelV2 struct {
	// Gets or sets the Android For Work VPN profile id associated with the application.
	AfwVpnProfileID *int `json:"AfwVpnProfileId,omitempty"`
	// Gets or sets flag to enable assume management for user installed iOS Apps.
	AllowManagement string `json:"AllowManagement,omitempty"`
	// Gets or sets the Android Management API VPN profile id associated with the application.
	AmapiVpnProfileID *int `json:"AmapiVpnProfileId,omitempty"`
	// Gets or sets a value indicating whether custom application attribute keys and values should be sent to the device.
	AppAttribute string `json:"AppAttribute,omitempty"`
	// Gets or sets the app attributes.
	AppAttributes []ApplicationConfigurationModelV2 `json:"AppAttributes,omitempty"`
	// Gets or sets a value indicating whether custom application configuration keys and values should be sent to the device.
	AppConfig string `json:"AppConfig,omitempty"`
	// Gets or sets the app configs.
	AppConfigs []ApplicationConfigurationModelV2 `json:"AppConfigs,omitempty"`
	// Gets or sets a value indicating whether application backup is enabled.
	ApplicationBackup string `json:"ApplicationBackup,omitempty"`
	// Gets or sets The application transforms ids attached to the application.
	ApplicationTransformIds []*int `json:"ApplicationTransformIds,omitempty"`
	// Gets or sets a value indicating whether the newest version of the app should be pushed to devices that have already downloaded the app.
	AutoUpdateDevicesWithPreviousVersion string `json:"AutoUpdateDevicesWithPreviousVersion,omitempty"`
	// Gets or sets the effective date time for the Application.
	EffectiveDate client.UEMTime `json:"EffectiveDate,omitempty"`
	// Gets or sets a value indicating whether the per app VPN flag for iOS devices is enabled.
	PerAppVpn string `json:"PerAppVpn,omitempty"`
	// Gets or sets a value indicating whether prevent application removal attribute keys and values should be sent to the device.
	PreventRemoval string `json:"PreventRemoval,omitempty"`
	// Gets or sets the push mode for the application.
	PushMode string `json:"PushMode,omitempty"`
	// Gets or sets the application rank.
	Rank *int `json:"Rank,omitempty"`
	// Gets or sets a value indicating whether the appslication should be removed on unenrollment.
	RemoveOnUnEnroll string `json:"RemoveOnUnEnroll,omitempty"`
	// Gets or sets the smart group id.
	SmartGroupID *int `json:"SmartGroupId,omitempty"`
	// Gets or sets the smart group name.
	SmartGroupName string `json:"SmartGroupName,omitempty"`
	// Gets or sets the Smart Group uuid.
	SmartGroupUUID string `json:"SmartGroupUuid,omitempty"`
	// Gets or sets a value indicating whether gets or sets the value whether to display in app catalog.
	VisibleInAppCatalog *bool `json:"VisibleInAppCatalog,omitempty"`
	// Gets or sets the VPN profile id associated with the application.
	VpnProfileID *int `json:"VpnProfileId,omitempty"`
	// Gets or sets the VPN Profile ID associated with the application.
	VpnProfileUUID string `json:"VpnProfileUuid,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// ApplicationBranchCacheStatisticsModelV2 represents The BranchCache statistics for the application deployments.

type ApplicationBranchCacheStatisticsModelV2 struct {
	// Summary of the BranchCache statistics for the application deployments.
	BranchcacheStatistics *BranchCacheStatisticsSummaryModelV2 `json:"branchcache_statistics,omitempty"`
	// List of BranchCache statistics of each device for an application deployment.
	Devices []DeviceBranchCacheStatisticsModelV2 `json:"devices,omitempty"`
}

// ApplicationCategoriesModelV2 represents The application categories model.

type ApplicationCategoriesModelV2 struct {
	// Gets or sets the category name.
	Name string `json:"Name,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// ApplicationCategoriesResultV2 represents Application Category List.

type ApplicationCategoriesResultV2 struct {
	// Gets or sets application details.
	Categories []ApplicationCategoryDetailsV2 `json:"Categories,omitempty"`
}

// ApplicationCategoriesV2Model represents The list of application categories.

type ApplicationCategoriesV2Model struct {
	// The list of application categories.
	Category []ApplicationCategoryV2Model `json:"category,omitempty"`
}

// ApplicationCategoryDetailsV2 represents Application Category Details.

type ApplicationCategoryDetailsV2 struct {
	// Gets or sets category Id.
	CategoryID *int `json:"CategoryId,omitempty"`
	// Gets or sets type of Category.
	CategoryType string `json:"CategoryType,omitempty"`
	// Gets or sets description of Category.
	Description string `json:"Description,omitempty"`
	// Gets or sets labelKey.
	LabelKey string `json:"LabelKey,omitempty"`
	// Gets or sets locationGroup Id.
	LocationGroupID *int `json:"LocationGroupId,omitempty"`
	// Gets or sets managedBy.
	ManagedBy string `json:"ManagedBy,omitempty"`
	// Gets or sets name of Category.
	Name string `json:"Name,omitempty"`
}

// ApplicationCategoryV2Model represents The application category.

type ApplicationCategoryV2Model struct {
	// The category identifier.
	CategoryID *int `json:"category_id,omitempty"`
	// The category name.
	Name string `json:"name,omitempty"`
}

// ApplicationConfigurationModelV2 represents The application configuration model.

type ApplicationConfigurationModelV2 struct {
	// Gets or sets configuration key.
	Key string `json:"Key,omitempty"`
	// Gets or sets nested Configurations.
	NestedConfigurations []ApplicationConfigurationModelV2 `json:"NestedConfigurations,omitempty"`
	// Gets or sets configuration value type, String = 1, Integer = 2, Boolean = 3, Choice = 9, Multiselect = 10, Hidden = 11, Bundle = 15, BundleArray = 16.
	Type string `json:"Type,omitempty"`
	// Gets or sets configuration value.
	Value string `json:"Value,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// ApplicationConfigurationV2Model represents Contains information about the application configuration.

type ApplicationConfigurationV2Model struct {
	// Configuration key.
	Key string `json:"key,omitempty"`
	// Type of the configuration value.
	Type *int `json:"type,omitempty"`
	// Configuration value.
	Value string `json:"value,omitempty"`
}

// ApplicationDeploymentProfileMapV1ModelV2 represents SDK profiles/Modular SDK profiles for assignments of an application.

type ApplicationDeploymentProfileMapV1ModelV2 struct {
	// Gets or sets configuration type.
	ConfigurationType string `json:"configuration_type,omitempty"`
	// Gets or sets Device Profile Uuid.
	DeviceProfileUUID string `json:"device_profile_uuid,omitempty"`
	// Gets or sets category.
	ProfileCategory string `json:"profile_category,omitempty"`
}

// ApplicationFilesOptionsV2Model represents The list of application files options.

type ApplicationFilesOptionsV2Model struct {
	// The list of application transforms.
	ApplicationTransformsList []ApplicationTransformV2Model `json:"application_transforms_list,omitempty"`
}

// ApplicationFiltersModelV2 represents The information related to application filters

type ApplicationFiltersModelV2 struct {
	// The label key corresponding to {WanderingWiFi.AirWatch.Entity.Device.ApplicationStatusReason} or {WanderingWiFi.AirWatch.Entity.Device.ApplicationDeploymentAction} enum member to be globalized for the purpose of display.
	DisplayKey string `json:"display_key,omitempty"`
	// The key value corresponding to {WanderingWiFi.AirWatch.Entity.Device.ApplicationStatusReason} or {WanderingWiFi.AirWatch.Entity.Device.ApplicationDeploymentAction} enum member.
	Key string `json:"key,omitempty"`
	// The value corresponding to {WanderingWiFi.AirWatch.Entity.Device.ApplicationStatusReason} or {WanderingWiFi.AirWatch.Entity.Device.ApplicationDeploymentAction} enum member.
	Value string `json:"value,omitempty"`
}

// ApplicationPolicyV1ModelV2 represents Default Application Policy Response Model.

type ApplicationPolicyV1ModelV2 struct {
	// The list of additional application privileges for this application.
	AdditionalApplicationPrivileges []string `json:"additional_application_privileges,omitempty"`
	// Indicates whether the application should be allowed to display widgets on the device's home screen.
	AllowHomeScreenWidgets *bool `json:"allow_home_screen_widgets,omitempty"`
	// Indicates whether modifications to the default values are allowed. Administrator's should not modify the default values of certain trusted applications.
	AllowModifyingDefaults *bool `json:"allow_modifying_defaults,omitempty"`
	// Indicates whether VPN Lockdown Exemption can be enabled or disabled.
	AllowVpnLockdownExemption *bool `json:"allow_vpn_lockdown_exemption,omitempty"`
	// Indicates whether work and personal versions of the app should be allowed to communicate with one another (e.g., calendar applications).
	ConnectedWorkAndPersonalApps *bool `json:"connected_work_and_personal_apps,omitempty"`
	// The default permission policy to apply for this application.
	DefaultPermissionPolicy string `json:"default_permission_policy,omitempty"`
	// The minimum version code for this application
	MinimumVersionCode *int `json:"minimum_version_code,omitempty"`
	// List of permissions requested by this application, overriding the default permission policy. Titles and descriptions are localized to the user and organization group.
	PermissionPolicyOverrides []RuntimeApplicationPermissionV2 `json:"permission_policy_overrides,omitempty"`
	// The user control settings to apply for this application.
	UserControlSettings string `json:"user_control_settings,omitempty"`
}

// ApplicationRequestV2Model represents Model for an application request for an app for which the app approval request workflow has been initiated.

type ApplicationRequestV2Model struct {
	// Gets or sets the status of the application request.
	ApprovalStatus *int `json:"approval_status"`
	// Gets or sets the device UUID for the application.
	DeviceUUID string `json:"device_uuid"`
	// Gets or sets the notes from the Workspace ONE Intelligence about the approval status of the request.
	Notes string `json:"notes"`
	// Gets or sets the Status change time for the current request when it is processed by a third party tool like Service Now.
	UpdatedAt client.UEMTime `json:"updated_at"`
	// Gets or sets the User who processed the request.
	UpdatedBy string `json:"updated_by"`
}

// ApplicationSearchV2Model represents The application search V2 model.

type ApplicationSearchV2Model struct {
	// The list of applications from the search operation.
	Applications []ApplicationV2Model `json:"applications,omitempty"`
	// The list of books from the search operation.
	Books []BookV2Model `json:"books,omitempty"`
	// The result set page index.
	Page *int `json:"page,omitempty"`
	// Maximum records per page.
	PageSize *int `json:"page_size,omitempty"`
	// Total number of results.
	Total *int `json:"total,omitempty"`
}

// ApplicationSupportedModelV2Model represents The application supported model.

type ApplicationSupportedModelV2Model struct {
	// The model identifier.
	ModelID *int `json:"model_id,omitempty"`
	// The model name.
	ModelName string `json:"model_name,omitempty"`
}

// ApplicationSupportedModelsModelV2 represents The application supported models model.

type ApplicationSupportedModelsModelV2 struct {
	// Gets or sets the model name.
	Name string `json:"Name,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// ApplicationSupportedModelsV2Model represents The list of application supported models.

type ApplicationSupportedModelsV2Model struct {
	// The list of application supported models.
	Model []ApplicationSupportedModelV2Model `json:"model,omitempty"`
}

// ApplicationTransformV2Model represents The application transform.

type ApplicationTransformV2Model struct {
	// The application transform unique identifier.
	ApplicationTransformUUID string `json:"application_transform_uuid,omitempty"`
	// The application unique identifier.
	ApplicationUUID string `json:"application_uuid,omitempty"`
	// The blob unique identifier of the uploaded tranform.
	TransformBlobUUID string `json:"transform_blob_uuid,omitempty"`
	// The name of the uploaded transform file.
	TransformFileName string `json:"transform_file_name,omitempty"`
}

// ApplicationUuidV2 represents UUID of an Application.

type ApplicationUuidV2 struct {
	// The UUID of the Application.
	AppUUIDs string `json:"appUuids,omitempty"`
}

// ApplicationV2Model represents The application V2 model.

type ApplicationV2Model struct {
	// The actual file version of the application.
	ActualFileVersion string `json:"actual_file_version,omitempty"`
	// The name of the uploaded application file.
	ApplicationFileName string `json:"application_file_name,omitempty"`
	// The application name.
	ApplicationName string `json:"application_name,omitempty"`
	// The rank of the application.
	ApplicationRank *int `json:"application_rank,omitempty"`
	// The size of the application.
	ApplicationSize string `json:"application_size,omitempty"`
	// The source of the application.
	ApplicationSource *int `json:"application_source,omitempty"`
	// The type of the application.
	ApplicationType string `json:"application_type,omitempty"`
	// The download URL for the application.
	ApplicationURL string `json:"application_url,omitempty"`
	// The version of the application.
	ApplicationVersion string `json:"application_version,omitempty"`
	// The number of devices to which application is assigned.
	AssignedDeviceCount *int `json:"assigned_device_count,omitempty"`
	// The assignment status of the application.
	AssignmentStatus string `json:"assignment_status,omitempty"`
	// The bundle identifier of the application.
	BundleID string `json:"bundle_id,omitempty"`
	// The list of application categories.
	CategoryList *ApplicationCategoriesV2Model `json:"category_list,omitempty"`
	// The comments of the application.
	Comments string `json:"comments,omitempty"`
	// The content gateway identifier.
	ContentGatewayID *int `json:"content_gateway_id,omitempty"`
	// The description of the application.
	Description string `json:"description,omitempty"`
	// The developer name of the application.
	Developer string `json:"developer,omitempty"`
	// The developer email of the application.
	DeveloperEmail string `json:"developer_email,omitempty"`
	// The developer phone of the application.
	DeveloperPhone string `json:"developer_phone,omitempty"`
	// The external store identifier.
	ExternalID string `json:"external_id,omitempty"`
	// The list of application files options.
	FilesOptions *ApplicationFilesOptionsV2Model `json:"files_options,omitempty"`
	// The uuid of the uploaded icon blob data.
	IconBlobUuID string `json:"icon_blob_uuId,omitempty"`
	// The name of the uploaded icon file.
	IconFileName string `json:"icon_file_name,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// The number of devices to which application is installed.
	InstalledDeviceCount *int `json:"installed_device_count,omitempty"`
	// Flag to indicate if auto update version in case of on-demand mode.
	IsAutoUpdateVersion *bool `json:"is_auto_update_version,omitempty"`
	// Flag to indicate if uploaded file is a dependency file.
	IsDependencyFile *bool `json:"is_dependency_file,omitempty"`
	// Flag to indicate if application will be used for product provisioning.
	IsEnableProvisioning *bool `json:"is_enable_provisioning,omitempty"`
	// Flag to indicate if the application can be reimbursed or not.
	IsReimbursable *bool `json:"is_reimbursable,omitempty"`
	// The large Icon URI.
	LargeIconUri string `json:"large_icon_uri,omitempty"`
	// The medium Icon URI.
	MediumIconUri string `json:"medium_icon_uri,omitempty"`
	// The name of the uploaded metadata file.
	MetadataFileName string `json:"metadata_file_name,omitempty"`
	// The MSI deployment parameters.
	MsiDeploymentParameters *MsiDeploymentParametersV2Model `json:"msi_deployment_parameters,omitempty"`
	// The number of devices to which application is assigned but not installed.
	NotInstalledDeviceCount *int `json:"not_installed_device_count,omitempty"`
	// The organization group identifier.
	OrganizationGroupUUID string `json:"organization_group_uuid,omitempty"`
	// The platform of the application.
	Platform string `json:"platform,omitempty"`
	// The push mode of the application.
	PushMode *int `json:"push_mode,omitempty"`
	// The root organization group name.
	RootOrganizationGroupName string `json:"root_organization_group_name,omitempty"`
	// The small Icon URI.
	SmallIconUri string `json:"small_icon_uri,omitempty"`
	// The list of assigned smart groups.
	SmartGroups []SmartGroupApplicationMapV2Model `json:"smart_groups,omitempty"`
	// The status of the application.
	Status string `json:"status,omitempty"`
	// The support email of the application.
	SupportEmail string `json:"support_email,omitempty"`
	// The support phone of the application.
	SupportPhone string `json:"support_phone,omitempty"`
	// The list of application supported models.
	SupportedModels *ApplicationSupportedModelsV2Model `json:"supported_models,omitempty"`
	// The supported processor architecture x86/x64. This is valid only for MSI/ZIP/EXE files with software distribution.
	SupportedProcessorArchitecture string `json:"supported_processor_architecture,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
	// The version identifier of the application.
	VersionIdentifier string `json:"version_identifier,omitempty"`
}

// ApplicationsProvisionProfileModelV2 represents A list of applications using the same provisioning profile.

type ApplicationsProvisionProfileModelV2 struct {
	// Application UUID
	AppUUIDs []string `json:"AppUuids,omitempty"`
	// date on which the provisioning profile was created
	CreationDate string `json:"CreationDate,omitempty"`
	// Type of the device
	DeviceType string `json:"DeviceType,omitempty"`
	// date on which the provisioning profile will expire
	ExpirationDate string `json:"ExpirationDate,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// AssignmentLicenseUsageV1ModelV2 represents Contains information about the assignment of a purchased application.

type AssignmentLicenseUsageV1ModelV2 struct {
	// Number of allocated licenses.
	Allocated *int `json:"allocated,omitempty"`
	// Number of redeemed licenses.
	Redeemed *int `json:"redeemed,omitempty"`
	// The smart group uuid assigned with the application.
	SmartGroupUUID string `json:"smart_group_uuid,omitempty"`
}

// BookV2Model represents The application V2 model.

type BookV2Model struct {
	// The number of device to which book is assigned.
	AssignedDeviceCount *int `json:"assigned_device_count,omitempty"`
	// The assignment status of the book.
	AssignmentStatus string `json:"assignment_status,omitempty"`
	// The book name.
	BookName string `json:"book_name,omitempty"`
	// The rank of the book.
	BookRank *int `json:"book_rank,omitempty"`
	// The size of the book.
	BookSize string `json:"book_size,omitempty"`
	// The source of the book.
	BookSource *int `json:"book_source,omitempty"`
	// The list of application categories.
	CategoryList *ApplicationCategoriesV2Model `json:"category_list,omitempty"`
	// The description of the book.
	Description string `json:"description,omitempty"`
	// The developer name of the book.
	Developer string `json:"developer,omitempty"`
	// The developer email of the book.
	DeveloperEmail string `json:"developer_email,omitempty"`
	// The developer phone of the book.
	DeveloperPhone string `json:"developer_phone,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// The number of device to which book is installed.
	InstalledDeviceCount *int `json:"installed_device_count,omitempty"`
	// The large Icon URI.
	LargeIconUri string `json:"large_icon_uri,omitempty"`
	// The medium Icon URI.
	MediumIconUri string `json:"medium_icon_uri,omitempty"`
	// The number of device to which book is assigned but not installed.
	NotInstalledDeviceCount *int `json:"not_installed_device_count,omitempty"`
	// The organization group identifier.
	OrganizationGroupUUID string `json:"organization_group_uuid,omitempty"`
	// The platform of the application.
	Platform *int `json:"platform,omitempty"`
	// The push mode of the application.
	PushMode *int `json:"push_mode,omitempty"`
	// The root organization group name.
	RootOrganizationGroupName string `json:"root_organization_group_name,omitempty"`
	// The small Icon URI.
	SmallIconUri string `json:"small_icon_uri,omitempty"`
	// The list of assigned smart groups.
	SmartGroups []SmartGroupApplicationMapV2Model `json:"smart_groups,omitempty"`
	// The status of the book.
	Status string `json:"status,omitempty"`
	// The support email of the book.
	SupportEmail string `json:"support_email,omitempty"`
	// The support phone of the book.
	SupportPhone string `json:"support_phone,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// BranchCacheStatisticsSummaryModelV2 represents Summary of the BranchCache statistics for the application deployments.

type BranchCacheStatisticsSummaryModelV2 struct {
	// Total number of bytes from the cache/peers for the application downloads for all devices.
	TotalCacheBytes *int64 `json:"total_cache_bytes,omitempty"`
	// Total number of devices that have the applications installed.
	TotalDeviceCount *int `json:"total_device_count,omitempty"`
	// Total number of devices with BranchCache mode enabled i.e. BranchCache mode on device is distributed/local/hosted
	TotalDevicesBranchcacheModeEnabled *int `json:"total_devices_branchcache_mode_enabled,omitempty"`
	// Total number of devices that did not utilize BranchCache in the application deployments i.e. application bytes from cache/peers was equal to zero for those devices.
	TotalDevicesNotUtilizedBranchcache *int `json:"total_devices_not_utilized_branchcache,omitempty"`
	// Total number of devices that utilized BranchCache in the application deployments i.e. application bytes from cache/peers was greater than zero for those devices.
	TotalDevicesUtilizedBranchcache *int `json:"total_devices_utilized_branchcache,omitempty"`
	// The savings as a percentage that is the bytes downloaded from the cache/peers over total application sizes for the device(s).
	TotalSavingsPercent *int `json:"total_savings_percent,omitempty"`
	// Total number of bytes from the server for the application downloads for all devices.
	TotalServerBytes *int64 `json:"total_server_bytes,omitempty"`
}

// BulkSearchRequestV2Model represents Request model for bulk search operations containing package identifiers and optional filtering parameters

type BulkSearchRequestV2Model struct {
	// Optional locale code for localized application descriptions and metadata (e.g., 'en-US', 'fr-FR', 'de-DE', 'cs-CZ'). The system will attempt to return content in the requested locale, falling back to en-US, then the application's default locale.
	Locale string `json:"locale,omitempty"`
	// List of unique package identifiers to search for in the enterprise repository. Each identifier should match the format used throughout the repository (e.g., 'Microsoft.VisualStudioCode').
	PackageIdentifiers []string `json:"package_identifiers"`
	// Primary operating system platform supported by this application package. Additional platform support may be available in specific versions.
	Platform *int `json:"platform,omitempty"`
}

// BulkSearchResponseV2Model represents Response model for bulk search operations containing found applications and identifiers that were not found

type BulkSearchResponseV2Model struct {
	// Collection of applications that were successfully found in the repository, matching the requested package identifiers and any applied filters.
	Applications []EnterpriseApplicationV2Model `json:"applications,omitempty"`
	// List of package identifiers from the original request that could not be found in the enterprise repository. This helps identify missing or incorrectly specified package names.
	NotFound []string `json:"not_found,omitempty"`
	// Total number of applications available in the enterprise application repository.
	TotalAppsCount *int `json:"total_apps_count,omitempty"`
}

// CustomScriptApiModelV2 represents Model class for the application uninstall process section applicable for .exe/.msi application(s).

type CustomScriptApiModelV2 struct {
	// Gets or sets the custom script type (Supported values: Input, Upload).
	CustomScriptType string `json:"CustomScriptType,omitempty"`
	// Gets or sets the application uninstall command provided.
	UninstallCommand string `json:"UninstallCommand,omitempty"`
	// Gets or sets the id value of the uninstall script file uploaded on the console. Supported file types : js,jse,ps1,ps1xml,psc1,psd1,psm1,pssc,cdxml,vbs,vbe,wsf,wsc.
	UninstallScriptBlobID *int `json:"UninstallScriptBlobId,omitempty"`
}

// DependencyInfoV2Model represents Comprehensive dependency information specifying all prerequisites required for successful package installation and operation across different dependency categories

type DependencyInfoV2Model struct {
	// External software or components that must be manually installed or configured outside of the enterprise repository. These dependencies cannot be automatically resolved by the package management system.
	ExternalDependencies []string `json:"external_dependencies,omitempty"`
	// Other application packages from the enterprise repository that must be installed before this package. This creates an installation order dependency chain.
	PackageDependencies []PackageDependencyV2Model `json:"package_dependencies,omitempty"`
	// List of Windows optional features that must be enabled on the target system before package installation. These are system-level features that may require administrative privileges to enable.
	WindowsFeatures []string `json:"windows_features,omitempty"`
	// Required Windows system libraries and runtime components that must be present on the target system. These are typically redistributable packages from Microsoft or other vendors.
	WindowsLibraries []string `json:"windows_libraries,omitempty"`
}

// DeploymentByCriteriaApiModelV2 represents A model class for application deployment.

type DeploymentByCriteriaApiModelV2 struct {
	// Model for application criteria modal dialog.
	AppCriteria *AppCriteriaApiModelV2 `json:"AppCriteria,omitempty"`
	// Gets or sets criteria type. Supported values- AppExists, AppDoesNotExist, FileExists, FileDoesNotExist, RegistryExists, RegistryDoesNotExist.
	CriteriaType string `json:"CriteriaType,omitempty"`
	// A model class for the file criteria.
	FileCriteria *FileCriteriaApiModelV2 `json:"FileCriteria,omitempty"`
	// Gets or sets logical condition. Supported values : End, And, Or.
	LogicalCondition string `json:"LogicalCondition,omitempty"`
	// A model class for the registry criteria.
	RegistryCriteria *RegistryCriteriaApiModelV2 `json:"RegistryCriteria,omitempty"`
}

// DeploymentByCustomScriptApiModelV2 represents Model class for using custom script section applicable for .exe/.msi application(s).

type DeploymentByCustomScriptApiModelV2 struct {
	// Gets or sets command to run the script.
	CommandToRunTheScript string `json:"CommandToRunTheScript,omitempty"`
	// Gets or sets the blob id of the script file associated. Supported file types : js, jse, ps1, ps1xml, psc1, psd1, psm1, pssc, cdxml, vbs, vbe, wsf, wsc.
	CustomScriptFileBlodID *int `json:"CustomScriptFileBlodId,omitempty"`
	// Gets or sets script Type (JScript, PowerShell, VBScript).
	ScriptType string `json:"ScriptType,omitempty"`
	// Gets or sets the success exit code.
	SuccessExitCode *int `json:"SuccessExitCode,omitempty"`
}

// DeviceBranchCacheStatisticsModelV2 represents The BranchCache statistics for the application deployments for a specific device.

type DeviceBranchCacheStatisticsModelV2 struct {
	// The application size in bytes.
	ApplicationSize *int64 `json:"application_size,omitempty"`
	// The application UUID.
	ApplicationUUID string `json:"application_uuid,omitempty"`
	// If BranchCache was leveraged for the application download for that device.
	BranchcacheEnabled string `json:"branchcache_enabled,omitempty"`
	// The number of bytes from the cache/peers for the application download for that device.
	CacheBytes *int64 `json:"cache_bytes,omitempty"`
	// The configured BranchCache mode for the device.
	ClientMode string `json:"client_mode,omitempty"`
	// The device UUID of the device to be queried.
	DeviceUUID string `json:"device_uuid,omitempty"`
	// The download sources that was used for an application download for that device.
	DownloadSources string `json:"download_sources,omitempty"`
	// The list of hosted server names if the device is configured in Hosted mode.
	HostedServers []string `json:"hosted_servers,omitempty"`
	// The number of bytes from the server for the application download for that device.
	ServerBytes *int64 `json:"server_bytes,omitempty"`
}

// DeviceInformationV2Model represents Contains information about the device on which the application to be installed.

type DeviceInformationV2Model struct {
	// UDID of the device
	DeviceUdid string `json:"device_udid,omitempty"`
	// Unique identifier of the device.
	DeviceUUID string `json:"device_uuid,omitempty"`
	// MAC Address of the device.
	MacAddress string `json:"mac_address,omitempty"`
	// Device serial number
	SerialNumber string `json:"serial_number,omitempty"`
}

// EnterpriseAppListResponseV2Model represents Response for search operation.

type EnterpriseAppListResponseV2Model struct {
	// Collection of application entries matching the search criteria or representing the requested page of all applications. Each entry contains essential application metadata for display and selection purposes.
	Applications []EnterpriseApplicationV2Model `json:"applications,omitempty"`
	// Boolean indicator that specifies whether additional pages of results are available beyond the current response. Use this to determine if pagination controls should be displayed to the user.
	HasMore *bool `json:"has_more,omitempty"`
	// Current page number in the result set, starting from 1. This field is only present for search operations and helps with pagination navigation and result context.
	PageCount *int `json:"page_count,omitempty"`
	// Opaque pagination token that can be used as the 'pageKey' parameter in subsequent requests to retrieve the next page of results. This field is null when no additional pages are available.
	PageKey string `json:"page_key,omitempty"`
	// Total number of applications available in the enterprise application repository.
	TotalAppsCount *int `json:"total_apps_count,omitempty"`
}

// EnterpriseAppPackageResponseV2Model represents Comprehensive response model for individual package operations, containing detailed package metadata, installer information, dependencies, and version management data

type EnterpriseAppPackageResponseV2Model struct {
	// ISO 8601 timestamp indicating when this package version was first added to the enterprise repository. This helps track package lifecycle and deployment history.
	CreatedAt client.UEMTime `json:"created_at,omitempty"`
	// Primary locale code for the application package, indicating the language and region for which the package was originally developed and contains the most complete localization.
	DefaultLocale string `json:"default_locale,omitempty"`
	// Comprehensive dependency information specifying all prerequisites required for successful package installation and operation across different dependency categories
	Dependencies *DependencyInfoV2Model `json:"dependencies,omitempty"`
	// Specifies the privilege elevation requirements for installing this package. This helps determine the necessary user permissions and security context for installation.
	ElevationRequirement *int `json:"elevation_requirement,omitempty"`
	// Boolean indicator specifying whether additional related information is available beyond the current response, such as more version history entries.
	HasMore *bool `json:"has_more,omitempty"`
	// App icon hash generated with SHA256 algorithm.
	IconSha256 string `json:"icon_sha256,omitempty"`
	// App icon url.
	IconURL string `json:"icon_url,omitempty"`
	// List of supported installation modes that determine the level of user interaction during package installation. This allows for flexible deployment scenarios from fully automated to interactive installations.
	InstallModes []*int `json:"install_modes,omitempty"`
	// Expected collection of exit codes from the installer indicating a successful installation. This is used to validate that the installation completed without errors.
	InstallerSuccessCodes []string `json:"installer_success_codes,omitempty"`
	// Command-line switches and parameters for customizing installer behavior during automated deployments and enterprise installation scenarios
	InstallerSwitches *InstallerSwitchesV2Model `json:"installer_switches,omitempty"`
	// Format type of the package installer, indicating the installation mechanism and expected behavior.
	InstallerType *int `json:"installer_type,omitempty"`
	// Collection of available installer packages for different architectures, platforms, or configurations. Each installer entry contains specific download and installation information.
	Installers []InstallerInfoV2Model `json:"installers,omitempty"`
	// Minimum operating system version required for successful installation and operation of the package. This helps prevent installation on incompatible systems.
	MinimumOsVersion string `json:"minimum_os_version,omitempty"`
	// Unique identifier for the application package. This identifier remains consistent across all versions of the package.
	PackageIdentifier string `json:"package_identifier,omitempty"`
	// Human-readable display name of the application package.
	PackageName string `json:"package_name,omitempty"`
	// Specific version string of the package being returned.
	PackageVersion string `json:"package_version,omitempty"`
	// Pagination token for retrieving additional related information such as version history or related packages. Use this token in subsequent requests to continue pagination.
	PageKey string `json:"page_key,omitempty"`
	// List of operating system platforms supported by this package version. Each entry specifies a platform where the package can be successfully installed and executed.
	Platform []*int `json:"platform,omitempty"`
	// Name of the organization or individual responsible for creating and maintaining the application package. This is typically the software vendor or developer.
	PublisherName string `json:"publisher_name,omitempty"`
	// Installation scope determining whether the package is installed for the current user only or system-wide for all users. This affects installation permissions and package visibility.
	Scope *int `json:"scope,omitempty"`
	// Concise description of the application's primary purpose and functionality, localized according to the requested locale preferences.
	ShortDescription string `json:"short_description,omitempty"`
	// ISO 8601 timestamp of the most recent modification to this package version's metadata or associated files. This helps identify when package information was last refreshed.
	UpdatedAt client.UEMTime `json:"updated_at,omitempty"`
}

// EnterpriseApplicationV2Model represents Enterprise application model optimized for listing and search operations, containing essential metadata for application discovery and selection

type EnterpriseApplicationV2Model struct {
	// Comprehensive list of all available versions for this application package, ordered from newest to oldest. This enables version-specific deployments and rollback scenarios.
	AllVersions []string `json:"all_versions,omitempty"`
	// Default locale code for the application, indicating the primary language and regional settings for which the application provides the most complete localization and documentation.
	DefaultLocale string `json:"default_locale,omitempty"`
	// App icon hash generated with SHA256 algorithm.
	IconSha256 string `json:"icon_sha256,omitempty"`
	// App icon url.
	IconURL string `json:"icon_url,omitempty"`
	// The application information for latest versioned package.
	LatestVersion *LatestVersionAppPackageModelV2 `json:"latest_version,omitempty"`
	// Short, memorable identifier or alias for the application that can be used as an alternative to the full package identifier for easier reference and command-line usage.
	Moniker string `json:"moniker,omitempty"`
	// Unique identifier for the application package used throughout the enterprise repository. This identifier follows consistent naming conventions and remains stable across package versions.
	PackageIdentifier string `json:"package_identifier,omitempty"`
	// User-friendly display name of the application, localized according to the requested locale preferences and suitable for presentation in user interfaces.
	PackageName string `json:"package_name,omitempty"`
	// Primary operating system platform supported by this application package.
	Platform string `json:"platform,omitempty"`
	// Name of the software publisher, vendor, or organization responsible for the application. This provides attribution and helps users identify trusted sources.
	Publisher string `json:"publisher,omitempty"`
	// Brief description of the application's primary functionality and purpose, localized based on the requested locale with fallback to available languages.
	ShortDescription string `json:"short_description,omitempty"`
	// List of tags associated with the application for categorization and enhanced search capabilities. Tags help users discover applications by functionality or use case.
	Tags []string `json:"tags,omitempty"`
}

// EntityIdV2 is a generated model type.

type EntityIdV2 struct {
	Value *int64 `json:"Value,omitempty"`
}

// EntityV1ModelV2 represents The Entity Model for the blob. Includes the numeric ID and Guid

type EntityV1ModelV2 struct {
	// The ID of the entity
	Value *int `json:"Value,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// FileCriteriaApiModelV2 represents A model class for the file criteria.

type FileCriteriaApiModelV2 struct {
	// Gets or sets the build version of the application.
	BuildNumber *int `json:"BuildNumber,omitempty"`
	// Gets or sets the major version of the application.
	MajorVersion *int `json:"MajorVersion,omitempty"`
	// Gets or sets the minor version of the application.
	MinorVersion *int `json:"MinorVersion,omitempty"`
	// Gets or sets the date on which the application was last modified.
	ModifiedOn string `json:"ModifiedOn,omitempty"`
	// Gets or sets the path of the application.
	Path string `json:"Path,omitempty"`
	// Gets or sets the fix version of the application.
	RevisionNumber *int `json:"RevisionNumber,omitempty"`
	// Gets or sets Criteria Operator. Supported values: Any, EqualTo, GreaterThan, LessThan NotEqualTo, GreaterThanOrEqualTo, LessThanOrEqualTo.
	VersionCondition string `json:"VersionCondition,omitempty"`
}

// HowToInstallApiModelV2 represents A model class for how to install options.

type HowToInstallApiModelV2 struct {
	// Gets or sets a value indicating whether admin privileges are needed for the installation of a package.
	AdminPrivileges *bool `json:"AdminPrivileges,omitempty"`
	// Gets or sets the device restart option. Supported values: DoNotRestart, ForceRestart, RestartIfNeeded.
	DeviceRestart string `json:"DeviceRestart,omitempty"`
	// Gets or sets the install command to install a package using the command line ex: "/quiet".
	InstallCommand string `json:"InstallCommand,omitempty"`
	// Gets or sets install context (Supported values: Device, User) where the package has to be installed.
	InstallContext string `json:"InstallContext,omitempty"`
	// Gets or sets the amount of time in minutes that the installation process can run before the installer considers the installation may have failed. Valid range 0 - 60.
	InstallTimeoutInMinutes *int `json:"InstallTimeoutInMinutes,omitempty"`
	// Gets or sets the success exit code.
	InstallerRebootExitCode string `json:"InstallerRebootExitCode,omitempty"`
	// Gets or sets the success exit code.
	InstallerSuccessExitCode string `json:"InstallerSuccessExitCode,omitempty"`
	// Gets or sets the numbers days after which device is force restarted.
	RestartDeadlineInDays *int `json:"RestartDeadlineInDays,omitempty"`
	// Gets or sets the number of times package installation operation will be retried. Valid range 0 - 10.
	RetryCount *int `json:"RetryCount,omitempty"`
	// Gets or sets the amount of time in minutes between retry operations. Valid range 0 - 10.
	RetryIntervalInMinutes *int `json:"RetryIntervalInMinutes,omitempty"`
	// Gets or sets the uninstall device restart option. Supported values: DoNotRestart, ForceRestart, RestartIfNeeded.
	UninstallDeviceRestart string `json:"UninstallDeviceRestart,omitempty"`
}

// ImportPackageRequestV2Model represents Request model for importing the specified package into UEM.

type ImportPackageRequestV2Model struct {
	// Target system architecture for the application package.
	Architecture *int `json:"architecture,omitempty"`
	// The type of installer used by the application package.
	InstallerType *int `json:"installer_type,omitempty"`
	// The installer meta-data locale.
	Locale string `json:"locale,omitempty"`
	// The organization group uuid.
	OrganizationGroupUUID string `json:"organization_group_uuid,omitempty"`
	// Unique package identifier to search for in the enterprise repository. The identifier should match the format used throughout the repository (e.g., 'Microsoft.VisualStudioCode').
	PackageIdentifier string `json:"package_identifier"`
	// Version of the application package to search for. Supports semantic versioning (e.g., '1.2.3').
	PackageVersion string `json:"package_version"`
	// The scope of installer.
	Scope *int `json:"scope,omitempty"`
}

// ImportPackageResponseV2Model represents Response model for blobIds returned after import.

type ImportPackageResponseV2Model struct {
	IconBlobID    *EntityIdV2 `json:"icon_blob_id,omitempty"`
	PackageBlobID *EntityIdV2 `json:"package_blob_id,omitempty"`
}

// InstallerInfoV2Model represents Detailed information about a specific installer package, including download location, architecture support, installation requirements, and configuration options

type InstallerInfoV2Model struct {
	// Target processor architecture for installer package.
	Architecture *int `json:"architecture,omitempty"`
	// Comprehensive dependency information specifying all prerequisites required for successful package installation and operation across different dependency categories
	Dependencies *DependencyInfoV2Model `json:"dependencies,omitempty"`
	// Specifies the privilege elevation requirements for installing this package. This helps determine the necessary user permissions and security context for installation.
	ElevationRequirement *int `json:"elevation_requirement,omitempty"`
	// Gets or sets Install command for the installers.
	InstallCommand string `json:"install_Command,omitempty"`
	// List of installation modes supported by this installer, determining the level of user interaction and feedback during the installation process.
	InstallModes []*int `json:"install_modes,omitempty"`
	// Extracted installer file name derived from the installer URL or response headers.
	InstallerFileName string `json:"installer_file_name,omitempty"`
	// Locale code indicating the primary language and regional settings for this specific installer. This may differ from the package's default locale for localized installer packages.
	InstallerLocale string `json:"installer_locale,omitempty"`
	// SHA-256 cryptographic hash of the installer file, providing integrity verification to ensure the downloaded installer has not been modified or corrupted.
	InstallerSha256 string `json:"installer_sha256,omitempty"`
	// Expected collection of exit codes from the installer indicating a successful installation. This is used to validate that the installation completed without errors.
	InstallerSuccessCodes []string `json:"installer_success_codes,omitempty"`
	// Command-line switches and parameters for customizing installer behavior during automated deployments and enterprise installation scenarios
	InstallerSwitches *InstallerSwitchesV2Model `json:"installer_switches,omitempty"`
	// Specific installer format and technology used by this installer package. This determines the installation behavior, required tools, and supported installation options.
	InstallerType *int `json:"installer_type,omitempty"`
	// Direct download URL for the installer package.
	InstallerURL string `json:"installer_url,omitempty"`
	// Minimum operating system version required for this specific installer. This may be more restrictive than the general package requirements due to architecture or feature dependencies.
	MinimumOsVersion string `json:"minimum_os_version,omitempty"`
	// List of specific platform variants supported by this installer, providing detailed compatibility information beyond the general platform classification.
	Platform []*int `json:"platform,omitempty"`
	// Installation scope determining whether the application will be installed for the current user only or system-wide for all users. This affects required permissions and application visibility.
	Scope *int `json:"scope,omitempty"`
}

// InstallerSwitchesV2Model represents Command-line switches and parameters for customizing installer behavior during automated deployments and enterprise installation scenarios

type InstallerSwitchesV2Model struct {
	// Additional custom command-line switches specific to this installer that provide advanced configuration options or specialized installation behaviors.
	Custom string `json:"custom,omitempty"`
	// Command-line parameter prefix for specifying a custom installation directory. The actual path should be appended to this switch value during installation.
	InstallLocation string `json:"install_location,omitempty"`
	// Command-line switch for interactive installation mode, allowing full user interaction with the installer interface. This is the default mode for manual installations.
	Interactive string `json:"interactive,omitempty"`
	// Command-line parameter prefix for enabling installer logging to a specified file. The log file path should be appended to this switch value for debugging and auditing purposes.
	Log string `json:"log,omitempty"`
	// Command-line switch for completely silent installation with no user interface or progress indication.
	Silent string `json:"silent,omitempty"`
	// Command-line switch for silent installation that displays a progress indicator but requires no user interaction.
	SilentWithProgress string `json:"silent_with_progress,omitempty"`
	// Command-line switch for performing an upgrade installation over an existing version of the application, preserving user settings and data where possible.
	Upgrade string `json:"upgrade,omitempty"`
}

// InternalAppModelV2 represents This model represents an internal application.

type InternalAppModelV2 struct {
	// Gets or sets the actual file version of the app.
	ActualFileVersion string `json:"ActualFileVersion,omitempty"`
	// Gets or sets airWatch internal application version.
	AirwatchAppVersion string `json:"AirwatchAppVersion,omitempty"`
	// Gets or sets the bundle id of the app.
	AppID string `json:"AppId,omitempty"`
	// Gets or sets app Provisioning UUID.
	AppProvisioningProfileUUID string `json:"AppProvisioningProfileUuid,omitempty"`
	// Gets or sets the size of the application in kilo bytes.
	AppSizeInKB *int `json:"AppSizeInKB,omitempty"`
	// Gets or sets the name of the application.
	ApplicationName string `json:"ApplicationName,omitempty"`
	// Gets or sets the URL of the application.
	ApplicationURL string `json:"ApplicationUrl,omitempty"`
	// Gets or sets app configs.
	Assignments []ApplicationAssignmentModelV2 `json:"Assignments,omitempty"`
	// Gets or sets a value indicating whether the management of user installed apps should be assumed.
	AssumeManagementOfUserInstalledApp string `json:"AssumeManagementOfUserInstalledApp,omitempty"`
	// Gets or sets the build version of the app.
	BuildVersion string `json:"BuildVersion,omitempty"`
	// Gets or sets the application category list.
	CategoryList []ApplicationCategoriesModelV2 `json:"CategoryList,omitempty"`
	// Gets or sets the change log for the application.
	ChangeLog string `json:"ChangeLog,omitempty"`
	// Gets or sets the comments for the application.
	Comments string `json:"Comments,omitempty"`
	// A model class for deployment options.
	DeploymentOptions *AppDeploymentOptionsModelV2 `json:"DeploymentOptions,omitempty"`
	// Gets or sets the number of devices to which current application is Assigned.
	DevicesAssignedCount *int `json:"DevicesAssignedCount,omitempty"`
	// Gets or sets devices on which current application is installed.
	DevicesInstalledCount *int `json:"DevicesInstalledCount,omitempty"`
	// Gets or sets number of devices to which current application is assigned, but not installed.
	DevicesNotInstalledCount *int `json:"DevicesNotInstalledCount,omitempty"`
	// Gets or sets the Unique Guid values of the excluded smart groups.
	ExcludedSmartGroupGuids []string `json:"ExcludedSmartGroupGuids,omitempty"`
	// Gets or sets the Smart group Ids to be excluded from receiving the application.
	ExcludedSmartGroupIds []*int `json:"ExcludedSmartGroupIds,omitempty"`
	// A model class for files options.
	FilesOptions *AppFilesOptionsModelV2 `json:"FilesOptions,omitempty"`
	// Gets or sets the LaunchCommand./&gt;.
	LaunchCommand string `json:"LaunchCommand,omitempty"`
	// Gets or sets the LaunchType./&gt;.
	LaunchType string `json:"LaunchType,omitempty"`
	// Summary model for the macOs Software pacakge.
	MacOsSoftwareDeploymentSummary *MacOsSoftwareDeploymentSummaryModelV2 `json:"MacOsSoftwareDeploymentSummary,omitempty"`
	// Gets or sets the managed by Organization Group of the app.
	ManagedBy string `json:"ManagedBy,omitempty"`
	// Gets or sets managed By Organization Group Uuid.
	ManagedByUUID string `json:"ManagedByUuid,omitempty"`
	// Gets or sets minimum Operating System Version of the application.
	MinimumOperatingSystem string `json:"MinimumOperatingSystem,omitempty"`
	// MSI deployment param model.
	MsiDeploymentParameters *MsiDeploymentParameterModelV2 `json:"MsiDeploymentParameters,omitempty"`
	// Gets or sets the platform.
	Platform string `json:"Platform,omitempty"`
	// Gets or sets user rating of the app.
	Rating *int `json:"Rating,omitempty"`
	// Gets or sets expiration date of the Device Provisioning Profile.
	RenewalDate client.UEMTime `json:"RenewalDate,omitempty"`
	// Gets or sets A value indicating whether the application uses AirWatch Software Development Kit.
	Sdk string `json:"Sdk,omitempty"`
	// Gets or sets the SDK profile id of the app if it uses SDK Profile.
	SdkProfileID *int `json:"SdkProfileId,omitempty"`
	// Gets or sets sdk Profile Uuid of the App if it uses SDK Profile.
	SdkProfileUUID string `json:"SdkProfileUuid,omitempty"`
	// Gets or sets status of the App.
	Status string `json:"Status,omitempty"`
	// Gets or sets The supported models of the app.
	SupportedModels []ApplicationSupportedModelsModelV2 `json:"SupportedModels,omitempty"`
	// Gets or sets The names of supported models of the app.
	SupportedModelsName []string `json:"SupportedModelsName,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// KioskBookmarkResponseV2Model represents List of kiosk bookmarks profiles.

type KioskBookmarkResponseV2Model struct {
	// Gets or sets profile icon blob guid.
	BlobUUID string `json:"blob_uuid,omitempty"`
	// Gets or sets profile name.
	ProfileName string `json:"profile_name,omitempty"`
	// Gets or sets title of the bookmark.
	Title string `json:"title,omitempty"`
	// Gets or sets bookmark url.
	URL string `json:"url,omitempty"`
}

// LatestAppPackageInstallerInfoModelV2 represents The latest versioned app package installation details.

type LatestAppPackageInstallerInfoModelV2 struct {
	// Target processor architecture for installer package.
	Architecture *int `json:"architecture,omitempty"`
	// Locale code indicating the primary language and regional settings for this specific installer. This may differ from the package's default locale for localized installer packages.
	InstallerLocale string `json:"installer_locale,omitempty"`
	// Specific installer format and technology used by this installer package. This determines the installation behavior, required tools, and supported installation options.
	InstallerType *int `json:"installer_type,omitempty"`
}

// LatestVersionAppPackageModelV2 represents The application information for latest versioned package.

type LatestVersionAppPackageModelV2 struct {
	// Collection of available installer packages for different architectures, platforms, or configurations. Each installer entry contains specific download and installation information.
	Installers []LatestAppPackageInstallerInfoModelV2 `json:"installers,omitempty"`
	// Latest version string of the package being returned.
	Version string `json:"version,omitempty"`
}

// LicensesSummaryV2Model represents Contains information about the licenses summary of a purchased application.

type LicensesSummaryV2Model struct {
	// Number of allocated licenses.
	Allocated *int `json:"allocated,omitempty"`
	// Number of licenses on hold.
	OnHold *int `json:"on_hold,omitempty"`
	// Number of redeemed licenses.
	Redeemed *int `json:"redeemed,omitempty"`
	// Total number of licenses.
	Total *int `json:"total,omitempty"`
	// Number of unallocated licenses.
	Unallocated *int `json:"unallocated,omitempty"`
}

// MacOsSoftwareDeploymentSummaryModelV2 represents Summary model for the macOs Software pacakge.

type MacOsSoftwareDeploymentSummaryModelV2 struct {
	// Gets or sets whether the installation type is managed or optional.
	IsManaged string `json:"IsManaged,omitempty"`
	// Gets or sets the pkginfo file as string.
	Pkginfo string `json:"Pkginfo,omitempty"`
}

// MsiDeploymentOptionsV1ModelV2 represents Assignment level msi deployment override options

type MsiDeploymentOptionsV1ModelV2 struct {
	// Gets or sets the device restart option. Supported values: DoNotRestart, ForceRestart, RestartIfNeeded.
	DeviceRestart *int `json:"device_restart,omitempty"`
	// Insatller Reboot Exit code of the assignment.
	InstallerRebootExitCode string `json:"installer_reboot_exit_code,omitempty"`
	// Insatller Success Exit code of the assignment
	InstallerSuccessExitCode string `json:"installer_success_exit_code,omitempty"`
	// Restart Deadline of the assignment
	RestartDeadlineInDays *int `json:"restart_deadline_in_days,omitempty"`
}

// MsiDeploymentParameterModelV2 represents MSI deployment param model.

type MsiDeploymentParameterModelV2 struct {
	// Gets or sets command line options to be used when calling MSIEXEC.exe.
	CommandLineArguments string `json:"CommandLineArguments,omitempty"`
	// Gets or sets amount of time, in minutes that the installation process can run before the installer considers the installation may have failed and no longer monitors the installation operation.Range 0-60.
	InstallTimeoutInMinutes *int `json:"InstallTimeoutInMinutes,omitempty"`
	// Gets or sets the number of times the download and installation operation will be retried before the installation will be marked as failed. With a limit of ‘10' attempts.
	RetryCount *int `json:"RetryCount,omitempty"`
	// Gets or sets amount of time, in minutes between retry operations. Range 0-10.
	RetryIntervalInMinutes *int `json:"RetryIntervalInMinutes,omitempty"`
}

// MsiDeploymentParametersV2Model represents The MSI deployment parameters.

type MsiDeploymentParametersV2Model struct {
	// The command line options to be used when calling MSIEXEC.exe.
	CommandLineArguments string `json:"command_line_arguments,omitempty"`
	// The windows msi install context.
	InstallContext *int `json:"install_context,omitempty"`
	// The amount of time in minutes that the installation process can run before the installer considers the installation may have failed and no longer monitors the installation operation. Range 0-60.
	InstallTimeoutInMinutes *int `json:"install_timeout_in_minutes,omitempty"`
	// The number of times the download and installation operation will be retried before the installation will be marked as failed. With a limit of 10 attempts.
	RetryCount *int `json:"retry_count,omitempty"`
	// The amount of time in minutes between retry operations. Range 0-10.
	RetryIntervalInMinutes *int `json:"retry_interval_in_minutes,omitempty"`
}

// Office365MamIntegrationPolicyResponseModelV2 represents The Office365 MAM Integration Policy Response Model.

type Office365MamIntegrationPolicyResponseModelV2 struct {
	// Gets or sets the android policy guid.
	AndroidPolicyUUID string `json:"android_policy_uuid,omitempty"`
	// Gets or sets the iOS policy guid.
	IosPolicyUUID string `json:"ios_policy_uuid,omitempty"`
}

// PackageDependencyV2Model represents Specification for a dependency on another package within the enterprise repository, including version requirements and compatibility constraints

type PackageDependencyV2Model struct {
	// Minimum version of the dependency package required for compatibility. Any version equal to or greater than this version will satisfy the dependency requirement.
	MinimumVersion string `json:"minimum_version,omitempty"`
	// Unique identifier of the required dependency package, matching the packageIdentifier format used throughout the enterprise repository.
	PackageIdentifier string `json:"package_identifier,omitempty"`
}

// PurchasedApplicationV2Model represents Contains information about a purchased application and its assignments.

type PurchasedApplicationV2Model struct {
	// Application's iTunes Store Identifier.
	AdamID string `json:"adam_id,omitempty"`
	// Application's assignments.
	Assignments []VppAssignmentV2Model `json:"assignments,omitempty"`
	// Application's categories.
	Categories []string `json:"categories,omitempty"`
	// Application's identifier.
	Identifier string `json:"identifier,omitempty"`
	// Contains information about the licenses summary of a purchased application.
	LicensesSummary *LicensesSummaryV2Model `json:"licenses_summary,omitempty"`
	// Application's name.
	Name string `json:"name,omitempty"`
	// Application's organization group UUID.
	OrganizationGroupUUID string `json:"organization_group_uuid,omitempty"`
	// Product type.
	ProductType *int `json:"product_type,omitempty"`
	// Application's UUID.
	UUID string `json:"uuid,omitempty"`
	// Type of licensing that the application is eligbile for.
	VppAppEligibility *int `json:"vpp_app_eligibility,omitempty"`
}

// RegistryCriteriaApiModelV2 represents A model class for the registry criteria.

type RegistryCriteriaApiModelV2 struct {
	// Gets or sets the name of new key to be created in the registry.
	KeyName string `json:"KeyName,omitempty"`
	// Gets or sets the type of key to be created in the registry. Supported values : String, Binary, DWord, QWord, MultiString, ExpandableString.
	KeyType string `json:"KeyType,omitempty"`
	// Gets or sets the value of the key to be created in the registry.
	KeyValue string `json:"KeyValue,omitempty"`
	// Gets or sets the path of the key in the registry.
	Path string `json:"Path,omitempty"`
	// Gets or sets Criteria Operator. Supported values: Any, EqualTo, GreaterThan, LessThan NotEqualTo, GreaterThanOrEqualTo, LessThanOrEqualTo.
	VersionCondition string `json:"VersionCondition,omitempty"`
}

// RuntimeApplicationPermissionV2 represents Object containing runtime permissions requested by an application.

type RuntimeApplicationPermissionV2 struct {
	// The localized description for this permission.
	Description string `json:"description,omitempty"`
	// The localized name for this permission.
	Name string `json:"name,omitempty"`
	// The permission id.
	PermissionID string `json:"permission_id,omitempty"`
	// The default permission policy for this permission.
	PermissionPolicy string `json:"permission_policy,omitempty"`
}

// SmartGroupApplicationMapV2Model represents The smart group applicaion mapping.

type SmartGroupApplicationMapV2Model struct {
	// The smart group name.
	Name string `json:"name,omitempty"`
	// The smart group unique identifier.
	SmartGroupUUID string `json:"smart_group_uuid,omitempty"`
}

// VppAssignmentV2Model represents Contains information about the assignment of a purchased application.

type VppAssignmentV2Model struct {
	// Number of allocated licenses.
	Allocated *int `json:"allocated,omitempty"`
	// Contains information about the application deployment parameters.
	DeploymentParameters *VppDeploymentParametersV2Model `json:"deployment_parameters,omitempty"`
	// Indicates whether the assignment is active.
	IsActive *bool `json:"is_active,omitempty"`
	// Number of redeemed licenses.
	Redeemed *int `json:"redeemed,omitempty"`
	// The smart group uuid assigned with the application.
	SmartGroupUUID string `json:"smart_group_uuid,omitempty"`
}

// VppDeploymentParametersV2Model represents Contains information about the application deployment parameters.

type VppDeploymentParametersV2Model struct {
	// Indicates whether to assume management of the user installed application.
	AllowManagement *bool `json:"allow_management,omitempty"`
	// Application's attributes.
	ApplicationAttributes []ApplicationConfigurationV2Model `json:"application_attributes,omitempty"`
	// Application's configurations.
	ApplicationConfigurations []ApplicationConfigurationV2Model `json:"application_configurations,omitempty"`
	// Type to deploy the application.
	AssignmentType *int `json:"assignment_type,omitempty"`
	// Indicates whether to prevent the application's backup.
	PreventApplicationBackup *bool `json:"prevent_application_backup,omitempty"`
	// Indicates whether to send the prevent removal application attributes.
	PreventRemoval *bool `json:"prevent_removal,omitempty"`
	// Indicates whether to remove the application on device's unenrollment.
	RemoveOnUnenroll *bool `json:"remove_on_unenroll,omitempty"`
	// Indicates whether to send the application attributes.
	SendApplicationAttributes *bool `json:"send_application_attributes,omitempty"`
	// Indicates whether to send the application configuration.
	SendApplicationConfiguration *bool `json:"send_application_configuration,omitempty"`
	// Indicates whether to use the VPN profile.
	UseVpn *bool `json:"use_vpn,omitempty"`
	// VPN profile's UUID.
	VpnProfileUUID string `json:"vpn_profile_uuid,omitempty"`
}

// WhenToCallInstallCompleteApiModelV2 represents A model class for how to install options.

type WhenToCallInstallCompleteApiModelV2 struct {
	// Gets or sets the criteria configured to identify application.
	CriteriaList []DeploymentByCriteriaApiModelV2 `json:"CriteriaList,omitempty"`
	// Model class for using custom script section applicable for .exe/.msi application(s).
	CustomScript *DeploymentByCustomScriptApiModelV2 `json:"CustomScript,omitempty"`
	// Gets or sets the way by which an application can be identified (Supported Values: DefiningCriteria, UsingCustomScript).
	IdentifyApplicationBy string `json:"IdentifyApplicationBy,omitempty"`
	// Gets or sets a value indicating whether value indicating whether additional criteria has to be used or not.
	UseAdditionalCriteria *bool `json:"UseAdditionalCriteria,omitempty"`
}

// WhenToInstallApiModelV2 represents A model class for when to install options.

type WhenToInstallApiModelV2 struct {
	// Gets or sets data Contingencies.
	DataContingencies []DeploymentByCriteriaApiModelV2 `json:"DataContingencies,omitempty"`
	// Gets or sets the device power required for installation of the package. Valid range 0 - 100.
	DevicePowerRequired *int `json:"DevicePowerRequired,omitempty"`
	// Gets or sets the disk space required for installation of the package in KB.
	DiskSpaceRequiredInKb *int `json:"DiskSpaceRequiredInKb,omitempty"`
	// Gets or sets the RAM required for the installation of the page in MB.
	RamRequiredInMb *int `json:"RamRequiredInMb,omitempty"`
}
