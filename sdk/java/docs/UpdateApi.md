# UpdateApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**checkForUpdate**](UpdateApi.md#checkForUpdate) | **GET** /v1/update:check | Check the self-hosted update channel for a newer build |
| [**downloadUpdateArtifact**](UpdateApi.md#downloadUpdateArtifact) | **GET** /v1/update-releases/{release_id}/artifact | Download an installer this server hosts itself |


<a id="checkForUpdate"></a>
# **checkForUpdate**
> UpdateCheckResult checkForUpdate(platform, arch, channel, currentVersion)

Check the self-hosted update channel for a newer build

The client-facing half of the self-hosted update channel. It is API-key gated but not user-authenticated: a controlled machine checks for updates before anyone logs in.  The server owns the index of published releases. The artifact lives at &#x60;download_url&#x60;: normally a host the operator controls, or this server itself when the installer was uploaded to it (then &#x60;download_url&#x60; is &#x60;GET /v1/update-releases/{release_id}/artifact&#x60; and &#x60;hosted_here&#x60; is true). The index is managed through the admin API (&#x60;POST /v1/admin/update-releases&#x60; and friends), which stays outside this business contract.  &#x60;current_version&#x60; is optional. Without it the newest release is returned and &#x60;update_available&#x60; is true whenever one exists; with it, &#x60;update_available&#x60; is true only when the published version is strictly newer, and &#x60;mandatory&#x60; mirrors the release flag once an update is available. &#x60;latest&#x60; is always present and is &#x60;null&#x60; when the channel has no release for that platform/arch. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.UpdateApi;

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

    UpdateApi apiInstance = new UpdateApi(defaultClient);
    String platform = "windows"; // String | Which operating system the caller is running.
    String arch = "x64"; // String | Which architecture the caller is running.
    String channel = "stable"; // String | Defaults to `stable`.
    String currentVersion = "currentVersion_example"; // String | The dotted numeric version the caller is running, e.g. `4.2.5.0`. A leading `v` is accepted.
    try {
      UpdateCheckResult result = apiInstance.checkForUpdate(platform, arch, channel, currentVersion);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling UpdateApi#checkForUpdate");
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
| **platform** | **String**| Which operating system the caller is running. | [enum: windows, macos, linux] |
| **arch** | **String**| Which architecture the caller is running. | [enum: x64, arm64, universal] |
| **channel** | **String**| Defaults to &#x60;stable&#x60;. | [optional] [enum: stable, beta, nightly] |
| **currentVersion** | **String**| The dotted numeric version the caller is running, e.g. &#x60;4.2.5.0&#x60;. A leading &#x60;v&#x60; is accepted. | [optional] |

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
| **200** | The newest release for that channel/platform/arch, or a null &#x60;latest&#x60; when the channel is empty. |  -  |
| **400** | The request was malformed or failed validation. |  -  |

<a id="downloadUpdateArtifact"></a>
# **downloadUpdateArtifact**
> File downloadUpdateArtifact(releaseId)

Download an installer this server hosts itself

The other half of a re-hosted release. &#x60;GET /v1/update:check&#x60; returns the &#x60;download_url&#x60; of the newest build, and for a release whose bytes were uploaded to this server that URL is this route, under the same API key the client already used for the check.  API-key gated but not user-authenticated, like the check itself. The response supports range requests, so a transfer that drops can resume.  A release whose bytes live on an external static host is not served here: fetching its route is a &#x60;404&#x60;, and the client should use the &#x60;download_url&#x60; from the check response instead. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.UpdateApi;

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

    UpdateApi apiInstance = new UpdateApi(defaultClient);
    Long releaseId = 56L; // Long | The `id` of the release from the check response.
    try {
      File result = apiInstance.downloadUpdateArtifact(releaseId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling UpdateApi#downloadUpdateArtifact");
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
| **releaseId** | **Long**| The &#x60;id&#x60; of the release from the check response. | |

### Return type

[**File**](File.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/octet-stream, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The installer bytes. |  * Content-Disposition - The stored file name, as an attachment. <br>  * Accept-Ranges - Present because range requests are supported. <br>  |
| **400** | The request was malformed or failed validation. |  -  |
| **404** | The addressed resource does not exist. |  -  |

