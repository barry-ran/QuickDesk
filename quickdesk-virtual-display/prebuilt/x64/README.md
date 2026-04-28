# Prebuilt Virtual Display Driver

Place the following files here after building and signing:

- `quickdesk_display.dll` — UMDF IDD driver binary (EV signed)
- `quickdesk_display.inf` — Driver installation information file
- `quickdesk_display.cat` — Driver catalog (EV signed, generated from signed DLL + INF)
- `nefconw.exe` — Nefarius device node management tool

These files are bundled into the Windows installer by `publish_qd_win.bat`.

## How to update

```bat
REM 1. Build the driver
scripts\build_vdd_win.bat Release
xcopy /Y output\x64\Release\drivers\vdd\* quickdesk-virtual-display\prebuilt\x64\

REM 2. Sign the DLL with your EV certificate
signtool sign /fd sha256 /tr http://timestamp.digicert.com /td sha256 /sha1 <thumbprint> quickdesk-virtual-display\prebuilt\x64\quickdesk_display.dll

REM 3. Regenerate .cat (must be done AFTER signing DLL)
scripts\gen_vdd_cat.bat

REM 4. Sign the .cat with the same EV certificate
signtool sign /fd sha256 /tr http://timestamp.digicert.com /td sha256 /sha1 <thumbprint> quickdesk-virtual-display\prebuilt\x64\quickdesk_display.cat
```

Then commit the updated binaries.
