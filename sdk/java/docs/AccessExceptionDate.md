

# AccessExceptionDate

Exactly one of `date` (+ optional `end_date`), `holiday`, or `repeat: \"lunar\"` with the lunar month/day. `mode` is `allow` or `deny`; an allow and a deny entry covering the same day are rejected. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**date** | **String** | First date of the exception (YYYY-MM-DD). |  [optional] |
|**endDate** | **String** | Last date of a range (YYYY-MM-DD); omit for a single day. |  [optional] |
|**mode** | [**ModeEnum**](#ModeEnum) |  |  |
|**repeat** | [**RepeatEnum**](#RepeatEnum) |  |  [optional] |
|**holiday** | **String** | A statutory holiday name from the server&#39;s calendar, e.g. 春节. |  [optional] |
|**holidayDay** | [**HolidayDayEnum**](#HolidayDayEnum) | Which dates of the holiday an entry covers (default &#x60;off&#x60;). |  [optional] |
|**lunarMonth** | **Integer** |  |  [optional] |
|**lunarDay** | **Integer** |  |  [optional] |



## Enum: ModeEnum

| Name | Value |
|---- | -----|
| ALLOW | &quot;allow&quot; |
| DENY | &quot;deny&quot; |



## Enum: RepeatEnum

| Name | Value |
|---- | -----|
| YEARLY | &quot;yearly&quot; |
| LUNAR | &quot;lunar&quot; |



## Enum: HolidayDayEnum

| Name | Value |
|---- | -----|
| OFF | &quot;off&quot; |
| WORK | &quot;work&quot; |



