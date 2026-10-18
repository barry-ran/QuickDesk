# RecordingsApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**appendRecordingArtifact**](RecordingsApi.md#appendRecordingArtifact) | **PATCH** /v1/me/recordings/{recording_id}/artifact | Append one chunk of a resumable recording upload |
| [**createRecordingArtifactTicket**](RecordingsApi.md#createRecordingArtifactTicket) | **POST** /v1/me/recordings/{recording_id}/artifact-ticket | Mint a short-lived playback URL for a recording |
| [**createRecordingThumbnailTicket**](RecordingsApi.md#createRecordingThumbnailTicket) | **POST** /v1/me/recordings/{recording_id}/thumbnail-ticket | Mint a short-lived preview URL for a recording |
| [**createSessionRecording**](RecordingsApi.md#createSessionRecording) | **POST** /v1/me/recordings | Index a new recording |
| [**deleteSessionRecording**](RecordingsApi.md#deleteSessionRecording) | **DELETE** /v1/me/recordings/{recording_id} | Delete a recording and its stored media |
| [**downloadRecordingArtifact**](RecordingsApi.md#downloadRecordingArtifact) | **GET** /v1/me/recordings/{recording_id}/artifact | Download a recording&#39;s media bytes |
| [**downloadRecordingArtifactByTicket**](RecordingsApi.md#downloadRecordingArtifactByTicket) | **GET** /v1/recordings/{recording_id}/artifact | Fetch recording media with a playback ticket |
| [**downloadRecordingThumbnail**](RecordingsApi.md#downloadRecordingThumbnail) | **GET** /v1/me/recordings/{recording_id}/thumbnail | Download a recording&#39;s preview image |
| [**downloadRecordingThumbnailByTicket**](RecordingsApi.md#downloadRecordingThumbnailByTicket) | **GET** /v1/recordings/{recording_id}/thumbnail | Fetch a recording preview with a playback ticket |
| [**getSessionRecording**](RecordingsApi.md#getSessionRecording) | **GET** /v1/me/recordings/{recording_id} | One recording the caller owns |
| [**listSessionRecordings**](RecordingsApi.md#listSessionRecordings) | **GET** /v1/me/recordings | The caller&#39;s session-recording index |
| [**updateSessionRecording**](RecordingsApi.md#updateSessionRecording) | **PATCH** /v1/me/recordings/{recording_id} | Update a recording&#39;s lifecycle metadata |
| [**uploadRecordingArtifact**](RecordingsApi.md#uploadRecordingArtifact) | **POST** /v1/me/recordings/{recording_id}/artifact | Upload a recording&#39;s media bytes |
| [**uploadRecordingThumbnail**](RecordingsApi.md#uploadRecordingThumbnail) | **POST** /v1/me/recordings/{recording_id}/thumbnail | Upload a recording&#39;s preview image |


<a id="appendRecordingArtifact"></a>
# **appendRecordingArtifact**
> SessionRecording appendRecordingArtifact(ext, uploadOffset, recordingId, uploadLength, uploadComplete, uploadProbe, body)

Append one chunk of a resumable recording upload

The resumable half of the upload contract, for a capture that does not fit in one request body or a connection that drops halfway. The client sends chunks; the server answers with the offset it actually holds in the &#x60;Upload-Offset&#x60; response header, so a retry resumes from that number instead of starting over.  Three headers drive it. &#x60;Upload-Offset&#x60; (required) is where the chunk starts and must equal the bytes already staged; otherwise the answer is &#x60;409&#x60; with the authoritative offset in the same header and the client resyncs. &#x60;Upload-Length&#x60; (optional, &#x60;*&#x60; while unknown) declares the total size, which lets the server finish by itself when the last byte lands and lets it refuse a chunk that would overshoot. &#x60;Upload-Complete: 1&#x60; says this chunk ends the upload, which is how a client finishes without ever declaring a total.  A status probe asks where the upload stands without staging bytes. Send &#x60;Upload-Probe: 1&#x60; with a valid &#x60;Upload-Offset&#x60; and no body; the answer is &#x60;202&#x60; with the current offset, so a client that crashed can re-learn where it stopped without a second endpoint. An empty body with a valid offset does the same thing for raw-HTTP callers, but a generated client cannot send a zero-length body, which is why the explicit header exists.  The staging file (&#x60;&lt;artifact-path&gt;.part&#x60;) is subject to the same size limit as the single-shot upload, an empty artifact is refused on every path, and the recording lifecycle deletes a staging file that has been untouched for six hours. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    String ext = ".mp4"; // String | The artifact extension, allow-listed exactly like the raw-body POST shape.
    Long uploadOffset = 56L; // Long | The byte offset this chunk starts at.
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    String uploadLength = "uploadLength_example"; // String | The total size of the artifact, or `*` while it is unknown.
    String uploadComplete = "0"; // String | Set to `1` when this chunk ends the upload.
    String uploadProbe = "0"; // String | Set to `1` to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body.
    File body = new File("/path/to/file"); // File | 
    try {
      SessionRecording result = apiInstance.appendRecordingArtifact(ext, uploadOffset, recordingId, uploadLength, uploadComplete, uploadProbe, body);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#appendRecordingArtifact");
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
| **ext** | **String**| The artifact extension, allow-listed exactly like the raw-body POST shape. | [enum: .mp4, .m4v, .mkv, .webm, .h264, .grf] |
| **uploadOffset** | **Long**| The byte offset this chunk starts at. | |
| **recordingId** | **Long**| The numeric id from a recording index row. | |
| **uploadLength** | **String**| The total size of the artifact, or &#x60;*&#x60; while it is unknown. | [optional] |
| **uploadComplete** | **String**| Set to &#x60;1&#x60; when this chunk ends the upload. | [optional] [enum: 0, 1] |
| **uploadProbe** | **String**| Set to &#x60;1&#x60; to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body. | [optional] [enum: 0, 1] |
| **body** | **File**|  | [optional] |

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
| **200** | The final chunk; the recording now carries the stored artifact. |  * Upload-Offset - Bytes staged, which for a finished upload is the artifact size. <br>  |
| **202** | The chunk was staged and more are expected. The authoritative offset is the &#x60;Upload-Offset&#x60; response header. The body is the recording row - the same &#x60;SessionRecording&#x60; schema as the &#x60;200&#x60; - because a generated client declares one success type per operation and would otherwise fail to decode a staging-only body.  |  * Upload-Offset - Bytes staged, which for a finished upload is the artifact size. <br>  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **409** | &#x60;Upload-Offset&#x60; did not match the bytes already staged. The header carries the authoritative offset, so the client can resume from it.  |  * Upload-Offset - Bytes staged, which for a finished upload is the artifact size. <br>  |

<a id="createRecordingArtifactTicket"></a>
# **createRecordingArtifactTicket**
> PlaybackTicket createRecordingArtifactTicket(recordingId)

Mint a short-lived playback URL for a recording

Owner-only. Returns a signed, time-limited &#x60;url&#x60; that can be fetched **without** an &#x60;Authorization&#x60; header, which is what makes playback work from &#x60;&lt;video src&gt;&#x60;. The ticket is bound to the recording, the owner and its purpose, so an artifact ticket cannot be replayed against the thumbnail route. Minting is stateless - nothing is written - and the link stops working when &#x60;expires_at&#x60; passes. &#x60;400&#x60; when playback tickets are disabled on the deployment (&#x60;RECORDING_URL_SECRET&#x60;/&#x60;API_KEY&#x60; unset); &#x60;404&#x60; when the recording has no stored artifact. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    try {
      PlaybackTicket result = apiInstance.createRecordingArtifactTicket(recordingId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#createRecordingArtifactTicket");
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
| **recordingId** | **Long**| The numeric id from a recording index row. | |

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
| **200** | A signed playback URL. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="createRecordingThumbnailTicket"></a>
# **createRecordingThumbnailTicket**
> PlaybackTicket createRecordingThumbnailTicket(recordingId)

Mint a short-lived preview URL for a recording

Same rules as the artifact ticket, but the URL addresses the thumbnail route and the ticket&#39;s purpose is checked on redemption. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    try {
      PlaybackTicket result = apiInstance.createRecordingThumbnailTicket(recordingId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#createRecordingThumbnailTicket");
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
| **recordingId** | **Long**| The numeric id from a recording index row. | |

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
| **200** | A signed preview URL. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="createSessionRecording"></a>
# **createSessionRecording**
> SessionRecording createSessionRecording(sessionRecordingInput)

Index a new recording

Called by the capturing side when a session starts (or when a finished capture is handed over), so the owner sees the recording while it is still being written and the row can carry the outcome afterwards. &#x60;status&#x60; is one of &#x60;recording&#x60;, &#x60;completed&#x60; or &#x60;failed&#x60;; &#x60;started_at&#x60; defaults to now. The media is uploaded separately to &#x60;.../artifact&#x60;.  The row belongs to the *caller*: &#x60;device_id&#x60; is recorded as a label and is not checked for ownership, so a producer can index a capture for a machine that is not bound to it. Only the caller can then read or delete the row. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    SessionRecordingInput sessionRecordingInput = new SessionRecordingInput(); // SessionRecordingInput | 
    try {
      SessionRecording result = apiInstance.createSessionRecording(sessionRecordingInput);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#createSessionRecording");
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
| **sessionRecordingInput** | [**SessionRecordingInput**](SessionRecordingInput.md)|  | |

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
| **201** | The created index row. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |

<a id="deleteSessionRecording"></a>
# **deleteSessionRecording**
> deleteSessionRecording(recordingId)

Delete a recording and its stored media

Removes the index row and the media files that belong to it, so the recording cannot be played back afterwards. There is no \&quot;already gone\&quot; success: a second delete is a &#x60;404&#x60;. The deletion is published on the owner&#39;s realtime stream and written to the audit trail as &#x60;recording.delete&#x60; with the actor&#39;s observed address. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    try {
      apiInstance.deleteSessionRecording(recordingId);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#deleteSessionRecording");
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
| **recordingId** | **Long**| The numeric id from a recording index row. | |

### Return type

null (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | Deleted. No body. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="downloadRecordingArtifact"></a>
# **downloadRecordingArtifact**
> File downloadRecordingArtifact(recordingId)

Download a recording&#39;s media bytes

Owner-only, &#x60;Authorization&#x60; (Bearer) required. The response is the stored file itself: the &#x60;Content-Type&#x60; follows the stored file&#39;s extension (&#x60;video/mp4&#x60; for &#x60;.mp4&#x60;, &#x60;application/octet-stream&#x60; otherwise) and &#x60;Content-Disposition&#x60; is &#x60;inline&#x60;. A browser cannot attach an &#x60;Authorization&#x60; header to &#x60;&lt;video src&gt;&#x60;, which is why the ticket routes below exist; this route is for API clients. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    try {
      File result = apiInstance.downloadRecordingArtifact(recordingId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#downloadRecordingArtifact");
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
| **recordingId** | **Long**| The numeric id from a recording index row. | |

### Return type

[**File**](File.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: video/mp4, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The media bytes. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="downloadRecordingArtifactByTicket"></a>
# **downloadRecordingArtifactByTicket**
> File downloadRecordingArtifactByTicket(t, recordingId)

Fetch recording media with a playback ticket

The URL returned by &#x60;POST /v1/me/recordings/{recording_id}/artifact-ticket&#x60;. Deliberately registered **outside** the API-key and user-auth groups, because a browser &#x60;&lt;video&gt;&#x60; element can send neither header - the signed &#x60;t&#x60; query parameter is the authorization, and the ticket is bound to owner and purpose. &#x60;401&#x60; for a missing, malformed or expired ticket; &#x60;404&#x60; when the ticket is for the wrong route or the recording is gone. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    String t = "t_example"; // String | The signed ticket from the artifact-ticket endpoint.
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    try {
      File result = apiInstance.downloadRecordingArtifactByTicket(t, recordingId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#downloadRecordingArtifactByTicket");
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
| **t** | **String**| The signed ticket from the artifact-ticket endpoint. | |
| **recordingId** | **Long**| The numeric id from a recording index row. | |

### Return type

[**File**](File.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: video/mp4, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The media bytes. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="downloadRecordingThumbnail"></a>
# **downloadRecordingThumbnail**
> File downloadRecordingThumbnail(recordingId)

Download a recording&#39;s preview image

Owner-only. Served with &#x60;Cache-Control: private, max-age&#x3D;300&#x60; because a preview is user-scoped and can change under the same URL. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    try {
      File result = apiInstance.downloadRecordingThumbnail(recordingId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#downloadRecordingThumbnail");
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
| **recordingId** | **Long**| The numeric id from a recording index row. | |

### Return type

[**File**](File.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: image/jpeg, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The preview image bytes. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="downloadRecordingThumbnailByTicket"></a>
# **downloadRecordingThumbnailByTicket**
> File downloadRecordingThumbnailByTicket(t, recordingId)

Fetch a recording preview with a playback ticket

The URL returned by &#x60;POST /v1/me/recordings/{recording_id}/thumbnail-ticket&#x60;. Same ticket rules as the artifact route; an artifact ticket is rejected here (&#x60;404&#x60;) because the purpose is part of the signature. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    String t = "t_example"; // String | The signed ticket from the thumbnail-ticket endpoint.
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    try {
      File result = apiInstance.downloadRecordingThumbnailByTicket(t, recordingId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#downloadRecordingThumbnailByTicket");
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
| **t** | **String**| The signed ticket from the thumbnail-ticket endpoint. | |
| **recordingId** | **Long**| The numeric id from a recording index row. | |

### Return type

[**File**](File.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: image/jpeg, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The preview image bytes. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="getSessionRecording"></a>
# **getSessionRecording**
> SessionRecording getSessionRecording(recordingId)

One recording the caller owns

The same row the list returns. Another user&#39;s id is a &#x60;404&#x60; rather than a &#x60;403&#x60;, so the route cannot be used to discover which recording ids exist. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    try {
      SessionRecording result = apiInstance.getSessionRecording(recordingId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#getSessionRecording");
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
| **recordingId** | **Long**| The numeric id from a recording index row. | |

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
| **200** | The recording. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="listSessionRecordings"></a>
# **listSessionRecordings**
> SessionRecordingList listSessionRecordings(cursor, limit)

The caller&#39;s session-recording index

Cursor-paginated, newest first. &#x60;next_cursor&#x60; is opaque: pass it back as &#x60;?cursor&#x3D;&#x60; to fetch the next page and stop when it is absent. The envelope here has no &#x60;total&#x60; because this endpoint does not count the whole set on every page, so an integrator must page until &#x60;next_cursor&#x60; is empty rather than trusting a count. This is the metadata index only; the media bytes live behind the per-recording &#x60;.../artifact&#x60; routes documented in &#x60;docs/user-api-docs.md&#x60;. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    String cursor = "cursor_example"; // String | 
    Integer limit = 56; // Integer | 
    try {
      SessionRecordingList result = apiInstance.listSessionRecordings(cursor, limit);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#listSessionRecordings");
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
| **cursor** | **String**|  | [optional] |
| **limit** | **Integer**|  | [optional] |

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
| **200** | One page of the caller&#39;s recordings, newest first. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |

<a id="updateSessionRecording"></a>
# **updateSessionRecording**
> SessionRecording updateSessionRecording(recordingId, sessionRecordingPatch)

Update a recording&#39;s lifecycle metadata

Partial update: only the fields present in the body are written, so a producer can patch &#x60;status&#x60; when a capture starts and then &#x60;size_bytes&#x60;/&#x60;ended_at&#x60; when the upload finishes. &#x60;status&#x60; accepts the same three values as create; anything else is a &#x60;400&#x60; and writes nothing. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    SessionRecordingPatch sessionRecordingPatch = new SessionRecordingPatch(); // SessionRecordingPatch | 
    try {
      SessionRecording result = apiInstance.updateSessionRecording(recordingId, sessionRecordingPatch);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#updateSessionRecording");
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
| **recordingId** | **Long**| The numeric id from a recording index row. | |
| **sessionRecordingPatch** | [**SessionRecordingPatch**](SessionRecordingPatch.md)|  | |

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
| **200** | The updated recording. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="uploadRecordingArtifact"></a>
# **uploadRecordingArtifact**
> SessionRecording uploadRecordingArtifact(recordingId, _file, ext)

Upload a recording&#39;s media bytes

Two request shapes, because the producers differ: &#x60;multipart/form-data&#x60; with a &#x60;file&#x60; part (the browser and Qt clients) or the raw body with &#x60;?ext&#x3D;&#x60;. The extension is allow-listed (&#x60;.mp4&#x60;, &#x60;.m4v&#x60;, &#x60;.mkv&#x60;, &#x60;.webm&#x60;, &#x60;.h264&#x60;, &#x60;.grf&#x60;), the bytes are written to a temporary file and renamed into place so a dropped upload can never leave a half-written file under the real name, and the row is then updated with &#x60;artifact_path&#x60;, &#x60;artifact_url&#x60;, &#x60;format&#x60; and &#x60;size_bytes&#x60; in one step.  An empty body is a &#x60;400&#x60;; a body over the configured limit (&#x60;RECORDING_MAX_BYTES&#x60;, exposed as &#x60;store.max_bytes&#x60;) is a &#x60;400&#x60; as well, never a truncated file. Uploading again replaces the stored artifact. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    File _file = new File("/path/to/file"); // File | 
    String ext = ".mp4"; // String | Required for the raw-body shape; ignored for multipart.
    try {
      SessionRecording result = apiInstance.uploadRecordingArtifact(recordingId, _file, ext);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#uploadRecordingArtifact");
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
| **recordingId** | **Long**| The numeric id from a recording index row. | |
| **_file** | **File**|  | |
| **ext** | **String**| Required for the raw-body shape; ignored for multipart. | [optional] [enum: .mp4, .m4v, .mkv, .webm, .h264, .grf] |

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
| **200** | The recording, now carrying the stored artifact. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="uploadRecordingThumbnail"></a>
# **uploadRecordingThumbnail**
> SessionRecording uploadRecordingThumbnail(recordingId, _file, ext)

Upload a recording&#39;s preview image

The same two shapes as the artifact upload (&#x60;multipart/form-data&#x60; with a &#x60;file&#x60; part, or the raw body with &#x60;?ext&#x3D;&#x60;), but the extension is limited to &#x60;.jpg&#x60;, &#x60;.jpeg&#x60; and &#x60;.png&#x60; and the size to 2 MiB - a preview is a small still, and an oversized one means the wrong file was sent.  The bytes are sniffed before they are stored rather than trusted from the name or the declared content type: the index renders this image inline, so a file that only claims to be a JPEG is a &#x60;400&#x60;. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.RecordingsApi;

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

    RecordingsApi apiInstance = new RecordingsApi(defaultClient);
    Long recordingId = 56L; // Long | The numeric id from a recording index row.
    File _file = new File("/path/to/file"); // File | 
    String ext = ".jpg"; // String | Required for the raw-body shape; ignored for multipart.
    try {
      SessionRecording result = apiInstance.uploadRecordingThumbnail(recordingId, _file, ext);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling RecordingsApi#uploadRecordingThumbnail");
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
| **recordingId** | **Long**| The numeric id from a recording index row. | |
| **_file** | **File**|  | |
| **ext** | **String**| Required for the raw-body shape; ignored for multipart. | [optional] [enum: .jpg, .jpeg, .png] |

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
| **200** | The recording, now carrying the preview. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **404** | The addressed resource does not exist. |  -  |

