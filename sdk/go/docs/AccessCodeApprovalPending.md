# AccessCodeApprovalPending

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApprovalId** | **int64** |  | 
**ApprovalSecret** | **string** | Shown once; exchanged for a signal token at the claim endpoint. | 
**Status** | **string** |  | 
**ExpiresAt** | **time.Time** |  | 
**PollAfterMs** | **int32** |  | 

## Methods

### NewAccessCodeApprovalPending

`func NewAccessCodeApprovalPending(approvalId int64, approvalSecret string, status string, expiresAt time.Time, pollAfterMs int32, ) *AccessCodeApprovalPending`

NewAccessCodeApprovalPending instantiates a new AccessCodeApprovalPending object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAccessCodeApprovalPendingWithDefaults

`func NewAccessCodeApprovalPendingWithDefaults() *AccessCodeApprovalPending`

NewAccessCodeApprovalPendingWithDefaults instantiates a new AccessCodeApprovalPending object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApprovalId

`func (o *AccessCodeApprovalPending) GetApprovalId() int64`

GetApprovalId returns the ApprovalId field if non-nil, zero value otherwise.

### GetApprovalIdOk

`func (o *AccessCodeApprovalPending) GetApprovalIdOk() (*int64, bool)`

GetApprovalIdOk returns a tuple with the ApprovalId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovalId

`func (o *AccessCodeApprovalPending) SetApprovalId(v int64)`

SetApprovalId sets ApprovalId field to given value.


### GetApprovalSecret

`func (o *AccessCodeApprovalPending) GetApprovalSecret() string`

GetApprovalSecret returns the ApprovalSecret field if non-nil, zero value otherwise.

### GetApprovalSecretOk

`func (o *AccessCodeApprovalPending) GetApprovalSecretOk() (*string, bool)`

GetApprovalSecretOk returns a tuple with the ApprovalSecret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovalSecret

`func (o *AccessCodeApprovalPending) SetApprovalSecret(v string)`

SetApprovalSecret sets ApprovalSecret field to given value.


### GetStatus

`func (o *AccessCodeApprovalPending) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *AccessCodeApprovalPending) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *AccessCodeApprovalPending) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetExpiresAt

`func (o *AccessCodeApprovalPending) GetExpiresAt() time.Time`

GetExpiresAt returns the ExpiresAt field if non-nil, zero value otherwise.

### GetExpiresAtOk

`func (o *AccessCodeApprovalPending) GetExpiresAtOk() (*time.Time, bool)`

GetExpiresAtOk returns a tuple with the ExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresAt

`func (o *AccessCodeApprovalPending) SetExpiresAt(v time.Time)`

SetExpiresAt sets ExpiresAt field to given value.


### GetPollAfterMs

`func (o *AccessCodeApprovalPending) GetPollAfterMs() int32`

GetPollAfterMs returns the PollAfterMs field if non-nil, zero value otherwise.

### GetPollAfterMsOk

`func (o *AccessCodeApprovalPending) GetPollAfterMsOk() (*int32, bool)`

GetPollAfterMsOk returns a tuple with the PollAfterMs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPollAfterMs

`func (o *AccessCodeApprovalPending) SetPollAfterMs(v int32)`

SetPollAfterMs sets PollAfterMs field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


