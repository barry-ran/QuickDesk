# AccessWindow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Days** | Pointer to **[]int32** |  | [optional] 
**Start** | **string** |  | 
**End** | **string** |  | 

## Methods

### NewAccessWindow

`func NewAccessWindow(start string, end string, ) *AccessWindow`

NewAccessWindow instantiates a new AccessWindow object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAccessWindowWithDefaults

`func NewAccessWindowWithDefaults() *AccessWindow`

NewAccessWindowWithDefaults instantiates a new AccessWindow object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDays

`func (o *AccessWindow) GetDays() []int32`

GetDays returns the Days field if non-nil, zero value otherwise.

### GetDaysOk

`func (o *AccessWindow) GetDaysOk() (*[]int32, bool)`

GetDaysOk returns a tuple with the Days field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDays

`func (o *AccessWindow) SetDays(v []int32)`

SetDays sets Days field to given value.

### HasDays

`func (o *AccessWindow) HasDays() bool`

HasDays returns a boolean if a field has been set.

### GetStart

`func (o *AccessWindow) GetStart() string`

GetStart returns the Start field if non-nil, zero value otherwise.

### GetStartOk

`func (o *AccessWindow) GetStartOk() (*string, bool)`

GetStartOk returns a tuple with the Start field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStart

`func (o *AccessWindow) SetStart(v string)`

SetStart sets Start field to given value.


### GetEnd

`func (o *AccessWindow) GetEnd() string`

GetEnd returns the End field if non-nil, zero value otherwise.

### GetEndOk

`func (o *AccessWindow) GetEndOk() (*string, bool)`

GetEndOk returns a tuple with the End field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnd

`func (o *AccessWindow) SetEnd(v string)`

SetEnd sets End field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


