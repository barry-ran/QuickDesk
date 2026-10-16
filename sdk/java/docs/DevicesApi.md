# DevicesApi

All URIs are relative to *https://your-signaling-host*

| Method | HTTP request | Description |
|------------- | ------------- | -------------|
| [**approveAccessApproval**](DevicesApi.md#approveAccessApproval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/approve | Approve a pending access request |
| [**claimAccessApproval**](DevicesApi.md#claimAccessApproval) | **POST** /v1/devices/{device_id}/access-approvals:claim | Exchange an approved request for a signal token |
| [**createAccessGrant**](DevicesApi.md#createAccessGrant) | **POST** /v1/me/devices/{device_id}/access-grants | Issue a time-boxed guest access code |
| [**denyAccessApproval**](DevicesApi.md#denyAccessApproval) | **POST** /v1/me/devices/{device_id}/access-approvals/{approval_id}/deny | Deny a pending access request |
| [**getAccessApprovalSummary**](DevicesApi.md#getAccessApprovalSummary) | **GET** /v1/me/devices/{device_id}/access-approvals:summary | Aggregated view of a device&#39;s approval queue |
| [**getAccessPolicy**](DevicesApi.md#getAccessPolicy) | **GET** /v1/me/devices/{device_id}/access-policy | The device&#39;s access policy |
| [**listAccessApprovals**](DevicesApi.md#listAccessApprovals) | **GET** /v1/me/devices/{device_id}/access-approvals | Second-factor approvals awaiting (or already decided by) the owner |
| [**listAccessGrants**](DevicesApi.md#listAccessGrants) | **GET** /v1/me/devices/{device_id}/access-grants | Guest access codes issued for a device |
| [**listMyDevices**](DevicesApi.md#listMyDevices) | **GET** /v1/me/devices | Devices owned by the authenticated user |
| [**putAccessPolicy**](DevicesApi.md#putAccessPolicy) | **PUT** /v1/me/devices/{device_id}/access-policy | Create or replace the device&#39;s access policy |
| [**revokeAccessGrant**](DevicesApi.md#revokeAccessGrant) | **DELETE** /v1/me/devices/{device_id}/access-grants/{grant_id} | Revoke a guest access code |
| [**verifyAccessCode**](DevicesApi.md#verifyAccessCode) | **POST** /v1/devices/{device_id}/access-code:verify | Verify a device access code and mint a signal token |


<a id="approveAccessApproval"></a>
# **approveAccessApproval**
> AccessApproval approveAccessApproval(deviceId, approvalId, approveAccessApprovalRequest)

Approve a pending access request

Owner-only. The body is optional; a &#x60;reason&#x60; is stored on the row and shown to the requester. A request that was already decided or has expired returns &#x60;409&#x60; with the current status instead of pretending the decision landed. The conformance smoke pins the &#x60;404&#x60; shape with an unknown id; the happy path needs a real pending approval (an online device plus &#x60;approval_mode&#x60;), so it is covered by the end-to-end run in &#x60;docs/dev/自托管端到端联调记录.md&#x60; rather than the smoke. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

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

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    Long approvalId = 56L; // Long | The numeric id from an approval row.
    ApproveAccessApprovalRequest approveAccessApprovalRequest = new ApproveAccessApprovalRequest(); // ApproveAccessApprovalRequest | 
    try {
      AccessApproval result = apiInstance.approveAccessApproval(deviceId, approvalId, approveAccessApprovalRequest);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#approveAccessApproval");
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
| **approvalId** | **Long**| The numeric id from an approval row. | |
| **approveAccessApprovalRequest** | [**ApproveAccessApprovalRequest**](ApproveAccessApprovalRequest.md)|  | [optional] |

### Return type

[**AccessApproval**](AccessApproval.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The approval after the decision. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **409** | The request was already decided or has expired. |  -  |

<a id="claimAccessApproval"></a>
# **claimAccessApproval**
> AccessCodeVerifyResponse claimAccessApproval(deviceId, approvalClaimRequest)

Exchange an approved request for a signal token

The requesting controller calls this with the &#x60;approval_id&#x60; and &#x60;approval_secret&#x60; it received from &#x60;access-code:verify&#x60;. It is API-key gated rather than user-authenticated because the caller has no account. &#x60;202&#x60; means the owner has not decided yet - keep polling with the same id and secret; &#x60;200&#x60; returns the signal token; &#x60;403&#x60; means denied or expired; &#x60;409&#x60; means the approval was already spent or the host went offline before the exchange. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

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

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    ApprovalClaimRequest approvalClaimRequest = new ApprovalClaimRequest(); // ApprovalClaimRequest | 
    try {
      AccessCodeVerifyResponse result = apiInstance.claimAccessApproval(deviceId, approvalClaimRequest);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#claimAccessApproval");
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
| **approvalClaimRequest** | [**ApprovalClaimRequest**](ApprovalClaimRequest.md)|  | |

### Return type

[**AccessCodeVerifyResponse**](AccessCodeVerifyResponse.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The owner approved and the signal token is ready. |  -  |
| **202** | Still waiting for the owner to decide. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **409** | The approval was already spent, or the host went offline before the exchange could happen.  |  -  |

<a id="createAccessGrant"></a>
# **createAccessGrant**
> AccessGrant createAccessGrant(deviceId, accessGrantInput)

Issue a time-boxed guest access code

The plaintext code is returned **once**, in the creation response; a grant is limited by expiry and uses, may be bound to a client id and/or source addresses, and is revoked with the DELETE below. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

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

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    AccessGrantInput accessGrantInput = new AccessGrantInput(); // AccessGrantInput | 
    try {
      AccessGrant result = apiInstance.createAccessGrant(deviceId, accessGrantInput);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#createAccessGrant");
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
| **accessGrantInput** | [**AccessGrantInput**](AccessGrantInput.md)|  | |

### Return type

[**AccessGrant**](AccessGrant.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | The created grant, including its one-time plaintext code. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="denyAccessApproval"></a>
# **denyAccessApproval**
> AccessApproval denyAccessApproval(deviceId, approvalId, approveAccessApprovalRequest)

Deny a pending access request

Owner-only, same shape as &#x60;approve&#x60;. Denying also arms the per-source denial cooldown (&#x60;APPROVAL_DENIAL_COOLDOWN_SECONDS&#x60;), so the same client cannot immediately re-request. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

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

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    Long approvalId = 56L; // Long | The numeric id from an approval row.
    ApproveAccessApprovalRequest approveAccessApprovalRequest = new ApproveAccessApprovalRequest(); // ApproveAccessApprovalRequest | 
    try {
      AccessApproval result = apiInstance.denyAccessApproval(deviceId, approvalId, approveAccessApprovalRequest);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#denyAccessApproval");
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
| **approvalId** | **Long**| The numeric id from an approval row. | |
| **approveAccessApprovalRequest** | [**ApproveAccessApprovalRequest**](ApproveAccessApprovalRequest.md)|  | [optional] |

### Return type

[**AccessApproval**](AccessApproval.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The approval after the decision. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **409** | The request was already decided or has expired. |  -  |

<a id="getAccessApprovalSummary"></a>
# **getAccessApprovalSummary**
> ApprovalSummary getAccessApprovalSummary(deviceId, since, limit)

Aggregated view of a device&#39;s approval queue

The row list shows requests one at a time; this is what makes \&quot;one address has been refused six times this afternoon\&quot; legible. The window is keyed on &#x60;created_at&#x60; for every count. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

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

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    OffsetDateTime since = OffsetDateTime.now(); // OffsetDateTime | RFC3339 start of the window; defaults to the last 24 hours.
    Integer limit = 56; // Integer | How many refused sources to list (default 5, capped at 10).
    try {
      ApprovalSummary result = apiInstance.getAccessApprovalSummary(deviceId, since, limit);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#getAccessApprovalSummary");
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
| **since** | **OffsetDateTime**| RFC3339 start of the window; defaults to the last 24 hours. | [optional] |
| **limit** | **Integer**| How many refused sources to list (default 5, capped at 10). | [optional] |

### Return type

[**ApprovalSummary**](ApprovalSummary.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The window&#39;s counts and refused sources. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="getAccessPolicy"></a>
# **getAccessPolicy**
> DeviceAccessPolicy getAccessPolicy(deviceId)

The device&#39;s access policy

Returns the documented \&quot;no policy\&quot; defaults when the owner never saved one, so a client can render the form without a second existence check. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

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

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    try {
      DeviceAccessPolicy result = apiInstance.getAccessPolicy(deviceId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#getAccessPolicy");
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

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The policy, or its defaults. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="listAccessApprovals"></a>
# **listAccessApprovals**
> ListAccessApprovals200Response listAccessApprovals(deviceId, status, limit)

Second-factor approvals awaiting (or already decided by) the owner

&#x60;status&#x60; is a comma-separated subset of &#x60;pending, approved, denied, expired, consumed&#x60; and defaults to &#x60;pending&#x60;. Approval is created by the device-side verify route when the policy asks for it; approving or denying happens on the sub-resource routes documented in &#x60;docs/user-api-docs.md&#x60; §5.3.5 (not yet part of this contract). 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

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

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    String status = "status_example"; // String | 
    Integer limit = 56; // Integer | 
    try {
      ListAccessApprovals200Response result = apiInstance.listAccessApprovals(deviceId, status, limit);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#listAccessApprovals");
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
| **status** | **String**|  | [optional] |
| **limit** | **Integer**|  | [optional] |

### Return type

[**ListAccessApprovals200Response**](ListAccessApprovals200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The matching approvals, newest first. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="listAccessGrants"></a>
# **listAccessGrants**
> ListAccessGrants200Response listAccessGrants(deviceId)

Guest access codes issued for a device

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

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

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    try {
      ListAccessGrants200Response result = apiInstance.listAccessGrants(deviceId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#listAccessGrants");
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

### Return type

[**ListAccessGrants200Response**](ListAccessGrants200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The device&#39;s grants, newest first. |  -  |
| **401** | Missing or expired credentials. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="listMyDevices"></a>
# **listMyDevices**
> ListMyDevices200Response listMyDevices()

Devices owned by the authenticated user

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

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

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    try {
      ListMyDevices200Response result = apiInstance.listMyDevices();
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#listMyDevices");
      System.err.println("Status code: " + e.getCode());
      System.err.println("Reason: " + e.getResponseBody());
      System.err.println("Response headers: " + e.getResponseHeaders());
      e.printStackTrace();
    }
  }
}
```

### Parameters
This endpoint does not need any parameter.

### Return type

[**ListMyDevices200Response**](ListMyDevices200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The caller&#39;s devices. |  -  |
| **401** | Missing or expired credentials. |  -  |

<a id="putAccessPolicy"></a>
# **putAccessPolicy**
> DeviceAccessPolicy putAccessPolicy(deviceId, deviceAccessPolicyInput)

Create or replace the device&#39;s access policy

A **full replace**: every field not sent is reset to its zero value. The policy is enforced on the device-side verify route before the access code is compared, so a blocked client never gets to guess codes. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

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

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    DeviceAccessPolicyInput deviceAccessPolicyInput = new DeviceAccessPolicyInput(); // DeviceAccessPolicyInput | 
    try {
      DeviceAccessPolicy result = apiInstance.putAccessPolicy(deviceId, deviceAccessPolicyInput);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#putAccessPolicy");
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
| **deviceAccessPolicyInput** | [**DeviceAccessPolicyInput**](DeviceAccessPolicyInput.md)|  | |

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The stored policy. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="revokeAccessGrant"></a>
# **revokeAccessGrant**
> RevokeAccessGrant200Response revokeAccessGrant(deviceId, grantId)

Revoke a guest access code

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

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

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    Long grantId = 56L; // Long | 
    try {
      RevokeAccessGrant200Response result = apiInstance.revokeAccessGrant(deviceId, grantId);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#revokeAccessGrant");
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
| **grantId** | **Long**|  | |

### Return type

[**RevokeAccessGrant200Response**](RevokeAccessGrant200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The grant is no longer redeemable. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |

<a id="verifyAccessCode"></a>
# **verifyAccessCode**
> AccessCodeVerifyResponse verifyAccessCode(deviceId, accessCodeVerifyRequest)

Verify a device access code and mint a signal token

The controller-side entry point. On success it returns a signal token for the relay. When the device policy wants the owner to confirm the session it returns &#x60;202&#x60; with a pending approval instead, and the caller polls &#x60;/v1/devices/{device_id}/access-approvals:claim&#x60;. A rejected code is a &#x60;403&#x60; problem with &#x60;code: INVALID_CODE&#x60;.  This route is reachable without a user token (the controlling client is not logged in) but still requires the API key or an allowed Origin. 

### Example
```java
// Import classes:
import org.openapitools.client.ApiClient;
import org.openapitools.client.ApiException;
import org.openapitools.client.Configuration;
import org.openapitools.client.auth.*;
import org.openapitools.client.models.*;
import org.openapitools.client.api.DevicesApi;

public class Example {
  public static void main(String[] args) {
    ApiClient defaultClient = Configuration.getDefaultApiClient();
    defaultClient.setBasePath("https://your-signaling-host");
    
    // Configure API key authorization: ApiKeyAuth
    ApiKeyAuth ApiKeyAuth = (ApiKeyAuth) defaultClient.getAuthentication("ApiKeyAuth");
    ApiKeyAuth.setApiKey("YOUR API KEY");
    // Uncomment the following line to set a prefix for the API key, e.g. "Token" (defaults to null)
    //ApiKeyAuth.setApiKeyPrefix("Token");

    DevicesApi apiInstance = new DevicesApi(defaultClient);
    String deviceId = "deviceId_example"; // String | The device id shown on the controlled machine.
    AccessCodeVerifyRequest accessCodeVerifyRequest = new AccessCodeVerifyRequest(); // AccessCodeVerifyRequest | 
    try {
      AccessCodeVerifyResponse result = apiInstance.verifyAccessCode(deviceId, accessCodeVerifyRequest);
      System.out.println(result);
    } catch (ApiException e) {
      System.err.println("Exception when calling DevicesApi#verifyAccessCode");
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
| **accessCodeVerifyRequest** | [**AccessCodeVerifyRequest**](AccessCodeVerifyRequest.md)|  | |

### Return type

[**AccessCodeVerifyResponse**](AccessCodeVerifyResponse.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json, application/problem+json

### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The code was accepted and a signal token was minted. |  -  |
| **202** | The device policy requires the owner to approve this session. |  -  |
| **400** | The request was malformed or failed validation. |  -  |
| **403** | Authenticated but not allowed (includes a rejected access code). |  -  |
| **404** | The addressed resource does not exist. |  -  |
| **429** | Too many requests; retry after the advertised delay. |  * Retry-After - Seconds to wait before retrying. <br>  |

