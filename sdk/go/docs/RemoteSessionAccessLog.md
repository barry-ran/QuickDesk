# RemoteSessionAccessLog

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **int64** |  | 
**DeviceId** | **string** |  | 
**OwnerId** | **int64** | The device&#39;s owner at the time of the attempt; 0 when unbound. | 
**ClientId** | **string** |  | 
**SessionId** | **string** |  | 
**Ip** | **string** |  | 
**Result** | **string** | The canonical access result, e.g. &#x60;session_started&#x60;. | 
**Reason** | **string** |  | 
**CreatedAt** | **time.Time** |  | 

## Methods

### NewRemoteSessionAccessLog

`func NewRemoteSessionAccessLog(id int64, deviceId string, ownerId int64, clientId string, sessionId string, ip string, result string, reason string, createdAt time.Time, ) *RemoteSessionAccessLog`

NewRemoteSessionAccessLog instantiates a new RemoteSessionAccessLog object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRemoteSessionAccessLogWithDefaults

`func NewRemoteSessionAccessLogWithDefaults() *RemoteSessionAccessLog`

NewRemoteSessionAccessLogWithDefaults instantiates a new RemoteSessionAccessLog object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *RemoteSessionAccessLog) GetId() int64`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *RemoteSessionAccessLog) GetIdOk() (*int64, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *RemoteSessionAccessLog) SetId(v int64)`

SetId sets Id field to given value.


### GetDeviceId

`func (o *RemoteSessionAccessLog) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *RemoteSessionAccessLog) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *RemoteSessionAccessLog) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetOwnerId

`func (o *RemoteSessionAccessLog) GetOwnerId() int64`

GetOwnerId returns the OwnerId field if non-nil, zero value otherwise.

### GetOwnerIdOk

`func (o *RemoteSessionAccessLog) GetOwnerIdOk() (*int64, bool)`

GetOwnerIdOk returns a tuple with the OwnerId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOwnerId

`func (o *RemoteSessionAccessLog) SetOwnerId(v int64)`

SetOwnerId sets OwnerId field to given value.


### GetClientId

`func (o *RemoteSessionAccessLog) GetClientId() string`

GetClientId returns the ClientId field if non-nil, zero value otherwise.

### GetClientIdOk

`func (o *RemoteSessionAccessLog) GetClientIdOk() (*string, bool)`

GetClientIdOk returns a tuple with the ClientId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClientId

`func (o *RemoteSessionAccessLog) SetClientId(v string)`

SetClientId sets ClientId field to given value.


### GetSessionId

`func (o *RemoteSessionAccessLog) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *RemoteSessionAccessLog) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *RemoteSessionAccessLog) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.


### GetIp

`func (o *RemoteSessionAccessLog) GetIp() string`

GetIp returns the Ip field if non-nil, zero value otherwise.

### GetIpOk

`func (o *RemoteSessionAccessLog) GetIpOk() (*string, bool)`

GetIpOk returns a tuple with the Ip field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIp

`func (o *RemoteSessionAccessLog) SetIp(v string)`

SetIp sets Ip field to given value.


### GetResult

`func (o *RemoteSessionAccessLog) GetResult() string`

GetResult returns the Result field if non-nil, zero value otherwise.

### GetResultOk

`func (o *RemoteSessionAccessLog) GetResultOk() (*string, bool)`

GetResultOk returns a tuple with the Result field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResult

`func (o *RemoteSessionAccessLog) SetResult(v string)`

SetResult sets Result field to given value.


### GetReason

`func (o *RemoteSessionAccessLog) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *RemoteSessionAccessLog) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *RemoteSessionAccessLog) SetReason(v string)`

SetReason sets Reason field to given value.


### GetCreatedAt

`func (o *RemoteSessionAccessLog) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *RemoteSessionAccessLog) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *RemoteSessionAccessLog) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


