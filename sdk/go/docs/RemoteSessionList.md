# RemoteSessionList

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Items** | [**[]RemoteSession**](RemoteSession.md) |  | 
**NextCursor** | Pointer to **string** |  | [optional] 

## Methods

### NewRemoteSessionList

`func NewRemoteSessionList(items []RemoteSession, ) *RemoteSessionList`

NewRemoteSessionList instantiates a new RemoteSessionList object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRemoteSessionListWithDefaults

`func NewRemoteSessionListWithDefaults() *RemoteSessionList`

NewRemoteSessionListWithDefaults instantiates a new RemoteSessionList object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetItems

`func (o *RemoteSessionList) GetItems() []RemoteSession`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *RemoteSessionList) GetItemsOk() (*[]RemoteSession, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *RemoteSessionList) SetItems(v []RemoteSession)`

SetItems sets Items field to given value.


### GetNextCursor

`func (o *RemoteSessionList) GetNextCursor() string`

GetNextCursor returns the NextCursor field if non-nil, zero value otherwise.

### GetNextCursorOk

`func (o *RemoteSessionList) GetNextCursorOk() (*string, bool)`

GetNextCursorOk returns a tuple with the NextCursor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextCursor

`func (o *RemoteSessionList) SetNextCursor(v string)`

SetNextCursor sets NextCursor field to given value.

### HasNextCursor

`func (o *RemoteSessionList) HasNextCursor() bool`

HasNextCursor returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


