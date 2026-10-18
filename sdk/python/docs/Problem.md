# Problem

RFC 7807 problem details, served as `application/problem+json`.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**type** | **str** |  | 
**title** | **str** |  | 
**status** | **int** |  | 
**detail** | **str** |  | [optional] 
**instance** | **str** |  | [optional] 
**code** | **str** | A stable machine-readable code, e.g. &#x60;INVALID_CODE&#x60; or &#x60;RATE_LIMITED&#x60;. | 
**trace_id** | **str** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.problem import Problem

# TODO update the JSON string below
json = "{}"
# create an instance of Problem from a JSON string
problem_instance = Problem.from_json(json)
# print the JSON string representation of the object
print(Problem.to_json())

# convert the object into a dict
problem_dict = problem_instance.to_dict()
# create an instance of Problem from a dict
problem_from_dict = Problem.from_dict(problem_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


