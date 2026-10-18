# PlaybackTicket

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Url** | **string** | Absolute path (with the signed &#x60;t&#x60; parameter) to fetch. | 
**ExpiresAt** | **time.Time** |  | 

## Methods

### NewPlaybackTicket

`func NewPlaybackTicket(url string, expiresAt time.Time, ) *PlaybackTicket`

NewPlaybackTicket instantiates a new PlaybackTicket object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPlaybackTicketWithDefaults

`func NewPlaybackTicketWithDefaults() *PlaybackTicket`

NewPlaybackTicketWithDefaults instantiates a new PlaybackTicket object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetUrl

`func (o *PlaybackTicket) GetUrl() string`

GetUrl returns the Url field if non-nil, zero value otherwise.

### GetUrlOk

`func (o *PlaybackTicket) GetUrlOk() (*string, bool)`

GetUrlOk returns a tuple with the Url field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUrl

`func (o *PlaybackTicket) SetUrl(v string)`

SetUrl sets Url field to given value.


### GetExpiresAt

`func (o *PlaybackTicket) GetExpiresAt() time.Time`

GetExpiresAt returns the ExpiresAt field if non-nil, zero value otherwise.

### GetExpiresAtOk

`func (o *PlaybackTicket) GetExpiresAtOk() (*time.Time, bool)`

GetExpiresAtOk returns a tuple with the ExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresAt

`func (o *PlaybackTicket) SetExpiresAt(v time.Time)`

SetExpiresAt sets ExpiresAt field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


