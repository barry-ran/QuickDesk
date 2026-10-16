# RemoteSession

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **int64** |  | 
**SessionId** | **string** | The signaling session id the server assigned; unique per session. | 
**UserId** | **int64** | The device&#39;s owner. 0 when the device was unbound at the time. | 
**DeviceId** | **string** |  | 
**ClientId** | **string** | The controller-declared client id, the same value the device access log records. | 
**ClientIp** | **string** |  | 
**StartedAt** | **time.Time** |  | 
**EndedAt** | **NullableTime** |  | 
**EndReason** | **string** | Empty while the session is open; &#x60;disconnected&#x60; when the signaling socket went away on its own, &#x60;revoked&#x60; when the owner or an administrator closed it on purpose, &#x60;server_restart&#x60; for rows a restart found still open. | 
**Active** | **bool** |  | 
**DurationSeconds** | **int64** |  | 
**CreatedAt** | **time.Time** |  | 
**UpdatedAt** | **time.Time** |  | 

## Methods

### NewRemoteSession

`func NewRemoteSession(id int64, sessionId string, userId int64, deviceId string, clientId string, clientIp string, startedAt time.Time, endedAt NullableTime, endReason string, active bool, durationSeconds int64, createdAt time.Time, updatedAt time.Time, ) *RemoteSession`

NewRemoteSession instantiates a new RemoteSession object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRemoteSessionWithDefaults

`func NewRemoteSessionWithDefaults() *RemoteSession`

NewRemoteSessionWithDefaults instantiates a new RemoteSession object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *RemoteSession) GetId() int64`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *RemoteSession) GetIdOk() (*int64, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *RemoteSession) SetId(v int64)`

SetId sets Id field to given value.


### GetSessionId

`func (o *RemoteSession) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *RemoteSession) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *RemoteSession) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.


### GetUserId

`func (o *RemoteSession) GetUserId() int64`

GetUserId returns the UserId field if non-nil, zero value otherwise.

### GetUserIdOk

`func (o *RemoteSession) GetUserIdOk() (*int64, bool)`

GetUserIdOk returns a tuple with the UserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserId

`func (o *RemoteSession) SetUserId(v int64)`

SetUserId sets UserId field to given value.


### GetDeviceId

`func (o *RemoteSession) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *RemoteSession) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *RemoteSession) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetClientId

`func (o *RemoteSession) GetClientId() string`

GetClientId returns the ClientId field if non-nil, zero value otherwise.

### GetClientIdOk

`func (o *RemoteSession) GetClientIdOk() (*string, bool)`

GetClientIdOk returns a tuple with the ClientId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClientId

`func (o *RemoteSession) SetClientId(v string)`

SetClientId sets ClientId field to given value.


### GetClientIp

`func (o *RemoteSession) GetClientIp() string`

GetClientIp returns the ClientIp field if non-nil, zero value otherwise.

### GetClientIpOk

`func (o *RemoteSession) GetClientIpOk() (*string, bool)`

GetClientIpOk returns a tuple with the ClientIp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClientIp

`func (o *RemoteSession) SetClientIp(v string)`

SetClientIp sets ClientIp field to given value.


### GetStartedAt

`func (o *RemoteSession) GetStartedAt() time.Time`

GetStartedAt returns the StartedAt field if non-nil, zero value otherwise.

### GetStartedAtOk

`func (o *RemoteSession) GetStartedAtOk() (*time.Time, bool)`

GetStartedAtOk returns a tuple with the StartedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartedAt

`func (o *RemoteSession) SetStartedAt(v time.Time)`

SetStartedAt sets StartedAt field to given value.


### GetEndedAt

`func (o *RemoteSession) GetEndedAt() time.Time`

GetEndedAt returns the EndedAt field if non-nil, zero value otherwise.

### GetEndedAtOk

`func (o *RemoteSession) GetEndedAtOk() (*time.Time, bool)`

GetEndedAtOk returns a tuple with the EndedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEndedAt

`func (o *RemoteSession) SetEndedAt(v time.Time)`

SetEndedAt sets EndedAt field to given value.


### SetEndedAtNil

`func (o *RemoteSession) SetEndedAtNil(b bool)`

 SetEndedAtNil sets the value for EndedAt to be an explicit nil

### UnsetEndedAt
`func (o *RemoteSession) UnsetEndedAt()`

UnsetEndedAt ensures that no value is present for EndedAt, not even an explicit nil
### GetEndReason

`func (o *RemoteSession) GetEndReason() string`

GetEndReason returns the EndReason field if non-nil, zero value otherwise.

### GetEndReasonOk

`func (o *RemoteSession) GetEndReasonOk() (*string, bool)`

GetEndReasonOk returns a tuple with the EndReason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEndReason

`func (o *RemoteSession) SetEndReason(v string)`

SetEndReason sets EndReason field to given value.


### GetActive

`func (o *RemoteSession) GetActive() bool`

GetActive returns the Active field if non-nil, zero value otherwise.

### GetActiveOk

`func (o *RemoteSession) GetActiveOk() (*bool, bool)`

GetActiveOk returns a tuple with the Active field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetActive

`func (o *RemoteSession) SetActive(v bool)`

SetActive sets Active field to given value.


### GetDurationSeconds

`func (o *RemoteSession) GetDurationSeconds() int64`

GetDurationSeconds returns the DurationSeconds field if non-nil, zero value otherwise.

### GetDurationSecondsOk

`func (o *RemoteSession) GetDurationSecondsOk() (*int64, bool)`

GetDurationSecondsOk returns a tuple with the DurationSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDurationSeconds

`func (o *RemoteSession) SetDurationSeconds(v int64)`

SetDurationSeconds sets DurationSeconds field to given value.


### GetCreatedAt

`func (o *RemoteSession) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *RemoteSession) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *RemoteSession) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetUpdatedAt

`func (o *RemoteSession) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *RemoteSession) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *RemoteSession) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


