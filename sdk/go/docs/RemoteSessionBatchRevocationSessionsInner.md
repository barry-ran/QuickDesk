# RemoteSessionBatchRevocationSessionsInner

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SessionId** | **string** |  | 
**Revoked** | **bool** |  | 
**Reason** | Pointer to **string** | Why the session could not be closed; present only when &#x60;revoked&#x60; is false. | [optional] 

## Methods

### NewRemoteSessionBatchRevocationSessionsInner

`func NewRemoteSessionBatchRevocationSessionsInner(sessionId string, revoked bool, ) *RemoteSessionBatchRevocationSessionsInner`

NewRemoteSessionBatchRevocationSessionsInner instantiates a new RemoteSessionBatchRevocationSessionsInner object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRemoteSessionBatchRevocationSessionsInnerWithDefaults

`func NewRemoteSessionBatchRevocationSessionsInnerWithDefaults() *RemoteSessionBatchRevocationSessionsInner`

NewRemoteSessionBatchRevocationSessionsInnerWithDefaults instantiates a new RemoteSessionBatchRevocationSessionsInner object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSessionId

`func (o *RemoteSessionBatchRevocationSessionsInner) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *RemoteSessionBatchRevocationSessionsInner) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *RemoteSessionBatchRevocationSessionsInner) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.


### GetRevoked

`func (o *RemoteSessionBatchRevocationSessionsInner) GetRevoked() bool`

GetRevoked returns the Revoked field if non-nil, zero value otherwise.

### GetRevokedOk

`func (o *RemoteSessionBatchRevocationSessionsInner) GetRevokedOk() (*bool, bool)`

GetRevokedOk returns a tuple with the Revoked field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevoked

`func (o *RemoteSessionBatchRevocationSessionsInner) SetRevoked(v bool)`

SetRevoked sets Revoked field to given value.


### GetReason

`func (o *RemoteSessionBatchRevocationSessionsInner) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *RemoteSessionBatchRevocationSessionsInner) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *RemoteSessionBatchRevocationSessionsInner) SetReason(v string)`

SetReason sets Reason field to given value.

### HasReason

`func (o *RemoteSessionBatchRevocationSessionsInner) HasReason() bool`

HasReason returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


