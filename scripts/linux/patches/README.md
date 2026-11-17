# quickdesk-remoting Linux patch

`quickdesk-remoting-linux.patch` is the working-tree diff that produced the
Linux x64 core binaries shipped in `output/core-linux-x64/` and bundled in the
AppImage.

Base commit: `barry-ran/quickdesk-remoting` `0a502e844e10081c11c0a81eb40171466de252e4`
("feat: 清理quickdesk中google相关代码").

The patch ports the QuickDesk core to Chromium 140 (the tree shipped in the
`v1.0.4` release): unsafe-buffer annotations, `URLLoaderFactoryParams` as a
mojom `StructPtr`, `AttachStringForUpload` for JSON bodies, an IO message pump
for the client, off-network-thread host state persistence, and `X-API-Key` on
all `/v1` requests. It also carries the native-messaging client and shared
memory frame consumer sources, plus the `quickdesk-skill` host extension that
bridges skill messages between the Qt layer and a connected client
(`remoting/quickdesk/host/quickdesk_skill_extension.*`).

Additional fixes carried by this patch:

- **Clipboard MIME**: host `OnSessionClipboard` now sets
  `kMimeTypeTextUtf8` before writing the text, so `clipboard_sync` is not
  dropped as unsupported (previously host→client clipboard silently failed).
- **Clipboard initial push**: `ClipboardX11::Start()` calls
  `XServerClipboard::RequestClipboardContent()`, so a Linux host sends its
  current clipboard as soon as a client session starts. Without it the host
  only pushed on `XFixes` owner changes, and `get_clipboard` stayed empty until
  the host user copied something. Windows/macOS official cores do not have
  this change and still require a copy on the host after connecting.
- **Client crash on stream removal**: `WebrtcVideoRendererAdapter`'s
  destructor indexes `GetVideoTracks()[0]` after a `DCHECK` that release
  builds compile out. During host renegotiation the media stream can be
  removed after its video track, so the empty vector indexed out of bounds and
  libc++ hardening aborted the client process. Both the destructor and
  `SetMediaStream` now bail out on an empty track list. This is a
  cross-platform core bug; Windows/macOS builds using the official
  `quickdesk_core.dll` / `libremoting_core.so` can still hit it.
- **Frame size clamp**: `SharedMemoryFrameConsumer::EnsureInitialized()`
  rejects frames whose I420 size exceeds `kMaxPackedSize` in addition to the
  per-dimension `kMaxDimension` check, so a hostile/buggy frame cannot write
  past the shared-memory mapping.
- **DCHECK off**: `shared_memory_frame_consumer.h` marks
  `sequence_checker_` `[[maybe_unused]]` so the tree compiles with
  `dcheck_always_on = false`. Keep DCHECKs off for release builds: an
  always-on DCHECK build aborts on the second client session
  (`network_service.cc: !g_network_service`) and on a second host viewer
  (`quickdesk_host.cc: kConnecting == state_`), which end users would see as
  a crash, not an assertion.

## Skill capability interop

The skill bridge only works when **both** endpoints advertise the
`qd-skill-bridge` capability after negotiation. This patch registers
`QuickDeskSkillExtension` in `quickdesk_host.cc`, so Linux hosts built from it
advertise the capability.

The official Windows/macOS remoting cores (`quickdesk_core.dll` /
`libremoting_core.so` from the `v1.0.4` release) announce an empty extension
capability slot (their capability string has a double space where
`qd-skill-bridge` would be). They contain `SkillBridgeExtension` code but never
register it, and the public source at base commit `0a502e844e` has no
registration either. Result: sessions to those hosts negotiate no skill
capability and `skillExec` / `skillListTools` time out (the MCP reports this
with a `hint`). There is no client-side workaround — a Windows/macOS core
build that registers the extension is required. Until then use `device_id:
"local"` (runs on the MCP host machine) or a Linux remote.

## Apply

```sh
cd <chromium-src>
git apply /path/to/quickdesk-remoting-linux.patch
```

## Build

```sh
cd <chromium-src>
gn gen out/qd --args='
  is_debug=false
  dcheck_always_on=false
  symbol_level=0
  target_cpu="x64"
  proprietary_codecs=true
  ffmpeg_branding="Chrome"
  enable_nacl=false'
ninja -C out/qd remoting/quickdesk:quickdesk_all
```

Ship `out/qd/quickdesk_host`, `out/qd/quickdesk_client`,
`out/qd/libremoting_core.so` (the host links it) and `out/qd/icudtl.dat`
together. Package the AppImage with `scripts/linux/package_appimage.sh`
(`CORE_DIR` points at that directory).
