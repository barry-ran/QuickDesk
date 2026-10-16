# RemoteSessionBatchRevocation

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** |  | 
**Revoked** | **int32** | How many sessions were closed and confirmed. | 
**Failed** | **int32** | How many were live but could not be closed. | 
**Truncated** | **bool** | True when more live sessions existed than one call handles (the cap is 200). | 
**Sessions** | [**[]RemoteSessionBatchRevocationSessionsInner**](RemoteSessionBatchRevocationSessionsInner.md) | One entry per session the call attempted, newest first. | 

## Methods

### NewRemoteSessionBatchRevocation

`func NewRemoteSessionBatchRevocation(deviceId string, revoked int32, failed int32, truncated bool, sessions []RemoteSessionBatchRevocationSessionsInner, ) *RemoteSessionBatchRevocation`

NewRemoteSessionBatchRevocation instantiates a new RemoteSessionBatchRevocation object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRemoteSessionBatchRevocationWithDefaults

`func NewRemoteSessionBatchRevocationWithDefaults() *RemoteSessionBatchRevocation`

NewRemoteSessionBatchRevocationWithDefaults instantiates a new RemoteSessionBatchRevocation object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDeviceId

`func (o *RemoteSessionBatchRevocation) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *RemoteSessionBatchRevocation) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *RemoteSessionBatchRevocation) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetRevoked

`func (o *RemoteSessionBatchRevocation) GetRevoked() int32`

GetRevoked returns the Revoked field if non-nil, zero value otherwise.

### GetRevokedOk

`func (o *RemoteSessionBatchRevocation) GetRevokedOk() (*int32, bool)`

GetRevokedOk returns a tuple with the Revoked field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevoked

`func (o *RemoteSessionBatchRevocation) SetRevoked(v int32)`

SetRevoked sets Revoked field to given value.


### GetFailed

`func (o *RemoteSessionBatchRevocation) GetFailed() int32`

GetFailed returns the Failed field if non-nil, zero value otherwise.

### GetFailedOk

`func (o *RemoteSessionBatchRevocation) GetFailedOk() (*int32, bool)`

GetFailedOk returns a tuple with the Failed field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFailed

`func (o *RemoteSessionBatchRevocation) SetFailed(v int32)`

SetFailed sets Failed field to given value.


### GetTruncated

`func (o *RemoteSessionBatchRevocation) GetTruncated() bool`

GetTruncated returns the Truncated field if non-nil, zero value otherwise.

### GetTruncatedOk

`func (o *RemoteSessionBatchRevocation) GetTruncatedOk() (*bool, bool)`

GetTruncatedOk returns a tuple with the Truncated field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTruncated

`func (o *RemoteSessionBatchRevocation) SetTruncated(v bool)`

SetTruncated sets Truncated field to given value.


### GetSessions

`func (o *RemoteSessionBatchRevocation) GetSessions() []RemoteSessionBatchRevocationSessionsInner`

GetSessions returns the Sessions field if non-nil, zero value otherwise.

### GetSessionsOk

`func (o *RemoteSessionBatchRevocation) GetSessionsOk() (*[]RemoteSessionBatchRevocationSessionsInner, bool)`

GetSessionsOk returns a tuple with the Sessions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessions

`func (o *RemoteSessionBatchRevocation) SetSessions(v []RemoteSessionBatchRevocationSessionsInner)`

SetSessions sets Sessions field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


