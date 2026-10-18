

# SessionRecordingInput

The index row a producer creates before (or instead of) uploading media. `device_id` is recorded as a label; it is not checked against the caller's devices, because a headless producer may be indexing a capture for a machine it is not bound to. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**deviceId** | **String** | The device the capture came from. |  |
|**deviceName** | **String** |  |  [optional] |
|**sessionId** | **String** | The control session this recording belongs to, when there is one. |  [optional] |
|**status** | [**StatusEnum**](#StatusEnum) | How far the capture got when this row was written. |  |
|**format** | **String** | Free-form label; the upload route overwrites it from the file extension. |  [optional] |
|**artifactPath** | **String** | Server-side path. Set by the upload route; clients do not need to send it. |  [optional] |
|**artifactUrl** | **String** |  |  [optional] |
|**thumbnailPath** | **String** |  |  [optional] |
|**durationSeconds** | **Integer** |  |  [optional] |
|**sizeBytes** | **Integer** |  |  [optional] |
|**startedAt** | **OffsetDateTime** | Defaults to the time the row was created when omitted. |  [optional] |
|**endedAt** | **OffsetDateTime** |  |  [optional] |



## Enum: StatusEnum

| Name | Value |
|---- | -----|
| RECORDING | &quot;recording&quot; |
| COMPLETED | &quot;completed&quot; |
| FAILED | &quot;failed&quot; |



