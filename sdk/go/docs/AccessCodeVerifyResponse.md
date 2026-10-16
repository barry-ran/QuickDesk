# AccessCodeVerifyResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Status** | **string** | Which shape this response is. | 
**SignalToken** | Pointer to **string** | Present when &#x60;status&#x60; is &#x60;granted&#x60;. | [optional] 
**ExpiresAt** | Pointer to **time.Time** | When the signal token (granted) or the approval (pending) expires. | [optional] 
**ApprovalId** | Pointer to **int64** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. | [optional] 
**ApprovalSecret** | Pointer to **string** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. Shown once; exchanged for a signal token at the claim endpoint.  | [optional] 
**PollAfterMs** | Pointer to **int32** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. | [optional] 

## Methods

### NewAccessCodeVerifyResponse

`func NewAccessCodeVerifyResponse(status string, ) *AccessCodeVerifyResponse`

NewAccessCodeVerifyResponse instantiates a new AccessCodeVerifyResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAccessCodeVerifyResponseWithDefaults

`func NewAccessCodeVerifyResponseWithDefaults() *AccessCodeVerifyResponse`

NewAccessCodeVerifyResponseWithDefaults instantiates a new AccessCodeVerifyResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetStatus

`func (o *AccessCodeVerifyResponse) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *AccessCodeVerifyResponse) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *AccessCodeVerifyResponse) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetSignalToken

`func (o *AccessCodeVerifyResponse) GetSignalToken() string`

GetSignalToken returns the SignalToken field if non-nil, zero value otherwise.

### GetSignalTokenOk

`func (o *AccessCodeVerifyResponse) GetSignalTokenOk() (*string, bool)`

GetSignalTokenOk returns a tuple with the SignalToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSignalToken

`func (o *AccessCodeVerifyResponse) SetSignalToken(v string)`

SetSignalToken sets SignalToken field to given value.

### HasSignalToken

`func (o *AccessCodeVerifyResponse) HasSignalToken() bool`

HasSignalToken returns a boolean if a field has been set.

### GetExpiresAt

`func (o *AccessCodeVerifyResponse) GetExpiresAt() time.Time`

GetExpiresAt returns the ExpiresAt field if non-nil, zero value otherwise.

### GetExpiresAtOk

`func (o *AccessCodeVerifyResponse) GetExpiresAtOk() (*time.Time, bool)`

GetExpiresAtOk returns a tuple with the ExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresAt

`func (o *AccessCodeVerifyResponse) SetExpiresAt(v time.Time)`

SetExpiresAt sets ExpiresAt field to given value.

### HasExpiresAt

`func (o *AccessCodeVerifyResponse) HasExpiresAt() bool`

HasExpiresAt returns a boolean if a field has been set.

### GetApprovalId

`func (o *AccessCodeVerifyResponse) GetApprovalId() int64`

GetApprovalId returns the ApprovalId field if non-nil, zero value otherwise.

### GetApprovalIdOk

`func (o *AccessCodeVerifyResponse) GetApprovalIdOk() (*int64, bool)`

GetApprovalIdOk returns a tuple with the ApprovalId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovalId

`func (o *AccessCodeVerifyResponse) SetApprovalId(v int64)`

SetApprovalId sets ApprovalId field to given value.

### HasApprovalId

`func (o *AccessCodeVerifyResponse) HasApprovalId() bool`

HasApprovalId returns a boolean if a field has been set.

### GetApprovalSecret

`func (o *AccessCodeVerifyResponse) GetApprovalSecret() string`

GetApprovalSecret returns the ApprovalSecret field if non-nil, zero value otherwise.

### GetApprovalSecretOk

`func (o *AccessCodeVerifyResponse) GetApprovalSecretOk() (*string, bool)`

GetApprovalSecretOk returns a tuple with the ApprovalSecret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovalSecret

`func (o *AccessCodeVerifyResponse) SetApprovalSecret(v string)`

SetApprovalSecret sets ApprovalSecret field to given value.

### HasApprovalSecret

`func (o *AccessCodeVerifyResponse) HasApprovalSecret() bool`

HasApprovalSecret returns a boolean if a field has been set.

### GetPollAfterMs

`func (o *AccessCodeVerifyResponse) GetPollAfterMs() int32`

GetPollAfterMs returns the PollAfterMs field if non-nil, zero value otherwise.

### GetPollAfterMsOk

`func (o *AccessCodeVerifyResponse) GetPollAfterMsOk() (*int32, bool)`

GetPollAfterMsOk returns a tuple with the PollAfterMs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPollAfterMs

`func (o *AccessCodeVerifyResponse) SetPollAfterMs(v int32)`

SetPollAfterMs sets PollAfterMs field to given value.

### HasPollAfterMs

`func (o *AccessCodeVerifyResponse) HasPollAfterMs() bool`

HasPollAfterMs returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


