

# AccessCodeVerifyResponse

The answer to `access-code:verify` and to the approval claim endpoint. `status` says which shape it is: `granted` (HTTP 200) carries `signal_token`, `pending` (HTTP 202) carries the approval the controller polls. The approval fields live on this schema rather than only on the 202 schema because a generated client declares one success type per operation - a Go caller receives the 202 body as this type, and without these properties it could not read the approval it is supposed to poll. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**status** | [**StatusEnum**](#StatusEnum) | Which shape this response is. |  |
|**signalToken** | **String** | Present when &#x60;status&#x60; is &#x60;granted&#x60;. |  [optional] |
|**expiresAt** | **OffsetDateTime** | When the signal token (granted) or the approval (pending) expires. |  [optional] |
|**approvalId** | **Long** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. |  [optional] |
|**approvalSecret** | **String** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. Shown once; exchanged for a signal token at the claim endpoint.  |  [optional] |
|**pollAfterMs** | **Integer** | Present when &#x60;status&#x60; is &#x60;pending&#x60;. |  [optional] |



## Enum: StatusEnum

| Name | Value |
|---- | -----|
| GRANTED | &quot;granted&quot; |
| PENDING | &quot;pending&quot; |



