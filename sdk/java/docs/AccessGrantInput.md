

# AccessGrantInput


## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**label** | **String** |  |  [optional] |
|**expiresInMinutes** | **Integer** |  |  |
|**maxUses** | **Integer** | 0 means \&quot;no use limit\&quot;; the grant still expires. |  [optional] |
|**code** | **String** | Optional caller-chosen code; omitted means the server generates one. |  [optional] |
|**boundClientId** | **String** | Self-reported controller identity the code is bound to. |  [optional] |
|**boundIps** | **List&lt;String&gt;** |  |  [optional] |



