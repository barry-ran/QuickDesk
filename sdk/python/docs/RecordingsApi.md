# quickdesk_signaling.RecordingsApi

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**append_recording_artifact**](RecordingsApi.md#append_recording_artifact) | **PATCH** /v1/me/recordings/{recording_id}/artifact | Append one chunk of a resumable recording upload
[**create_recording_artifact_ticket**](RecordingsApi.md#create_recording_artifact_ticket) | **POST** /v1/me/recordings/{recording_id}/artifact-ticket | Mint a short-lived playback URL for a recording
[**create_recording_thumbnail_ticket**](RecordingsApi.md#create_recording_thumbnail_ticket) | **POST** /v1/me/recordings/{recording_id}/thumbnail-ticket | Mint a short-lived preview URL for a recording
[**create_session_recording**](RecordingsApi.md#create_session_recording) | **POST** /v1/me/recordings | Index a new recording
[**delete_session_recording**](RecordingsApi.md#delete_session_recording) | **DELETE** /v1/me/recordings/{recording_id} | Delete a recording and its stored media
[**download_recording_artifact**](RecordingsApi.md#download_recording_artifact) | **GET** /v1/me/recordings/{recording_id}/artifact | Download a recording&#39;s media bytes
[**download_recording_artifact_by_ticket**](RecordingsApi.md#download_recording_artifact_by_ticket) | **GET** /v1/recordings/{recording_id}/artifact | Fetch recording media with a playback ticket
[**download_recording_thumbnail**](RecordingsApi.md#download_recording_thumbnail) | **GET** /v1/me/recordings/{recording_id}/thumbnail | Download a recording&#39;s preview image
[**download_recording_thumbnail_by_ticket**](RecordingsApi.md#download_recording_thumbnail_by_ticket) | **GET** /v1/recordings/{recording_id}/thumbnail | Fetch a recording preview with a playback ticket
[**get_session_recording**](RecordingsApi.md#get_session_recording) | **GET** /v1/me/recordings/{recording_id} | One recording the caller owns
[**list_session_recordings**](RecordingsApi.md#list_session_recordings) | **GET** /v1/me/recordings | The caller&#39;s session-recording index
[**update_session_recording**](RecordingsApi.md#update_session_recording) | **PATCH** /v1/me/recordings/{recording_id} | Update a recording&#39;s lifecycle metadata
[**upload_recording_artifact**](RecordingsApi.md#upload_recording_artifact) | **POST** /v1/me/recordings/{recording_id}/artifact | Upload a recording&#39;s media bytes
[**upload_recording_thumbnail**](RecordingsApi.md#upload_recording_thumbnail) | **POST** /v1/me/recordings/{recording_id}/thumbnail | Upload a recording&#39;s preview image


# **append_recording_artifact**
> SessionRecording append_recording_artifact(ext, upload_offset, recording_id, upload_length=upload_length, upload_complete=upload_complete, upload_probe=upload_probe, body=body)

Append one chunk of a resumable recording upload

The resumable half of the upload contract, for a capture that does not
fit in one request body or a connection that drops halfway. The client
sends chunks; the server answers with the offset it actually holds in
the `Upload-Offset` response header, so a retry resumes from that number
instead of starting over.

Three headers drive it. `Upload-Offset` (required) is where the chunk
starts and must equal the bytes already staged; otherwise the answer is
`409` with the authoritative offset in the same header and the client
resyncs. `Upload-Length` (optional, `*` while unknown) declares the total
size, which lets the server finish by itself when the last byte lands and
lets it refuse a chunk that would overshoot. `Upload-Complete: 1` says
this chunk ends the upload, which is how a client finishes without ever
declaring a total.

A status probe asks where the upload stands without staging bytes. Send
`Upload-Probe: 1` with a valid `Upload-Offset` and no body; the answer is
`202` with the current offset, so a client that crashed can re-learn
where it stopped without a second endpoint. An empty body with a valid
offset does the same thing for raw-HTTP callers, but a generated client
cannot send a zero-length body, which is why the explicit header exists.

The staging file (`<artifact-path>.part`) is subject to the same size
limit as the single-shot upload, an empty artifact is refused on every
path, and the recording lifecycle deletes a staging file that has been
untouched for six hours.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.session_recording import SessionRecording
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    ext = 'ext_example' # str | The artifact extension, allow-listed exactly like the raw-body POST shape.
    upload_offset = 56 # int | The byte offset this chunk starts at.
    recording_id = 56 # int | The numeric id from a recording index row.
    upload_length = 'upload_length_example' # str | The total size of the artifact, or `*` while it is unknown. (optional)
    upload_complete = 'upload_complete_example' # str | Set to `1` when this chunk ends the upload. (optional)
    upload_probe = 'upload_probe_example' # str | Set to `1` to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body. (optional)
    body = None # bytearray |  (optional)

    try:
        # Append one chunk of a resumable recording upload
        api_response = api_instance.append_recording_artifact(ext, upload_offset, recording_id, upload_length=upload_length, upload_complete=upload_complete, upload_probe=upload_probe, body=body)
        print("The response of RecordingsApi->append_recording_artifact:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->append_recording_artifact: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ext** | **str**| The artifact extension, allow-listed exactly like the raw-body POST shape. | 
 **upload_offset** | **int**| The byte offset this chunk starts at. | 
 **recording_id** | **int**| The numeric id from a recording index row. | 
 **upload_length** | **str**| The total size of the artifact, or &#x60;*&#x60; while it is unknown. | [optional] 
 **upload_complete** | **str**| Set to &#x60;1&#x60; when this chunk ends the upload. | [optional] 
 **upload_probe** | **str**| Set to &#x60;1&#x60; to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body. | [optional] 
 **body** | **bytearray**|  | [optional] 

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/octet-stream
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The final chunk; the recording now carries the stored artifact. |  * Upload-Offset - Bytes staged, which for a finished upload is the artifact size. <br>  |
**202** | The chunk was staged and more are expected. The authoritative offset is the &#x60;Upload-Offset&#x60; response header. The body is the recording row - the same &#x60;SessionRecording&#x60; schema as the &#x60;200&#x60; - because a generated client declares one success type per operation and would otherwise fail to decode a staging-only body.  |  * Upload-Offset - Bytes staged, which for a finished upload is the artifact size. <br>  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |
**409** | &#x60;Upload-Offset&#x60; did not match the bytes already staged. The header carries the authoritative offset, so the client can resume from it.  |  * Upload-Offset - Bytes staged, which for a finished upload is the artifact size. <br>  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_recording_artifact_ticket**
> PlaybackTicket create_recording_artifact_ticket(recording_id)

Mint a short-lived playback URL for a recording

Owner-only. Returns a signed, time-limited `url` that can be fetched
**without** an `Authorization` header, which is what makes playback work
from `<video src>`. The ticket is bound to the recording, the owner and
its purpose, so an artifact ticket cannot be replayed against the
thumbnail route. Minting is stateless - nothing is written - and the
link stops working when `expires_at` passes. `400` when playback tickets
are disabled on the deployment (`RECORDING_URL_SECRET`/`API_KEY` unset);
`404` when the recording has no stored artifact.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.playback_ticket import PlaybackTicket
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    recording_id = 56 # int | The numeric id from a recording index row.

    try:
        # Mint a short-lived playback URL for a recording
        api_response = api_instance.create_recording_artifact_ticket(recording_id)
        print("The response of RecordingsApi->create_recording_artifact_ticket:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->create_recording_artifact_ticket: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recording_id** | **int**| The numeric id from a recording index row. | 

### Return type

[**PlaybackTicket**](PlaybackTicket.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A signed playback URL. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_recording_thumbnail_ticket**
> PlaybackTicket create_recording_thumbnail_ticket(recording_id)

Mint a short-lived preview URL for a recording

Same rules as the artifact ticket, but the URL addresses the thumbnail
route and the ticket's purpose is checked on redemption.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.playback_ticket import PlaybackTicket
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    recording_id = 56 # int | The numeric id from a recording index row.

    try:
        # Mint a short-lived preview URL for a recording
        api_response = api_instance.create_recording_thumbnail_ticket(recording_id)
        print("The response of RecordingsApi->create_recording_thumbnail_ticket:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->create_recording_thumbnail_ticket: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recording_id** | **int**| The numeric id from a recording index row. | 

### Return type

[**PlaybackTicket**](PlaybackTicket.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | A signed preview URL. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **create_session_recording**
> SessionRecording create_session_recording(session_recording_input)

Index a new recording

Called by the capturing side when a session starts (or when a finished
capture is handed over), so the owner sees the recording while it is
still being written and the row can carry the outcome afterwards.
`status` is one of `recording`, `completed` or `failed`; `started_at`
defaults to now. The media is uploaded separately to `.../artifact`.

The row belongs to the *caller*: `device_id` is recorded as a label and
is not checked for ownership, so a producer can index a capture for a
machine that is not bound to it. Only the caller can then read or delete
the row.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.session_recording import SessionRecording
from quickdesk_signaling.models.session_recording_input import SessionRecordingInput
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    session_recording_input = quickdesk_signaling.SessionRecordingInput() # SessionRecordingInput | 

    try:
        # Index a new recording
        api_response = api_instance.create_session_recording(session_recording_input)
        print("The response of RecordingsApi->create_session_recording:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->create_session_recording: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **session_recording_input** | [**SessionRecordingInput**](SessionRecordingInput.md)|  | 

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**201** | The created index row. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **delete_session_recording**
> delete_session_recording(recording_id)

Delete a recording and its stored media

Removes the index row and the media files that belong to it, so the
recording cannot be played back afterwards. There is no "already gone"
success: a second delete is a `404`. The deletion is published on the
owner's realtime stream and written to the audit trail as
`recording.delete` with the actor's observed address.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    recording_id = 56 # int | The numeric id from a recording index row.

    try:
        # Delete a recording and its stored media
        api_instance.delete_session_recording(recording_id)
    except Exception as e:
        print("Exception when calling RecordingsApi->delete_session_recording: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recording_id** | **int**| The numeric id from a recording index row. | 

### Return type

void (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**204** | Deleted. No body. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **download_recording_artifact**
> bytearray download_recording_artifact(recording_id)

Download a recording's media bytes

Owner-only, `Authorization` (Bearer) required. The response is the
stored file itself: the `Content-Type` follows the stored file's
extension (`video/mp4` for `.mp4`, `application/octet-stream`
otherwise) and `Content-Disposition` is `inline`. A browser cannot
attach an `Authorization` header to `<video src>`, which is why the
ticket routes below exist; this route is for API clients.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    recording_id = 56 # int | The numeric id from a recording index row.

    try:
        # Download a recording's media bytes
        api_response = api_instance.download_recording_artifact(recording_id)
        print("The response of RecordingsApi->download_recording_artifact:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->download_recording_artifact: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recording_id** | **int**| The numeric id from a recording index row. | 

### Return type

**bytearray**

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: video/mp4, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The media bytes. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **download_recording_artifact_by_ticket**
> bytearray download_recording_artifact_by_ticket(t, recording_id)

Fetch recording media with a playback ticket

The URL returned by `POST /v1/me/recordings/{recording_id}/artifact-ticket`.
Deliberately registered **outside** the API-key and user-auth groups,
because a browser `<video>` element can send neither header - the signed
`t` query parameter is the authorization, and the ticket is bound to
owner and purpose. `401` for a missing, malformed or expired ticket;
`404` when the ticket is for the wrong route or the recording is gone.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    t = 't_example' # str | The signed ticket from the artifact-ticket endpoint.
    recording_id = 56 # int | The numeric id from a recording index row.

    try:
        # Fetch recording media with a playback ticket
        api_response = api_instance.download_recording_artifact_by_ticket(t, recording_id)
        print("The response of RecordingsApi->download_recording_artifact_by_ticket:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->download_recording_artifact_by_ticket: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **t** | **str**| The signed ticket from the artifact-ticket endpoint. | 
 **recording_id** | **int**| The numeric id from a recording index row. | 

### Return type

**bytearray**

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: video/mp4, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The media bytes. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **download_recording_thumbnail**
> bytearray download_recording_thumbnail(recording_id)

Download a recording's preview image

Owner-only. Served with `Cache-Control: private, max-age=300` because a
preview is user-scoped and can change under the same URL.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    recording_id = 56 # int | The numeric id from a recording index row.

    try:
        # Download a recording's preview image
        api_response = api_instance.download_recording_thumbnail(recording_id)
        print("The response of RecordingsApi->download_recording_thumbnail:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->download_recording_thumbnail: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recording_id** | **int**| The numeric id from a recording index row. | 

### Return type

**bytearray**

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: image/jpeg, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The preview image bytes. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **download_recording_thumbnail_by_ticket**
> bytearray download_recording_thumbnail_by_ticket(t, recording_id)

Fetch a recording preview with a playback ticket

The URL returned by `POST /v1/me/recordings/{recording_id}/thumbnail-ticket`.
Same ticket rules as the artifact route; an artifact ticket is rejected
here (`404`) because the purpose is part of the signature.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    t = 't_example' # str | The signed ticket from the thumbnail-ticket endpoint.
    recording_id = 56 # int | The numeric id from a recording index row.

    try:
        # Fetch a recording preview with a playback ticket
        api_response = api_instance.download_recording_thumbnail_by_ticket(t, recording_id)
        print("The response of RecordingsApi->download_recording_thumbnail_by_ticket:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->download_recording_thumbnail_by_ticket: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **t** | **str**| The signed ticket from the thumbnail-ticket endpoint. | 
 **recording_id** | **int**| The numeric id from a recording index row. | 

### Return type

**bytearray**

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: image/jpeg, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The preview image bytes. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **get_session_recording**
> SessionRecording get_session_recording(recording_id)

One recording the caller owns

The same row the list returns. Another user's id is a `404` rather than
a `403`, so the route cannot be used to discover which recording ids
exist.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.session_recording import SessionRecording
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    recording_id = 56 # int | The numeric id from a recording index row.

    try:
        # One recording the caller owns
        api_response = api_instance.get_session_recording(recording_id)
        print("The response of RecordingsApi->get_session_recording:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->get_session_recording: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recording_id** | **int**| The numeric id from a recording index row. | 

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The recording. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **list_session_recordings**
> SessionRecordingList list_session_recordings(cursor=cursor, limit=limit)

The caller's session-recording index

Cursor-paginated, newest first. `next_cursor` is opaque: pass it back
as `?cursor=` to fetch the next page and stop when it is absent. The
envelope here has no `total` because this endpoint does not count the
whole set on every page, so an integrator must page until
`next_cursor` is empty rather than trusting a count. This is the
metadata index only; the media bytes live behind the per-recording
`.../artifact` routes documented in `docs/user-api-docs.md`.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.session_recording_list import SessionRecordingList
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    cursor = 'cursor_example' # str |  (optional)
    limit = 56 # int |  (optional)

    try:
        # The caller's session-recording index
        api_response = api_instance.list_session_recordings(cursor=cursor, limit=limit)
        print("The response of RecordingsApi->list_session_recordings:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->list_session_recordings: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **cursor** | **str**|  | [optional] 
 **limit** | **int**|  | [optional] 

### Return type

[**SessionRecordingList**](SessionRecordingList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | One page of the caller&#39;s recordings, newest first. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **update_session_recording**
> SessionRecording update_session_recording(recording_id, session_recording_patch)

Update a recording's lifecycle metadata

Partial update: only the fields present in the body are written, so a
producer can patch `status` when a capture starts and then
`size_bytes`/`ended_at` when the upload finishes. `status` accepts the
same three values as create; anything else is a `400` and writes
nothing.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.session_recording import SessionRecording
from quickdesk_signaling.models.session_recording_patch import SessionRecordingPatch
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    recording_id = 56 # int | The numeric id from a recording index row.
    session_recording_patch = quickdesk_signaling.SessionRecordingPatch() # SessionRecordingPatch | 

    try:
        # Update a recording's lifecycle metadata
        api_response = api_instance.update_session_recording(recording_id, session_recording_patch)
        print("The response of RecordingsApi->update_session_recording:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->update_session_recording: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recording_id** | **int**| The numeric id from a recording index row. | 
 **session_recording_patch** | [**SessionRecordingPatch**](SessionRecordingPatch.md)|  | 

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The updated recording. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **upload_recording_artifact**
> SessionRecording upload_recording_artifact(recording_id, file, ext=ext)

Upload a recording's media bytes

Two request shapes, because the producers differ: `multipart/form-data`
with a `file` part (the browser and Qt clients) or the raw body with
`?ext=`. The extension is allow-listed (`.mp4`, `.m4v`, `.mkv`, `.webm`,
`.h264`, `.grf`), the bytes are written to a temporary file and renamed
into place so a dropped upload can never leave a half-written file under
the real name, and the row is then updated with `artifact_path`,
`artifact_url`, `format` and `size_bytes` in one step.

An empty body is a `400`; a body over the configured limit
(`RECORDING_MAX_BYTES`, exposed as `store.max_bytes`) is a `400` as well,
never a truncated file. Uploading again replaces the stored artifact.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.session_recording import SessionRecording
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    recording_id = 56 # int | The numeric id from a recording index row.
    file = None # bytearray | 
    ext = 'ext_example' # str | Required for the raw-body shape; ignored for multipart. (optional)

    try:
        # Upload a recording's media bytes
        api_response = api_instance.upload_recording_artifact(recording_id, file, ext=ext)
        print("The response of RecordingsApi->upload_recording_artifact:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->upload_recording_artifact: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recording_id** | **int**| The numeric id from a recording index row. | 
 **file** | **bytearray**|  | 
 **ext** | **str**| Required for the raw-body shape; ignored for multipart. | [optional] 

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: multipart/form-data, application/octet-stream
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The recording, now carrying the stored artifact. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **upload_recording_thumbnail**
> SessionRecording upload_recording_thumbnail(recording_id, file, ext=ext)

Upload a recording's preview image

The same two shapes as the artifact upload (`multipart/form-data` with a
`file` part, or the raw body with `?ext=`), but the extension is limited
to `.jpg`, `.jpeg` and `.png` and the size to 2 MiB - a preview is a
small still, and an oversized one means the wrong file was sent.

The bytes are sniffed before they are stored rather than trusted from
the name or the declared content type: the index renders this image
inline, so a file that only claims to be a JPEG is a `400`.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.session_recording import SessionRecording
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
    api_instance = quickdesk_signaling.RecordingsApi(api_client)
    recording_id = 56 # int | The numeric id from a recording index row.
    file = None # bytearray | 
    ext = 'ext_example' # str | Required for the raw-body shape; ignored for multipart. (optional)

    try:
        # Upload a recording's preview image
        api_response = api_instance.upload_recording_thumbnail(recording_id, file, ext=ext)
        print("The response of RecordingsApi->upload_recording_thumbnail:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling RecordingsApi->upload_recording_thumbnail: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **recording_id** | **int**| The numeric id from a recording index row. | 
 **file** | **bytearray**|  | 
 **ext** | **str**| Required for the raw-body shape; ignored for multipart. | [optional] 

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: multipart/form-data, application/octet-stream
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The recording, now carrying the preview. |  -  |
**400** | The request was malformed or failed validation. |  -  |
**401** | Missing or expired credentials. |  -  |
**404** | The addressed resource does not exist. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

