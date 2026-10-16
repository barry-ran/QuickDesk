

# ApprovalSummary


## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**since** | **OffsetDateTime** | Inclusive start of the window the counts cover. |  |
|**total** | **Long** | Every request created inside the window, whatever its outcome. |  |
|**byStatus** | [**ApprovalSummaryByStatus**](ApprovalSummaryByStatus.md) |  |  |
|**deniedSources** | [**List&lt;ApprovalSummaryDeniedSourcesInner&gt;**](ApprovalSummaryDeniedSourcesInner.md) | Refused sources inside the window, most refusals first. |  |
|**denialCooldownSeconds** | **Integer** | The live APPROVAL_DENIAL_COOLDOWN_SECONDS, so an owner can tell \&quot;refused right now\&quot; from \&quot;refused a while ago\&quot;. |  |
|**denialCooldownMultiplier** | **Integer** | The live APPROVAL_DENIAL_COOLDOWN_MULTIPLIER; 1 means each further consecutive refusal costs the same as the last. |  |
|**denialCooldownMaxSeconds** | **Integer** | The live APPROVAL_DENIAL_COOLDOWN_MAX_SECONDS, the ceiling the escalation cannot pass; 0 means uncapped. |  |



