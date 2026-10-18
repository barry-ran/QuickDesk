# DeviceAccessPolicy


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**remote_disabled** | **bool** | When true, no controller may start a session at all. | [optional] 
**allow_ips** | **List[str]** | Bare IPs or CIDR blocks; empty means \&quot;no IP restriction\&quot;. | [optional] 
**max_sessions** | **int** | Concurrent controller cap; 0 means unlimited, negative is rejected. | [optional] 
**allowed_windows** | [**List[AccessWindow]**](AccessWindow.md) | Recurring windows in which access is allowed; empty means always. | [optional] 
**timezone** | **str** | IANA zone the windows are evaluated in; empty means the server&#39;s local zone. | [optional] 
**exception_dates** | [**List[AccessExceptionDate]**](AccessExceptionDate.md) | Per-date overrides (holidays, one-off closures, lunar dates). | [optional] 
**approval_required** | **bool** | Legacy switch kept for older clients; prefer &#x60;approval_mode&#x60;. | [optional] 
**approval_mode** | **str** | When the owner must approve a correct code before a session starts. | [optional] 
**approval_trusted_ips** | **List[str]** | Sources exempted by &#x60;approval_mode: untrusted_ip&#x60;. | [optional] 
**device_id** | **str** |  | 
**updated_at** | **datetime** | Omitted when the owner never saved a policy. | [optional] 

## Example

```python
from quickdesk_signaling.models.device_access_policy import DeviceAccessPolicy

# TODO update the JSON string below
json = "{}"
# create an instance of DeviceAccessPolicy from a JSON string
device_access_policy_instance = DeviceAccessPolicy.from_json(json)
# print the JSON string representation of the object
print(DeviceAccessPolicy.to_json())

# convert the object into a dict
device_access_policy_dict = device_access_policy_instance.to_dict()
# create an instance of DeviceAccessPolicy from a dict
device_access_policy_from_dict = DeviceAccessPolicy.from_dict(device_access_policy_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


