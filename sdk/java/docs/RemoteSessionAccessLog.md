

# RemoteSessionAccessLog

One device access-log row that carries a session id - in practice the `session_started` entry the server wrote when the controller authenticated, which is what pairs \"an attempt\" with \"the session it became\". The full timeline is documented separately in `docs/user-api-docs.md` (§5.3.1); this is the part a session owns. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**id** | **Long** |  |  |
|**deviceId** | **String** |  |  |
|**ownerId** | **Long** | The device&#39;s owner at the time of the attempt; 0 when unbound. |  |
|**clientId** | **String** |  |  |
|**sessionId** | **String** |  |  |
|**ip** | **String** |  |  |
|**result** | **String** | The canonical access result, e.g. &#x60;session_started&#x60;. |  |
|**reason** | **String** |  |  |
|**createdAt** | **OffsetDateTime** |  |  |



