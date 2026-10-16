

# NotificationChannel


## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**id** | **Long** |  |  |
|**userId** | **Long** |  |  |
|**name** | **String** |  |  |
|**url** | **URI** |  |  |
|**events** | **String** | A JSON-encoded array of subscribed event types (a string, not an array, because that is how the column is stored). Decode it before matching against &#x60;supported_events&#x60;.  |  |
|**enabled** | **Boolean** |  |  |
|**lastTriggered** | **OffsetDateTime** |  |  [optional] |
|**lastStatus** | **Integer** | HTTP status of the most recent delivery attempt; 0 means none yet. |  |
|**createdAt** | **OffsetDateTime** |  |  [optional] |
|**updatedAt** | **OffsetDateTime** |  |  [optional] |



