# AuthSessionResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**User** | [**User**](User.md) |  | 
**AccessToken** | **string** |  | 
**AccessExpiresAt** | **time.Time** |  | 
**RefreshToken** | **string** |  | 
**RefreshExpiresAt** | **time.Time** |  | 

## Methods

### NewAuthSessionResponse

`func NewAuthSessionResponse(user User, accessToken string, accessExpiresAt time.Time, refreshToken string, refreshExpiresAt time.Time, ) *AuthSessionResponse`

NewAuthSessionResponse instantiates a new AuthSessionResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAuthSessionResponseWithDefaults

`func NewAuthSessionResponseWithDefaults() *AuthSessionResponse`

NewAuthSessionResponseWithDefaults instantiates a new AuthSessionResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetUser

`func (o *AuthSessionResponse) GetUser() User`

GetUser returns the User field if non-nil, zero value otherwise.

### GetUserOk

`func (o *AuthSessionResponse) GetUserOk() (*User, bool)`

GetUserOk returns a tuple with the User field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUser

`func (o *AuthSessionResponse) SetUser(v User)`

SetUser sets User field to given value.


### GetAccessToken

`func (o *AuthSessionResponse) GetAccessToken() string`

GetAccessToken returns the AccessToken field if non-nil, zero value otherwise.

### GetAccessTokenOk

`func (o *AuthSessionResponse) GetAccessTokenOk() (*string, bool)`

GetAccessTokenOk returns a tuple with the AccessToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccessToken

`func (o *AuthSessionResponse) SetAccessToken(v string)`

SetAccessToken sets AccessToken field to given value.


### GetAccessExpiresAt

`func (o *AuthSessionResponse) GetAccessExpiresAt() time.Time`

GetAccessExpiresAt returns the AccessExpiresAt field if non-nil, zero value otherwise.

### GetAccessExpiresAtOk

`func (o *AuthSessionResponse) GetAccessExpiresAtOk() (*time.Time, bool)`

GetAccessExpiresAtOk returns a tuple with the AccessExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccessExpiresAt

`func (o *AuthSessionResponse) SetAccessExpiresAt(v time.Time)`

SetAccessExpiresAt sets AccessExpiresAt field to given value.


### GetRefreshToken

`func (o *AuthSessionResponse) GetRefreshToken() string`

GetRefreshToken returns the RefreshToken field if non-nil, zero value otherwise.

### GetRefreshTokenOk

`func (o *AuthSessionResponse) GetRefreshTokenOk() (*string, bool)`

GetRefreshTokenOk returns a tuple with the RefreshToken field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRefreshToken

`func (o *AuthSessionResponse) SetRefreshToken(v string)`

SetRefreshToken sets RefreshToken field to given value.


### GetRefreshExpiresAt

`func (o *AuthSessionResponse) GetRefreshExpiresAt() time.Time`

GetRefreshExpiresAt returns the RefreshExpiresAt field if non-nil, zero value otherwise.

### GetRefreshExpiresAtOk

`func (o *AuthSessionResponse) GetRefreshExpiresAtOk() (*time.Time, bool)`

GetRefreshExpiresAtOk returns a tuple with the RefreshExpiresAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRefreshExpiresAt

`func (o *AuthSessionResponse) SetRefreshExpiresAt(v time.Time)`

SetRefreshExpiresAt sets RefreshExpiresAt field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


