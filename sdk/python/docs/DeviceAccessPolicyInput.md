# DeviceAccessPolicyInput

Full replace: fields not sent are reset to their zero value. 

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

## Example

```python
from quickdesk_signaling.models.device_access_policy_input import DeviceAccessPolicyInput

# TODO update the JSON string below
json = "{}"
# create an instance of DeviceAccessPolicyInput from a JSON string
device_access_policy_input_instance = DeviceAccessPolicyInput.from_json(json)
# print the JSON string representation of the object
print(DeviceAccessPolicyInput.to_json())

# convert the object into a dict
device_access_policy_input_dict = device_access_policy_input_instance.to_dict()
# create an instance of DeviceAccessPolicyInput from a dict
device_access_policy_input_from_dict = DeviceAccessPolicyInput.from_dict(device_access_policy_input_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


