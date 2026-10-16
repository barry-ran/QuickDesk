# DeviceAccessPolicyInput

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

## Methods

### NewDeviceAccessPolicyInput

`func NewDeviceAccessPolicyInput() *DeviceAccessPolicyInput`

NewDeviceAccessPolicyInput instantiates a new DeviceAccessPolicyInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDeviceAccessPolicyInputWithDefaults

`func NewDeviceAccessPolicyInputWithDefaults() *DeviceAccessPolicyInput`

NewDeviceAccessPolicyInputWithDefaults instantiates a new DeviceAccessPolicyInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRemoteDisabled

`func (o *DeviceAccessPolicyInput) GetRemoteDisabled() bool`

GetRemoteDisabled returns the RemoteDisabled field if non-nil, zero value otherwise.

### GetRemoteDisabledOk

`func (o *DeviceAccessPolicyInput) GetRemoteDisabledOk() (*bool, bool)`

GetRemoteDisabledOk returns a tuple with the RemoteDisabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRemoteDisabled

`func (o *DeviceAccessPolicyInput) SetRemoteDisabled(v bool)`

SetRemoteDisabled sets RemoteDisabled field to given value.

### HasRemoteDisabled

`func (o *DeviceAccessPolicyInput) HasRemoteDisabled() bool`

HasRemoteDisabled returns a boolean if a field has been set.

### GetAllowIps

`func (o *DeviceAccessPolicyInput) GetAllowIps() []string`

GetAllowIps returns the AllowIps field if non-nil, zero value otherwise.

### GetAllowIpsOk

`func (o *DeviceAccessPolicyInput) GetAllowIpsOk() (*[]string, bool)`

GetAllowIpsOk returns a tuple with the AllowIps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllowIps

`func (o *DeviceAccessPolicyInput) SetAllowIps(v []string)`

SetAllowIps sets AllowIps field to given value.

### HasAllowIps

`func (o *DeviceAccessPolicyInput) HasAllowIps() bool`

HasAllowIps returns a boolean if a field has been set.

### GetMaxSessions

`func (o *DeviceAccessPolicyInput) GetMaxSessions() int32`

GetMaxSessions returns the MaxSessions field if non-nil, zero value otherwise.

### GetMaxSessionsOk

`func (o *DeviceAccessPolicyInput) GetMaxSessionsOk() (*int32, bool)`

GetMaxSessionsOk returns a tuple with the MaxSessions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaxSessions

`func (o *DeviceAccessPolicyInput) SetMaxSessions(v int32)`

SetMaxSessions sets MaxSessions field to given value.

### HasMaxSessions

`func (o *DeviceAccessPolicyInput) HasMaxSessions() bool`

HasMaxSessions returns a boolean if a field has been set.

### GetAllowedWindows

`func (o *DeviceAccessPolicyInput) GetAllowedWindows() []AccessWindow`

GetAllowedWindows returns the AllowedWindows field if non-nil, zero value otherwise.

### GetAllowedWindowsOk

`func (o *DeviceAccessPolicyInput) GetAllowedWindowsOk() (*[]AccessWindow, bool)`

GetAllowedWindowsOk returns a tuple with the AllowedWindows field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllowedWindows

`func (o *DeviceAccessPolicyInput) SetAllowedWindows(v []AccessWindow)`

SetAllowedWindows sets AllowedWindows field to given value.

### HasAllowedWindows

`func (o *DeviceAccessPolicyInput) HasAllowedWindows() bool`

HasAllowedWindows returns a boolean if a field has been set.

### GetTimezone

`func (o *DeviceAccessPolicyInput) GetTimezone() string`

GetTimezone returns the Timezone field if non-nil, zero value otherwise.

### GetTimezoneOk

`func (o *DeviceAccessPolicyInput) GetTimezoneOk() (*string, bool)`

GetTimezoneOk returns a tuple with the Timezone field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimezone

`func (o *DeviceAccessPolicyInput) SetTimezone(v string)`

SetTimezone sets Timezone field to given value.

### HasTimezone

`func (o *DeviceAccessPolicyInput) HasTimezone() bool`

HasTimezone returns a boolean if a field has been set.

### GetExceptionDates

`func (o *DeviceAccessPolicyInput) GetExceptionDates() []AccessExceptionDate`

GetExceptionDates returns the ExceptionDates field if non-nil, zero value otherwise.

### GetExceptionDatesOk

`func (o *DeviceAccessPolicyInput) GetExceptionDatesOk() (*[]AccessExceptionDate, bool)`

GetExceptionDatesOk returns a tuple with the ExceptionDates field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExceptionDates

`func (o *DeviceAccessPolicyInput) SetExceptionDates(v []AccessExceptionDate)`

SetExceptionDates sets ExceptionDates field to given value.

### HasExceptionDates

`func (o *DeviceAccessPolicyInput) HasExceptionDates() bool`

HasExceptionDates returns a boolean if a field has been set.

### GetApprovalRequired

`func (o *DeviceAccessPolicyInput) GetApprovalRequired() bool`

GetApprovalRequired returns the ApprovalRequired field if non-nil, zero value otherwise.

### GetApprovalRequiredOk

`func (o *DeviceAccessPolicyInput) GetApprovalRequiredOk() (*bool, bool)`

GetApprovalRequiredOk returns a tuple with the ApprovalRequired field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovalRequired

`func (o *DeviceAccessPolicyInput) SetApprovalRequired(v bool)`

SetApprovalRequired sets ApprovalRequired field to given value.

### HasApprovalRequired

`func (o *DeviceAccessPolicyInput) HasApprovalRequired() bool`

HasApprovalRequired returns a boolean if a field has been set.

### GetApprovalMode

`func (o *DeviceAccessPolicyInput) GetApprovalMode() string`

GetApprovalMode returns the ApprovalMode field if non-nil, zero value otherwise.

### GetApprovalModeOk

`func (o *DeviceAccessPolicyInput) GetApprovalModeOk() (*string, bool)`

GetApprovalModeOk returns a tuple with the ApprovalMode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovalMode

`func (o *DeviceAccessPolicyInput) SetApprovalMode(v string)`

SetApprovalMode sets ApprovalMode field to given value.

### HasApprovalMode

`func (o *DeviceAccessPolicyInput) HasApprovalMode() bool`

HasApprovalMode returns a boolean if a field has been set.

### GetApprovalTrustedIps

`func (o *DeviceAccessPolicyInput) GetApprovalTrustedIps() []string`

GetApprovalTrustedIps returns the ApprovalTrustedIps field if non-nil, zero value otherwise.

### GetApprovalTrustedIpsOk

`func (o *DeviceAccessPolicyInput) GetApprovalTrustedIpsOk() (*[]string, bool)`

GetApprovalTrustedIpsOk returns a tuple with the ApprovalTrustedIps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovalTrustedIps

`func (o *DeviceAccessPolicyInput) SetApprovalTrustedIps(v []string)`

SetApprovalTrustedIps sets ApprovalTrustedIps field to given value.

### HasApprovalTrustedIps

`func (o *DeviceAccessPolicyInput) HasApprovalTrustedIps() bool`

HasApprovalTrustedIps returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


