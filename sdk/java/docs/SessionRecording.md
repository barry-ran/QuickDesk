

# SessionRecording

One indexed recording. The row is the control-plane metadata; the media artifact is produced by the control engine and fetched through the artifact routes. `artifact_path` is a server-side path and is not a download URL - clients should call `.../artifact-ticket` and use the URL they get back. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**id** | **Integer** |  |  |
|**userId** | **Integer** |  |  |
|**deviceId** | **String** |  |  |
|**deviceName** | **String** |  |  |
|**sessionId** | **String** |  |  |
|**status** | [**StatusEnum**](#StatusEnum) | &#x60;recording&#x60; while the capture is in progress, then &#x60;completed&#x60; or &#x60;failed&#x60;. A stale &#x60;recording&#x60; row is swept to &#x60;failed&#x60; by the recording lifecycle worker.  |  |
|**format** | **String** | Container/codec hint, e.g. &#x60;mp4&#x60;, &#x60;h264&#x60;, &#x60;grf&#x60;. |  |
|**artifactPath** | **String** | Server-side stored path; not a client-fetchable URL. |  |
|**artifactUrl** | **String** |  |  |
|**thumbnailPath** | **String** |  |  |
|**thumbnailUrl** | **String** |  |  |
|**durationSeconds** | **Integer** |  |  |
|**sizeBytes** | **Long** |  |  |
|**startedAt** | **OffsetDateTime** |  |  |
|**endedAt** | **OffsetDateTime** |  |  [optional] |
|**createdAt** | **OffsetDateTime** |  |  |
|**updatedAt** | **OffsetDateTime** |  |  |
|**user** | **Object** | The owning user, present when the server eager-loads it. Optional - clients must not depend on this being present.  |  [optional] |



## Enum: StatusEnum

| Name | Value |
|---- | -----|
| RECORDING | &quot;recording&quot; |
| COMPLETED | &quot;completed&quot; |
| FAILED | &quot;failed&quot; |



