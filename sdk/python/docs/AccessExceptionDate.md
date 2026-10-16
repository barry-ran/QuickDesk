# AccessExceptionDate

Exactly one of `date` (+ optional `end_date`), `holiday`, or `repeat: \"lunar\"` with the lunar month/day. `mode` is `allow` or `deny`; an allow and a deny entry covering the same day are rejected. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**var_date** | **str** | First date of the exception (YYYY-MM-DD). | [optional] 
**end_date** | **str** | Last date of a range (YYYY-MM-DD); omit for a single day. | [optional] 
**mode** | **str** |  | 
**repeat** | **str** |  | [optional] 
**holiday** | **str** | A statutory holiday name from the server&#39;s calendar, e.g. 春节. | [optional] 
**holiday_day** | **str** | Which dates of the holiday an entry covers (default &#x60;off&#x60;). | [optional] 
**lunar_month** | **int** |  | [optional] 
**lunar_day** | **int** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.access_exception_date import AccessExceptionDate

# TODO update the JSON string below
json = "{}"
# create an instance of AccessExceptionDate from a JSON string
access_exception_date_instance = AccessExceptionDate.from_json(json)
# print the JSON string representation of the object
print(AccessExceptionDate.to_json())

# convert the object into a dict
access_exception_date_dict = access_exception_date_instance.to_dict()
# create an instance of AccessExceptionDate from a dict
access_exception_date_from_dict = AccessExceptionDate.from_dict(access_exception_date_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


