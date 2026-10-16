# \RemoteSessionsAPI

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetRemoteSession**](RemoteSessionsAPI.md#GetRemoteSession) | **Get** /v1/me/remote-sessions/{session_id} | One control session with the records that share its id
[**ListRemoteSessions**](RemoteSessionsAPI.md#ListRemoteSessions) | **Get** /v1/me/remote-sessions | The caller&#39;s control sessions, open and finished



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

