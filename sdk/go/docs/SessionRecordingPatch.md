# SessionRecordingPatch

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Status** | Pointer to **string** |  | [optional] 
**Format** | Pointer to **string** |  | [optional] 
**ArtifactPath** | Pointer to **string** |  | [optional] 
**ArtifactUrl** | Pointer to **string** |  | [optional] 
**ThumbnailPath** | Pointer to **string** |  | [optional] 
**DurationSeconds** | Pointer to **int32** |  | [optional] 
**SizeBytes** | Pointer to **int32** |  | [optional] 
**EndedAt** | Pointer to **NullableTime** |  | [optional] 

## Methods

### NewSessionRecordingPatch

`func NewSessionRecordingPatch() *SessionRecordingPatch`

NewSessionRecordingPatch instantiates a new SessionRecordingPatch object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionRecordingPatchWithDefaults

`func NewSessionRecordingPatchWithDefaults() *SessionRecordingPatch`

NewSessionRecordingPatchWithDefaults instantiates a new SessionRecordingPatch object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetStatus

`func (o *SessionRecordingPatch) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *SessionRecordingPatch) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *SessionRecordingPatch) SetStatus(v string)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *SessionRecordingPatch) HasStatus() bool`

HasStatus returns a boolean if a field has been set.

### GetFormat

`func (o *SessionRecordingPatch) GetFormat() string`

GetFormat returns the Format field if non-nil, zero value otherwise.

### GetFormatOk

`func (o *SessionRecordingPatch) GetFormatOk() (*string, bool)`

GetFormatOk returns a tuple with the Format field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFormat

`func (o *SessionRecordingPatch) SetFormat(v string)`

SetFormat sets Format field to given value.

### HasFormat

`func (o *SessionRecordingPatch) HasFormat() bool`

HasFormat returns a boolean if a field has been set.

### GetArtifactPath

`func (o *SessionRecordingPatch) GetArtifactPath() string`

GetArtifactPath returns the ArtifactPath field if non-nil, zero value otherwise.

### GetArtifactPathOk

`func (o *SessionRecordingPatch) GetArtifactPathOk() (*string, bool)`

GetArtifactPathOk returns a tuple with the ArtifactPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArtifactPath

`func (o *SessionRecordingPatch) SetArtifactPath(v string)`

SetArtifactPath sets ArtifactPath field to given value.

### HasArtifactPath

`func (o *SessionRecordingPatch) HasArtifactPath() bool`

HasArtifactPath returns a boolean if a field has been set.

### GetArtifactUrl

`func (o *SessionRecordingPatch) GetArtifactUrl() string`

GetArtifactUrl returns the ArtifactUrl field if non-nil, zero value otherwise.

### GetArtifactUrlOk

`func (o *SessionRecordingPatch) GetArtifactUrlOk() (*string, bool)`

GetArtifactUrlOk returns a tuple with the ArtifactUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArtifactUrl

`func (o *SessionRecordingPatch) SetArtifactUrl(v string)`

SetArtifactUrl sets ArtifactUrl field to given value.

### HasArtifactUrl

`func (o *SessionRecordingPatch) HasArtifactUrl() bool`

HasArtifactUrl returns a boolean if a field has been set.

### GetThumbnailPath

`func (o *SessionRecordingPatch) GetThumbnailPath() string`

GetThumbnailPath returns the ThumbnailPath field if non-nil, zero value otherwise.

### GetThumbnailPathOk

`func (o *SessionRecordingPatch) GetThumbnailPathOk() (*string, bool)`

GetThumbnailPathOk returns a tuple with the ThumbnailPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetThumbnailPath

`func (o *SessionRecordingPatch) SetThumbnailPath(v string)`

SetThumbnailPath sets ThumbnailPath field to given value.

### HasThumbnailPath

`func (o *SessionRecordingPatch) HasThumbnailPath() bool`

HasThumbnailPath returns a boolean if a field has been set.

### GetDurationSeconds

`func (o *SessionRecordingPatch) GetDurationSeconds() int32`

GetDurationSeconds returns the DurationSeconds field if non-nil, zero value otherwise.

### GetDurationSecondsOk

`func (o *SessionRecordingPatch) GetDurationSecondsOk() (*int32, bool)`

GetDurationSecondsOk returns a tuple with the DurationSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDurationSeconds

`func (o *SessionRecordingPatch) SetDurationSeconds(v int32)`

SetDurationSeconds sets DurationSeconds field to given value.

### HasDurationSeconds

`func (o *SessionRecordingPatch) HasDurationSeconds() bool`

HasDurationSeconds returns a boolean if a field has been set.

### GetSizeBytes

`func (o *SessionRecordingPatch) GetSizeBytes() int32`

GetSizeBytes returns the SizeBytes field if non-nil, zero value otherwise.

### GetSizeBytesOk

`func (o *SessionRecordingPatch) GetSizeBytesOk() (*int32, bool)`

GetSizeBytesOk returns a tuple with the SizeBytes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSizeBytes

`func (o *SessionRecordingPatch) SetSizeBytes(v int32)`

SetSizeBytes sets SizeBytes field to given value.

### HasSizeBytes

`func (o *SessionRecordingPatch) HasSizeBytes() bool`

HasSizeBytes returns a boolean if a field has been set.

### GetEndedAt

`func (o *SessionRecordingPatch) GetEndedAt() time.Time`

GetEndedAt returns the EndedAt field if non-nil, zero value otherwise.

### GetEndedAtOk

`func (o *SessionRecordingPatch) GetEndedAtOk() (*time.Time, bool)`

GetEndedAtOk returns a tuple with the EndedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEndedAt

`func (o *SessionRecordingPatch) SetEndedAt(v time.Time)`

SetEndedAt sets EndedAt field to given value.

### HasEndedAt

`func (o *SessionRecordingPatch) HasEndedAt() bool`

HasEndedAt returns a boolean if a field has been set.

### SetEndedAtNil

`func (o *SessionRecordingPatch) SetEndedAtNil(b bool)`

 SetEndedAtNil sets the value for EndedAt to be an explicit nil

### UnsetEndedAt
`func (o *SessionRecordingPatch) UnsetEndedAt()`

UnsetEndedAt ensures that no value is present for EndedAt, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


