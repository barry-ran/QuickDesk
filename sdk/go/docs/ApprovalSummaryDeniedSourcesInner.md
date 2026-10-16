# ApprovalSummaryDeniedSourcesInner

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ClientIp** | **string** |  | 
**Denials** | **int64** |  | 
**LastDeniedAt** | Pointer to **NullableTime** |  | [optional] 

## Methods

### NewApprovalSummaryDeniedSourcesInner

`func NewApprovalSummaryDeniedSourcesInner(clientIp string, denials int64, ) *ApprovalSummaryDeniedSourcesInner`

NewApprovalSummaryDeniedSourcesInner instantiates a new ApprovalSummaryDeniedSourcesInner object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewApprovalSummaryDeniedSourcesInnerWithDefaults

`func NewApprovalSummaryDeniedSourcesInnerWithDefaults() *ApprovalSummaryDeniedSourcesInner`

NewApprovalSummaryDeniedSourcesInnerWithDefaults instantiates a new ApprovalSummaryDeniedSourcesInner object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetClientIp

`func (o *ApprovalSummaryDeniedSourcesInner) GetClientIp() string`

GetClientIp returns the ClientIp field if non-nil, zero value otherwise.

### GetClientIpOk

`func (o *ApprovalSummaryDeniedSourcesInner) GetClientIpOk() (*string, bool)`

GetClientIpOk returns a tuple with the ClientIp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClientIp

`func (o *ApprovalSummaryDeniedSourcesInner) SetClientIp(v string)`

SetClientIp sets ClientIp field to given value.


### GetDenials

`func (o *ApprovalSummaryDeniedSourcesInner) GetDenials() int64`

GetDenials returns the Denials field if non-nil, zero value otherwise.

### GetDenialsOk

`func (o *ApprovalSummaryDeniedSourcesInner) GetDenialsOk() (*int64, bool)`

GetDenialsOk returns a tuple with the Denials field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDenials

`func (o *ApprovalSummaryDeniedSourcesInner) SetDenials(v int64)`

SetDenials sets Denials field to given value.


### GetLastDeniedAt

`func (o *ApprovalSummaryDeniedSourcesInner) GetLastDeniedAt() time.Time`

GetLastDeniedAt returns the LastDeniedAt field if non-nil, zero value otherwise.

### GetLastDeniedAtOk

`func (o *ApprovalSummaryDeniedSourcesInner) GetLastDeniedAtOk() (*time.Time, bool)`

GetLastDeniedAtOk returns a tuple with the LastDeniedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastDeniedAt

`func (o *ApprovalSummaryDeniedSourcesInner) SetLastDeniedAt(v time.Time)`

SetLastDeniedAt sets LastDeniedAt field to given value.

### HasLastDeniedAt

`func (o *ApprovalSummaryDeniedSourcesInner) HasLastDeniedAt() bool`

HasLastDeniedAt returns a boolean if a field has been set.

### SetLastDeniedAtNil

`func (o *ApprovalSummaryDeniedSourcesInner) SetLastDeniedAtNil(b bool)`

 SetLastDeniedAtNil sets the value for LastDeniedAt to be an explicit nil

### UnsetLastDeniedAt
`func (o *ApprovalSummaryDeniedSourcesInner) UnsetLastDeniedAt()`

UnsetLastDeniedAt ensures that no value is present for LastDeniedAt, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


