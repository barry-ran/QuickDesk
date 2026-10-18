# SessionRecording

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **int32** |  | 
**UserId** | **int32** |  | 
**DeviceId** | **string** |  | 
**DeviceName** | **string** |  | 
**SessionId** | **string** |  | 
**Status** | **string** | &#x60;recording&#x60; while the capture is in progress, then &#x60;completed&#x60; or &#x60;failed&#x60;. A stale &#x60;recording&#x60; row is swept to &#x60;failed&#x60; by the recording lifecycle worker.  | 
**Format** | **string** | Container/codec hint, e.g. &#x60;mp4&#x60;, &#x60;h264&#x60;, &#x60;grf&#x60;. | 
**ArtifactPath** | **string** | Server-side stored path; not a client-fetchable URL. | 
**ArtifactUrl** | **string** |  | 
**ThumbnailPath** | **string** |  | 
**ThumbnailUrl** | **string** |  | 
**DurationSeconds** | **int32** |  | 
**SizeBytes** | **int64** |  | 
**StartedAt** | **time.Time** |  | 
**EndedAt** | Pointer to **time.Time** |  | [optional] 
**CreatedAt** | **time.Time** |  | 
**UpdatedAt** | **time.Time** |  | 
**User** | Pointer to **map[string]interface{}** | The owning user, present when the server eager-loads it. Optional - clients must not depend on this being present.  | [optional] 

## Methods

### NewSessionRecording

`func NewSessionRecording(id int32, userId int32, deviceId string, deviceName string, sessionId string, status string, format string, artifactPath string, artifactUrl string, thumbnailPath string, thumbnailUrl string, durationSeconds int32, sizeBytes int64, startedAt time.Time, createdAt time.Time, updatedAt time.Time, ) *SessionRecording`

NewSessionRecording instantiates a new SessionRecording object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionRecordingWithDefaults

`func NewSessionRecordingWithDefaults() *SessionRecording`

NewSessionRecordingWithDefaults instantiates a new SessionRecording object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *SessionRecording) GetId() int32`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *SessionRecording) GetIdOk() (*int32, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *SessionRecording) SetId(v int32)`

SetId sets Id field to given value.


### GetUserId

`func (o *SessionRecording) GetUserId() int32`

GetUserId returns the UserId field if non-nil, zero value otherwise.

### GetUserIdOk

`func (o *SessionRecording) GetUserIdOk() (*int32, bool)`

GetUserIdOk returns a tuple with the UserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserId

`func (o *SessionRecording) SetUserId(v int32)`

SetUserId sets UserId field to given value.


### GetDeviceId

`func (o *SessionRecording) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *SessionRecording) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *SessionRecording) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetDeviceName

`func (o *SessionRecording) GetDeviceName() string`

GetDeviceName returns the DeviceName field if non-nil, zero value otherwise.

### GetDeviceNameOk

`func (o *SessionRecording) GetDeviceNameOk() (*string, bool)`

GetDeviceNameOk returns a tuple with the DeviceName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceName

`func (o *SessionRecording) SetDeviceName(v string)`

SetDeviceName sets DeviceName field to given value.


### GetSessionId

`func (o *SessionRecording) GetSessionId() string`

GetSessionId returns the SessionId field if non-nil, zero value otherwise.

### GetSessionIdOk

`func (o *SessionRecording) GetSessionIdOk() (*string, bool)`

GetSessionIdOk returns a tuple with the SessionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSessionId

`func (o *SessionRecording) SetSessionId(v string)`

SetSessionId sets SessionId field to given value.


### GetStatus

`func (o *SessionRecording) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *SessionRecording) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *SessionRecording) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetFormat

`func (o *SessionRecording) GetFormat() string`

GetFormat returns the Format field if non-nil, zero value otherwise.

### GetFormatOk

`func (o *SessionRecording) GetFormatOk() (*string, bool)`

GetFormatOk returns a tuple with the Format field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFormat

`func (o *SessionRecording) SetFormat(v string)`

SetFormat sets Format field to given value.


### GetArtifactPath

`func (o *SessionRecording) GetArtifactPath() string`

GetArtifactPath returns the ArtifactPath field if non-nil, zero value otherwise.

### GetArtifactPathOk

`func (o *SessionRecording) GetArtifactPathOk() (*string, bool)`

GetArtifactPathOk returns a tuple with the ArtifactPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArtifactPath

`func (o *SessionRecording) SetArtifactPath(v string)`

SetArtifactPath sets ArtifactPath field to given value.


### GetArtifactUrl

`func (o *SessionRecording) GetArtifactUrl() string`

GetArtifactUrl returns the ArtifactUrl field if non-nil, zero value otherwise.

### GetArtifactUrlOk

`func (o *SessionRecording) GetArtifactUrlOk() (*string, bool)`

GetArtifactUrlOk returns a tuple with the ArtifactUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArtifactUrl

`func (o *SessionRecording) SetArtifactUrl(v string)`

SetArtifactUrl sets ArtifactUrl field to given value.


### GetThumbnailPath

`func (o *SessionRecording) GetThumbnailPath() string`

GetThumbnailPath returns the ThumbnailPath field if non-nil, zero value otherwise.

### GetThumbnailPathOk

`func (o *SessionRecording) GetThumbnailPathOk() (*string, bool)`

GetThumbnailPathOk returns a tuple with the ThumbnailPath field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetThumbnailPath

`func (o *SessionRecording) SetThumbnailPath(v string)`

SetThumbnailPath sets ThumbnailPath field to given value.


### GetThumbnailUrl

`func (o *SessionRecording) GetThumbnailUrl() string`

GetThumbnailUrl returns the ThumbnailUrl field if non-nil, zero value otherwise.

### GetThumbnailUrlOk

`func (o *SessionRecording) GetThumbnailUrlOk() (*string, bool)`

GetThumbnailUrlOk returns a tuple with the ThumbnailUrl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetThumbnailUrl

`func (o *SessionRecording) SetThumbnailUrl(v string)`

SetThumbnailUrl sets ThumbnailUrl field to given value.


### GetDurationSeconds

`func (o *SessionRecording) GetDurationSeconds() int32`

GetDurationSeconds returns the DurationSeconds field if non-nil, zero value otherwise.

### GetDurationSecondsOk

`func (o *SessionRecording) GetDurationSecondsOk() (*int32, bool)`

GetDurationSecondsOk returns a tuple with the DurationSeconds field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDurationSeconds

`func (o *SessionRecording) SetDurationSeconds(v int32)`

SetDurationSeconds sets DurationSeconds field to given value.


### GetSizeBytes

`func (o *SessionRecording) GetSizeBytes() int64`

GetSizeBytes returns the SizeBytes field if non-nil, zero value otherwise.

### GetSizeBytesOk

`func (o *SessionRecording) GetSizeBytesOk() (*int64, bool)`

GetSizeBytesOk returns a tuple with the SizeBytes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSizeBytes

`func (o *SessionRecording) SetSizeBytes(v int64)`

SetSizeBytes sets SizeBytes field to given value.


### GetStartedAt

`func (o *SessionRecording) GetStartedAt() time.Time`

GetStartedAt returns the StartedAt field if non-nil, zero value otherwise.

### GetStartedAtOk

`func (o *SessionRecording) GetStartedAtOk() (*time.Time, bool)`

GetStartedAtOk returns a tuple with the StartedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStartedAt

`func (o *SessionRecording) SetStartedAt(v time.Time)`

SetStartedAt sets StartedAt field to given value.


### GetEndedAt

`func (o *SessionRecording) GetEndedAt() time.Time`

GetEndedAt returns the EndedAt field if non-nil, zero value otherwise.

### GetEndedAtOk

`func (o *SessionRecording) GetEndedAtOk() (*time.Time, bool)`

GetEndedAtOk returns a tuple with the EndedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEndedAt

`func (o *SessionRecording) SetEndedAt(v time.Time)`

SetEndedAt sets EndedAt field to given value.

### HasEndedAt

`func (o *SessionRecording) HasEndedAt() bool`

HasEndedAt returns a boolean if a field has been set.

### GetCreatedAt

`func (o *SessionRecording) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *SessionRecording) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *SessionRecording) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.


### GetUpdatedAt

`func (o *SessionRecording) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *SessionRecording) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *SessionRecording) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.


### GetUser

`func (o *SessionRecording) GetUser() map[string]interface{}`

GetUser returns the User field if non-nil, zero value otherwise.

### GetUserOk

`func (o *SessionRecording) GetUserOk() (*map[string]interface{}, bool)`

GetUserOk returns a tuple with the User field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUser

`func (o *SessionRecording) SetUser(v map[string]interface{})`

SetUser sets User field to given value.

### HasUser

`func (o *SessionRecording) HasUser() bool`

HasUser returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


