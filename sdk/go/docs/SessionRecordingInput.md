# SessionRecordingInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DeviceId** | **string** | The device the capture came from. | 
**DeviceName** | Pointer to **string** |  | [optional] 
**SessionId** | Pointer to **string** | The control session this recording belongs to, when there is one. | [optional] 
**Status** | **string** | How far the capture got when this row was written. | 
**Format** | Pointer to **string** | Free-form label; the upload route overwrites it from the file extension. | [optional] 
**ArtifactPath** | Pointer to **string** | Server-side path. Set by the upload route; clients do not need to send it. | [optional] 
**ArtifactUrl** | Pointer to **string** |  | [optional] 
**ThumbnailPath** | Pointer to **string** |  | [optional] 
**DurationSeconds** | Pointer to **int32** |  | [optional] 
**SizeBytes** | Pointer to **int32** |  | [optional] 
**StartedAt** | Pointer to **NullableTime** | Defaults to the time the row was created when omitted. | [optional] 
**EndedAt** | Pointer to **NullableTime** |  | [optional] 

## Methods

### NewSessionRecordingInput

`func NewSessionRecordingInput(deviceId string, status string, ) *SessionRecordingInput`

NewSessionRecordingInput instantiates a new SessionRecordingInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionRecordingInputWithDefaults

`func NewSessionRecordingInputWithDefaults() *SessionRecordingInput`

NewSessionRecordingInputWithDefaults instantiates a new SessionRecordingInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDeviceId

`func (o *SessionRecordingInput) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *SessionRecordingInput) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *SessionRecordingInput) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetDeviceName

`func (o *SessionRecordingInput) GetDeviceName() string`

GetDeviceName returns the DeviceName field if non-nil, zero value otherwise.

### GetDeviceNameOk

`func (o *SessionRecordingInput) GetDeviceNameOk() (*string, bool)`

GetDeviceNameOk returns a tuple with the DeviceName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceName

`func (o *SessionRecordingInput) SetDeviceName(v string)`

SetDeviceName sets DeviceName field to given value.

### HasDeviceName

`func (o *SessionRecordingInput) HasDeviceName() bool`

HasDeviceName returns a boolean if a field has been set.

### GetSessionId

`func (o *SessionRecordingInput) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *SessionRecordingInput) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *SessionRecordingInput) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.

### HasSessionId

`func (o *SessionRecordingInput) HasSessionId() bool`

HasSessionId returns a boolean if a field has been set.

### GetStatus

`func (o *SessionRecordingInput) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *SessionRecordingInput) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *SessionRecordingInput) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetFormat

`func (o *SessionRecordingInput) GetFormat() string`

GetFormat returns the Format field if non-nil, zero value otherwise.

### GetFormatOk

`func (o *SessionRecordingInput) GetFormatOk() (*string, bool)`

GetFormatOk returns a tuple with the Format field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFormat

`func (o *SessionRecordingInput) SetFormat(v string)`

SetFormat sets Format field to given value.

### HasFormat

`func (o *SessionRecordingInput) HasFormat() bool`

HasFormat returns a boolean if a field has been set.

### GetArtifactPath

`func (o *SessionRecordingInput) GetArtifactPath() string`

GetArtifactPath returns the ArtifactPath field if non-nil, zero value otherwise.

### GetArtifactPathOk

`func (o *SessionRecordingInput) GetArtifactPathOk() (*string, bool)`

GetArtifactPathOk returns a tuple with the ArtifactPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArtifactPath

`func (o *SessionRecordingInput) SetArtifactPath(v string)`

SetArtifactPath sets ArtifactPath field to given value.

### HasArtifactPath

`func (o *SessionRecordingInput) HasArtifactPath() bool`

HasArtifactPath returns a boolean if a field has been set.

### GetArtifactUrl

`func (o *SessionRecordingInput) GetArtifactUrl() string`

GetArtifactUrl returns the ArtifactUrl field if non-nil, zero value otherwise.

### GetArtifactUrlOk

`func (o *SessionRecordingInput) GetArtifactUrlOk() (*string, bool)`

GetArtifactUrlOk returns a tuple with the ArtifactUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArtifactUrl

`func (o *SessionRecordingInput) SetArtifactUrl(v string)`

SetArtifactUrl sets ArtifactUrl field to given value.

### HasArtifactUrl

`func (o *SessionRecordingInput) HasArtifactUrl() bool`

HasArtifactUrl returns a boolean if a field has been set.

### GetThumbnailPath

`func (o *SessionRecordingInput) GetThumbnailPath() string`

GetThumbnailPath returns the ThumbnailPath field if non-nil, zero value otherwise.

### GetThumbnailPathOk

`func (o *SessionRecordingInput) GetThumbnailPathOk() (*string, bool)`

GetThumbnailPathOk returns a tuple with the ThumbnailPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetThumbnailPath

`func (o *SessionRecordingInput) SetThumbnailPath(v string)`

SetThumbnailPath sets ThumbnailPath field to given value.

### HasThumbnailPath

`func (o *SessionRecordingInput) HasThumbnailPath() bool`

HasThumbnailPath returns a boolean if a field has been set.

### GetDurationSeconds

`func (o *SessionRecordingInput) GetDurationSeconds() int32`

GetDurationSeconds returns the DurationSeconds field if non-nil, zero value otherwise.

### GetDurationSecondsOk

`func (o *SessionRecordingInput) GetDurationSecondsOk() (*int32, bool)`

GetDurationSecondsOk returns a tuple with the DurationSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDurationSeconds

`func (o *SessionRecordingInput) SetDurationSeconds(v int32)`

SetDurationSeconds sets DurationSeconds field to given value.

### HasDurationSeconds

`func (o *SessionRecordingInput) HasDurationSeconds() bool`

HasDurationSeconds returns a boolean if a field has been set.

### GetSizeBytes

`func (o *SessionRecordingInput) GetSizeBytes() int32`

GetSizeBytes returns the SizeBytes field if non-nil, zero value otherwise.

### GetSizeBytesOk

`func (o *SessionRecordingInput) GetSizeBytesOk() (*int32, bool)`

GetSizeBytesOk returns a tuple with the SizeBytes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSizeBytes

`func (o *SessionRecordingInput) SetSizeBytes(v int32)`

SetSizeBytes sets SizeBytes field to given value.

### HasSizeBytes

`func (o *SessionRecordingInput) HasSizeBytes() bool`

HasSizeBytes returns a boolean if a field has been set.

### GetStartedAt

`func (o *SessionRecordingInput) GetStartedAt() time.Time`

GetStartedAt returns the StartedAt field if non-nil, zero value otherwise.

### GetStartedAtOk

`func (o *SessionRecordingInput) GetStartedAtOk() (*time.Time, bool)`

GetStartedAtOk returns a tuple with the StartedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartedAt

`func (o *SessionRecordingInput) SetStartedAt(v time.Time)`

SetStartedAt sets StartedAt field to given value.

### HasStartedAt

`func (o *SessionRecordingInput) HasStartedAt() bool`

HasStartedAt returns a boolean if a field has been set.

### SetStartedAtNil

`func (o *SessionRecordingInput) SetStartedAtNil(b bool)`

 SetStartedAtNil sets the value for StartedAt to be an explicit nil

### UnsetStartedAt
`func (o *SessionRecordingInput) UnsetStartedAt()`

UnsetStartedAt ensures that no value is present for StartedAt, not even an explicit nil
### GetEndedAt

`func (o *SessionRecordingInput) GetEndedAt() time.Time`

GetEndedAt returns the EndedAt field if non-nil, zero value otherwise.

### GetEndedAtOk

`func (o *SessionRecordingInput) GetEndedAtOk() (*time.Time, bool)`

GetEndedAtOk returns a tuple with the EndedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEndedAt

`func (o *SessionRecordingInput) SetEndedAt(v time.Time)`

SetEndedAt sets EndedAt field to given value.

### HasEndedAt

`func (o *SessionRecordingInput) HasEndedAt() bool`

HasEndedAt returns a boolean if a field has been set.

### SetEndedAtNil

`func (o *SessionRecordingInput) SetEndedAtNil(b bool)`

 SetEndedAtNil sets the value for EndedAt to be an explicit nil

### UnsetEndedAt
`func (o *SessionRecordingInput) UnsetEndedAt()`

UnsetEndedAt ensures that no value is present for EndedAt, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


