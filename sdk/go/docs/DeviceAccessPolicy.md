# DeviceAccessPolicy

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RemoteDisabled** | Pointer to **bool** | When true, no controller may start a session at all. | [optional] 
**AllowIps** | Pointer to **[]string** | Bare IPs or CIDR blocks; empty means \&quot;no IP restriction\&quot;. | [optional] 
**MaxSessions** | Pointer to **int32** | Concurrent controller cap; 0 means unlimited, negative is rejected. | [optional] 
**AllowedWindows** | Pointer to [**[]AccessWindow**](AccessWindow.md) | Recurring windows in which access is allowed; empty means always. | [optional] 
**Timezone** | Pointer to **string** | IANA zone the windows are evaluated in; empty means the server&#39;s local zone. | [optional] 
**ExceptionDates** | Pointer to [**[]AccessExceptionDate**](AccessExceptionDate.md) | Per-date overrides (holidays, one-off closures, lunar dates). | [optional] 
**ApprovalRequired** | Pointer to **bool** | Legacy switch kept for older clients; prefer &#x60;approval_mode&#x60;. | [optional] 
**ApprovalMode** | Pointer to **string** | When the owner must approve a correct code before a session starts. | [optional] 
**ApprovalTrustedIps** | Pointer to **[]string** | Sources exempted by &#x60;approval_mode: untrusted_ip&#x60;. | [optional] 
**DeviceId** | **string** |  | 
**UpdatedAt** | Pointer to **NullableTime** | Omitted when the owner never saved a policy. | [optional] 

## Methods

### NewDeviceAccessPolicy

`func NewDeviceAccessPolicy(deviceId string, ) *DeviceAccessPolicy`

NewDeviceAccessPolicy instantiates a new DeviceAccessPolicy object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeviceAccessPolicyWithDefaults

`func NewDeviceAccessPolicyWithDefaults() *DeviceAccessPolicy`

NewDeviceAccessPolicyWithDefaults instantiates a new DeviceAccessPolicy object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRemoteDisabled

`func (o *DeviceAccessPolicy) GetRemoteDisabled() bool`

GetRemoteDisabled returns the RemoteDisabled field if non-nil, zero value otherwise.

### GetRemoteDisabledOk

`func (o *DeviceAccessPolicy) GetRemoteDisabledOk() (*bool, bool)`

GetRemoteDisabledOk returns a tuple with the RemoteDisabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRemoteDisabled

`func (o *DeviceAccessPolicy) SetRemoteDisabled(v bool)`

SetRemoteDisabled sets RemoteDisabled field to given value.

### HasRemoteDisabled

`func (o *DeviceAccessPolicy) HasRemoteDisabled() bool`

HasRemoteDisabled returns a boolean if a field has been set.

### GetAllowIps

`func (o *DeviceAccessPolicy) GetAllowIps() []string`

GetAllowIps returns the AllowIps field if non-nil, zero value otherwise.

### GetAllowIpsOk

`func (o *DeviceAccessPolicy) GetAllowIpsOk() (*[]string, bool)`

GetAllowIpsOk returns a tuple with the AllowIps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllowIps

`func (o *DeviceAccessPolicy) SetAllowIps(v []string)`

SetAllowIps sets AllowIps field to given value.

### HasAllowIps

`func (o *DeviceAccessPolicy) HasAllowIps() bool`

HasAllowIps returns a boolean if a field has been set.

### GetMaxSessions

`func (o *DeviceAccessPolicy) GetMaxSessions() int32`

GetMaxSessions returns the MaxSessions field if non-nil, zero value otherwise.

### GetMaxSessionsOk

`func (o *DeviceAccessPolicy) GetMaxSessionsOk() (*int32, bool)`

GetMaxSessionsOk returns a tuple with the MaxSessions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaxSessions

`func (o *DeviceAccessPolicy) SetMaxSessions(v int32)`

SetMaxSessions sets MaxSessions field to given value.

### HasMaxSessions

`func (o *DeviceAccessPolicy) HasMaxSessions() bool`

HasMaxSessions returns a boolean if a field has been set.

### GetAllowedWindows

`func (o *DeviceAccessPolicy) GetAllowedWindows() []AccessWindow`

GetAllowedWindows returns the AllowedWindows field if non-nil, zero value otherwise.

### GetAllowedWindowsOk

`func (o *DeviceAccessPolicy) GetAllowedWindowsOk() (*[]AccessWindow, bool)`

GetAllowedWindowsOk returns a tuple with the AllowedWindows field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllowedWindows

`func (o *DeviceAccessPolicy) SetAllowedWindows(v []AccessWindow)`

SetAllowedWindows sets AllowedWindows field to given value.

### HasAllowedWindows

`func (o *DeviceAccessPolicy) HasAllowedWindows() bool`

HasAllowedWindows returns a boolean if a field has been set.

### GetTimezone

`func (o *DeviceAccessPolicy) GetTimezone() string`

GetTimezone returns the Timezone field if non-nil, zero value otherwise.

### GetTimezoneOk

`func (o *DeviceAccessPolicy) GetTimezoneOk() (*string, bool)`

GetTimezoneOk returns a tuple with the Timezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimezone

`func (o *DeviceAccessPolicy) SetTimezone(v string)`

SetTimezone sets Timezone field to given value.

### HasTimezone

`func (o *DeviceAccessPolicy) HasTimezone() bool`

HasTimezone returns a boolean if a field has been set.

### GetExceptionDates

`func (o *DeviceAccessPolicy) GetExceptionDates() []AccessExceptionDate`

GetExceptionDates returns the ExceptionDates field if non-nil, zero value otherwise.

### GetExceptionDatesOk

`func (o *DeviceAccessPolicy) GetExceptionDatesOk() (*[]AccessExceptionDate, bool)`

GetExceptionDatesOk returns a tuple with the ExceptionDates field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExceptionDates

`func (o *DeviceAccessPolicy) SetExceptionDates(v []AccessExceptionDate)`

SetExceptionDates sets ExceptionDates field to given value.

### HasExceptionDates

`func (o *DeviceAccessPolicy) HasExceptionDates() bool`

HasExceptionDates returns a boolean if a field has been set.

### GetApprovalRequired

`func (o *DeviceAccessPolicy) GetApprovalRequired() bool`

GetApprovalRequired returns the ApprovalRequired field if non-nil, zero value otherwise.

### GetApprovalRequiredOk

`func (o *DeviceAccessPolicy) GetApprovalRequiredOk() (*bool, bool)`

GetApprovalRequiredOk returns a tuple with the ApprovalRequired field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovalRequired

`func (o *DeviceAccessPolicy) SetApprovalRequired(v bool)`

SetApprovalRequired sets ApprovalRequired field to given value.

### HasApprovalRequired

`func (o *DeviceAccessPolicy) HasApprovalRequired() bool`

HasApprovalRequired returns a boolean if a field has been set.

### GetApprovalMode

`func (o *DeviceAccessPolicy) GetApprovalMode() string`

GetApprovalMode returns the ApprovalMode field if non-nil, zero value otherwise.

### GetApprovalModeOk

`func (o *DeviceAccessPolicy) GetApprovalModeOk() (*string, bool)`

GetApprovalModeOk returns a tuple with the ApprovalMode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovalMode

`func (o *DeviceAccessPolicy) SetApprovalMode(v string)`

SetApprovalMode sets ApprovalMode field to given value.

### HasApprovalMode

`func (o *DeviceAccessPolicy) HasApprovalMode() bool`

HasApprovalMode returns a boolean if a field has been set.

### GetApprovalTrustedIps

`func (o *DeviceAccessPolicy) GetApprovalTrustedIps() []string`

GetApprovalTrustedIps returns the ApprovalTrustedIps field if non-nil, zero value otherwise.

### GetApprovalTrustedIpsOk

`func (o *DeviceAccessPolicy) GetApprovalTrustedIpsOk() (*[]string, bool)`

GetApprovalTrustedIpsOk returns a tuple with the ApprovalTrustedIps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovalTrustedIps

`func (o *DeviceAccessPolicy) SetApprovalTrustedIps(v []string)`

SetApprovalTrustedIps sets ApprovalTrustedIps field to given value.

### HasApprovalTrustedIps

`func (o *DeviceAccessPolicy) HasApprovalTrustedIps() bool`

HasApprovalTrustedIps returns a boolean if a field has been set.

### GetDeviceId

`func (o *DeviceAccessPolicy) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *DeviceAccessPolicy) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *DeviceAccessPolicy) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetUpdatedAt

`func (o *DeviceAccessPolicy) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *DeviceAccessPolicy) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *DeviceAccessPolicy) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *DeviceAccessPolicy) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### SetUpdatedAtNil

`func (o *DeviceAccessPolicy) SetUpdatedAtNil(b bool)`

 SetUpdatedAtNil sets the value for UpdatedAt to be an explicit nil

### UnsetUpdatedAt
`func (o *DeviceAccessPolicy) UnsetUpdatedAt()`

UnsetUpdatedAt ensures that no value is present for UpdatedAt, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


