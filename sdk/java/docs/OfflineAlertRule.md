

# OfflineAlertRule


## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**deviceId** | **String** |  |  |
|**enabled** | **Boolean** |  |  |
|**afterMinutes** | **Integer** |  |  |
|**repeatMinutes** | **Integer** |  |  |
|**snoozedUntil** | **OffsetDateTime** |  |  [optional] |
|**maintenanceWindows** | [**List&lt;MaintenanceWindow&gt;**](MaintenanceWindow.md) |  |  |
|**maintenanceTimezone** | **String** |  |  [optional] |
|**inMaintenanceWindow** | **Boolean** | Computed at read time, so a client can explain a quiet period without re-implementing the schedule. |  |
|**offlineSince** | **OffsetDateTime** | The evaluator&#39;s episode state; present only while the device is in an outage. |  [optional] |
|**lastAlertedAt** | **OffsetDateTime** |  |  [optional] |
|**updatedAt** | **OffsetDateTime** |  |  [optional] |



