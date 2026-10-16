# UpdateRelease

One build published to the self-hosted update channel. The artifact itself lives at `download_url`.

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**id** | **int** |  | 
**channel** | **str** |  | 
**platform** | **str** |  | 
**arch** | **str** |  | 
**version** | **str** | Dotted numeric version, e.g. &#x60;4.2.5.0&#x60;. | 
**download_url** | **str** | Absolute http(s) URL of the installer or archive. | 
**sha256** | **str** | Lowercase hex SHA-256 of the artifact; clients verify what they downloaded. | 
**size_bytes** | **int** |  | 
**notes** | **str** |  | [optional] 
**mandatory** | **bool** | When true, the update is not optional for older clients. | 
**published_at** | **datetime** |  | 

## Example

```python
from quickdesk_signaling.models.update_release import UpdateRelease

# TODO update the JSON string below
json = "{}"
# create an instance of UpdateRelease from a JSON string
update_release_instance = UpdateRelease.from_json(json)
# print the JSON string representation of the object
print(UpdateRelease.to_json())

# convert the object into a dict
update_release_dict = update_release_instance.to_dict()
# create an instance of UpdateRelease from a dict
update_release_from_dict = UpdateRelease.from_dict(update_release_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


