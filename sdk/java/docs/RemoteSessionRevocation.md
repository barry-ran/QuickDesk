

# RemoteSessionRevocation

The result of ending one live control session. `revoked` is always `true` - the endpoint answers `409` rather than returning a false here, so a caller never has to interpret a success response as \"maybe\". 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**sessionId** | **String** |  |  |
|**deviceId** | **String** |  |  |
|**revoked** | **Boolean** |  |  |



