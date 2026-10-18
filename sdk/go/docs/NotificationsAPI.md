# \NotificationsAPI

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetOfflineAlert**](NotificationsAPI.md#GetOfflineAlert) | **Get** /v1/me/devices/{device_id}/offline-alert | The owner&#39;s offline-alert rule for a device
[**ListNotificationChannels**](NotificationsAPI.md#ListNotificationChannels) | **Get** /v1/me/notification-channels | The user&#39;s notification channels
[**PutOfflineAlert**](NotificationsAPI.md#PutOfflineAlert) | **Put** /v1/me/devices/{device_id}/offline-alert | Create or replace the owner&#39;s offline-alert rule



## GetOfflineAlert

> OfflineAlertRule GetOfflineAlert(ctx, deviceId).Execute()

The owner's offline-alert rule for a device



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
	resp, r, err := apiClient.NotificationsAPI.GetOfflineAlert(context.Background(), deviceId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `NotificationsAPI.GetOfflineAlert``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetOfflineAlert`: OfflineAlertRule
	fmt.Fprintf(os.Stdout, "Response from `NotificationsAPI.GetOfflineAlert`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetOfflineAlertRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**OfflineAlertRule**](OfflineAlertRule.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListNotificationChannels

> NotificationChannelList ListNotificationChannels(ctx).Execute()

The user's notification channels



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
	resp, r, err := apiClient.NotificationsAPI.ListNotificationChannels(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `NotificationsAPI.ListNotificationChannels``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListNotificationChannels`: NotificationChannelList
	fmt.Fprintf(os.Stdout, "Response from `NotificationsAPI.ListNotificationChannels`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListNotificationChannelsRequest struct via the builder pattern


### Return type

[**NotificationChannelList**](NotificationChannelList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## PutOfflineAlert

> OfflineAlertRule PutOfflineAlert(ctx, deviceId).OfflineAlertInput(offlineAlertInput).Execute()

Create or replace the owner's offline-alert rule

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
	offlineAlertInput := *openapiclient.NewOfflineAlertInput(false) // OfflineAlertInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.NotificationsAPI.PutOfflineAlert(context.Background(), deviceId).OfflineAlertInput(offlineAlertInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `NotificationsAPI.PutOfflineAlert``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `PutOfflineAlert`: OfflineAlertRule
	fmt.Fprintf(os.Stdout, "Response from `NotificationsAPI.PutOfflineAlert`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**deviceId** | **string** | The device id shown on the controlled machine. | 

### Other Parameters

Other parameters are passed through a pointer to a apiPutOfflineAlertRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **offlineAlertInput** | [**OfflineAlertInput**](OfflineAlertInput.md) |  | 

### Return type

[**OfflineAlertRule**](OfflineAlertRule.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

