# ApprovalSummary

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Since** | **time.Time** | Inclusive start of the window the counts cover. | 
**Total** | **int64** | Every request created inside the window, whatever its outcome. | 
**ByStatus** | [**ApprovalSummaryByStatus**](ApprovalSummaryByStatus.md) |  | 
**DeniedSources** | [**[]ApprovalSummaryDeniedSourcesInner**](ApprovalSummaryDeniedSourcesInner.md) | Refused sources inside the window, most refusals first. | 
**DenialCooldownSeconds** | **int32** | The live APPROVAL_DENIAL_COOLDOWN_SECONDS, so an owner can tell \&quot;refused right now\&quot; from \&quot;refused a while ago\&quot;. | 
**DenialCooldownMultiplier** | **int32** | The live APPROVAL_DENIAL_COOLDOWN_MULTIPLIER; 1 means each further consecutive refusal costs the same as the last. | 
**DenialCooldownMaxSeconds** | **int32** | The live APPROVAL_DENIAL_COOLDOWN_MAX_SECONDS, the ceiling the escalation cannot pass; 0 means uncapped. | 

## Methods

### NewApprovalSummary

`func NewApprovalSummary(since time.Time, total int64, byStatus ApprovalSummaryByStatus, deniedSources []ApprovalSummaryDeniedSourcesInner, denialCooldownSeconds int32, denialCooldownMultiplier int32, denialCooldownMaxSeconds int32, ) *ApprovalSummary`

NewApprovalSummary instantiates a new ApprovalSummary object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewApprovalSummaryWithDefaults

`func NewApprovalSummaryWithDefaults() *ApprovalSummary`

NewApprovalSummaryWithDefaults instantiates a new ApprovalSummary object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSince

`func (o *ApprovalSummary) GetSince() time.Time`

GetSince returns the Since field if non-nil, zero value otherwise.

### GetSinceOk

`func (o *ApprovalSummary) GetSinceOk() (*time.Time, bool)`

GetSinceOk returns a tuple with the Since field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSince

`func (o *ApprovalSummary) SetSince(v time.Time)`

SetSince sets Since field to given value.


### GetTotal

`func (o *ApprovalSummary) GetTotal() int64`

GetTotal returns the Total field if non-nil, zero value otherwise.

### GetTotalOk

`func (o *ApprovalSummary) GetTotalOk() (*int64, bool)`

GetTotalOk returns a tuple with the Total field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotal

`func (o *ApprovalSummary) SetTotal(v int64)`

SetTotal sets Total field to given value.


### GetByStatus

`func (o *ApprovalSummary) GetByStatus() ApprovalSummaryByStatus`

GetByStatus returns the ByStatus field if non-nil, zero value otherwise.

### GetByStatusOk

`func (o *ApprovalSummary) GetByStatusOk() (*ApprovalSummaryByStatus, bool)`

GetByStatusOk returns a tuple with the ByStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetByStatus

`func (o *ApprovalSummary) SetByStatus(v ApprovalSummaryByStatus)`

SetByStatus sets ByStatus field to given value.


### GetDeniedSources

`func (o *ApprovalSummary) GetDeniedSources() []ApprovalSummaryDeniedSourcesInner`

GetDeniedSources returns the DeniedSources field if non-nil, zero value otherwise.

### GetDeniedSourcesOk

`func (o *ApprovalSummary) GetDeniedSourcesOk() (*[]ApprovalSummaryDeniedSourcesInner, bool)`

GetDeniedSourcesOk returns a tuple with the DeniedSources field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeniedSources

`func (o *ApprovalSummary) SetDeniedSources(v []ApprovalSummaryDeniedSourcesInner)`

SetDeniedSources sets DeniedSources field to given value.


### GetDenialCooldownSeconds

`func (o *ApprovalSummary) GetDenialCooldownSeconds() int32`

GetDenialCooldownSeconds returns the DenialCooldownSeconds field if non-nil, zero value otherwise.

### GetDenialCooldownSecondsOk

`func (o *ApprovalSummary) GetDenialCooldownSecondsOk() (*int32, bool)`

GetDenialCooldownSecondsOk returns a tuple with the DenialCooldownSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDenialCooldownSeconds

`func (o *ApprovalSummary) SetDenialCooldownSeconds(v int32)`

SetDenialCooldownSeconds sets DenialCooldownSeconds field to given value.


### GetDenialCooldownMultiplier

`func (o *ApprovalSummary) GetDenialCooldownMultiplier() int32`

GetDenialCooldownMultiplier returns the DenialCooldownMultiplier field if non-nil, zero value otherwise.

### GetDenialCooldownMultiplierOk

`func (o *ApprovalSummary) GetDenialCooldownMultiplierOk() (*int32, bool)`

GetDenialCooldownMultiplierOk returns a tuple with the DenialCooldownMultiplier field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDenialCooldownMultiplier

`func (o *ApprovalSummary) SetDenialCooldownMultiplier(v int32)`

SetDenialCooldownMultiplier sets DenialCooldownMultiplier field to given value.


### GetDenialCooldownMaxSeconds

`func (o *ApprovalSummary) GetDenialCooldownMaxSeconds() int32`

GetDenialCooldownMaxSeconds returns the DenialCooldownMaxSeconds field if non-nil, zero value otherwise.

### GetDenialCooldownMaxSecondsOk

`func (o *ApprovalSummary) GetDenialCooldownMaxSecondsOk() (*int32, bool)`

GetDenialCooldownMaxSecondsOk returns a tuple with the DenialCooldownMaxSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDenialCooldownMaxSeconds

`func (o *ApprovalSummary) SetDenialCooldownMaxSeconds(v int32)`

SetDenialCooldownMaxSeconds sets DenialCooldownMaxSeconds field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


