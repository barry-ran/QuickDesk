# OfflineAlertInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Enabled** | **bool** |  | 
**AfterMinutes** | Pointer to **int32** | Omitting it (or sending 0) stores the default of 10. | [optional] 
**RepeatMinutes** | Pointer to **int32** | 0 (the default) sends one alert per outage. | [optional] 
**SnoozedUntil** | Pointer to **NullableTime** | Silences alerts until this instant; null clears an existing snooze. | [optional] 
**MaintenanceWindows** | Pointer to [**[]MaintenanceWindow**](MaintenanceWindow.md) | Recurring quiet periods evaluated in &#x60;maintenance_timezone&#x60;. | [optional] 
**MaintenanceTimezone** | Pointer to **string** | IANA zone name for the windows; empty means the server&#39;s local zone. | [optional] 

## Methods

### NewOfflineAlertInput

`func NewOfflineAlertInput(enabled bool, ) *OfflineAlertInput`

NewOfflineAlertInput instantiates a new OfflineAlertInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewOfflineAlertInputWithDefaults

`func NewOfflineAlertInputWithDefaults() *OfflineAlertInput`

NewOfflineAlertInputWithDefaults instantiates a new OfflineAlertInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetEnabled

`func (o *OfflineAlertInput) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *OfflineAlertInput) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *OfflineAlertInput) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetAfterMinutes

`func (o *OfflineAlertInput) GetAfterMinutes() int32`

GetAfterMinutes returns the AfterMinutes field if non-nil, zero value otherwise.

### GetAfterMinutesOk

`func (o *OfflineAlertInput) GetAfterMinutesOk() (*int32, bool)`

GetAfterMinutesOk returns a tuple with the AfterMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAfterMinutes

`func (o *OfflineAlertInput) SetAfterMinutes(v int32)`

SetAfterMinutes sets AfterMinutes field to given value.

### HasAfterMinutes

`func (o *OfflineAlertInput) HasAfterMinutes() bool`

HasAfterMinutes returns a boolean if a field has been set.

### GetRepeatMinutes

`func (o *OfflineAlertInput) GetRepeatMinutes() int32`

GetRepeatMinutes returns the RepeatMinutes field if non-nil, zero value otherwise.

### GetRepeatMinutesOk

`func (o *OfflineAlertInput) GetRepeatMinutesOk() (*int32, bool)`

GetRepeatMinutesOk returns a tuple with the RepeatMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepeatMinutes

`func (o *OfflineAlertInput) SetRepeatMinutes(v int32)`

SetRepeatMinutes sets RepeatMinutes field to given value.

### HasRepeatMinutes

`func (o *OfflineAlertInput) HasRepeatMinutes() bool`

HasRepeatMinutes returns a boolean if a field has been set.

### GetSnoozedUntil

`func (o *OfflineAlertInput) GetSnoozedUntil() time.Time`

GetSnoozedUntil returns the SnoozedUntil field if non-nil, zero value otherwise.

### GetSnoozedUntilOk

`func (o *OfflineAlertInput) GetSnoozedUntilOk() (*time.Time, bool)`

GetSnoozedUntilOk returns a tuple with the SnoozedUntil field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSnoozedUntil

`func (o *OfflineAlertInput) SetSnoozedUntil(v time.Time)`

SetSnoozedUntil sets SnoozedUntil field to given value.

### HasSnoozedUntil

`func (o *OfflineAlertInput) HasSnoozedUntil() bool`

HasSnoozedUntil returns a boolean if a field has been set.

### SetSnoozedUntilNil

`func (o *OfflineAlertInput) SetSnoozedUntilNil(b bool)`

 SetSnoozedUntilNil sets the value for SnoozedUntil to be an explicit nil

### UnsetSnoozedUntil
`func (o *OfflineAlertInput) UnsetSnoozedUntil()`

UnsetSnoozedUntil ensures that no value is present for SnoozedUntil, not even an explicit nil
### GetMaintenanceWindows

`func (o *OfflineAlertInput) GetMaintenanceWindows() []MaintenanceWindow`

GetMaintenanceWindows returns the MaintenanceWindows field if non-nil, zero value otherwise.

### GetMaintenanceWindowsOk

`func (o *OfflineAlertInput) GetMaintenanceWindowsOk() (*[]MaintenanceWindow, bool)`

GetMaintenanceWindowsOk returns a tuple with the MaintenanceWindows field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaintenanceWindows

`func (o *OfflineAlertInput) SetMaintenanceWindows(v []MaintenanceWindow)`

SetMaintenanceWindows sets MaintenanceWindows field to given value.

### HasMaintenanceWindows

`func (o *OfflineAlertInput) HasMaintenanceWindows() bool`

HasMaintenanceWindows returns a boolean if a field has been set.

### GetMaintenanceTimezone

`func (o *OfflineAlertInput) GetMaintenanceTimezone() string`

GetMaintenanceTimezone returns the MaintenanceTimezone field if non-nil, zero value otherwise.

### GetMaintenanceTimezoneOk

`func (o *OfflineAlertInput) GetMaintenanceTimezoneOk() (*string, bool)`

GetMaintenanceTimezoneOk returns a tuple with the MaintenanceTimezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaintenanceTimezone

`func (o *OfflineAlertInput) SetMaintenanceTimezone(v string)`

SetMaintenanceTimezone sets MaintenanceTimezone field to given value.

### HasMaintenanceTimezone

`func (o *OfflineAlertInput) HasMaintenanceTimezone() bool`

HasMaintenanceTimezone returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


