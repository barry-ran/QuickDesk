# quickdesk_signaling.UpdateApi

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**check_for_update**](UpdateApi.md#check_for_update) | **GET** /v1/update:check | Check the self-hosted update channel for a newer build


# **check_for_update**
> UpdateCheckResult check_for_update(platform, arch, channel=channel, current_version=current_version)

Check the self-hosted update channel for a newer build

The client-facing half of the self-hosted update channel. It is API-key
gated but not user-authenticated: a controlled machine checks for
updates before anyone logs in.

The server owns the index of published releases; the artifact itself
lives at `download_url` on whatever host the operator controls. The
index is managed through the admin API (`POST /v1/admin/update-releases`
and friends), which stays outside this business contract.

`current_version` is optional. Without it the newest release is returned
and `update_available` is true whenever one exists; with it,
`update_available` is true only when the published version is strictly
newer, and `mandatory` mirrors the release flag once an update is
available. `latest` is always present and is `null` when the channel has
no release for that platform/arch.


### Example

* Api Key Authentication (ApiKeyAuth):
* Bearer (opaque) Authentication (BearerAuth):

```python
import quickdesk_signaling
from quickdesk_signaling.models.update_check_result import UpdateCheckResult
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
    api_instance = quickdesk_signaling.UpdateApi(api_client)
    platform = 'platform_example' # str | Which operating system the caller is running.
    arch = 'arch_example' # str | Which architecture the caller is running.
    channel = 'channel_example' # str | Defaults to `stable`. (optional)
    current_version = 'current_version_example' # str | The dotted numeric version the caller is running, e.g. `4.2.5.0`. A leading `v` is accepted. (optional)

    try:
        # Check the self-hosted update channel for a newer build
        api_response = api_instance.check_for_update(platform, arch, channel=channel, current_version=current_version)
        print("The response of UpdateApi->check_for_update:\n")
        pprint(api_response)
    except Exception as e:
        print("Exception when calling UpdateApi->check_for_update: %s\n" % e)
```



### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **platform** | **str**| Which operating system the caller is running. | 
 **arch** | **str**| Which architecture the caller is running. | 
 **channel** | **str**| Defaults to &#x60;stable&#x60;. | [optional] 
 **current_version** | **str**| The dotted numeric version the caller is running, e.g. &#x60;4.2.5.0&#x60;. A leading &#x60;v&#x60; is accepted. | [optional] 

### Return type

[**UpdateCheckResult**](UpdateCheckResult.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details

| Status code | Description | Response headers |
|-------------|-------------|------------------|
**200** | The newest release for that channel/platform/arch, or a null &#x60;latest&#x60; when the channel is empty. |  -  |
**400** | The request was malformed or failed validation. |  -  |

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

