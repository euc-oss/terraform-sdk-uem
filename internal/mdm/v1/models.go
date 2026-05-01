// Code generated. DO NOT EDIT.

package mdmv1

// CertificateV1 represents Model representing the Certificate Resource.

type CertificateV1 struct {
	// Gets or sets certificate Data in BASE64 Encoded form.
	CertificatePayload string `json:"CertificatePayload,omitempty"`
	// Gets or sets certificate Password.
	Password string `json:"Password,omitempty"`
}

// DeviceSensorAssignedSmartGroupV1Model represents A model holding the details of smart groups assigned to device sensors.

type DeviceSensorAssignedSmartGroupV1Model struct {
	// Name of the smart group.
	Name string `json:"name,omitempty"`
	// Unique identifier for the smart group.
	SmartGroupUUID string `json:"smart_group_uuid,omitempty"`
}

// DeviceSensorListResponseV1Model represents A model holding the details of device sensors for the Organization Group and the count of device sensors.

type DeviceSensorListResponseV1Model struct {
	// A list of failed templates with error details.
	ResultSet []DeviceSensorResponseV1Model `json:"result_set,omitempty"`
	// Total number of device sensors for the Organization Group.
	TotalResults *int `json:"total_results,omitempty"`
}

// DeviceSensorRequestV1Model represents Request model for creating a device sensor

type DeviceSensorRequestV1Model struct {
	// Description of the device sensor.
	Description string `json:"description,omitempty"`
	// Event triggers defining the trigger for the data collection.
	EventTrigger []*int `json:"event_trigger,omitempty"`
	// Execution architecture under which the script would be run on device.
	ExecutionArchitecture string `json:"execution_architecture,omitempty"`
	// Execution context under which the script would be run on device.
	ExecutionContext string `json:"execution_context,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// Name of the device sensor.
	Name string `json:"name"`
	// Organization group uuid
	OrganizationGroupUUID string `json:"organization_group_uuid"`
	// Platform for which the device sensor will be created.
	Platform string `json:"platform"`
	// Response type of the data.
	QueryResponseType string `json:"query_response_type,omitempty"`
	// Query type of the script.
	QueryType string `json:"query_type"`
	// Schedule trigger (in hours) defining the trigger for the data collection.
	ScheduleTrigger string `json:"schedule_trigger,omitempty"`
	// Script to be executed on device.
	ScriptData string `json:"script_data"`
	// Defines timeout for the script execution in seconds.
	Timeout *int `json:"timeout,omitempty"`
	// Trigger type of the script.
	TriggerType string `json:"trigger_type"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// DeviceSensorResponseV1Model represents A model holding the details of a device sensor

type DeviceSensorResponseV1Model struct {
	// Assigned smart groups to the device sensor.
	AssignedSmartGroups []DeviceSensorAssignedSmartGroupV1Model `json:"assigned_smart_groups,omitempty"`
	// Description of the device sensor.
	Description string `json:"description,omitempty"`
	// Event triggers defining the trigger for the data collection.
	EventTrigger []*int `json:"event_trigger,omitempty"`
	// Execution architecture under which the script would be run on device.
	ExecutionArchitecture string `json:"execution_architecture,omitempty"`
	// Execution context under which the script would be run on device.
	ExecutionContext string `json:"execution_context,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// Specifies if the sensor is read only with respect to the current organization group.
	IsReadOnly *bool `json:"is_read_only,omitempty"`
	// Name of the device sensor.
	Name string `json:"name,omitempty"`
	// Organization group name the device sensor is managed by.
	OrganizationGroupName string `json:"organization_group_name,omitempty"`
	// Identifier of the organization group.
	OrganizationGroupUUID string `json:"organization_group_uuid,omitempty"`
	// Platform for which the device sensor will be created.
	Platform string `json:"platform,omitempty"`
	// Response type of the data.
	QueryResponseType string `json:"query_response_type,omitempty"`
	// Query type of the script.
	QueryType string `json:"query_type,omitempty"`
	// Schedule trigger (in hours) defining the trigger for the data collection.
	ScheduleTrigger string `json:"schedule_trigger,omitempty"`
	// Script to be executed on device.
	ScriptData string `json:"script_data,omitempty"`
	// Defines timeout(in seconds) for the script execution.
	Timeout *int `json:"timeout,omitempty"`
	// Trigger type of the script.
	TriggerType string `json:"trigger_type,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// DeviceSensorUpdateV1Model represents Request model for updating a device sensor

type DeviceSensorUpdateV1Model struct {
	// Description of the device sensor.
	Description string `json:"description,omitempty"`
	// Event triggers defining the trigger for the data collection.
	EventTrigger []*int `json:"event_trigger,omitempty"`
	// Execution architecture under which the script would be run on device.
	ExecutionArchitecture string `json:"execution_architecture,omitempty"`
	// Execution context under which the script would be run on device.
	ExecutionContext string `json:"execution_context,omitempty"`
	// Gets or sets identifier.
	ID *int `json:"id,omitempty"`
	// Platform for which the device sensor will be created.
	Platform string `json:"platform,omitempty"`
	// Query type of the script.
	QueryType string `json:"query_type,omitempty"`
	// Schedule trigger (in hours) defining the trigger for the data collection.
	ScheduleTrigger string `json:"schedule_trigger,omitempty"`
	// Script to be executed on device.
	ScriptData string `json:"script_data,omitempty"`
	// Defines timeout for the script execution in seconds.
	Timeout *int `json:"timeout,omitempty"`
	// Trigger type of the script.
	TriggerType string `json:"trigger_type,omitempty"`
	// Gets or sets current objects UUID.
	UUID string `json:"uuid,omitempty"`
}

// DeviceSensorsBulkDeleteRequestV1Model represents Request with list of device sensor identifiers.

type DeviceSensorsBulkDeleteRequestV1Model struct {
	// Organization Group uuid
	OrganizationGroupUUID string `json:"organization_group_uuid"`
	// List of device sensor identifiers.
	SensorUUIDs []string `json:"sensor_uuids"`
}

// EntityIdV1 is a generated model type.

type EntityIdV1 struct {
	Value *int64 `json:"Value,omitempty"`
}

// SmartGroupSearchModelV1 represents Represents Smart Group Search Model.

type SmartGroupSearchModelV1 struct {
	// Gets or sets number of entities to which smart group is assigned.
	Assignments *int `json:"Assignments,omitempty"`
	// Gets or sets number of devices in Smart Group.
	Devices *int `json:"Devices,omitempty"`
	// Gets or sets number of entities from which the smart group is excluded.
	Exclusions *int `json:"Exclusions,omitempty"`
	// Gets or sets managedBy Organization Group Identifier.
	ManagedByOrganizationGroupID string `json:"ManagedByOrganizationGroupId,omitempty"`
	// Gets or sets managedBy Organization Group Name.
	ManagedByOrganizationGroupName string `json:"ManagedByOrganizationGroupName,omitempty"`
	// Gets or sets managedBy Organization Group Identifier.
	ManagedByOrganizationGroupUUID string `json:"ManagedByOrganizationGroupUuid,omitempty"`
	// Gets or sets smart Group Name.
	Name string `json:"Name,omitempty"`
	// Gets or sets smart Group Identifier.
	SmartGroupID *int `json:"SmartGroupID,omitempty"`
	// Gets or sets smart Group Identifier.
	SmartGroupUUID string `json:"SmartGroupUuid,omitempty"`
}

// SmartGroupSearchResultV1 represents This holds the details of response for Smart group search.

type SmartGroupSearchResultV1 struct {
	Page     *int `json:"Page,omitempty"`
	PageSize *int `json:"PageSize,omitempty"`
	// Gets or sets list of Smart group details resulted in the search operation.
	SmartGroups []SmartGroupSearchModelV1 `json:"SmartGroups,omitempty"`
	Total       *int                      `json:"Total,omitempty"`
}
