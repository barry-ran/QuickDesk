

# ApprovalAlreadyDecided

Returned by approve/deny when the request was already decided (or expired) so the click is reported honestly instead of silently succeeding. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**status** | [**StatusEnum**](#StatusEnum) |  |  |
|**message** | **String** |  |  |



## Enum: StatusEnum

| Name | Value |
|---- | -----|
| PENDING | &quot;pending&quot; |
| APPROVED | &quot;approved&quot; |
| DENIED | &quot;denied&quot; |
| EXPIRED | &quot;expired&quot; |
| CONSUMED | &quot;consumed&quot; |



