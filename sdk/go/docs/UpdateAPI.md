# \UpdateAPI

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CheckForUpdate**](UpdateAPI.md#CheckForUpdate) | **Get** /v1/update:check | Check the self-hosted update channel for a newer build
[**DownloadUpdateArtifact**](UpdateAPI.md#DownloadUpdateArtifact) | **Get** /v1/update-releases/{release_id}/artifact | Download an installer this server hosts itself



## CheckForUpdate

> UpdateCheckResult CheckForUpdate(ctx).Platform(platform).Arch(arch).Channel(channel).CurrentVersion(currentVersion).Execute()

Check the self-hosted update channel for a newer build



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
	platform := "platform_example" // string | Which operating system the caller is running.
	arch := "arch_example" // string | Which architecture the caller is running.
	channel := "channel_example" // string | Defaults to `stable`. (optional)
	currentVersion := "currentVersion_example" // string | The dotted numeric version the caller is running, e.g. `4.2.5.0`. A leading `v` is accepted. (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.UpdateAPI.CheckForUpdate(context.Background()).Platform(platform).Arch(arch).Channel(channel).CurrentVersion(currentVersion).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `UpdateAPI.CheckForUpdate``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CheckForUpdate`: UpdateCheckResult
	fmt.Fprintf(os.Stdout, "Response from `UpdateAPI.CheckForUpdate`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCheckForUpdateRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **platform** | **string** | Which operating system the caller is running. | 
 **arch** | **string** | Which architecture the caller is running. | 
 **channel** | **string** | Defaults to &#x60;stable&#x60;. | 
 **currentVersion** | **string** | The dotted numeric version the caller is running, e.g. &#x60;4.2.5.0&#x60;. A leading &#x60;v&#x60; is accepted. | 

### Return type

[**UpdateCheckResult**](UpdateCheckResult.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DownloadUpdateArtifact

> *os.File DownloadUpdateArtifact(ctx, releaseId).Execute()

Download an installer this server hosts itself



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
	releaseId := int64(789) // int64 | The `id` of the release from the check response.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.UpdateAPI.DownloadUpdateArtifact(context.Background(), releaseId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `UpdateAPI.DownloadUpdateArtifact``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DownloadUpdateArtifact`: *os.File
	fmt.Fprintf(os.Stdout, "Response from `UpdateAPI.DownloadUpdateArtifact`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**releaseId** | **int64** | The &#x60;id&#x60; of the release from the check response. | 

### Other Parameters

Other parameters are passed through a pointer to a apiDownloadUpdateArtifactRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[***os.File**](*os.File.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/octet-stream, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

