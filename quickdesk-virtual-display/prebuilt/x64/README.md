# Prebuilt Virtual Display Driver

Place the following files here after building locally with `scripts\build_vdd_win.bat Release`:

- `quickdesk_display.dll` — UMDF IDD driver binary
- `quickdesk_display.inf` — Driver installation information file
- `quickdesk_display.cat` — Driver catalog (signed)
- `nefconw.exe` — Nefarius device node management tool

These files are bundled into the Windows installer by `publish_qd_win.bat`.

## How to update

```bat
scripts\build_vdd_win.bat Release
xcopy /Y output\x64\Release\drivers\vdd\* quickdesk-virtual-display\prebuilt\x64\
```

Then commit the updated binaries.
