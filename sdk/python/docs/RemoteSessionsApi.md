# quickdesk_signaling.RemoteSessionsApi

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**list_remote_sessions**](RemoteSessionsApi.md#list_remote_sessions) | **GET** /v1/me/remote-sessions | The caller&#39;s control sessions, open and finished


# **list_remote_sessions**
> RemoteSessionList list_remote_sessions(status=status, device_id=device_id, cursor=cursor, limit=limit)

The caller's control sessions, open and finished

Who is in the caller's machines right now, and who has been.

A remote session is one controller holding a signaling session against a
device. It is opened when that controller's signaling socket completes
authentication and closed when that socket ends, which is the whole
lifetime this server can observe: the media path is peer to peer, so a
session whose signaling channel is gone is over as far as the control
plane is concerned.

`status=active` answers "who is in right now", `ended` is history, and
the default lists both. `device_id` narrows to one machine.
Cursor-paginated, newest first, `next_cursor` absent on the last page.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.remote_session_list import RemoteSessionList
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
    api_instance = quickdesk_signaling.RemoteSessionsApi(api_client)
    status = 'status_example' # str | Defaults to `all`. (optional)
    device_id = 'device_id_example' # str |  (optional)
    cursor = 'cursor_example' # str |  (optional)
    limit = 56 # int |  (optional)

    try:
        # The caller's control sessions, open and finished
        api_response = api_instance.list_remote_sessions(status=status, device_id=device_id, cursor=cursor, limit=limit)
        print("The response of RemoteSessionsApi->list_remote_sessions:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RemoteSessionsApi->list_remote_sessions: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **status** | **str**| Defaults to &#x60;all&#x60;. | [optional] 
 **device_id** | **str**|  | [optional] 
 **cursor** | **str**|  | [optional] 
 **limit** | **int**|  | [optional] 

### Return type

[**RemoteSessionList**](RemoteSessionList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | One page of the caller&#39;s control sessions, newest first. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

