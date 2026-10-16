# AccessExceptionDate

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Date** | Pointer to **string** | First date of the exception (YYYY-MM-DD). | [optional] 
**EndDate** | Pointer to **string** | Last date of a range (YYYY-MM-DD); omit for a single day. | [optional] 
**Mode** | **string** |  | 
**Repeat** | Pointer to **string** |  | [optional] 
**Holiday** | Pointer to **string** | A statutory holiday name from the server&#39;s calendar, e.g. 春节. | [optional] 
**HolidayDay** | Pointer to **string** | Which dates of the holiday an entry covers (default &#x60;off&#x60;). | [optional] 
**LunarMonth** | Pointer to **int32** |  | [optional] 
**LunarDay** | Pointer to **int32** |  | [optional] 

## Methods

### NewAccessExceptionDate

`func NewAccessExceptionDate(mode string, ) *AccessExceptionDate`

NewAccessExceptionDate instantiates a new AccessExceptionDate object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAccessExceptionDateWithDefaults

`func NewAccessExceptionDateWithDefaults() *AccessExceptionDate`

NewAccessExceptionDateWithDefaults instantiates a new AccessExceptionDate object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDate

`func (o *AccessExceptionDate) GetDate() string`

GetDate returns the Date field if non-nil, zero value otherwise.

### GetDateOk

`func (o *AccessExceptionDate) GetDateOk() (*string, bool)`

GetDateOk returns a tuple with the Date field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDate

`func (o *AccessExceptionDate) SetDate(v string)`

SetDate sets Date field to given value.

### HasDate

`func (o *AccessExceptionDate) HasDate() bool`

HasDate returns a boolean if a field has been set.

### GetEndDate

`func (o *AccessExceptionDate) GetEndDate() string`

GetEndDate returns the EndDate field if non-nil, zero value otherwise.

### GetEndDateOk

`func (o *AccessExceptionDate) GetEndDateOk() (*string, bool)`

GetEndDateOk returns a tuple with the EndDate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEndDate

`func (o *AccessExceptionDate) SetEndDate(v string)`

SetEndDate sets EndDate field to given value.

### HasEndDate

`func (o *AccessExceptionDate) HasEndDate() bool`

HasEndDate returns a boolean if a field has been set.

### GetMode

`func (o *AccessExceptionDate) GetMode() string`

GetMode returns the Mode field if non-nil, zero value otherwise.

### GetModeOk

`func (o *AccessExceptionDate) GetModeOk() (*string, bool)`

GetModeOk returns a tuple with the Mode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMode

`func (o *AccessExceptionDate) SetMode(v string)`

SetMode sets Mode field to given value.


### GetRepeat

`func (o *AccessExceptionDate) GetRepeat() string`

GetRepeat returns the Repeat field if non-nil, zero value otherwise.

### GetRepeatOk

`func (o *AccessExceptionDate) GetRepeatOk() (*string, bool)`

GetRepeatOk returns a tuple with the Repeat field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRepeat

`func (o *AccessExceptionDate) SetRepeat(v string)`

SetRepeat sets Repeat field to given value.

### HasRepeat

`func (o *AccessExceptionDate) HasRepeat() bool`

HasRepeat returns a boolean if a field has been set.

### GetHoliday

`func (o *AccessExceptionDate) GetHoliday() string`

GetHoliday returns the Holiday field if non-nil, zero value otherwise.

### GetHolidayOk

`func (o *AccessExceptionDate) GetHolidayOk() (*string, bool)`

GetHolidayOk returns a tuple with the Holiday field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHoliday

`func (o *AccessExceptionDate) SetHoliday(v string)`

SetHoliday sets Holiday field to given value.

### HasHoliday

`func (o *AccessExceptionDate) HasHoliday() bool`

HasHoliday returns a boolean if a field has been set.

### GetHolidayDay

`func (o *AccessExceptionDate) GetHolidayDay() string`

GetHolidayDay returns the HolidayDay field if non-nil, zero value otherwise.

### GetHolidayDayOk

`func (o *AccessExceptionDate) GetHolidayDayOk() (*string, bool)`

GetHolidayDayOk returns a tuple with the HolidayDay field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHolidayDay

`func (o *AccessExceptionDate) SetHolidayDay(v string)`

SetHolidayDay sets HolidayDay field to given value.

### HasHolidayDay

`func (o *AccessExceptionDate) HasHolidayDay() bool`

HasHolidayDay returns a boolean if a field has been set.

### GetLunarMonth

`func (o *AccessExceptionDate) GetLunarMonth() int32`

GetLunarMonth returns the LunarMonth field if non-nil, zero value otherwise.

### GetLunarMonthOk

`func (o *AccessExceptionDate) GetLunarMonthOk() (*int32, bool)`

GetLunarMonthOk returns a tuple with the LunarMonth field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLunarMonth

`func (o *AccessExceptionDate) SetLunarMonth(v int32)`

SetLunarMonth sets LunarMonth field to given value.

### HasLunarMonth

`func (o *AccessExceptionDate) HasLunarMonth() bool`

HasLunarMonth returns a boolean if a field has been set.

### GetLunarDay

`func (o *AccessExceptionDate) GetLunarDay() int32`

GetLunarDay returns the LunarDay field if non-nil, zero value otherwise.

### GetLunarDayOk

`func (o *AccessExceptionDate) GetLunarDayOk() (*int32, bool)`

GetLunarDayOk returns a tuple with the LunarDay field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLunarDay

`func (o *AccessExceptionDate) SetLunarDay(v int32)`

SetLunarDay sets LunarDay field to given value.

### HasLunarDay

`func (o *AccessExceptionDate) HasLunarDay() bool`

HasLunarDay returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


