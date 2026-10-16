# quickdesk_signaling.AuthApi

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**create_session**](AuthApi.md#create_session) | **POST** /v1/auth/sessions | Exchange a username/password for a session


# **create_session**
> AuthSessionResponse create_session(auth_session_request)

Exchange a username/password for a session

### Example

* Api Key Authentication (ApiKeyAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.auth_session_request import AuthSessionRequest
from quickdesk_signaling.models.auth_session_response import AuthSessionResponse
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
    api_instance = quickdesk_signaling.AuthApi(api_client)
    auth_session_request = quickdesk_signaling.AuthSessionRequest() # AuthSessionRequest | 

    try:
        # Exchange a username/password for a session
        api_response = api_instance.create_session(auth_session_request)
        print("The response of AuthApi->create_session:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling AuthApi->create_session: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **auth_session_request** | [**AuthSessionRequest**](AuthSessionRequest.md)|  | 

### Return type

[**AuthSessionResponse**](AuthSessionResponse.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A session with an access token and a refresh token. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**429** | Too many requests; retry after the advertised delay. |  * Retry-After - Seconds to wait before retrying. <br>  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

