# NotificationChannelList

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Items** | [**[]NotificationChannel**](NotificationChannel.md) |  | 
**SupportedEvents** | **[]string** |  | 
**SignatureHeader** | **string** |  | 
**SignatureScheme** | **string** |  | 

## Methods

### NewNotificationChannelList

`func NewNotificationChannelList(items []NotificationChannel, supportedEvents []string, signatureHeader string, signatureScheme string, ) *NotificationChannelList`

NewNotificationChannelList instantiates a new NotificationChannelList object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewNotificationChannelListWithDefaults

`func NewNotificationChannelListWithDefaults() *NotificationChannelList`

NewNotificationChannelListWithDefaults instantiates a new NotificationChannelList object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetItems

`func (o *NotificationChannelList) GetItems() []NotificationChannel`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *NotificationChannelList) GetItemsOk() (*[]NotificationChannel, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *NotificationChannelList) SetItems(v []NotificationChannel)`

SetItems sets Items field to given value.


### GetSupportedEvents

`func (o *NotificationChannelList) GetSupportedEvents() []string`

GetSupportedEvents returns the SupportedEvents field if non-nil, zero value otherwise.

### GetSupportedEventsOk

`func (o *NotificationChannelList) GetSupportedEventsOk() (*[]string, bool)`

GetSupportedEventsOk returns a tuple with the SupportedEvents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSupportedEvents

`func (o *NotificationChannelList) SetSupportedEvents(v []string)`

SetSupportedEvents sets SupportedEvents field to given value.


### GetSignatureHeader

`func (o *NotificationChannelList) GetSignatureHeader() string`

GetSignatureHeader returns the SignatureHeader field if non-nil, zero value otherwise.

### GetSignatureHeaderOk

`func (o *NotificationChannelList) GetSignatureHeaderOk() (*string, bool)`

GetSignatureHeaderOk returns a tuple with the SignatureHeader field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSignatureHeader

`func (o *NotificationChannelList) SetSignatureHeader(v string)`

SetSignatureHeader sets SignatureHeader field to given value.


### GetSignatureScheme

`func (o *NotificationChannelList) GetSignatureScheme() string`

GetSignatureScheme returns the SignatureScheme field if non-nil, zero value otherwise.

### GetSignatureSchemeOk

`func (o *NotificationChannelList) GetSignatureSchemeOk() (*string, bool)`

GetSignatureSchemeOk returns a tuple with the SignatureScheme field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSignatureScheme

`func (o *NotificationChannelList) SetSignatureScheme(v string)`

SetSignatureScheme sets SignatureScheme field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


