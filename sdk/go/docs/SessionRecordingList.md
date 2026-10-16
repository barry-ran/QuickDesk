# SessionRecordingList

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Items** | [**[]SessionRecording**](SessionRecording.md) |  | 
**NextCursor** | Pointer to **string** |  | [optional] 

## Methods

### NewSessionRecordingList

`func NewSessionRecordingList(items []SessionRecording, ) *SessionRecordingList`

NewSessionRecordingList instantiates a new SessionRecordingList object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSessionRecordingListWithDefaults

`func NewSessionRecordingListWithDefaults() *SessionRecordingList`

NewSessionRecordingListWithDefaults instantiates a new SessionRecordingList object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetItems

`func (o *SessionRecordingList) GetItems() []SessionRecording`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *SessionRecordingList) GetItemsOk() (*[]SessionRecording, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *SessionRecordingList) SetItems(v []SessionRecording)`

SetItems sets Items field to given value.


### GetNextCursor

`func (o *SessionRecordingList) GetNextCursor() string`

GetNextCursor returns the NextCursor field if non-nil, zero value otherwise.

### GetNextCursorOk

`func (o *SessionRecordingList) GetNextCursorOk() (*string, bool)`

GetNextCursorOk returns a tuple with the NextCursor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextCursor

`func (o *SessionRecordingList) SetNextCursor(v string)`

SetNextCursor sets NextCursor field to given value.

### HasNextCursor

`func (o *SessionRecordingList) HasNextCursor() bool`

HasNextCursor returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


