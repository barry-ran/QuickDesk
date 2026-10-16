

# UpdateRelease

One build published to the self-hosted update channel. The artifact itself is always fetchable at `download_url`: either an absolute URL on a host the operator controls, or this server's own `GET /v1/update-releases/{release_id}/artifact` when the installer was uploaded here (`hosted_here` is true). A release with no fetchable bytes is a placeholder and is never returned to a client. 

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**id** | **Long** |  |  |
|**channel** | [**ChannelEnum**](#ChannelEnum) |  |  |
|**platform** | [**PlatformEnum**](#PlatformEnum) |  |  |
|**arch** | [**ArchEnum**](#ArchEnum) |  |  |
|**version** | **String** | Dotted numeric version, e.g. &#x60;4.2.5.0&#x60;. |  |
|**downloadUrl** | **URI** | Absolute http(s) URL of the installer or archive. |  |
|**hostedHere** | **Boolean** | True when this server stores and serves the bytes (so &#x60;download_url&#x60; points at &#x60;GET /v1/update-releases/{release_id}/artifact&#x60;), false when the installer lives on a host the operator manages. |  |
|**sha256** | **String** | Lowercase hex SHA-256 of the artifact; clients verify what they downloaded. |  |
|**sizeBytes** | **Long** |  |  |
|**notes** | **String** |  |  [optional] |
|**mandatory** | **Boolean** | When true, the update is not optional for older clients. |  |
|**publishedAt** | **OffsetDateTime** |  |  |



## Enum: ChannelEnum

| Name | Value |
|---- | -----|
| STABLE | &quot;stable&quot; |
| BETA | &quot;beta&quot; |
| NIGHTLY | &quot;nightly&quot; |



## Enum: PlatformEnum

| Name | Value |
|---- | -----|
| WINDOWS | &quot;windows&quot; |
| MACOS | &quot;macos&quot; |
| LINUX | &quot;linux&quot; |



## Enum: ArchEnum

| Name | Value |
|---- | -----|
| X64 | &quot;x64&quot; |
| ARM64 | &quot;arm64&quot; |
| UNIVERSAL | &quot;universal&quot; |



