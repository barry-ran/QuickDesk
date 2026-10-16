# \DevicesAPI

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ApproveAccessApproval**](DevicesAPI.md#ApproveAccessApproval) | **Post** /v1/me/devices/{device_id}/access-approvals/{approval_id}/approve | Approve a pending access request
[**ClaimAccessApproval**](DevicesAPI.md#ClaimAccessApproval) | **Post** /v1/devices/{device_id}/access-approvals:claim | Exchange an approved request for a signal token
[**CreateAccessGrant**](DevicesAPI.md#CreateAccessGrant) | **Post** /v1/me/devices/{device_id}/access-grants | Issue a time-boxed guest access code
[**DenyAccessApproval**](DevicesAPI.md#DenyAccessApproval) | **Post** /v1/me/devices/{device_id}/access-approvals/{approval_id}/deny | Deny a pending access request
[**GetAccessApprovalSummary**](DevicesAPI.md#GetAccessApprovalSummary) | **Get** /v1/me/devices/{device_id}/access-approvals:summary | Aggregated view of a device&#39;s approval queue
[**GetAccessPolicy**](DevicesAPI.md#GetAccessPolicy) | **Get** /v1/me/devices/{device_id}/access-policy | The device&#39;s access policy
[**ListAccessApprovals**](DevicesAPI.md#ListAccessApprovals) | **Get** /v1/me/devices/{device_id}/access-approvals | Second-factor approvals awaiting (or already decided by) the owner
[**ListAccessGrants**](DevicesAPI.md#ListAccessGrants) | **Get** /v1/me/devices/{device_id}/access-grants | Guest access codes issued for a device
[**ListMyDevices**](DevicesAPI.md#ListMyDevices) | **Get** /v1/me/devices | Devices owned by the authenticated user
[**PutAccessPolicy**](DevicesAPI.md#PutAccessPolicy) | **Put** /v1/me/devices/{device_id}/access-policy | Create or replace the device&#39;s access policy
[**RevokeAccessGrant**](DevicesAPI.md#RevokeAccessGrant) | **Delete** /v1/me/devices/{device_id}/access-grants/{grant_id} | Revoke a guest access code
[**VerifyAccessCode**](DevicesAPI.md#VerifyAccessCode) | **Post** /v1/devices/{device_id}/access-code:verify | Verify a device access code and mint a signal token



## ApproveAccessApproval

> AccessApproval ApproveAccessApproval(ctx, deviceId, approvalId).ApproveAccessApprovalRequest(approveAccessApprovalRequest).Execute()

Approve a pending access request



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.
	approvalId := int64(789) // int64 | The numeric id from an approval row.
	approveAccessApprovalRequest := *openapiclient.NewApproveAccessApprovalRequest() // ApproveAccessApprovalRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.ApproveAccessApproval(context.Background(), deviceId, approvalId).ApproveAccessApprovalRequest(approveAccessApprovalRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.ApproveAccessApproval``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ApproveAccessApproval`: AccessApproval
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.ApproveAccessApproval`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 
**approvalId** | **int64** | The numeric id from an approval row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiApproveAccessApprovalRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **approveAccessApprovalRequest** | [**ApproveAccessApprovalRequest**](ApproveAccessApprovalRequest.md) |  | 

### Return type

[**AccessApproval**](AccessApproval.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ClaimAccessApproval

> AccessCodeVerifyResponse ClaimAccessApproval(ctx, deviceId).ApprovalClaimRequest(approvalClaimRequest).Execute()

Exchange an approved request for a signal token



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.
	approvalClaimRequest := *openapiclient.NewApprovalClaimRequest(int64(123), "Secret_example") // ApprovalClaimRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.ClaimAccessApproval(context.Background(), deviceId).ApprovalClaimRequest(approvalClaimRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.ClaimAccessApproval``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ClaimAccessApproval`: AccessCodeVerifyResponse
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.ClaimAccessApproval`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiClaimAccessApprovalRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **approvalClaimRequest** | [**ApprovalClaimRequest**](ApprovalClaimRequest.md) |  | 

### Return type

[**AccessCodeVerifyResponse**](AccessCodeVerifyResponse.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateAccessGrant

> AccessGrant CreateAccessGrant(ctx, deviceId).AccessGrantInput(accessGrantInput).Execute()

Issue a time-boxed guest access code



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.
	accessGrantInput := *openapiclient.NewAccessGrantInput(int32(123)) // AccessGrantInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.CreateAccessGrant(context.Background(), deviceId).AccessGrantInput(accessGrantInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.CreateAccessGrant``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateAccessGrant`: AccessGrant
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.CreateAccessGrant`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiCreateAccessGrantRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **accessGrantInput** | [**AccessGrantInput**](AccessGrantInput.md) |  | 

### Return type

[**AccessGrant**](AccessGrant.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DenyAccessApproval

> AccessApproval DenyAccessApproval(ctx, deviceId, approvalId).ApproveAccessApprovalRequest(approveAccessApprovalRequest).Execute()

Deny a pending access request



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.
	approvalId := int64(789) // int64 | The numeric id from an approval row.
	approveAccessApprovalRequest := *openapiclient.NewApproveAccessApprovalRequest() // ApproveAccessApprovalRequest |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.DenyAccessApproval(context.Background(), deviceId, approvalId).ApproveAccessApprovalRequest(approveAccessApprovalRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.DenyAccessApproval``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DenyAccessApproval`: AccessApproval
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.DenyAccessApproval`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 
**approvalId** | **int64** | The numeric id from an approval row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiDenyAccessApprovalRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **approveAccessApprovalRequest** | [**ApproveAccessApprovalRequest**](ApproveAccessApprovalRequest.md) |  | 

### Return type

[**AccessApproval**](AccessApproval.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetAccessApprovalSummary

> ApprovalSummary GetAccessApprovalSummary(ctx, deviceId).Since(since).Limit(limit).Execute()

Aggregated view of a device's approval queue



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
    "time"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.
	since := time.Now() // time.Time | RFC3339 start of the window; defaults to the last 24 hours. (optional)
	limit := int32(56) // int32 | How many refused sources to list (default 5, capped at 10). (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.GetAccessApprovalSummary(context.Background(), deviceId).Since(since).Limit(limit).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.GetAccessApprovalSummary``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetAccessApprovalSummary`: ApprovalSummary
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.GetAccessApprovalSummary`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetAccessApprovalSummaryRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **since** | **time.Time** | RFC3339 start of the window; defaults to the last 24 hours. | 
 **limit** | **int32** | How many refused sources to list (default 5, capped at 10). | 

### Return type

[**ApprovalSummary**](ApprovalSummary.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetAccessPolicy

> DeviceAccessPolicy GetAccessPolicy(ctx, deviceId).Execute()

The device's access policy



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.GetAccessPolicy(context.Background(), deviceId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.GetAccessPolicy``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetAccessPolicy`: DeviceAccessPolicy
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.GetAccessPolicy`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetAccessPolicyRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListAccessApprovals

> ListAccessApprovals200Response ListAccessApprovals(ctx, deviceId).Status(status).Limit(limit).Execute()

Second-factor approvals awaiting (or already decided by) the owner



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.
	status := "status_example" // string |  (optional)
	limit := int32(56) // int32 |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.ListAccessApprovals(context.Background(), deviceId).Status(status).Limit(limit).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.ListAccessApprovals``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListAccessApprovals`: ListAccessApprovals200Response
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.ListAccessApprovals`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiListAccessApprovalsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **status** | **string** |  | 
 **limit** | **int32** |  | 

### Return type

[**ListAccessApprovals200Response**](ListAccessApprovals200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListAccessGrants

> ListAccessGrants200Response ListAccessGrants(ctx, deviceId).Execute()

Guest access codes issued for a device

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.ListAccessGrants(context.Background(), deviceId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.ListAccessGrants``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListAccessGrants`: ListAccessGrants200Response
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.ListAccessGrants`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiListAccessGrantsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**ListAccessGrants200Response**](ListAccessGrants200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListMyDevices

> ListMyDevices200Response ListMyDevices(ctx).Execute()

Devices owned by the authenticated user

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.ListMyDevices(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.ListMyDevices``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListMyDevices`: ListMyDevices200Response
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.ListMyDevices`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListMyDevicesRequest struct via the builder pattern


### Return type

[**ListMyDevices200Response**](ListMyDevices200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PutAccessPolicy

> DeviceAccessPolicy PutAccessPolicy(ctx, deviceId).DeviceAccessPolicyInput(deviceAccessPolicyInput).Execute()

Create or replace the device's access policy



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.
	deviceAccessPolicyInput := *openapiclient.NewDeviceAccessPolicyInput() // DeviceAccessPolicyInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.PutAccessPolicy(context.Background(), deviceId).DeviceAccessPolicyInput(deviceAccessPolicyInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.PutAccessPolicy``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PutAccessPolicy`: DeviceAccessPolicy
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.PutAccessPolicy`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiPutAccessPolicyRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **deviceAccessPolicyInput** | [**DeviceAccessPolicyInput**](DeviceAccessPolicyInput.md) |  | 

### Return type

[**DeviceAccessPolicy**](DeviceAccessPolicy.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RevokeAccessGrant

> RevokeAccessGrant200Response RevokeAccessGrant(ctx, deviceId, grantId).Execute()

Revoke a guest access code

### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.
	grantId := int64(789) // int64 | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.RevokeAccessGrant(context.Background(), deviceId, grantId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.RevokeAccessGrant``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RevokeAccessGrant`: RevokeAccessGrant200Response
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.RevokeAccessGrant`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 
**grantId** | **int64** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiRevokeAccessGrantRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**RevokeAccessGrant200Response**](RevokeAccessGrant200Response.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## VerifyAccessCode

> AccessCodeVerifyResponse VerifyAccessCode(ctx, deviceId).AccessCodeVerifyRequest(accessCodeVerifyRequest).Execute()

Verify a device access code and mint a signal token



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/ilh-github/go-remote-next/signaling"
)

func main() {
	deviceId := "deviceId_example" // string | The device id shown on the controlled machine.
	accessCodeVerifyRequest := *openapiclient.NewAccessCodeVerifyRequest("Code_example") // AccessCodeVerifyRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.DevicesAPI.VerifyAccessCode(context.Background(), deviceId).AccessCodeVerifyRequest(accessCodeVerifyRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `DevicesAPI.VerifyAccessCode``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `VerifyAccessCode`: AccessCodeVerifyResponse
	fmt.Fprintf(os.Stdout, "Response from `DevicesAPI.VerifyAccessCode`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiVerifyAccessCodeRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **accessCodeVerifyRequest** | [**AccessCodeVerifyRequest**](AccessCodeVerifyRequest.md) |  | 

### Return type

[**AccessCodeVerifyResponse**](AccessCodeVerifyResponse.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

