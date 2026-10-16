

# DeviceAccessPolicy


## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**remoteDisabled** | **Boolean** | When true, no controller may start a session at all. |  [optional] |
|**allowIps** | **List&lt;String&gt;** | Bare IPs or CIDR blocks; empty means \&quot;no IP restriction\&quot;. |  [optional] |
|**maxSessions** | **Integer** | Concurrent controller cap; 0 means unlimited, negative is rejected. |  [optional] |
|**allowedWindows** | [**List&lt;AccessWindow&gt;**](AccessWindow.md) | Recurring windows in which access is allowed; empty means always. |  [optional] |
|**timezone** | **String** | IANA zone the windows are evaluated in; empty means the server&#39;s local zone. |  [optional] |
|**exceptionDates** | **List&lt;AccessExceptionDate&gt;** | Per-date overrides (holidays, one-off closures, lunar dates). |  [optional] |
|**approvalRequired** | **Boolean** | Legacy switch kept for older clients; prefer &#x60;approval_mode&#x60;. |  [optional] |
|**approvalMode** | [**ApprovalModeEnum**](#ApprovalModeEnum) | When the owner must approve a correct code before a session starts. |  [optional] |
|**approvalTrustedIps** | **List&lt;String&gt;** | Sources exempted by &#x60;approval_mode: untrusted_ip&#x60;. |  [optional] |
|**deviceId** | **String** |  |  |
|**updatedAt** | **OffsetDateTime** | Omitted when the owner never saved a policy. |  [optional] |



## Enum: ApprovalModeEnum

| Name | Value |
|---- | -----|
| OFF | &quot;off&quot; |
| ALWAYS | &quot;always&quot; |
| GRANT | &quot;grant&quot; |
| UNTRUSTED_IP | &quot;untrusted_ip&quot; |



