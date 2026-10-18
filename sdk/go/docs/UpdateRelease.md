# UpdateRelease

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **int64** |  | 
**Channel** | **string** |  | 
**Platform** | **string** |  | 
**Arch** | **string** |  | 
**Version** | **string** | Dotted numeric version, e.g. &#x60;4.2.5.0&#x60;. | 
**DownloadUrl** | **string** | Absolute http(s) URL of the installer or archive. | 
**HostedHere** | **bool** | True when this server stores and serves the bytes (so &#x60;download_url&#x60; points at &#x60;GET /v1/update-releases/{release_id}/artifact&#x60;), false when the installer lives on a host the operator manages. | 
**Sha256** | **string** | Lowercase hex SHA-256 of the artifact; clients verify what they downloaded. | 
**SizeBytes** | **int64** |  | 
**Notes** | Pointer to **string** |  | [optional] 
**Mandatory** | **bool** | When true, the update is not optional for older clients. | 
**PublishedAt** | **time.Time** |  | 

## Methods

### NewUpdateRelease

`func NewUpdateRelease(id int64, channel string, platform string, arch string, version string, downloadUrl string, hostedHere bool, sha256 string, sizeBytes int64, mandatory bool, publishedAt time.Time, ) *UpdateRelease`

NewUpdateRelease instantiates a new UpdateRelease object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUpdateReleaseWithDefaults

`func NewUpdateReleaseWithDefaults() *UpdateRelease`

NewUpdateReleaseWithDefaults instantiates a new UpdateRelease object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *UpdateRelease) GetId() int64`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *UpdateRelease) GetIdOk() (*int64, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *UpdateRelease) SetId(v int64)`

SetId sets Id field to given value.


### GetChannel

`func (o *UpdateRelease) GetChannel() string`

GetChannel returns the Channel field if non-nil, zero value otherwise.

### GetChannelOk

`func (o *UpdateRelease) GetChannelOk() (*string, bool)`

GetChannelOk returns a tuple with the Channel field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChannel

`func (o *UpdateRelease) SetChannel(v string)`

SetChannel sets Channel field to given value.


### GetPlatform

`func (o *UpdateRelease) GetPlatform() string`

GetPlatform returns the Platform field if non-nil, zero value otherwise.

### GetPlatformOk

`func (o *UpdateRelease) GetPlatformOk() (*string, bool)`

GetPlatformOk returns a tuple with the Platform field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPlatform

`func (o *UpdateRelease) SetPlatform(v string)`

SetPlatform sets Platform field to given value.


### GetArch

`func (o *UpdateRelease) GetArch() string`

GetArch returns the Arch field if non-nil, zero value otherwise.

### GetArchOk

`func (o *UpdateRelease) GetArchOk() (*string, bool)`

GetArchOk returns a tuple with the Arch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArch

`func (o *UpdateRelease) SetArch(v string)`

SetArch sets Arch field to given value.


### GetVersion

`func (o *UpdateRelease) GetVersion() string`

GetVersion returns the Version field if non-nil, zero value otherwise.

### GetVersionOk

`func (o *UpdateRelease) GetVersionOk() (*string, bool)`

GetVersionOk returns a tuple with the Version field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVersion

`func (o *UpdateRelease) SetVersion(v string)`

SetVersion sets Version field to given value.


### GetDownloadUrl

`func (o *UpdateRelease) GetDownloadUrl() string`

GetDownloadUrl returns the DownloadUrl field if non-nil, zero value otherwise.

### GetDownloadUrlOk

`func (o *UpdateRelease) GetDownloadUrlOk() (*string, bool)`

GetDownloadUrlOk returns a tuple with the DownloadUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDownloadUrl

`func (o *UpdateRelease) SetDownloadUrl(v string)`

SetDownloadUrl sets DownloadUrl field to given value.


### GetHostedHere

`func (o *UpdateRelease) GetHostedHere() bool`

GetHostedHere returns the HostedHere field if non-nil, zero value otherwise.

### GetHostedHereOk

`func (o *UpdateRelease) GetHostedHereOk() (*bool, bool)`

GetHostedHereOk returns a tuple with the HostedHere field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHostedHere

`func (o *UpdateRelease) SetHostedHere(v bool)`

SetHostedHere sets HostedHere field to given value.


### GetSha256

`func (o *UpdateRelease) GetSha256() string`

GetSha256 returns the Sha256 field if non-nil, zero value otherwise.

### GetSha256Ok

`func (o *UpdateRelease) GetSha256Ok() (*string, bool)`

GetSha256Ok returns a tuple with the Sha256 field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSha256

`func (o *UpdateRelease) SetSha256(v string)`

SetSha256 sets Sha256 field to given value.


### GetSizeBytes

`func (o *UpdateRelease) GetSizeBytes() int64`

GetSizeBytes returns the SizeBytes field if non-nil, zero value otherwise.

### GetSizeBytesOk

`func (o *UpdateRelease) GetSizeBytesOk() (*int64, bool)`

GetSizeBytesOk returns a tuple with the SizeBytes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSizeBytes

`func (o *UpdateRelease) SetSizeBytes(v int64)`

SetSizeBytes sets SizeBytes field to given value.


### GetNotes

`func (o *UpdateRelease) GetNotes() string`

GetNotes returns the Notes field if non-nil, zero value otherwise.

### GetNotesOk

`func (o *UpdateRelease) GetNotesOk() (*string, bool)`

GetNotesOk returns a tuple with the Notes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNotes

`func (o *UpdateRelease) SetNotes(v string)`

SetNotes sets Notes field to given value.

### HasNotes

`func (o *UpdateRelease) HasNotes() bool`

HasNotes returns a boolean if a field has been set.

### GetMandatory

`func (o *UpdateRelease) GetMandatory() bool`

GetMandatory returns the Mandatory field if non-nil, zero value otherwise.

### GetMandatoryOk

`func (o *UpdateRelease) GetMandatoryOk() (*bool, bool)`

GetMandatoryOk returns a tuple with the Mandatory field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMandatory

`func (o *UpdateRelease) SetMandatory(v bool)`

SetMandatory sets Mandatory field to given value.


### GetPublishedAt

`func (o *UpdateRelease) GetPublishedAt() time.Time`

GetPublishedAt returns the PublishedAt field if non-nil, zero value otherwise.

### GetPublishedAtOk

`func (o *UpdateRelease) GetPublishedAtOk() (*time.Time, bool)`

GetPublishedAtOk returns a tuple with the PublishedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPublishedAt

`func (o *UpdateRelease) SetPublishedAt(v time.Time)`

SetPublishedAt sets PublishedAt field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


