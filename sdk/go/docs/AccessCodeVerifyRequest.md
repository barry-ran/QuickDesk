# AccessCodeVerifyRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Code** | **string** |  | 
**ClientId** | Pointer to **string** | Optional self-reported client identity, recorded on the access log. | [optional] 

## Methods

### NewAccessCodeVerifyRequest

`func NewAccessCodeVerifyRequest(code string, ) *AccessCodeVerifyRequest`

NewAccessCodeVerifyRequest instantiates a new AccessCodeVerifyRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAccessCodeVerifyRequestWithDefaults

`func NewAccessCodeVerifyRequestWithDefaults() *AccessCodeVerifyRequest`

NewAccessCodeVerifyRequestWithDefaults instantiates a new AccessCodeVerifyRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCode

`func (o *AccessCodeVerifyRequest) GetCode() string`

GetCode returns the Code field if non-nil, zero value otherwise.

### GetCodeOk

`func (o *AccessCodeVerifyRequest) GetCodeOk() (*string, bool)`

GetCodeOk returns a tuple with the Code field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCode

`func (o *AccessCodeVerifyRequest) SetCode(v string)`

SetCode sets Code field to given value.


### GetClientId

`func (o *AccessCodeVerifyRequest) GetClientId() string`

GetClientId returns the ClientId field if non-nil, zero value otherwise.

### GetClientIdOk

`func (o *AccessCodeVerifyRequest) GetClientIdOk() (*string, bool)`

GetClientIdOk returns a tuple with the ClientId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClientId

`func (o *AccessCodeVerifyRequest) SetClientId(v string)`

SetClientId sets ClientId field to given value.

### HasClientId

`func (o *AccessCodeVerifyRequest) HasClientId() bool`

HasClientId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


