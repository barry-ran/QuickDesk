

# RemoteSessionDetail

One control session with the records that share its `session_id`. Both lists are capped at 100 entries and present-but-empty when there is nothing to show, so a client never has to distinguish \"no recordings\" from \"field missing\". 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**session** | [**RemoteSession**](RemoteSession.md) |  |  |
|**recordings** | [**List&lt;SessionRecording&gt;**](SessionRecording.md) |  |  |
|**accessLog** | [**List&lt;RemoteSessionAccessLog&gt;**](RemoteSessionAccessLog.md) |  |  |
|**counts** | [**RemoteSessionDetailCounts**](RemoteSessionDetailCounts.md) |  |  |



