

# UpdateRelease

One build published to the self-hosted update channel. The artifact itself lives at `download_url`.

## Properties

| Name | Type | Description | Notes |
|------------ | ------------- | ------------- | -------------|
|**id** | **Long** |  |  |
|**channel** | [**ChannelEnum**](#ChannelEnum) |  |  |
|**platform** | [**PlatformEnum**](#PlatformEnum) |  |  |
|**arch** | [**ArchEnum**](#ArchEnum) |  |  |
|**version** | **String** | Dotted numeric version, e.g. &#x60;4.2.5.0&#x60;. |  |
|**downloadUrl** | **URI** | Absolute http(s) URL of the installer or archive. |  |
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



