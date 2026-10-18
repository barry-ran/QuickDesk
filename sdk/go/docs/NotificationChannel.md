# NotificationChannel

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **int64** |  | 
**UserId** | **int64** |  | 
**Name** | **string** |  | 
**Url** | **string** |  | 
**Events** | **string** | A JSON-encoded array of subscribed event types (a string, not an array, because that is how the column is stored). Decode it before matching against &#x60;supported_events&#x60;.  | 
**Enabled** | **bool** |  | 
**LastTriggered** | Pointer to **NullableTime** |  | [optional] 
**LastStatus** | **int32** | HTTP status of the most recent delivery attempt; 0 means none yet. | 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewNotificationChannel

`func NewNotificationChannel(id int64, userId int64, name string, url string, events string, enabled bool, lastStatus int32, ) *NotificationChannel`

NewNotificationChannel instantiates a new NotificationChannel object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewNotificationChannelWithDefaults

`func NewNotificationChannelWithDefaults() *NotificationChannel`

NewNotificationChannelWithDefaults instantiates a new NotificationChannel object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *NotificationChannel) GetId() int64`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *NotificationChannel) GetIdOk() (*int64, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *NotificationChannel) SetId(v int64)`

SetId sets Id field to given value.


### GetUserId

`func (o *NotificationChannel) GetUserId() int64`

GetUserId returns the UserId field if non-nil, zero value otherwise.

### GetUserIdOk

`func (o *NotificationChannel) GetUserIdOk() (*int64, bool)`

GetUserIdOk returns a tuple with the UserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserId

`func (o *NotificationChannel) SetUserId(v int64)`

SetUserId sets UserId field to given value.


### GetName

`func (o *NotificationChannel) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *NotificationChannel) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *NotificationChannel) SetName(v string)`

SetName sets Name field to given value.


### GetUrl

`func (o *NotificationChannel) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *NotificationChannel) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *NotificationChannel) SetUrl(v string)`

SetUrl sets Url field to given value.


### GetEvents

`func (o *NotificationChannel) GetEvents() string`

GetEvents returns the Events field if non-nil, zero value otherwise.

### GetEventsOk

`func (o *NotificationChannel) GetEventsOk() (*string, bool)`

GetEventsOk returns a tuple with the Events field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEvents

`func (o *NotificationChannel) SetEvents(v string)`

SetEvents sets Events field to given value.


### GetEnabled

`func (o *NotificationChannel) GetEnabled() bool`

GetEnabled returns the Enabled field if non-nil, zero value otherwise.

### GetEnabledOk

`func (o *NotificationChannel) GetEnabledOk() (*bool, bool)`

GetEnabledOk returns a tuple with the Enabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnabled

`func (o *NotificationChannel) SetEnabled(v bool)`

SetEnabled sets Enabled field to given value.


### GetLastTriggered

`func (o *NotificationChannel) GetLastTriggered() time.Time`

GetLastTriggered returns the LastTriggered field if non-nil, zero value otherwise.

### GetLastTriggeredOk

`func (o *NotificationChannel) GetLastTriggeredOk() (*time.Time, bool)`

GetLastTriggeredOk returns a tuple with the LastTriggered field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastTriggered

`func (o *NotificationChannel) SetLastTriggered(v time.Time)`

SetLastTriggered sets LastTriggered field to given value.

### HasLastTriggered

`func (o *NotificationChannel) HasLastTriggered() bool`

HasLastTriggered returns a boolean if a field has been set.

### SetLastTriggeredNil

`func (o *NotificationChannel) SetLastTriggeredNil(b bool)`

 SetLastTriggeredNil sets the value for LastTriggered to be an explicit nil

### UnsetLastTriggered
`func (o *NotificationChannel) UnsetLastTriggered()`

UnsetLastTriggered ensures that no value is present for LastTriggered, not even an explicit nil
### GetLastStatus

`func (o *NotificationChannel) GetLastStatus() int32`

GetLastStatus returns the LastStatus field if non-nil, zero value otherwise.

### GetLastStatusOk

`func (o *NotificationChannel) GetLastStatusOk() (*int32, bool)`

GetLastStatusOk returns a tuple with the LastStatus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastStatus

`func (o *NotificationChannel) SetLastStatus(v int32)`

SetLastStatus sets LastStatus field to given value.


### GetCreatedAt

`func (o *NotificationChannel) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *NotificationChannel) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *NotificationChannel) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *NotificationChannel) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *NotificationChannel) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *NotificationChannel) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *NotificationChannel) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *NotificationChannel) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


