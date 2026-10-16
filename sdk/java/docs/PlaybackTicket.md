

# PlaybackTicket

A signed, time-limited URL for one recording's media or preview. It carries no credentials, so it must be treated as a bearer token: anyone who holds the link can fetch the file until `expires_at`. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**url** | **String** | Absolute path (with the signed &#x60;t&#x60; parameter) to fetch. |  |
|**expiresAt** | **OffsetDateTime** |  |  |



