

# SessionRecordingPatch

Partial update - every field is optional and only the ones present are written, so a producer can patch the row as the capture progresses. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**status** | [**StatusEnum**](#StatusEnum) |  |  [optional] |
|**format** | **String** |  |  [optional] |
|**artifactPath** | **String** |  |  [optional] |
|**artifactUrl** | **String** |  |  [optional] |
|**thumbnailPath** | **String** |  |  [optional] |
|**durationSeconds** | **Integer** |  |  [optional] |
|**sizeBytes** | **Integer** |  |  [optional] |
|**endedAt** | **OffsetDateTime** |  |  [optional] |



## Enum: StatusEnum

| Name | Value |
|---- | -----|
| RECORDING | &quot;recording&quot; |
| COMPLETED | &quot;completed&quot; |
| FAILED | &quot;failed&quot; |



