# RemoteSessionsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**getRemoteSession**](RemoteSessionsApi.md#getRemoteSession) | **GET** /v1/me/remote-sessions/{session_id} | One control session with the records that share its id |
| [**listRemoteSessions**](RemoteSessionsApi.md#listRemoteSessions) | **GET** /v1/me/remote-sessions | The caller&#39;s control sessions, open and finished |


<a id="getRemoteSession"></a>
# **getRemoteSession**
> RemoteSessionDetail getRemoteSession(sessionId)

One control session with the records that share its id

What happened in one session: the session row itself, the recordings produced during it, and the device access-log entries that carry its &#x60;session_id&#x60; (in practice the &#x60;session_started&#x60; row written when the controller authenticated). The three tables always existed - this is the read that puts them back together.  Owner only. A session belonging to another account is reported as &#x60;404&#x60; rather than &#x60;403&#x60;, so a caller cannot probe which session ids exist. Both lists are capped at 100 entries and are empty arrays rather than absent when there is nothing to show. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RemoteSessionsApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("https://your-signaling-host");
    
    // Configure API key authorization: ApiKeyAuth
    ApiKeyAuth ApiKeyAuth = (ApiKeyAuth) defaultClient.getAuthentication("ApiKeyAuth");
    ApiKeyAuth.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //ApiKeyAuth.setApiKeyPrefix("Token");

    // Configure HTTP bearer authorization: BearerAuth
    HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
    BearerAuth.setBearerToken("BEARER TOKEN");

    RemoteSessionsApi apiInstance = new RemoteSessionsApi(defaultClient);
    String sessionId = "sessionId_example"; // String | The signaling session id from the list.
    try {
      RemoteSessionDetail result = apiInstance.getRemoteSession(sessionId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RemoteSessionsApi#getRemoteSession");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **sessionId** | **String**| The signaling session id from the list. | |

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
| **200** | The session and its related records. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="listRemoteSessions"></a>
# **listRemoteSessions**
> RemoteSessionList listRemoteSessions(status, deviceId, cursor, limit)

The caller&#39;s control sessions, open and finished

Who is in the caller&#39;s machines right now, and who has been.  A remote session is one controller holding a signaling session against a device. It is opened when that controller&#39;s signaling socket completes authentication and closed when that socket ends, which is the whole lifetime this server can observe: the media path is peer to peer, so a session whose signaling channel is gone is over as far as the control plane is concerned.  &#x60;status&#x3D;active&#x60; answers \&quot;who is in right now\&quot;, &#x60;ended&#x60; is history, and the default lists both. &#x60;device_id&#x60; narrows to one machine. Cursor-paginated, newest first, &#x60;next_cursor&#x60; absent on the last page. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RemoteSessionsApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("https://your-signaling-host");
    
    // Configure API key authorization: ApiKeyAuth
    ApiKeyAuth ApiKeyAuth = (ApiKeyAuth) defaultClient.getAuthentication("ApiKeyAuth");
    ApiKeyAuth.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //ApiKeyAuth.setApiKeyPrefix("Token");

    // Configure HTTP bearer authorization: BearerAuth
    HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
    BearerAuth.setBearerToken("BEARER TOKEN");

    RemoteSessionsApi apiInstance = new RemoteSessionsApi(defaultClient);
    String status = "active"; // String | Defaults to `all`.
    String deviceId = "deviceId_example"; // String | 
    String cursor = "cursor_example"; // String | 
    Integer limit = 56; // Integer | 
    try {
      RemoteSessionList result = apiInstance.listRemoteSessions(status, deviceId, cursor, limit);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RemoteSessionsApi#listRemoteSessions");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters

| Name | Type | Description  | Notes |
|------------- | ------------- | ------------- | -------------|
| **status** | **String**| Defaults to &#x60;all&#x60;. | [optional] [enum: active, ended, all] |
| **deviceId** | **String**|  | [optional] |
| **cursor** | **String**|  | [optional] |
| **limit** | **Integer**|  | [optional] |

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
| **200** | One page of the caller&#39;s control sessions, newest first. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |

