# OfflineAlertRule

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** |  | 
**Enabled** | **bool** |  | 
**AfterMinutes** | **int32** |  | 
**RepeatMinutes** | **int32** |  | 
**SnoozedUntil** | Pointer to **NullableTime** |  | [optional] 
**MaintenanceWindows** | [**[]MaintenanceWindow**](MaintenanceWindow.md) |  | 
**MaintenanceTimezone** | Pointer to **string** |  | [optional] 
**InMaintenanceWindow** | **bool** | Computed at read time, so a client can explain a quiet period without re-implementing the schedule. | 
**OfflineSince** | Pointer to **NullableTime** | The evaluator&#39;s episode state; present only while the device is in an outage. | [optional] 
**LastAlertedAt** | Pointer to **NullableTime** |  | [optional] 
**UpdatedAt** | Pointer to **NullableTime** |  | [optional] 

## Methods

### NewOfflineAlertRule

`func NewOfflineAlertRule(deviceId string, enabled bool, afterMinutes int32, repeatMinutes int32, maintenanceWindows []MaintenanceWindow, inMaintenanceWindow bool, ) *OfflineAlertRule`

NewOfflineAlertRule instantiates a new OfflineAlertRule object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewOfflineAlertRuleWithDefaults

`func NewOfflineAlertRuleWithDefaults() *OfflineAlertRule`

NewOfflineAlertRuleWithDefaults instantiates a new OfflineAlertRule object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDeviceId

`func (o *OfflineAlertRule) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *OfflineAlertRule) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *OfflineAlertRule) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetEnabled

`func (o *OfflineAlertRule) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *OfflineAlertRule) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *OfflineAlertRule) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetAfterMinutes

`func (o *OfflineAlertRule) GetAfterMinutes() int32`

GetAfterMinutes returns the AfterMinutes field if non-nil, zero value otherwise.

### GetAfterMinutesOk

`func (o *OfflineAlertRule) GetAfterMinutesOk() (*int32, bool)`

GetAfterMinutesOk returns a tuple with the AfterMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAfterMinutes

`func (o *OfflineAlertRule) SetAfterMinutes(v int32)`

SetAfterMinutes sets AfterMinutes field to given value.


### GetRepeatMinutes

`func (o *OfflineAlertRule) GetRepeatMinutes() int32`

GetRepeatMinutes returns the RepeatMinutes field if non-nil, zero value otherwise.

### GetRepeatMinutesOk

`func (o *OfflineAlertRule) GetRepeatMinutesOk() (*int32, bool)`

GetRepeatMinutesOk returns a tuple with the RepeatMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepeatMinutes

`func (o *OfflineAlertRule) SetRepeatMinutes(v int32)`

SetRepeatMinutes sets RepeatMinutes field to given value.


### GetSnoozedUntil

`func (o *OfflineAlertRule) GetSnoozedUntil() time.Time`

GetSnoozedUntil returns the SnoozedUntil field if non-nil, zero value otherwise.

### GetSnoozedUntilOk

`func (o *OfflineAlertRule) GetSnoozedUntilOk() (*time.Time, bool)`

GetSnoozedUntilOk returns a tuple with the SnoozedUntil field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSnoozedUntil

`func (o *OfflineAlertRule) SetSnoozedUntil(v time.Time)`

SetSnoozedUntil sets SnoozedUntil field to given value.

### HasSnoozedUntil

`func (o *OfflineAlertRule) HasSnoozedUntil() bool`

HasSnoozedUntil returns a boolean if a field has been set.

### SetSnoozedUntilNil

`func (o *OfflineAlertRule) SetSnoozedUntilNil(b bool)`

 SetSnoozedUntilNil sets the value for SnoozedUntil to be an explicit nil

### UnsetSnoozedUntil
`func (o *OfflineAlertRule) UnsetSnoozedUntil()`

UnsetSnoozedUntil ensures that no value is present for SnoozedUntil, not even an explicit nil
### GetMaintenanceWindows

`func (o *OfflineAlertRule) GetMaintenanceWindows() []MaintenanceWindow`

GetMaintenanceWindows returns the MaintenanceWindows field if non-nil, zero value otherwise.

### GetMaintenanceWindowsOk

`func (o *OfflineAlertRule) GetMaintenanceWindowsOk() (*[]MaintenanceWindow, bool)`

GetMaintenanceWindowsOk returns a tuple with the MaintenanceWindows field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaintenanceWindows

`func (o *OfflineAlertRule) SetMaintenanceWindows(v []MaintenanceWindow)`

SetMaintenanceWindows sets MaintenanceWindows field to given value.


### GetMaintenanceTimezone

`func (o *OfflineAlertRule) GetMaintenanceTimezone() string`

GetMaintenanceTimezone returns the MaintenanceTimezone field if non-nil, zero value otherwise.

### GetMaintenanceTimezoneOk

`func (o *OfflineAlertRule) GetMaintenanceTimezoneOk() (*string, bool)`

GetMaintenanceTimezoneOk returns a tuple with the MaintenanceTimezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaintenanceTimezone

`func (o *OfflineAlertRule) SetMaintenanceTimezone(v string)`

SetMaintenanceTimezone sets MaintenanceTimezone field to given value.

### HasMaintenanceTimezone

`func (o *OfflineAlertRule) HasMaintenanceTimezone() bool`

HasMaintenanceTimezone returns a boolean if a field has been set.

### GetInMaintenanceWindow

`func (o *OfflineAlertRule) GetInMaintenanceWindow() bool`

GetInMaintenanceWindow returns the InMaintenanceWindow field if non-nil, zero value otherwise.

### GetInMaintenanceWindowOk

`func (o *OfflineAlertRule) GetInMaintenanceWindowOk() (*bool, bool)`

GetInMaintenanceWindowOk returns a tuple with the InMaintenanceWindow field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInMaintenanceWindow

`func (o *OfflineAlertRule) SetInMaintenanceWindow(v bool)`

SetInMaintenanceWindow sets InMaintenanceWindow field to given value.


### GetOfflineSince

`func (o *OfflineAlertRule) GetOfflineSince() time.Time`

GetOfflineSince returns the OfflineSince field if non-nil, zero value otherwise.

### GetOfflineSinceOk

`func (o *OfflineAlertRule) GetOfflineSinceOk() (*time.Time, bool)`

GetOfflineSinceOk returns a tuple with the OfflineSince field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOfflineSince

`func (o *OfflineAlertRule) SetOfflineSince(v time.Time)`

SetOfflineSince sets OfflineSince field to given value.

### HasOfflineSince

`func (o *OfflineAlertRule) HasOfflineSince() bool`

HasOfflineSince returns a boolean if a field has been set.

### SetOfflineSinceNil

`func (o *OfflineAlertRule) SetOfflineSinceNil(b bool)`

 SetOfflineSinceNil sets the value for OfflineSince to be an explicit nil

### UnsetOfflineSince
`func (o *OfflineAlertRule) UnsetOfflineSince()`

UnsetOfflineSince ensures that no value is present for OfflineSince, not even an explicit nil
### GetLastAlertedAt

`func (o *OfflineAlertRule) GetLastAlertedAt() time.Time`

GetLastAlertedAt returns the LastAlertedAt field if non-nil, zero value otherwise.

### GetLastAlertedAtOk

`func (o *OfflineAlertRule) GetLastAlertedAtOk() (*time.Time, bool)`

GetLastAlertedAtOk returns a tuple with the LastAlertedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastAlertedAt

`func (o *OfflineAlertRule) SetLastAlertedAt(v time.Time)`

SetLastAlertedAt sets LastAlertedAt field to given value.

### HasLastAlertedAt

`func (o *OfflineAlertRule) HasLastAlertedAt() bool`

HasLastAlertedAt returns a boolean if a field has been set.

### SetLastAlertedAtNil

`func (o *OfflineAlertRule) SetLastAlertedAtNil(b bool)`

 SetLastAlertedAtNil sets the value for LastAlertedAt to be an explicit nil

### UnsetLastAlertedAt
`func (o *OfflineAlertRule) UnsetLastAlertedAt()`

UnsetLastAlertedAt ensures that no value is present for LastAlertedAt, not even an explicit nil
### GetUpdatedAt

`func (o *OfflineAlertRule) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *OfflineAlertRule) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *OfflineAlertRule) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *OfflineAlertRule) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### SetUpdatedAtNil

`func (o *OfflineAlertRule) SetUpdatedAtNil(b bool)`

 SetUpdatedAtNil sets the value for UpdatedAt to be an explicit nil

### UnsetUpdatedAt
`func (o *OfflineAlertRule) UnsetUpdatedAt()`

UnsetUpdatedAt ensures that no value is present for UpdatedAt, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


