

# Device


## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**deviceId** | **String** |  |  |
|**deviceName** | **String** |  |  |
|**remark** | **String** | The owner&#39;s private note; empty when none is set. |  [optional] |
|**online** | **Boolean** | Whether the machine currently holds a live connection. |  |
|**loggedIn** | **Boolean** | True only when the device is online *and* logged in, so a parked machine is not reported as reachable. |  |
|**accessCode** | **String** | The current access code. It rotates, so treat it as short-lived. |  [optional] |
|**os** | **String** |  |  [optional] |
|**osVersion** | **String** |  |  [optional] |
|**appVersion** | **String** |  |  [optional] |
|**lastSeenAt** | **String** | RFC3339 UTC, omitted when the device has never been seen. |  [optional] |



