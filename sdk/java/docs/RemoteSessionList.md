

# RemoteSessionList

Cursor page of control sessions. `next_cursor` is absent on the last page and `total` is deliberately absent, the same as every other paginated list here: the endpoint does not count the whole set per page. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**items** | [**List&lt;RemoteSession&gt;**](RemoteSession.md) |  |  |
|**nextCursor** | **String** |  |  [optional] |



