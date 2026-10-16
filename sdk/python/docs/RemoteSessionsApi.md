# quickdesk_signaling.RemoteSessionsApi

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**get_remote_session**](RemoteSessionsApi.md#get_remote_session) | **GET** /v1/me/remote-sessions/{session_id} | One control session with the records that share its id
[**list_device_remote_sessions**](RemoteSessionsApi.md#list_device_remote_sessions) | **GET** /v1/me/devices/{device_id}/sessions | One device&#39;s control sessions, each with its related records
[**list_remote_sessions**](RemoteSessionsApi.md#list_remote_sessions) | **GET** /v1/me/remote-sessions | The caller&#39;s control sessions, open and finished
[**revoke_remote_session**](RemoteSessionsApi.md#revoke_remote_session) | **POST** /v1/me/remote-sessions/{session_id}/revoke | End one of the caller&#39;s control sessions


# **get_remote_session**
> RemoteSessionDetail get_remote_session(session_id)

One control session with the records that share its id

What happened in one session: the session row itself, the recordings
produced during it, and the device access-log entries that carry its
`session_id` (in practice the `session_started` row written when the
controller authenticated). The three tables always existed - this is the
read that puts them back together.

Owner only. A session belonging to another account is reported as `404`
rather than `403`, so a caller cannot probe which session ids exist.
Both lists are capped at 100 entries and are empty arrays rather than
absent when there is nothing to show.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.remote_session_detail import RemoteSessionDetail
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
    session_id = 'session_id_example' # str | The signaling session id from the list.

    try:
        # One control session with the records that share its id
        api_response = api_instance.get_remote_session(session_id)
        print("The response of RemoteSessionsApi->get_remote_session:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RemoteSessionsApi->get_remote_session: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **session_id** | **str**| The signaling session id from the list. | 

### Return type

[**RemoteSessionDetail**](RemoteSessionDetail.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The session and its related records. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_device_remote_sessions**
> RemoteSessionDetailList list_device_remote_sessions(device_id, status=status, cursor=cursor, limit=limit)

One device's control sessions, each with its related records

The same history as `GET /v1/me/remote-sessions?device_id=`, but every
entry arrives already joined with the records that share its session id -
the shape a device page needs, so a client never has to know a session id
(or make a second request per row) to reach the join.

`status` is `active`, `ended` or omitted/`all`; `cursor`/`limit` page it
newest first like every other list here. The related arrays are batched
across the page, so a page costs three queries rather than three per row,
and they are present-but-empty rather than absent.

A device the caller does not own is `403` and an unknown device is `404`,
matching `.../access-logs`. (The session detail read answers `404` for
somebody else's session instead, because a session id is looked up rather
than named by its owner.)


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.remote_session_detail_list import RemoteSessionDetailList
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
    device_id = 'device_id_example' # str | The device id shown on the controlled machine.
    status = 'status_example' # str | Defaults to `all`. (optional)
    cursor = 'cursor_example' # str |  (optional)
    limit = 56 # int |  (optional)

    try:
        # One device's control sessions, each with its related records
        api_response = api_instance.list_device_remote_sessions(device_id, status=status, cursor=cursor, limit=limit)
        print("The response of RemoteSessionsApi->list_device_remote_sessions:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RemoteSessionsApi->list_device_remote_sessions: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **device_id** | **str**| The device id shown on the controlled machine. | 
 **status** | **str**| Defaults to &#x60;all&#x60;. | [optional] 
 **cursor** | **str**|  | [optional] 
 **limit** | **int**|  | [optional] 

### Return type

[**RemoteSessionDetailList**](RemoteSessionDetailList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | One page of the device&#39;s control sessions, newest first. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**403** | Authenticated but not allowed (includes a rejected access code). |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

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

# **revoke_remote_session**
> RemoteSessionRevocation revoke_remote_session(session_id)

End one of the caller's control sessions

The owner's "throw them out". A remote session lives exactly as long as
the controller's signaling socket, so ending the session means closing
that socket; the session row is stamped `end_reason: revoked` and the
owner's realtime stream carries `remote_session.ended` with that reason.
Closing the socket also frees the device's controller slot, so a revoked
session stops counting against the access policy's `max_sessions`.

Owner only, with the same concealment as the detail read: a session that
belongs to another account is a `404`, not a `403`. The socket may be
held by a different server instance - the row records which one admitted
it - so the request is addressed at that instance and this call waits for
the session to actually close: a `200` means it happened, not that a
command was merely published. A session that is already over, or whose
owning instance did not close it in time, is a `409`, because the one
thing worse than refusing a revoke is reporting a success that did not
happen.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.remote_session_revocation import RemoteSessionRevocation
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
    session_id = 'session_id_example' # str | The signaling session id from the list.

    try:
        # End one of the caller's control sessions
        api_response = api_instance.revoke_remote_session(session_id)
        print("The response of RemoteSessionsApi->revoke_remote_session:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RemoteSessionsApi->revoke_remote_session: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **session_id** | **str**| The signaling session id from the list. | 

### Return type

[**RemoteSessionRevocation**](RemoteSessionRevocation.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The session was live here and its socket has been closed. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |
**409** | The session exists but cannot be revoked from here - it is already over, or it is live on another instance.  |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

