# quickdesk_signaling.DevicesApi

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**approve_access_approval**](DevicesApi.md#approve_access_approval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/approve | Approve a pending access request
[**claim_access_approval**](DevicesApi.md#claim_access_approval) | **POST** /v1/devices/{device_id}/access-approvals:claim | Exchange an approved request for a signal token
[**create_access_grant**](DevicesApi.md#create_access_grant) | **POST** /v1/me/devices/{device_id}/access-grants | Issue a time-boxed guest access code
[**deny_access_approval**](DevicesApi.md#deny_access_approval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/deny | Deny a pending access request
[**get_access_approval_summary**](DevicesApi.md#get_access_approval_summary) | **GET** /v1/me/devices/{device_id}/access-approvals:summary | Aggregated view of a device&#39;s approval queue
[**get_access_policy**](DevicesApi.md#get_access_policy) | **GET** /v1/me/devices/{device_id}/access-policy | The device&#39;s access policy
[**list_access_approvals**](DevicesApi.md#list_access_approvals) | **GET** /v1/me/devices/{device_id}/access-approvals | Second-factor approvals awaiting (or already decided by) the owner
[**list_access_grants**](DevicesApi.md#list_access_grants) | **GET** /v1/me/devices/{device_id}/access-grants | Guest access codes issued for a device
[**list_my_devices**](DevicesApi.md#list_my_devices) | **GET** /v1/me/devices | Devices owned by the authenticated user
[**put_access_policy**](DevicesApi.md#put_access_policy) | **PUT** /v1/me/devices/{device_id}/access-policy | Create or replace the device&#39;s access policy
[**revoke_access_grant**](DevicesApi.md#revoke_access_grant) | **DELETE** /v1/me/devices/{device_id}/access-grants/{grant_id} | Revoke a guest access code
[**verify_access_code**](DevicesApi.md#verify_access_code) | **POST** /v1/devices/{device_id}/access-code:verify | Verify a device access code and mint a signal token


# **approve_access_approval**
> AccessApproval approve_access_approval(device_id, approval_id, approve_access_approval_request=approve_access_approval_request)

Approve a pending access request

Owner-only. The body is optional; a `reason` is stored on the row and
shown to the requester. A request that was already decided or has
expired returns `409` with the current status instead of pretending the
decision landed. The conformance smoke pins the `404` shape with an
unknown id; the happy path needs a real pending approval (an online
device plus `approval_mode`), so it is covered by the end-to-end run in
`docs/dev/自托管端到端联调记录.md` rather than the smoke.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.access_approval import AccessApproval
from quickdesk_signaling.models.approve_access_approval_request import ApproveAccessApprovalRequest
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.
    approval_id = 56 # int | The numeric id from an approval row.
    approve_access_approval_request = quickdesk_signaling.ApproveAccessApprovalRequest() # ApproveAccessApprovalRequest |  (optional)

    try:
        # Approve a pending access request
        api_response = api_instance.approve_access_approval(device_id, approval_id, approve_access_approval_request=approve_access_approval_request)
        print("The response of DevicesApi->approve_access_approval:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->approve_access_approval: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 
 **approval_id** | **int**| The numeric id from an approval row. | 
 **approve_access_approval_request** | [**ApproveAccessApprovalRequest**](ApproveAccessApprovalRequest.md)|  | [optional] 

### Return type

[**AccessApproval**](AccessApproval.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The approval after the decision. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |
**409** | The request was already decided or has expired. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **claim_access_approval**
> AccessCodeVerifyResponse claim_access_approval(device_id, approval_claim_request)

Exchange an approved request for a signal token

The requesting controller calls this with the `approval_id` and
`approval_secret` it received from `access-code:verify`. It is API-key
gated rather than user-authenticated because the caller has no account.
`202` means the owner has not decided yet - keep polling with the same
id and secret; `200` returns the signal token; `403` means denied or
expired; `409` means the approval was already spent or the host went
offline before the exchange.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.access_code_verify_response import AccessCodeVerifyResponse
from quickdesk_signaling.models.approval_claim_request import ApprovalClaimRequest
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.
    approval_claim_request = quickdesk_signaling.ApprovalClaimRequest() # ApprovalClaimRequest | 

    try:
        # Exchange an approved request for a signal token
        api_response = api_instance.claim_access_approval(device_id, approval_claim_request)
        print("The response of DevicesApi->claim_access_approval:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->claim_access_approval: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 
 **approval_claim_request** | [**ApprovalClaimRequest**](ApprovalClaimRequest.md)|  | 

### Return type

[**AccessCodeVerifyResponse**](AccessCodeVerifyResponse.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The owner approved and the signal token is ready. |  -  |
**202** | Still waiting for the owner to decide. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |
**409** | The approval was already spent, or the host went offline before the exchange could happen.  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_access_grant**
> AccessGrant create_access_grant(device_id, access_grant_input)

Issue a time-boxed guest access code

The plaintext code is returned **once**, in the creation response; a
grant is limited by expiry and uses, may be bound to a client id and/or
source addresses, and is revoked with the DELETE below.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.access_grant import AccessGrant
from quickdesk_signaling.models.access_grant_input import AccessGrantInput
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.
    access_grant_input = quickdesk_signaling.AccessGrantInput() # AccessGrantInput | 

    try:
        # Issue a time-boxed guest access code
        api_response = api_instance.create_access_grant(device_id, access_grant_input)
        print("The response of DevicesApi->create_access_grant:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->create_access_grant: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 
 **access_grant_input** | [**AccessGrantInput**](AccessGrantInput.md)|  | 

### Return type

[**AccessGrant**](AccessGrant.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | The created grant, including its one-time plaintext code. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **deny_access_approval**
> AccessApproval deny_access_approval(device_id, approval_id, approve_access_approval_request=approve_access_approval_request)

Deny a pending access request

Owner-only, same shape as `approve`. Denying also arms the per-source
denial cooldown (`APPROVAL_DENIAL_COOLDOWN_SECONDS`), so the same client
cannot immediately re-request.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.access_approval import AccessApproval
from quickdesk_signaling.models.approve_access_approval_request import ApproveAccessApprovalRequest
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.
    approval_id = 56 # int | The numeric id from an approval row.
    approve_access_approval_request = quickdesk_signaling.ApproveAccessApprovalRequest() # ApproveAccessApprovalRequest |  (optional)

    try:
        # Deny a pending access request
        api_response = api_instance.deny_access_approval(device_id, approval_id, approve_access_approval_request=approve_access_approval_request)
        print("The response of DevicesApi->deny_access_approval:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->deny_access_approval: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 
 **approval_id** | **int**| The numeric id from an approval row. | 
 **approve_access_approval_request** | [**ApproveAccessApprovalRequest**](ApproveAccessApprovalRequest.md)|  | [optional] 

### Return type

[**AccessApproval**](AccessApproval.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The approval after the decision. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |
**409** | The request was already decided or has expired. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_access_approval_summary**
> ApprovalSummary get_access_approval_summary(device_id, since=since, limit=limit)

Aggregated view of a device's approval queue

The row list shows requests one at a time; this is what makes "one
address has been refused six times this afternoon" legible. The window
is keyed on `created_at` for every count.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.approval_summary import ApprovalSummary
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.
    since = '2013-10-20T19:20:30+01:00' # datetime | RFC3339 start of the window; defaults to the last 24 hours. (optional)
    limit = 56 # int | How many refused sources to list (default 5, capped at 10). (optional)

    try:
        # Aggregated view of a device's approval queue
        api_response = api_instance.get_access_approval_summary(device_id, since=since, limit=limit)
        print("The response of DevicesApi->get_access_approval_summary:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->get_access_approval_summary: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 
 **since** | **datetime**| RFC3339 start of the window; defaults to the last 24 hours. | [optional] 
 **limit** | **int**| How many refused sources to list (default 5, capped at 10). | [optional] 

### Return type

[**ApprovalSummary**](ApprovalSummary.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The window&#39;s counts and refused sources. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_access_policy**
> DeviceAccessPolicy get_access_policy(device_id)

The device's access policy

Returns the documented "no policy" defaults when the owner never saved
one, so a client can render the form without a second existence check.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.device_access_policy import DeviceAccessPolicy
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.

    try:
        # The device's access policy
        api_response = api_instance.get_access_policy(device_id)
        print("The response of DevicesApi->get_access_policy:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->get_access_policy: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The policy, or its defaults. |  -  |
**401** | Missing or expired credentials. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_access_approvals**
> ListAccessApprovals200Response list_access_approvals(device_id, status=status, limit=limit)

Second-factor approvals awaiting (or already decided by) the owner

`status` is a comma-separated subset of `pending, approved, denied,
expired, consumed` and defaults to `pending`. Approval is created by the
device-side verify route when the policy asks for it; approving or
denying happens on the sub-resource routes documented in
`docs/user-api-docs.md` §5.3.5 (not yet part of this contract).


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.list_access_approvals200_response import ListAccessApprovals200Response
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.
    status = 'status_example' # str |  (optional)
    limit = 56 # int |  (optional)

    try:
        # Second-factor approvals awaiting (or already decided by) the owner
        api_response = api_instance.list_access_approvals(device_id, status=status, limit=limit)
        print("The response of DevicesApi->list_access_approvals:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->list_access_approvals: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 
 **status** | **str**|  | [optional] 
 **limit** | **int**|  | [optional] 

### Return type

[**ListAccessApprovals200Response**](ListAccessApprovals200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The matching approvals, newest first. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_access_grants**
> ListAccessGrants200Response list_access_grants(device_id)

Guest access codes issued for a device

### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.list_access_grants200_response import ListAccessGrants200Response
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.

    try:
        # Guest access codes issued for a device
        api_response = api_instance.list_access_grants(device_id)
        print("The response of DevicesApi->list_access_grants:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->list_access_grants: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 

### Return type

[**ListAccessGrants200Response**](ListAccessGrants200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The device&#39;s grants, newest first. |  -  |
**401** | Missing or expired credentials. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_my_devices**
> ListMyDevices200Response list_my_devices()

Devices owned by the authenticated user

### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.list_my_devices200_response import ListMyDevices200Response
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)

    try:
        # Devices owned by the authenticated user
        api_response = api_instance.list_my_devices()
        print("The response of DevicesApi->list_my_devices:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->list_my_devices: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**ListMyDevices200Response**](ListMyDevices200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The caller&#39;s devices. |  -  |
**401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **put_access_policy**
> DeviceAccessPolicy put_access_policy(device_id, device_access_policy_input)

Create or replace the device's access policy

A **full replace**: every field not sent is reset to its zero value. The
policy is enforced on the device-side verify route before the access
code is compared, so a blocked client never gets to guess codes.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.device_access_policy import DeviceAccessPolicy
from quickdesk_signaling.models.device_access_policy_input import DeviceAccessPolicyInput
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.
    device_access_policy_input = quickdesk_signaling.DeviceAccessPolicyInput() # DeviceAccessPolicyInput | 

    try:
        # Create or replace the device's access policy
        api_response = api_instance.put_access_policy(device_id, device_access_policy_input)
        print("The response of DevicesApi->put_access_policy:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->put_access_policy: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 
 **device_access_policy_input** | [**DeviceAccessPolicyInput**](DeviceAccessPolicyInput.md)|  | 

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The stored policy. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **revoke_access_grant**
> RevokeAccessGrant200Response revoke_access_grant(device_id, grant_id)

Revoke a guest access code

### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.revoke_access_grant200_response import RevokeAccessGrant200Response
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Configure Bearer authorization (opaque): BearerAuth
configuration = quickdesk_signaling.Configuration(
    access_token = os.environ["BEARER_TOKEN"]
)

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.
    grant_id = 56 # int | 

    try:
        # Revoke a guest access code
        api_response = api_instance.revoke_access_grant(device_id, grant_id)
        print("The response of DevicesApi->revoke_access_grant:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->revoke_access_grant: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 
 **grant_id** | **int**|  | 

### Return type

[**RevokeAccessGrant200Response**](RevokeAccessGrant200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The grant is no longer redeemable. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **verify_access_code**
> AccessCodeVerifyResponse verify_access_code(device_id, access_code_verify_request)

Verify a device access code and mint a signal token

The controller-side entry point. On success it returns a signal token
for the relay. When the device policy wants the owner to confirm the
session it returns `202` with a pending approval instead, and the caller
polls `/v1/devices/{device_id}/access-approvals:claim`. A rejected code
is a `403` problem with `code: INVALID_CODE`.

This route is reachable without a user token (the controlling client is
not logged in) but still requires the API key or an allowed Origin.


### Example

* Api Key Authentication (ApiKeyAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.access_code_verify_request import AccessCodeVerifyRequest
from quickdesk_signaling.models.access_code_verify_response import AccessCodeVerifyResponse
from quickdesk_signaling.rest import ApiException
from pprint import pprint

# Defining the host is optional and defaults to https://your-signaling-host
# See configuration.py for a list of all supported configuration parameters.
configuration = quickdesk_signaling.Configuration(
    host = "https://your-signaling-host"
)

# The client must configure the authentication and authorization parameters
# in accordance with the API server security policy.
# Examples for each auth method are provided below, use the example that
# satisfies your auth use case.

# Configure API key authorization: ApiKeyAuth
configuration.api_key['ApiKeyAuth'] = os.environ["API_KEY"]

# Uncomment below to setup prefix (e.g. Bearer) for API key, if needed
# configuration.api_key_prefix['ApiKeyAuth'] = 'Bearer'

# Enter a context with an instance of the API client
with quickdesk_signaling.ApiClient(configuration) as api_client:
    # Create an instance of the API class
    api_instance = quickdesk_signaling.DevicesApi(api_client)
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.
    access_code_verify_request = quickdesk_signaling.AccessCodeVerifyRequest() # AccessCodeVerifyRequest | 

    try:
        # Verify a device access code and mint a signal token
        api_response = api_instance.verify_access_code(device_id, access_code_verify_request)
        print("The response of DevicesApi->verify_access_code:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling DevicesApi->verify_access_code: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 
 **access_code_verify_request** | [**AccessCodeVerifyRequest**](AccessCodeVerifyRequest.md)|  | 

### Return type

[**AccessCodeVerifyResponse**](AccessCodeVerifyResponse.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The code was accepted and a signal token was minted. |  -  |
**202** | The device policy requires the owner to approve this session. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |
**429** | Too many requests; retry after the advertised delay. |  * Retry-After - Seconds to wait before retrying. <br>  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

