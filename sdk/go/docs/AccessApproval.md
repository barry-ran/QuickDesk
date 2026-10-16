# AccessApproval

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **int64** |  | 
**DeviceId** | **string** |  | 
**ClientId** | **string** | Self-reported by the requesting controller. | 
**ClientIp** | **string** | The address the server observed (trusted-proxy aware). | 
**Source** | **string** |  | 
**GrantId** | Pointer to **NullableInt64** |  | [optional] 
**Status** | **string** |  | 
**ExpiresAt** | **time.Time** |  | 
**DecidedAt** | Pointer to **NullableTime** |  | [optional] 
**Reason** | Pointer to **string** |  | [optional] 
**CreatedAt** | **time.Time** |  | 

## Methods

### NewAccessApproval

`func NewAccessApproval(id int64, deviceId string, clientId string, clientIp string, source string, status string, expiresAt time.Time, createdAt time.Time, ) *AccessApproval`

NewAccessApproval instantiates a new AccessApproval object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAccessApprovalWithDefaults

`func NewAccessApprovalWithDefaults() *AccessApproval`

NewAccessApprovalWithDefaults instantiates a new AccessApproval object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *AccessApproval) GetId() int64`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *AccessApproval) GetIdOk() (*int64, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *AccessApproval) SetId(v int64)`

SetId sets Id field to given value.


### GetDeviceId

`func (o *AccessApproval) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *AccessApproval) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *AccessApproval) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetClientId

`func (o *AccessApproval) GetClientId() string`

GetClientId returns the ClientId field if non-nil, zero value otherwise.

### GetClientIdOk

`func (o *AccessApproval) GetClientIdOk() (*string, bool)`

GetClientIdOk returns a tuple with the ClientId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClientId

`func (o *AccessApproval) SetClientId(v string)`

SetClientId sets ClientId field to given value.


### GetClientIp

`func (o *AccessApproval) GetClientIp() string`

GetClientIp returns the ClientIp field if non-nil, zero value otherwise.

### GetClientIpOk

`func (o *AccessApproval) GetClientIpOk() (*string, bool)`

GetClientIpOk returns a tuple with the ClientIp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClientIp

`func (o *AccessApproval) SetClientIp(v string)`

SetClientIp sets ClientIp field to given value.


### GetSource

`func (o *AccessApproval) GetSource() string`

GetSource returns the Source field if non-nil, zero value otherwise.

### GetSourceOk

`func (o *AccessApproval) GetSourceOk() (*string, bool)`

GetSourceOk returns a tuple with the Source field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSource

`func (o *AccessApproval) SetSource(v string)`

SetSource sets Source field to given value.


### GetGrantId

`func (o *AccessApproval) GetGrantId() int64`

GetGrantId returns the GrantId field if non-nil, zero value otherwise.

### GetGrantIdOk

`func (o *AccessApproval) GetGrantIdOk() (*int64, bool)`

GetGrantIdOk returns a tuple with the GrantId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGrantId

`func (o *AccessApproval) SetGrantId(v int64)`

SetGrantId sets GrantId field to given value.

### HasGrantId

`func (o *AccessApproval) HasGrantId() bool`

HasGrantId returns a boolean if a field has been set.

### SetGrantIdNil

`func (o *AccessApproval) SetGrantIdNil(b bool)`

 SetGrantIdNil sets the value for GrantId to be an explicit nil

### UnsetGrantId
`func (o *AccessApproval) UnsetGrantId()`

UnsetGrantId ensures that no value is present for GrantId, not even an explicit nil
### GetStatus

`func (o *AccessApproval) GetStatus() string`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *AccessApproval) GetStatusOk() (*string, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *AccessApproval) SetStatus(v string)`

SetStatus sets Status field to given value.


### GetExpiresAt

`func (o *AccessApproval) GetExpiresAt() time.Time`

GetExpiresAt returns the ExpiresAt field if non-nil, zero value otherwise.

### GetExpiresAtOk

`func (o *AccessApproval) GetExpiresAtOk() (*time.Time, bool)`

GetExpiresAtOk returns a tuple with the ExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresAt

`func (o *AccessApproval) SetExpiresAt(v time.Time)`

SetExpiresAt sets ExpiresAt field to given value.


### GetDecidedAt

`func (o *AccessApproval) GetDecidedAt() time.Time`

GetDecidedAt returns the DecidedAt field if non-nil, zero value otherwise.

### GetDecidedAtOk

`func (o *AccessApproval) GetDecidedAtOk() (*time.Time, bool)`

GetDecidedAtOk returns a tuple with the DecidedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDecidedAt

`func (o *AccessApproval) SetDecidedAt(v time.Time)`

SetDecidedAt sets DecidedAt field to given value.

### HasDecidedAt

`func (o *AccessApproval) HasDecidedAt() bool`

HasDecidedAt returns a boolean if a field has been set.

### SetDecidedAtNil

`func (o *AccessApproval) SetDecidedAtNil(b bool)`

 SetDecidedAtNil sets the value for DecidedAt to be an explicit nil

### UnsetDecidedAt
`func (o *AccessApproval) UnsetDecidedAt()`

UnsetDecidedAt ensures that no value is present for DecidedAt, not even an explicit nil
### GetReason

`func (o *AccessApproval) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *AccessApproval) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *AccessApproval) SetReason(v string)`

SetReason sets Reason field to given value.

### HasReason

`func (o *AccessApproval) HasReason() bool`

HasReason returns a boolean if a field has been set.

### GetCreatedAt

`func (o *AccessApproval) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *AccessApproval) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *AccessApproval) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


