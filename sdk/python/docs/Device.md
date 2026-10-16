# Device


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **str** |  | 
**device_name** | **str** |  | 
**remark** | **str** | The owner&#39;s private note; empty when none is set. | [optional] 
**online** | **bool** | Whether the machine currently holds a live connection. | 
**logged_in** | **bool** | True only when the device is online *and* logged in, so a parked machine is not reported as reachable. | 
**access_code** | **str** | The current access code. It rotates, so treat it as short-lived. | [optional] 
**os** | **str** |  | [optional] 
**os_version** | **str** |  | [optional] 
**app_version** | **str** |  | [optional] 
**last_seen_at** | **str** | RFC3339 UTC, omitted when the device has never been seen. | [optional] 

## Example

```python
from quickdesk_signaling.models.device import Device

# TODO update the JSON string below
json = "{}"
# create an instance of Device from a JSON string
device_instance = Device.from_json(json)
# print the JSON string representation of the object
print(Device.to_json())

# convert the object into a dict
device_dict = device_instance.to_dict()
# create an instance of Device from a dict
device_from_dict = Device.from_dict(device_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


