# ApprovalClaimRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ApprovalId** | **int64** |  | 
**Secret** | **string** | The &#x60;approval_secret&#x60; returned once by &#x60;access-code:verify&#x60;. | 

## Methods

### NewApprovalClaimRequest

`func NewApprovalClaimRequest(approvalId int64, secret string, ) *ApprovalClaimRequest`

NewApprovalClaimRequest instantiates a new ApprovalClaimRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewApprovalClaimRequestWithDefaults

`func NewApprovalClaimRequestWithDefaults() *ApprovalClaimRequest`

NewApprovalClaimRequestWithDefaults instantiates a new ApprovalClaimRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetApprovalId

`func (o *ApprovalClaimRequest) GetApprovalId() int64`

GetApprovalId returns the ApprovalId field if non-nil, zero value otherwise.

### GetApprovalIdOk

`func (o *ApprovalClaimRequest) GetApprovalIdOk() (*int64, bool)`

GetApprovalIdOk returns a tuple with the ApprovalId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetApprovalId

`func (o *ApprovalClaimRequest) SetApprovalId(v int64)`

SetApprovalId sets ApprovalId field to given value.


### GetSecret

`func (o *ApprovalClaimRequest) GetSecret() string`

GetSecret returns the Secret field if non-nil, zero value otherwise.

### GetSecretOk

`func (o *ApprovalClaimRequest) GetSecretOk() (*string, bool)`

GetSecretOk returns a tuple with the Secret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecret

`func (o *ApprovalClaimRequest) SetSecret(v string)`

SetSecret sets Secret field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


