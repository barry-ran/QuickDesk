# AccessGrant

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | **int64** |  | 
**DeviceId** | **string** |  | 
**Code** | **string** | Returned only in the creation response; treat it as a secret. | 
**Label** | Pointer to **string** |  | [optional] 
**ExpiresAt** | **time.Time** |  | 
**MaxUses** | **int32** |  | 
**Uses** | **int32** |  | 
**Active** | **bool** |  | 
**RevokedAt** | Pointer to **NullableTime** |  | [optional] 
**LastUsedAt** | Pointer to **NullableTime** |  | [optional] 
**BoundClientId** | **string** |  | 
**BoundIps** | **[]string** |  | 
**LastUsedClientId** | Pointer to **string** |  | [optional] 
**LastUsedIp** | Pointer to **string** |  | [optional] 
**CreatedAt** | **time.Time** |  | 

## Methods

### NewAccessGrant

`func NewAccessGrant(id int64, deviceId string, code string, expiresAt time.Time, maxUses int32, uses int32, active bool, boundClientId string, boundIps []string, createdAt time.Time, ) *AccessGrant`

NewAccessGrant instantiates a new AccessGrant object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAccessGrantWithDefaults

`func NewAccessGrantWithDefaults() *AccessGrant`

NewAccessGrantWithDefaults instantiates a new AccessGrant object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *AccessGrant) GetId() int64`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *AccessGrant) GetIdOk() (*int64, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *AccessGrant) SetId(v int64)`

SetId sets Id field to given value.


### GetDeviceId

`func (o *AccessGrant) GetDeviceId() string`

GetDeviceId returns the DeviceId field if non-nil, zero value otherwise.

### GetDeviceIdOk

`func (o *AccessGrant) GetDeviceIdOk() (*string, bool)`

GetDeviceIdOk returns a tuple with the DeviceId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDeviceId

`func (o *AccessGrant) SetDeviceId(v string)`

SetDeviceId sets DeviceId field to given value.


### GetCode

`func (o *AccessGrant) GetCode() string`

GetCode returns the Code field if non-nil, zero value otherwise.

### GetCodeOk

`func (o *AccessGrant) GetCodeOk() (*string, bool)`

GetCodeOk returns a tuple with the Code field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCode

`func (o *AccessGrant) SetCode(v string)`

SetCode sets Code field to given value.


### GetLabel

`func (o *AccessGrant) GetLabel() string`

GetLabel returns the Label field if non-nil, zero value otherwise.

### GetLabelOk

`func (o *AccessGrant) GetLabelOk() (*string, bool)`

GetLabelOk returns a tuple with the Label field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabel

`func (o *AccessGrant) SetLabel(v string)`

SetLabel sets Label field to given value.

### HasLabel

`func (o *AccessGrant) HasLabel() bool`

HasLabel returns a boolean if a field has been set.

### GetExpiresAt

`func (o *AccessGrant) GetExpiresAt() time.Time`

GetExpiresAt returns the ExpiresAt field if non-nil, zero value otherwise.

### GetExpiresAtOk

`func (o *AccessGrant) GetExpiresAtOk() (*time.Time, bool)`

GetExpiresAtOk returns a tuple with the ExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpiresAt

`func (o *AccessGrant) SetExpiresAt(v time.Time)`

SetExpiresAt sets ExpiresAt field to given value.


### GetMaxUses

`func (o *AccessGrant) GetMaxUses() int32`

GetMaxUses returns the MaxUses field if non-nil, zero value otherwise.

### GetMaxUsesOk

`func (o *AccessGrant) GetMaxUsesOk() (*int32, bool)`

GetMaxUsesOk returns a tuple with the MaxUses field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMaxUses

`func (o *AccessGrant) SetMaxUses(v int32)`

SetMaxUses sets MaxUses field to given value.


### GetUses

`func (o *AccessGrant) GetUses() int32`

GetUses returns the Uses field if non-nil, zero value otherwise.

### GetUsesOk

`func (o *AccessGrant) GetUsesOk() (*int32, bool)`

GetUsesOk returns a tuple with the Uses field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUses

`func (o *AccessGrant) SetUses(v int32)`

SetUses sets Uses field to given value.


### GetActive

`func (o *AccessGrant) GetActive() bool`

GetActive returns the Active field if non-nil, zero value otherwise.

### GetActiveOk

`func (o *AccessGrant) GetActiveOk() (*bool, bool)`

GetActiveOk returns a tuple with the Active field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetActive

`func (o *AccessGrant) SetActive(v bool)`

SetActive sets Active field to given value.


### GetRevokedAt

`func (o *AccessGrant) GetRevokedAt() time.Time`

GetRevokedAt returns the RevokedAt field if non-nil, zero value otherwise.

### GetRevokedAtOk

`func (o *AccessGrant) GetRevokedAtOk() (*time.Time, bool)`

GetRevokedAtOk returns a tuple with the RevokedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRevokedAt

`func (o *AccessGrant) SetRevokedAt(v time.Time)`

SetRevokedAt sets RevokedAt field to given value.

### HasRevokedAt

`func (o *AccessGrant) HasRevokedAt() bool`

HasRevokedAt returns a boolean if a field has been set.

### SetRevokedAtNil

`func (o *AccessGrant) SetRevokedAtNil(b bool)`

 SetRevokedAtNil sets the value for RevokedAt to be an explicit nil

### UnsetRevokedAt
`func (o *AccessGrant) UnsetRevokedAt()`

UnsetRevokedAt ensures that no value is present for RevokedAt, not even an explicit nil
### GetLastUsedAt

`func (o *AccessGrant) GetLastUsedAt() time.Time`

GetLastUsedAt returns the LastUsedAt field if non-nil, zero value otherwise.

### GetLastUsedAtOk

`func (o *AccessGrant) GetLastUsedAtOk() (*time.Time, bool)`

GetLastUsedAtOk returns a tuple with the LastUsedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastUsedAt

`func (o *AccessGrant) SetLastUsedAt(v time.Time)`

SetLastUsedAt sets LastUsedAt field to given value.

### HasLastUsedAt

`func (o *AccessGrant) HasLastUsedAt() bool`

HasLastUsedAt returns a boolean if a field has been set.

### SetLastUsedAtNil

`func (o *AccessGrant) SetLastUsedAtNil(b bool)`

 SetLastUsedAtNil sets the value for LastUsedAt to be an explicit nil

### UnsetLastUsedAt
`func (o *AccessGrant) UnsetLastUsedAt()`

UnsetLastUsedAt ensures that no value is present for LastUsedAt, not even an explicit nil
### GetBoundClientId

`func (o *AccessGrant) GetBoundClientId() string`

GetBoundClientId returns the BoundClientId field if non-nil, zero value otherwise.

### GetBoundClientIdOk

`func (o *AccessGrant) GetBoundClientIdOk() (*string, bool)`

GetBoundClientIdOk returns a tuple with the BoundClientId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBoundClientId

`func (o *AccessGrant) SetBoundClientId(v string)`

SetBoundClientId sets BoundClientId field to given value.


### GetBoundIps

`func (o *AccessGrant) GetBoundIps() []string`

GetBoundIps returns the BoundIps field if non-nil, zero value otherwise.

### GetBoundIpsOk

`func (o *AccessGrant) GetBoundIpsOk() (*[]string, bool)`

GetBoundIpsOk returns a tuple with the BoundIps field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBoundIps

`func (o *AccessGrant) SetBoundIps(v []string)`

SetBoundIps sets BoundIps field to given value.


### GetLastUsedClientId

`func (o *AccessGrant) GetLastUsedClientId() string`

GetLastUsedClientId returns the LastUsedClientId field if non-nil, zero value otherwise.

### GetLastUsedClientIdOk

`func (o *AccessGrant) GetLastUsedClientIdOk() (*string, bool)`

GetLastUsedClientIdOk returns a tuple with the LastUsedClientId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastUsedClientId

`func (o *AccessGrant) SetLastUsedClientId(v string)`

SetLastUsedClientId sets LastUsedClientId field to given value.

### HasLastUsedClientId

`func (o *AccessGrant) HasLastUsedClientId() bool`

HasLastUsedClientId returns a boolean if a field has been set.

### GetLastUsedIp

`func (o *AccessGrant) GetLastUsedIp() string`

GetLastUsedIp returns the LastUsedIp field if non-nil, zero value otherwise.

### GetLastUsedIpOk

`func (o *AccessGrant) GetLastUsedIpOk() (*string, bool)`

GetLastUsedIpOk returns a tuple with the LastUsedIp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastUsedIp

`func (o *AccessGrant) SetLastUsedIp(v string)`

SetLastUsedIp sets LastUsedIp field to given value.

### HasLastUsedIp

`func (o *AccessGrant) HasLastUsedIp() bool`

HasLastUsedIp returns a boolean if a field has been set.

### GetCreatedAt

`func (o *AccessGrant) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *AccessGrant) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *AccessGrant) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


