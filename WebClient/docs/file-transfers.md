# Browser file-transfer behavior

The remote-view upload picker, file drop and file paste use the existing
`filetransfer-<id>` WebRTC channels. Every task gets an ID immediately; each
DataChannelHandler serializes uploads and de-duplicates identical pending
browser file selections (name, size and lastModified). Different sessions use
different IDs in the same page. No remoting or wire-protocol changes are needed.

- Ordinary text paste is not intercepted by the file path. File paste works only
  when the browser supplies File objects in the paste event; arbitrary local
  paths and native Explorer virtual-file clipboard behavior are not supported.
- Known directory drops / directory selections are rejected. There is no folder
  traversal, automatic archive extraction or remote command execution.
- Queued cancellation sends nothing. Active cancellation sends the existing
  CANCELED error (enum 1), closes even a connecting channel, and stops later
  local reads/sends. This is a local stop, not a promise that a remote partial
  file was removed.
- Upload success requires the host Success response after End. Channel close,
  read/send errors and acknowledgement timeout never imply success.
- Downloads with File System Access support stream to the user-selected local
  destination; success is sent only after queued writes and close succeed.
  Cancelling while createWritable is pending aborts the late writable.
- Browsers without that API use a buffered Blob download, capped at 64 MiB of
  accumulated file content. Streaming pending writes are capped at 8 MiB, with
  an abort if local writes cannot keep up. These are application buffering
  bounds, not a claim about total browser/process memory.
- Metadata size is advisory in the existing protocol. End followed by successful
  local finalization is authoritative. File size varints use safe integer math
  rather than 32-bit bitwise shifts; sizes above JS safe integer range fail.
- Filenames and transfer errors are displayed as text, never injected as HTML.

This change targets the remote-view file transfer channel. The separate skill-
bridge file-manager panel is not changed.

## Tests

`npm test` includes actual DataChannelHandler tests with fake channels and async
file/disk primitives, including cancellation races, stale callbacks, queued
same-name tasks, read failures, 64-bit metadata, advisory size, browser fallback
limits and download close errors.

`npm run test:e2e:file-transfer` runs Chromium with the actual remote-view UI
handlers, browser File/DataTransfer/paste events, and mocked peer channels.
Session/auth is stubbed; no real host connection or external file transfer occurs.
The regular WebClient workflow runs unit tests, build, existing mobile browser
tests and this new transfer browser test.

Manual validation with the packaged native host remains necessary for the
remote destination/overwrite and partial-file cleanup behavior, which belongs
to the unchanged host implementation.
