# AccessGrantInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Label** | Pointer to **string** |  | [optional] 
**ExpiresInMinutes** | **int32** |  | 
**MaxUses** | Pointer to **int32** | 0 means \&quot;no use limit\&quot;; the grant still expires. | [optional] 
**Code** | Pointer to **string** | Optional caller-chosen code; omitted means the server generates one. | [optional] 
**BoundClientId** | Pointer to **string** | Self-reported controller identity the code is bound to. | [optional] 
**BoundIps** | Pointer to **[]string** |  | [optional] 

## Methods

### NewAccessGrantInput

`func NewAccessGrantInput(expiresInMinutes int32, ) *AccessGrantInput`

NewAccessGrantInput instantiates a new AccessGrantInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAccessGrantInputWithDefaults

`func NewAccessGrantInputWithDefaults() *AccessGrantInput`

NewAccessGrantInputWithDefaults instantiates a new AccessGrantInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLabel

`func (o *AccessGrantInput) GetLabel() string`

GetLabel returns the Label field if non-nil, zero value otherwise.

### GetLabelOk

`func (o *AccessGrantInput) GetLabelOk() (*string, bool)`

GetLabelOk returns a tuple with the Label field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabel

`func (o *AccessGrantInput) SetLabel(v string)`

SetLabel sets Label field to given value.

### HasLabel

`func (o *AccessGrantInput) HasLabel() bool`

HasLabel returns a boolean if a field has been set.

### GetExpiresInMinutes

`func (o *AccessGrantInput) GetExpiresInMinutes() int32`

GetExpiresInMinutes returns the ExpiresInMinutes field if non-nil, zero value otherwise.

### GetExpiresInMinutesOk

`func (o *AccessGrantInput) GetExpiresInMinutesOk() (*int32, bool)`

GetExpiresInMinutesOk returns a tuple with the ExpiresInMinutes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresInMinutes

`func (o *AccessGrantInput) SetExpiresInMinutes(v int32)`

SetExpiresInMinutes sets ExpiresInMinutes field to given value.


### GetMaxUses

`func (o *AccessGrantInput) GetMaxUses() int32`

GetMaxUses returns the MaxUses field if non-nil, zero value otherwise.

### GetMaxUsesOk

`func (o *AccessGrantInput) GetMaxUsesOk() (*int32, bool)`

GetMaxUsesOk returns a tuple with the MaxUses field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaxUses

`func (o *AccessGrantInput) SetMaxUses(v int32)`

SetMaxUses sets MaxUses field to given value.

### HasMaxUses

`func (o *AccessGrantInput) HasMaxUses() bool`

HasMaxUses returns a boolean if a field has been set.

### GetCode

`func (o *AccessGrantInput) GetCode() string`

GetCode returns the Code field if non-nil, zero value otherwise.

### GetCodeOk

`func (o *AccessGrantInput) GetCodeOk() (*string, bool)`

GetCodeOk returns a tuple with the Code field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCode

`func (o *AccessGrantInput) SetCode(v string)`

SetCode sets Code field to given value.

### HasCode

`func (o *AccessGrantInput) HasCode() bool`

HasCode returns a boolean if a field has been set.

### GetBoundClientId

`func (o *AccessGrantInput) GetBoundClientId() string`

GetBoundClientId returns the BoundClientId field if non-nil, zero value otherwise.

### GetBoundClientIdOk

`func (o *AccessGrantInput) GetBoundClientIdOk() (*string, bool)`

GetBoundClientIdOk returns a tuple with the BoundClientId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBoundClientId

`func (o *AccessGrantInput) SetBoundClientId(v string)`

SetBoundClientId sets BoundClientId field to given value.

### HasBoundClientId

`func (o *AccessGrantInput) HasBoundClientId() bool`

HasBoundClientId returns a boolean if a field has been set.

### GetBoundIps

`func (o *AccessGrantInput) GetBoundIps() []string`

GetBoundIps returns the BoundIps field if non-nil, zero value otherwise.

### GetBoundIpsOk

`func (o *AccessGrantInput) GetBoundIpsOk() (*[]string, bool)`

GetBoundIpsOk returns a tuple with the BoundIps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBoundIps

`func (o *AccessGrantInput) SetBoundIps(v []string)`

SetBoundIps sets BoundIps field to given value.

### HasBoundIps

`func (o *AccessGrantInput) HasBoundIps() bool`

HasBoundIps returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


