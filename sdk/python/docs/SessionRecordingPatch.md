# SessionRecordingPatch

Partial update - every field is optional and only the ones present are written, so a producer can patch the row as the capture progresses. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**status** | **str** |  | [optional] 
**format** | **str** |  | [optional] 
**artifact_path** | **str** |  | [optional] 
**artifact_url** | **str** |  | [optional] 
**thumbnail_path** | **str** |  | [optional] 
**duration_seconds** | **int** |  | [optional] 
**size_bytes** | **int** |  | [optional] 
**ended_at** | **datetime** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.session_recording_patch import SessionRecordingPatch

# TODO update the JSON string below
json = "{}"
# create an instance of SessionRecordingPatch from a JSON string
session_recording_patch_instance = SessionRecordingPatch.from_json(json)
# print the JSON string representation of the object
print(SessionRecordingPatch.to_json())

# convert the object into a dict
session_recording_patch_dict = session_recording_patch_instance.to_dict()
# create an instance of SessionRecordingPatch from a dict
session_recording_patch_from_dict = SessionRecordingPatch.from_dict(session_recording_patch_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


