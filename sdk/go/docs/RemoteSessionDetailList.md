# RemoteSessionDetailList

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Items** | [**[]RemoteSessionDetail**](RemoteSessionDetail.md) |  | 
**NextCursor** | Pointer to **string** |  | [optional] 

## Methods

### NewRemoteSessionDetailList

`func NewRemoteSessionDetailList(items []RemoteSessionDetail, ) *RemoteSessionDetailList`

NewRemoteSessionDetailList instantiates a new RemoteSessionDetailList object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRemoteSessionDetailListWithDefaults

`func NewRemoteSessionDetailListWithDefaults() *RemoteSessionDetailList`

NewRemoteSessionDetailListWithDefaults instantiates a new RemoteSessionDetailList object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetItems

`func (o *RemoteSessionDetailList) GetItems() []RemoteSessionDetail`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *RemoteSessionDetailList) GetItemsOk() (*[]RemoteSessionDetail, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *RemoteSessionDetailList) SetItems(v []RemoteSessionDetail)`

SetItems sets Items field to given value.


### GetNextCursor

`func (o *RemoteSessionDetailList) GetNextCursor() string`

GetNextCursor returns the NextCursor field if non-nil, zero value otherwise.

### GetNextCursorOk

`func (o *RemoteSessionDetailList) GetNextCursorOk() (*string, bool)`

GetNextCursorOk returns a tuple with the NextCursor field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextCursor

`func (o *RemoteSessionDetailList) SetNextCursor(v string)`

SetNextCursor sets NextCursor field to given value.

### HasNextCursor

`func (o *RemoteSessionDetailList) HasNextCursor() bool`

HasNextCursor returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


