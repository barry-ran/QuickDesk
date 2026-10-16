# UpdateCheckResult

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**UpdateAvailable** | **bool** |  | 
**Mandatory** | **bool** |  | 
**Channel** | **string** |  | 
**Platform** | **string** |  | 
**Arch** | **string** |  | 
**CurrentVersion** | Pointer to **string** | Echoed when the caller supplied it. | [optional] 
**Latest** | [**NullableUpdateRelease**](UpdateRelease.md) |  | 

## Methods

### NewUpdateCheckResult

`func NewUpdateCheckResult(updateAvailable bool, mandatory bool, channel string, platform string, arch string, latest NullableUpdateRelease, ) *UpdateCheckResult`

NewUpdateCheckResult instantiates a new UpdateCheckResult object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUpdateCheckResultWithDefaults

`func NewUpdateCheckResultWithDefaults() *UpdateCheckResult`

NewUpdateCheckResultWithDefaults instantiates a new UpdateCheckResult object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetUpdateAvailable

`func (o *UpdateCheckResult) GetUpdateAvailable() bool`

GetUpdateAvailable returns the UpdateAvailable field if non-nil, zero value otherwise.

### GetUpdateAvailableOk

`func (o *UpdateCheckResult) GetUpdateAvailableOk() (*bool, bool)`

GetUpdateAvailableOk returns a tuple with the UpdateAvailable field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdateAvailable

`func (o *UpdateCheckResult) SetUpdateAvailable(v bool)`

SetUpdateAvailable sets UpdateAvailable field to given value.


### GetMandatory

`func (o *UpdateCheckResult) GetMandatory() bool`

GetMandatory returns the Mandatory field if non-nil, zero value otherwise.

### GetMandatoryOk

`func (o *UpdateCheckResult) GetMandatoryOk() (*bool, bool)`

GetMandatoryOk returns a tuple with the Mandatory field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMandatory

`func (o *UpdateCheckResult) SetMandatory(v bool)`

SetMandatory sets Mandatory field to given value.


### GetChannel

`func (o *UpdateCheckResult) GetChannel() string`

GetChannel returns the Channel field if non-nil, zero value otherwise.

### GetChannelOk

`func (o *UpdateCheckResult) GetChannelOk() (*string, bool)`

GetChannelOk returns a tuple with the Channel field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChannel

`func (o *UpdateCheckResult) SetChannel(v string)`

SetChannel sets Channel field to given value.


### GetPlatform

`func (o *UpdateCheckResult) GetPlatform() string`

GetPlatform returns the Platform field if non-nil, zero value otherwise.

### GetPlatformOk

`func (o *UpdateCheckResult) GetPlatformOk() (*string, bool)`

GetPlatformOk returns a tuple with the Platform field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPlatform

`func (o *UpdateCheckResult) SetPlatform(v string)`

SetPlatform sets Platform field to given value.


### GetArch

`func (o *UpdateCheckResult) GetArch() string`

GetArch returns the Arch field if non-nil, zero value otherwise.

### GetArchOk

`func (o *UpdateCheckResult) GetArchOk() (*string, bool)`

GetArchOk returns a tuple with the Arch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArch

`func (o *UpdateCheckResult) SetArch(v string)`

SetArch sets Arch field to given value.


### GetCurrentVersion

`func (o *UpdateCheckResult) GetCurrentVersion() string`

GetCurrentVersion returns the CurrentVersion field if non-nil, zero value otherwise.

### GetCurrentVersionOk

`func (o *UpdateCheckResult) GetCurrentVersionOk() (*string, bool)`

GetCurrentVersionOk returns a tuple with the CurrentVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentVersion

`func (o *UpdateCheckResult) SetCurrentVersion(v string)`

SetCurrentVersion sets CurrentVersion field to given value.

### HasCurrentVersion

`func (o *UpdateCheckResult) HasCurrentVersion() bool`

HasCurrentVersion returns a boolean if a field has been set.

### GetLatest

`func (o *UpdateCheckResult) GetLatest() UpdateRelease`

GetLatest returns the Latest field if non-nil, zero value otherwise.

### GetLatestOk

`func (o *UpdateCheckResult) GetLatestOk() (*UpdateRelease, bool)`

GetLatestOk returns a tuple with the Latest field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLatest

`func (o *UpdateCheckResult) SetLatest(v UpdateRelease)`

SetLatest sets Latest field to given value.


### SetLatestNil

`func (o *UpdateCheckResult) SetLatestNil(b bool)`

 SetLatestNil sets the value for Latest to be an explicit nil

### UnsetLatest
`func (o *UpdateCheckResult) UnsetLatest()`

UnsetLatest ensures that no value is present for Latest, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


