# \AuthAPI

All URIs are relative to *https://your-signaling-host*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateSession**](AuthAPI.md#CreateSession) | **Post** /v1/auth/sessions | Exchange a username/password for a session



## CreateSession

> AuthSessionResponse CreateSession(ctx).AuthSessionRequest(authSessionRequest).Execute()

Exchange a username/password for a session

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
	authSessionRequest := *openapiclient.NewAuthSessionRequest("Identifier_example", "Password_example") // AuthSessionRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.AuthAPI.CreateSession(context.Background()).AuthSessionRequest(authSessionRequest).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthAPI.CreateSession``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `CreateSession`: AuthSessionResponse
	fmt.Fprintf(os.Stdout, "Response from `AuthAPI.CreateSession`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiCreateSessionRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **authSessionRequest** | [**AuthSessionRequest**](AuthSessionRequest.md) |  | 

### Return type

[**AuthSessionResponse**](AuthSessionResponse.md)

### Authorization

[ApiKeyAuth](../README.md#ApiKeyAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

