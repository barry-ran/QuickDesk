# \RemoteSessionsAPI

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetRemoteSession**](RemoteSessionsAPI.md#GetRemoteSession) | **Get** /v1/me/remote-sessions/{session_id} | One control session with the records that share its id
[**ListDeviceRemoteSessions**](RemoteSessionsAPI.md#ListDeviceRemoteSessions) | **Get** /v1/me/devices/{device_id}/sessions | One device&#39;s control sessions, each with its related records
[**ListRemoteSessions**](RemoteSessionsAPI.md#ListRemoteSessions) | **Get** /v1/me/remote-sessions | The caller&#39;s control sessions, open and finished
[**RevokeDeviceSessions**](RemoteSessionsAPI.md#RevokeDeviceSessions) | **Post** /v1/me/devices/{device_id}/sessions:revoke | End every live control session on one device
[**RevokeRemoteSession**](RemoteSessionsAPI.md#RevokeRemoteSession) | **Post** /v1/me/remote-sessions/{session_id}/revoke | End one of the caller&#39;s control sessions



## GetRemoteSession

> RemoteSessionDetail GetRemoteSession(ctx, sessionId).Execute()

One control session with the records that share its id



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
	sessionId := "sessionId_example" // string | The signaling session id from the list.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RemoteSessionsAPI.GetRemoteSession(context.Background(), sessionId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RemoteSessionsAPI.GetRemoteSession``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetRemoteSession`: RemoteSessionDetail
	fmt.Fprintf(os.Stdout, "Response from `RemoteSessionsAPI.GetRemoteSession`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**sessionId** | **string** | The signaling session id from the list. | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetRemoteSessionRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**RemoteSessionDetail**](RemoteSessionDetail.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListDeviceRemoteSessions

> RemoteSessionDetailList ListDeviceRemoteSessions(ctx, deviceId).Status(status).Cursor(cursor).Limit(limit).Execute()

One device's control sessions, each with its related records



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
	status := "status_example" // string | Defaults to `all`. (optional)
	cursor := "cursor_example" // string |  (optional)
	limit := int32(56) // int32 |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RemoteSessionsAPI.ListDeviceRemoteSessions(context.Background(), deviceId).Status(status).Cursor(cursor).Limit(limit).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RemoteSessionsAPI.ListDeviceRemoteSessions``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListDeviceRemoteSessions`: RemoteSessionDetailList
	fmt.Fprintf(os.Stdout, "Response from `RemoteSessionsAPI.ListDeviceRemoteSessions`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiListDeviceRemoteSessionsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **status** | **string** | Defaults to &#x60;all&#x60;. | 
 **cursor** | **string** |  | 
 **limit** | **int32** |  | 

### Return type

[**RemoteSessionDetailList**](RemoteSessionDetailList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListRemoteSessions

> RemoteSessionList ListRemoteSessions(ctx).Status(status).DeviceId(deviceId).Cursor(cursor).Limit(limit).Execute()

The caller's control sessions, open and finished



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
	status := "status_example" // string | Defaults to `all`. (optional)
	deviceId := "deviceId_example" // string |  (optional)
	cursor := "cursor_example" // string |  (optional)
	limit := int32(56) // int32 |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RemoteSessionsAPI.ListRemoteSessions(context.Background()).Status(status).DeviceId(deviceId).Cursor(cursor).Limit(limit).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RemoteSessionsAPI.ListRemoteSessions``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListRemoteSessions`: RemoteSessionList
	fmt.Fprintf(os.Stdout, "Response from `RemoteSessionsAPI.ListRemoteSessions`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListRemoteSessionsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **status** | **string** | Defaults to &#x60;all&#x60;. | 
 **deviceId** | **string** |  | 
 **cursor** | **string** |  | 
 **limit** | **int32** |  | 

### Return type

[**RemoteSessionList**](RemoteSessionList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RevokeDeviceSessions

> RemoteSessionBatchRevocation RevokeDeviceSessions(ctx, deviceId).Execute()

End every live control session on one device



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
	resp, r, err := apiClient.RemoteSessionsAPI.RevokeDeviceSessions(context.Background(), deviceId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RemoteSessionsAPI.RevokeDeviceSessions``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RevokeDeviceSessions`: RemoteSessionBatchRevocation
	fmt.Fprintf(os.Stdout, "Response from `RemoteSessionsAPI.RevokeDeviceSessions`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiRevokeDeviceSessionsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**RemoteSessionBatchRevocation**](RemoteSessionBatchRevocation.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RevokeRemoteSession

> RemoteSessionRevocation RevokeRemoteSession(ctx, sessionId).Execute()

End one of the caller's control sessions



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
	sessionId := "sessionId_example" // string | The signaling session id from the list.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RemoteSessionsAPI.RevokeRemoteSession(context.Background(), sessionId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RemoteSessionsAPI.RevokeRemoteSession``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RevokeRemoteSession`: RemoteSessionRevocation
	fmt.Fprintf(os.Stdout, "Response from `RemoteSessionsAPI.RevokeRemoteSession`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**sessionId** | **string** | The signaling session id from the list. | 

### Other Parameters

Other parameters are passed through a pointer to a apiRevokeRemoteSessionRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**RemoteSessionRevocation**](RemoteSessionRevocation.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

