# RemoteSessionsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**getRemoteSession**](RemoteSessionsApi.md#getRemoteSession) | **GET** /v1/me/remote-sessions/{session_id} | One control session with the records that share its id |
| [**listDeviceRemoteSessions**](RemoteSessionsApi.md#listDeviceRemoteSessions) | **GET** /v1/me/devices/{device_id}/sessions | One device&#39;s control sessions, each with its related records |
| [**listRemoteSessions**](RemoteSessionsApi.md#listRemoteSessions) | **GET** /v1/me/remote-sessions | The caller&#39;s control sessions, open and finished |
| [**revokeRemoteSession**](RemoteSessionsApi.md#revokeRemoteSession) | **POST** /v1/me/remote-sessions/{session_id}/revoke | End one of the caller&#39;s control sessions |


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

<a id="listDeviceRemoteSessions"></a>
# **listDeviceRemoteSessions**
> RemoteSessionDetailList listDeviceRemoteSessions(deviceId, status, cursor, limit)

One device&#39;s control sessions, each with its related records

The same history as &#x60;GET /v1/me/remote-sessions?device_id&#x3D;&#x60;, but every entry arrives already joined with the records that share its session id - the shape a device page needs, so a client never has to know a session id (or make a second request per row) to reach the join.  &#x60;status&#x60; is &#x60;active&#x60;, &#x60;ended&#x60; or omitted/&#x60;all&#x60;; &#x60;cursor&#x60;/&#x60;limit&#x60; page it newest first like every other list here. The related arrays are batched across the page, so a page costs three queries rather than three per row, and they are present-but-empty rather than absent.  A device the caller does not own is &#x60;403&#x60; and an unknown device is &#x60;404&#x60;, matching &#x60;.../access-logs&#x60;. (The session detail read answers &#x60;404&#x60; for somebody else&#39;s session instead, because a session id is looked up rather than named by its owner.) 

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
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    String status = "active"; // String | Defaults to `all`.
    String cursor = "cursor_example"; // String | 
    Integer limit = 56; // Integer | 
    try {
      RemoteSessionDetailList result = apiInstance.listDeviceRemoteSessions(deviceId, status, cursor, limit);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RemoteSessionsApi#listDeviceRemoteSessions");
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
| **deviceId** | **String**| The device id shown on the controlled machine. | |
| **status** | **String**| Defaults to &#x60;all&#x60;. | [optional] [enum: active, ended, all] |
| **cursor** | **String**|  | [optional] |
| **limit** | **Integer**|  | [optional] |

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
| **200** | One page of the device&#39;s control sessions, newest first. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
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

<a id="revokeRemoteSession"></a>
# **revokeRemoteSession**
> RemoteSessionRevocation revokeRemoteSession(sessionId)

End one of the caller&#39;s control sessions

The owner&#39;s \&quot;throw them out\&quot;. A remote session lives exactly as long as the controller&#39;s signaling socket, so ending the session means closing that socket; the session row is stamped &#x60;end_reason: revoked&#x60; and the owner&#39;s realtime stream carries &#x60;remote_session.ended&#x60; with that reason. Closing the socket also frees the device&#39;s controller slot, so a revoked session stops counting against the access policy&#39;s &#x60;max_sessions&#x60;.  Owner only, with the same concealment as the detail read: a session that belongs to another account is a &#x60;404&#x60;, not a &#x60;403&#x60;. The socket may be held by a different server instance - the row records which one admitted it - so the request is addressed at that instance and this call waits for the session to actually close: a &#x60;200&#x60; means it happened, not that a command was merely published. A session that is already over, or whose owning instance did not close it in time, is a &#x60;409&#x60;, because the one thing worse than refusing a revoke is reporting a success that did not happen. 

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
      RemoteSessionRevocation result = apiInstance.revokeRemoteSession(sessionId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RemoteSessionsApi#revokeRemoteSession");
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

[**RemoteSessionRevocation**](RemoteSessionRevocation.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The session was live here and its socket has been closed. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **409** | The session exists but cannot be revoked from here - it is already over, or it is live on another instance.  |  -  |

