

# RemoteSession

One controller holding, or having held, a signaling session against a device. `active` and `duration_seconds` are derived: the first from `ended_at`, the second is the run length so far while the session is still open, so a client never has to pick its own notion of \"now\". 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**id** | **Long** |  |  |
|**sessionId** | **String** | The signaling session id the server assigned; unique per session. |  |
|**userId** | **Long** | The device&#39;s owner. 0 when the device was unbound at the time. |  |
|**deviceId** | **String** |  |  |
|**clientId** | **String** | The controller-declared client id, the same value the device access log records. |  |
|**clientIp** | **String** |  |  |
|**startedAt** | **OffsetDateTime** |  |  |
|**endedAt** | **OffsetDateTime** |  |  |
|**endReason** | **String** | Empty while the session is open; &#x60;disconnected&#x60; when the signaling socket went away on its own, &#x60;revoked&#x60; when the owner or an administrator closed it on purpose, &#x60;server_restart&#x60; for rows a restart found still open. |  |
|**active** | **Boolean** |  |  |
|**durationSeconds** | **Long** |  |  |
|**createdAt** | **OffsetDateTime** |  |  |
|**updatedAt** | **OffsetDateTime** |  |  |



