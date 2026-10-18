# RemoteSessionRevocation

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SessionId** | **string** |  | 
**DeviceId** | **string** |  | 
**Revoked** | **bool** |  | 

## Methods

### NewRemoteSessionRevocation

`func NewRemoteSessionRevocation(sessionId string, deviceId string, revoked bool, ) *RemoteSessionRevocation`

NewRemoteSessionRevocation instantiates a new RemoteSessionRevocation object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRemoteSessionRevocationWithDefaults

`func NewRemoteSessionRevocationWithDefaults() *RemoteSessionRevocation`

NewRemoteSessionRevocationWithDefaults instantiates a new RemoteSessionRevocation object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSessionId

`func (o *RemoteSessionRevocation) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *RemoteSessionRevocation) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *RemoteSessionRevocation) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.


### GetDeviceId

`func (o *RemoteSessionRevocation) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *RemoteSessionRevocation) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *RemoteSessionRevocation) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetRevoked

`func (o *RemoteSessionRevocation) GetRevoked() bool`

GetRevoked returns the Revoked field if non-nil, zero value otherwise.

### GetRevokedOk

`func (o *RemoteSessionRevocation) GetRevokedOk() (*bool, bool)`

GetRevokedOk returns a tuple with the Revoked field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevoked

`func (o *RemoteSessionRevocation) SetRevoked(v bool)`

SetRevoked sets Revoked field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


