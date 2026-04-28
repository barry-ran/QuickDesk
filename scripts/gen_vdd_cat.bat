@echo off
REM ---------------------------------------------------------------
REM Generate .cat catalog for signed VDD driver
REM
REM Run this AFTER signing quickdesk_display.dll with your EV cert.
REM The generated .cat must then also be signed with the same cert.
REM
REM Usage: gen_vdd_cat.bat [prebuilt dir]
REM   Default dir: quickdesk-virtual-display\prebuilt\x64
REM
REM Requires: WDK (inf2cat.exe)
REM ---------------------------------------------------------------

echo=
echo ---------------------------------------------------------------
echo Generate VDD Catalog (.cat)
echo ---------------------------------------------------------------

set script_path=%~dp0
set driver_dir=%~1
if "%driver_dir%"=="" set driver_dir=%script_path%..\quickdesk-virtual-display\prebuilt\x64

echo [*] driver dir: %driver_dir%

:: check required files exist
if not exist "%driver_dir%\quickdesk_display.dll" (
    echo [!] error: quickdesk_display.dll not found in %driver_dir%
    exit /b 1
)
if not exist "%driver_dir%\quickdesk_display.inf" (
    echo [!] error: quickdesk_display.inf not found in %driver_dir%
    exit /b 1
)

:: find inf2cat.exe
set INF2CAT=
for /f "delims=" %%i in ('where inf2cat.exe 2^>nul') do set INF2CAT=%%i
if "%INF2CAT%"=="" (
    if exist "C:\Program Files (x86)\Windows Kits\10\bin\10.0.26100.0\x86\inf2cat.exe" (
        set "INF2CAT=C:\Program Files (x86)\Windows Kits\10\bin\10.0.26100.0\x86\inf2cat.exe"
    )
)
if "%INF2CAT%"=="" (
    echo [!] error: inf2cat.exe not found. Install WDK first.
    exit /b 1
)
echo [*] inf2cat: %INF2CAT%

:: delete old .cat
if exist "%driver_dir%\quickdesk_display.cat" (
    del /q "%driver_dir%\quickdesk_display.cat"
    echo [*] removed old .cat
)

:: generate catalog
echo [*] generating catalog...
"%INF2CAT%" /os:10_x64 /driver:"%driver_dir%"
if %errorlevel% neq 0 (
    echo [!] inf2cat failed with error %errorlevel%
    exit /b 1
)

if exist "%driver_dir%\quickdesk_display.cat" (
    echo [*] catalog generated: %driver_dir%\quickdesk_display.cat
    echo=
    echo [!] IMPORTANT: Now sign the .cat with your EV certificate:
    echo     signtool sign /fd sha256 /tr http://timestamp.digicert.com /td sha256 /sha1 ^<thumbprint^> "%driver_dir%\quickdesk_display.cat"
) else (
    echo [!] error: .cat file was not generated
    exit /b 1
)
