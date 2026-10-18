

# AccessCodeApprovalPending

The HTTP 202 body of `access-code:verify` - the same fields as `AccessCodeVerifyResponse` with `status: pending`. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**approvalId** | **Long** |  |  |
|**approvalSecret** | **String** | Shown once; exchanged for a signal token at the claim endpoint. |  |
|**status** | [**StatusEnum**](#StatusEnum) |  |  |
|**expiresAt** | **OffsetDateTime** |  |  |
|**pollAfterMs** | **Integer** |  |  |



## Enum: StatusEnum

| Name | Value |
|---- | -----|
| PENDING | &quot;pending&quot; |



