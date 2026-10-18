# \RecordingsAPI

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AppendRecordingArtifact**](RecordingsAPI.md#AppendRecordingArtifact) | **Patch** /v1/me/recordings/{recording_id}/artifact | Append one chunk of a resumable recording upload
[**CreateRecordingArtifactTicket**](RecordingsAPI.md#CreateRecordingArtifactTicket) | **Post** /v1/me/recordings/{recording_id}/artifact-ticket | Mint a short-lived playback URL for a recording
[**CreateRecordingThumbnailTicket**](RecordingsAPI.md#CreateRecordingThumbnailTicket) | **Post** /v1/me/recordings/{recording_id}/thumbnail-ticket | Mint a short-lived preview URL for a recording
[**CreateSessionRecording**](RecordingsAPI.md#CreateSessionRecording) | **Post** /v1/me/recordings | Index a new recording
[**DeleteSessionRecording**](RecordingsAPI.md#DeleteSessionRecording) | **Delete** /v1/me/recordings/{recording_id} | Delete a recording and its stored media
[**DownloadRecordingArtifact**](RecordingsAPI.md#DownloadRecordingArtifact) | **Get** /v1/me/recordings/{recording_id}/artifact | Download a recording&#39;s media bytes
[**DownloadRecordingArtifactByTicket**](RecordingsAPI.md#DownloadRecordingArtifactByTicket) | **Get** /v1/recordings/{recording_id}/artifact | Fetch recording media with a playback ticket
[**DownloadRecordingThumbnail**](RecordingsAPI.md#DownloadRecordingThumbnail) | **Get** /v1/me/recordings/{recording_id}/thumbnail | Download a recording&#39;s preview image
[**DownloadRecordingThumbnailByTicket**](RecordingsAPI.md#DownloadRecordingThumbnailByTicket) | **Get** /v1/recordings/{recording_id}/thumbnail | Fetch a recording preview with a playback ticket
[**GetSessionRecording**](RecordingsAPI.md#GetSessionRecording) | **Get** /v1/me/recordings/{recording_id} | One recording the caller owns
[**ListSessionRecordings**](RecordingsAPI.md#ListSessionRecordings) | **Get** /v1/me/recordings | The caller&#39;s session-recording index
[**UpdateSessionRecording**](RecordingsAPI.md#UpdateSessionRecording) | **Patch** /v1/me/recordings/{recording_id} | Update a recording&#39;s lifecycle metadata
[**UploadRecordingArtifact**](RecordingsAPI.md#UploadRecordingArtifact) | **Post** /v1/me/recordings/{recording_id}/artifact | Upload a recording&#39;s media bytes
[**UploadRecordingThumbnail**](RecordingsAPI.md#UploadRecordingThumbnail) | **Post** /v1/me/recordings/{recording_id}/thumbnail | Upload a recording&#39;s preview image



## AppendRecordingArtifact

> SessionRecording AppendRecordingArtifact(ctx, recordingId).Ext(ext).UploadOffset(uploadOffset).UploadLength(uploadLength).UploadComplete(uploadComplete).UploadProbe(uploadProbe).Body(body).Execute()

Append one chunk of a resumable recording upload



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
	ext := "ext_example" // string | The artifact extension, allow-listed exactly like the raw-body POST shape.
	uploadOffset := int64(789) // int64 | The byte offset this chunk starts at.
	recordingId := int64(789) // int64 | The numeric id from a recording index row.
	uploadLength := "uploadLength_example" // string | The total size of the artifact, or `*` while it is unknown. (optional)
	uploadComplete := "uploadComplete_example" // string | Set to `1` when this chunk ends the upload. (optional)
	uploadProbe := "uploadProbe_example" // string | Set to `1` to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body. (optional)
	body := os.NewFile(1234, "some_file") // *os.File |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.AppendRecordingArtifact(context.Background(), recordingId).Ext(ext).UploadOffset(uploadOffset).UploadLength(uploadLength).UploadComplete(uploadComplete).UploadProbe(uploadProbe).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.AppendRecordingArtifact``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AppendRecordingArtifact`: SessionRecording
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.AppendRecordingArtifact`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiAppendRecordingArtifactRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ext** | **string** | The artifact extension, allow-listed exactly like the raw-body POST shape. | 
 **uploadOffset** | **int64** | The byte offset this chunk starts at. | 

 **uploadLength** | **string** | The total size of the artifact, or &#x60;*&#x60; while it is unknown. | 
 **uploadComplete** | **string** | Set to &#x60;1&#x60; when this chunk ends the upload. | 
 **uploadProbe** | **string** | Set to &#x60;1&#x60; to ask for the current offset without staging any bytes. The generated clients use this instead of an empty body. | 
 **body** | ***os.File** |  | 

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/octet-stream
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateRecordingArtifactTicket

> PlaybackTicket CreateRecordingArtifactTicket(ctx, recordingId).Execute()

Mint a short-lived playback URL for a recording



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
	recordingId := int64(789) // int64 | The numeric id from a recording index row.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.CreateRecordingArtifactTicket(context.Background(), recordingId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.CreateRecordingArtifactTicket``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateRecordingArtifactTicket`: PlaybackTicket
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.CreateRecordingArtifactTicket`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiCreateRecordingArtifactTicketRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**PlaybackTicket**](PlaybackTicket.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateRecordingThumbnailTicket

> PlaybackTicket CreateRecordingThumbnailTicket(ctx, recordingId).Execute()

Mint a short-lived preview URL for a recording



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
	recordingId := int64(789) // int64 | The numeric id from a recording index row.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.CreateRecordingThumbnailTicket(context.Background(), recordingId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.CreateRecordingThumbnailTicket``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateRecordingThumbnailTicket`: PlaybackTicket
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.CreateRecordingThumbnailTicket`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiCreateRecordingThumbnailTicketRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**PlaybackTicket**](PlaybackTicket.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## CreateSessionRecording

> SessionRecording CreateSessionRecording(ctx).SessionRecordingInput(sessionRecordingInput).Execute()

Index a new recording



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
	sessionRecordingInput := *openapiclient.NewSessionRecordingInput("DeviceId_example", "Status_example") // SessionRecordingInput | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.CreateSessionRecording(context.Background()).SessionRecordingInput(sessionRecordingInput).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.CreateSessionRecording``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateSessionRecording`: SessionRecording
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.CreateSessionRecording`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateSessionRecordingRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **sessionRecordingInput** | [**SessionRecordingInput**](SessionRecordingInput.md) |  | 

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteSessionRecording

> DeleteSessionRecording(ctx, recordingId).Execute()

Delete a recording and its stored media



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
	recordingId := int64(789) // int64 | The numeric id from a recording index row.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.RecordingsAPI.DeleteSessionRecording(context.Background(), recordingId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.DeleteSessionRecording``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteSessionRecordingRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DownloadRecordingArtifact

> *os.File DownloadRecordingArtifact(ctx, recordingId).Execute()

Download a recording's media bytes



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
	recordingId := int64(789) // int64 | The numeric id from a recording index row.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.DownloadRecordingArtifact(context.Background(), recordingId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.DownloadRecordingArtifact``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DownloadRecordingArtifact`: *os.File
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.DownloadRecordingArtifact`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiDownloadRecordingArtifactRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[***os.File**](*os.File.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: video/mp4, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DownloadRecordingArtifactByTicket

> *os.File DownloadRecordingArtifactByTicket(ctx, recordingId).T(t).Execute()

Fetch recording media with a playback ticket



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
	t := "t_example" // string | The signed ticket from the artifact-ticket endpoint.
	recordingId := int64(789) // int64 | The numeric id from a recording index row.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.DownloadRecordingArtifactByTicket(context.Background(), recordingId).T(t).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.DownloadRecordingArtifactByTicket``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DownloadRecordingArtifactByTicket`: *os.File
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.DownloadRecordingArtifactByTicket`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiDownloadRecordingArtifactByTicketRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **t** | **string** | The signed ticket from the artifact-ticket endpoint. | 


### Return type

[***os.File**](*os.File.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: video/mp4, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DownloadRecordingThumbnail

> *os.File DownloadRecordingThumbnail(ctx, recordingId).Execute()

Download a recording's preview image



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
	recordingId := int64(789) // int64 | The numeric id from a recording index row.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.DownloadRecordingThumbnail(context.Background(), recordingId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.DownloadRecordingThumbnail``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DownloadRecordingThumbnail`: *os.File
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.DownloadRecordingThumbnail`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiDownloadRecordingThumbnailRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[***os.File**](*os.File.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: image/jpeg, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DownloadRecordingThumbnailByTicket

> *os.File DownloadRecordingThumbnailByTicket(ctx, recordingId).T(t).Execute()

Fetch a recording preview with a playback ticket



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
	t := "t_example" // string | The signed ticket from the thumbnail-ticket endpoint.
	recordingId := int64(789) // int64 | The numeric id from a recording index row.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.DownloadRecordingThumbnailByTicket(context.Background(), recordingId).T(t).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.DownloadRecordingThumbnailByTicket``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `DownloadRecordingThumbnailByTicket`: *os.File
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.DownloadRecordingThumbnailByTicket`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiDownloadRecordingThumbnailByTicketRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **t** | **string** | The signed ticket from the thumbnail-ticket endpoint. | 


### Return type

[***os.File**](*os.File.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: image/jpeg, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetSessionRecording

> SessionRecording GetSessionRecording(ctx, recordingId).Execute()

One recording the caller owns



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
	recordingId := int64(789) // int64 | The numeric id from a recording index row.

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.GetSessionRecording(context.Background(), recordingId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.GetSessionRecording``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GetSessionRecording`: SessionRecording
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.GetSessionRecording`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetSessionRecordingRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListSessionRecordings

> SessionRecordingList ListSessionRecordings(ctx).Cursor(cursor).Limit(limit).Execute()

The caller's session-recording index



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
	cursor := "cursor_example" // string |  (optional)
	limit := int32(56) // int32 |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.ListSessionRecordings(context.Background()).Cursor(cursor).Limit(limit).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.ListSessionRecordings``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ListSessionRecordings`: SessionRecordingList
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.ListSessionRecordings`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListSessionRecordingsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **cursor** | **string** |  | 
 **limit** | **int32** |  | 

### Return type

[**SessionRecordingList**](SessionRecordingList.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UpdateSessionRecording

> SessionRecording UpdateSessionRecording(ctx, recordingId).SessionRecordingPatch(sessionRecordingPatch).Execute()

Update a recording's lifecycle metadata



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
	recordingId := int64(789) // int64 | The numeric id from a recording index row.
	sessionRecordingPatch := *openapiclient.NewSessionRecordingPatch() // SessionRecordingPatch | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.UpdateSessionRecording(context.Background(), recordingId).SessionRecordingPatch(sessionRecordingPatch).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.UpdateSessionRecording``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UpdateSessionRecording`: SessionRecording
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.UpdateSessionRecording`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiUpdateSessionRecordingRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **sessionRecordingPatch** | [**SessionRecordingPatch**](SessionRecordingPatch.md) |  | 

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UploadRecordingArtifact

> SessionRecording UploadRecordingArtifact(ctx, recordingId).File(file).Ext(ext).Execute()

Upload a recording's media bytes



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
	recordingId := int64(789) // int64 | The numeric id from a recording index row.
	file := os.NewFile(1234, "some_file") // *os.File | 
	ext := "ext_example" // string | Required for the raw-body shape; ignored for multipart. (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.UploadRecordingArtifact(context.Background(), recordingId).File(file).Ext(ext).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.UploadRecordingArtifact``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UploadRecordingArtifact`: SessionRecording
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.UploadRecordingArtifact`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiUploadRecordingArtifactRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **file** | ***os.File** |  | 
 **ext** | **string** | Required for the raw-body shape; ignored for multipart. | 

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: multipart/form-data, application/octet-stream
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UploadRecordingThumbnail

> SessionRecording UploadRecordingThumbnail(ctx, recordingId).File(file).Ext(ext).Execute()

Upload a recording's preview image



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
	recordingId := int64(789) // int64 | The numeric id from a recording index row.
	file := os.NewFile(1234, "some_file") // *os.File | 
	ext := "ext_example" // string | Required for the raw-body shape; ignored for multipart. (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RecordingsAPI.UploadRecordingThumbnail(context.Background(), recordingId).File(file).Ext(ext).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RecordingsAPI.UploadRecordingThumbnail``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UploadRecordingThumbnail`: SessionRecording
	fmt.Fprintf(os.Stdout, "Response from `RecordingsAPI.UploadRecordingThumbnail`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**recordingId** | **int64** | The numeric id from a recording index row. | 

### Other Parameters

Other parameters are passed through a pointer to a apiUploadRecordingThumbnailRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **file** | ***os.File** |  | 
 **ext** | **string** | Required for the raw-body shape; ignored for multipart. | 

### Return type

[**SessionRecording**](SessionRecording.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth), [BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: multipart/form-data, application/octet-stream
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

