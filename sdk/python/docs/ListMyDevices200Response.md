# ListMyDevices200Response


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[Device]**](Device.md) |  | 

## Example

```python
from quickdesk_signaling.models.list_my_devices200_response import ListMyDevices200Response

# TODO update the JSON string below
json = "{}"
# create an instance of ListMyDevices200Response from a JSON string
list_my_devices200_response_instance = ListMyDevices200Response.from_json(json)
# print the JSON string representation of the object
print(ListMyDevices200Response.to_json())

# convert the object into a dict
list_my_devices200_response_dict = list_my_devices200_response_instance.to_dict()
# create an instance of ListMyDevices200Response from a dict
list_my_devices200_response_from_dict = ListMyDevices200Response.from_dict(list_my_devices200_response_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


