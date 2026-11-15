# Building QuickDesk on Linux

The Qt client builds and runs on Linux (X11 session or Xwayland). The remoting
core (host/client binaries) is **not** published as a Linux archive yet, so it has
to be provided once before the app can host or connect.

## Prerequisites

- Debian 12 / Ubuntu 22.04 or newer, CMake >= 3.21, GCC >= 12, Ninja or Make.
- Qt 6.8 (Quick, QuickControls2, Multimedia, WebSockets, Sql, Svg).
  With [aqtinstall](https://github.com/miurahr/aqtinstall):
  `python -m aqt install-qt linux desktop 6.8.3 linux_gcc_64 -O ~/Qt`
- Optional, same as on other platforms: Rust for `quickdesk-mcp`, Go for
  `SignalingServer`, Node.js for `WebClient`.

## Remoting core

`cmake/quickdesk-remoting.cmake` prefers a core that is already on disk and only
downloads one on Windows/macOS. Place the Linux build here:

```
QuickDesk/3rdparty/quickdesk-remoting/x64/
    quickdesk_host
    quickdesk_client
    icudtl.dat
    *.so            # e.g. libremoting_core.so, if your build produces one
```

Build it from [quickdesk-remoting](https://github.com/barry-ran/quickdesk-remoting).
Without a core the Qt shell still configures and builds, but it cannot host or
open sessions.

## Build

```sh
cmake -S . -B build -DCMAKE_PREFIX_PATH="$HOME/Qt/6.8.3/gcc_64" -DCMAKE_BUILD_TYPE=Release
cmake --build build -j"$(nproc)"
```

The binary ends up in `build/QuickDesk` (or `output/x64/Release/QuickDesk` in the
in-tree layout used by the packaging script).

## Package an AppImage

```sh
CORE_DIR=/path/to/Linux-x64 bash scripts/linux/package_appimage.sh
```

`scripts/linux/package_appimage.sh` expects `CORE_DIR` to contain the core files
above; it bundles the core, the MCP bridge, the skill host and RapidOCR into
`output/appimage/QuickDesk-x86_64.AppImage`.

The generated `AppRun` does three things and they are deliberate:

- forces `QT_QPA_PLATFORM=xcb`, because no Wayland platform plugin is bundled;
- unsets `WAYLAND_DISPLAY` / `XDG_SESSION_TYPE` when running through Xwayland,
  because the host captures the desktop through X11 and a Wayland session marker
  makes Chromium pick a capture path that produces no frames;
- keeps the AppImage path as `argv[0]`, because `QStandardPaths` derives the
  data directory from it — settings must survive between launches.

## Running

```sh
./output/appimage/QuickDesk-x86_64.AppImage
```

Logs and configuration live in `~/.local/share/QuickDesk-x86_64.AppImage/`.

## Known limitation: Wayland capture

The host captures via X11. On a KDE/GNOME Wayland desktop, Xwayland's root window
has no content, so a session connects and negotiates the display size but the
picture stays black. To validate frames, run the host in an X11 session or under
Xvfb (`Xvfb :99 -screen 0 1920x1080x24 &` with `DISPLAY=:99`).
