

# AccessApproval


## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**id** | **Long** |  |  |
|**deviceId** | **String** |  |  |
|**clientId** | **String** | Self-reported by the requesting controller. |  |
|**clientIp** | **String** | The address the server observed (trusted-proxy aware). |  |
|**source** | [**SourceEnum**](#SourceEnum) |  |  |
|**grantId** | **Long** |  |  [optional] |
|**status** | [**StatusEnum**](#StatusEnum) |  |  |
|**expiresAt** | **OffsetDateTime** |  |  |
|**decidedAt** | **OffsetDateTime** |  |  [optional] |
|**reason** | **String** |  |  [optional] |
|**createdAt** | **OffsetDateTime** |  |  |



## Enum: SourceEnum

| Name | Value |
|---- | -----|
| DEVICE_CODE | &quot;device_code&quot; |
| GRANT | &quot;grant&quot; |



## Enum: StatusEnum

| Name | Value |
|---- | -----|
| PENDING | &quot;pending&quot; |
| APPROVED | &quot;approved&quot; |
| DENIED | &quot;denied&quot; |
| EXPIRED | &quot;expired&quot; |
| CONSUMED | &quot;consumed&quot; |



