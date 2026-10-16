# ApprovalClaimPending

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Status** | **string** |  | 
**ExpiresAt** | **time.Time** |  | 
**PollAfterMs** | **int32** |  | 

## Methods

### NewApprovalClaimPending

`func NewApprovalClaimPending(status string, expiresAt time.Time, pollAfterMs int32, ) *ApprovalClaimPending`

NewApprovalClaimPending instantiates a new ApprovalClaimPending object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewApprovalClaimPendingWithDefaults

`func NewApprovalClaimPendingWithDefaults() *ApprovalClaimPending`

NewApprovalClaimPendingWithDefaults instantiates a new ApprovalClaimPending object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetStatus

`func (o *ApprovalClaimPending) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *ApprovalClaimPending) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *ApprovalClaimPending) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetExpiresAt

`func (o *ApprovalClaimPending) GetExpiresAt() time.Time`

GetExpiresAt returns the ExpiresAt field if non-nil, zero value otherwise.

### GetExpiresAtOk

`func (o *ApprovalClaimPending) GetExpiresAtOk() (*time.Time, bool)`

GetExpiresAtOk returns a tuple with the ExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresAt

`func (o *ApprovalClaimPending) SetExpiresAt(v time.Time)`

SetExpiresAt sets ExpiresAt field to given value.


### GetPollAfterMs

`func (o *ApprovalClaimPending) GetPollAfterMs() int32`

GetPollAfterMs returns the PollAfterMs field if non-nil, zero value otherwise.

### GetPollAfterMsOk

`func (o *ApprovalClaimPending) GetPollAfterMsOk() (*int32, bool)`

GetPollAfterMsOk returns a tuple with the PollAfterMs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPollAfterMs

`func (o *ApprovalClaimPending) SetPollAfterMs(v int32)`

SetPollAfterMs sets PollAfterMs field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


