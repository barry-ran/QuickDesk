# quickdesk_signaling.MeApi

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_me**](MeApi.md#get_me) | **GET** /v1/me | The authenticated user


# **get_me**
> User get_me()

The authenticated user

### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.user import User
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
    api_instance = quickdesk_signaling.MeApi(api_client)

    try:
        # The authenticated user
        api_response = api_instance.get_me()
        print("The response of MeApi->get_me:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling MeApi->get_me: %s\n" % e)
```



### Parameters

This endpoint does not need any parameter.

### Return type

[**User**](User.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The current user. |  -  |
**401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

