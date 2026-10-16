# SessionRecordingInput

The index row a producer creates before (or instead of) uploading media. `device_id` is recorded as a label; it is not checked against the caller's devices, because a headless producer may be indexing a capture for a machine it is not bound to. 

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**device_id** | **str** | The device the capture came from. | 
**device_name** | **str** |  | [optional] 
**session_id** | **str** | The control session this recording belongs to, when there is one. | [optional] 
**status** | **str** | How far the capture got when this row was written. | 
**format** | **str** | Free-form label; the upload route overwrites it from the file extension. | [optional] 
**artifact_path** | **str** | Server-side path. Set by the upload route; clients do not need to send it. | [optional] 
**artifact_url** | **str** |  | [optional] 
**thumbnail_path** | **str** |  | [optional] 
**duration_seconds** | **int** |  | [optional] 
**size_bytes** | **int** |  | [optional] 
**started_at** | **datetime** | Defaults to the time the row was created when omitted. | [optional] 
**ended_at** | **datetime** |  | [optional] 

## Example

```python
from quickdesk_signaling.models.session_recording_input import SessionRecordingInput

# TODO update the JSON string below
json = "{}"
# create an instance of SessionRecordingInput from a JSON string
session_recording_input_instance = SessionRecordingInput.from_json(json)
# print the JSON string representation of the object
print(SessionRecordingInput.to_json())

# convert the object into a dict
session_recording_input_dict = session_recording_input_instance.to_dict()
# create an instance of SessionRecordingInput from a dict
session_recording_input_from_dict = SessionRecordingInput.from_dict(session_recording_input_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


