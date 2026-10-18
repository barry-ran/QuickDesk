# MaintenanceWindow

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Days** | Pointer to **[]int32** | Weekdays 0-6 (0 &#x3D; Sunday); empty means every day. | [optional] 
**Start** | **string** |  | 
**End** | **string** |  | 

## Methods

### NewMaintenanceWindow

`func NewMaintenanceWindow(start string, end string, ) *MaintenanceWindow`

NewMaintenanceWindow instantiates a new MaintenanceWindow object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewMaintenanceWindowWithDefaults

`func NewMaintenanceWindowWithDefaults() *MaintenanceWindow`

NewMaintenanceWindowWithDefaults instantiates a new MaintenanceWindow object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDays

`func (o *MaintenanceWindow) GetDays() []int32`

GetDays returns the Days field if non-nil, zero value otherwise.

### GetDaysOk

`func (o *MaintenanceWindow) GetDaysOk() (*[]int32, bool)`

GetDaysOk returns a tuple with the Days field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDays

`func (o *MaintenanceWindow) SetDays(v []int32)`

SetDays sets Days field to given value.

### HasDays

`func (o *MaintenanceWindow) HasDays() bool`

HasDays returns a boolean if a field has been set.

### GetStart

`func (o *MaintenanceWindow) GetStart() string`

GetStart returns the Start field if non-nil, zero value otherwise.

### GetStartOk

`func (o *MaintenanceWindow) GetStartOk() (*string, bool)`

GetStartOk returns a tuple with the Start field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStart

`func (o *MaintenanceWindow) SetStart(v string)`

SetStart sets Start field to given value.


### GetEnd

`func (o *MaintenanceWindow) GetEnd() string`

GetEnd returns the End field if non-nil, zero value otherwise.

### GetEndOk

`func (o *MaintenanceWindow) GetEndOk() (*string, bool)`

GetEndOk returns a tuple with the End field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnd

`func (o *MaintenanceWindow) SetEnd(v string)`

SetEnd sets End field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


