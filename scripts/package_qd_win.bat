@echo off

echo=
echo=
echo ---------------------------------------------------------------
echo check ENV
echo ---------------------------------------------------------------

:: get script absolute path
set script_path=%~dp0
set old_cd=%cd%
cd /d %~dp0

SETLOCAL EnableDelayedExpansion
set build_mode=Release
set errno=1

echo=
echo=
echo ---------------------------------------------------------------
echo parse arguments
echo ---------------------------------------------------------------

:parse_args
if "%1"=="" goto args_done
if /i "%1"=="debug" set build_mode=Debug
if /i "%1"=="release" set build_mode=Release
shift
goto parse_args
:args_done

echo [*] build mode: %build_mode%
echo=

set publish_path=%script_path%..\publish\%build_mode%
set installer_script=%script_path%installer\quickdesk.iss
set icon_path=%script_path%..\QuickDesk\res\QuickDesk.ico
set output_path=%script_path%..\publish

:: read version from file
set /p app_version=<"%script_path%..\version"

echo [*] publish path: %publish_path%
echo [*] installer script: %installer_script%
echo [*] icon path: %icon_path%
echo [*] output path: %output_path%
echo [*] app version: %app_version%
echo=

:: check if publish path exists
if not exist "%publish_path%" (
    echo [!] error: publish path does not exist: %publish_path%
    echo [!] please run publish_qd_win.bat %build_mode% first
    goto return
)

echo [*] verifying Windows publish dependencies...
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%script_path%verify_qd_win_release.ps1" -PackageDir "%publish_path%"
if not %errorlevel%==0 (
    echo [!] error: Windows publish dependency verification failed
    goto return
)
echo [*] Windows publish dependency verification passed

echo=
echo=
echo ---------------------------------------------------------------
echo create installer
echo ---------------------------------------------------------------

:: try to find Inno Setup compiler
set iscc_path=
if exist "C:\Program Files (x86)\Inno Setup 6\ISCC.exe" (
    set "iscc_path=C:\Program Files (x86)\Inno Setup 6\ISCC.exe"
) else if exist "C:\Program Files\Inno Setup 6\ISCC.exe" (
    set "iscc_path=C:\Program Files\Inno Setup 6\ISCC.exe"
) else (
    where iscc >nul 2>nul
    if !errorlevel!==0 (
        set "iscc_path=iscc"
    ) else (
        echo [!] error: Inno Setup 6 not found
        echo [!] please install Inno Setup 6 or add ISCC.exe to PATH
        goto return
    )
)

echo [*] ISCC: %iscc_path%
echo [*] building installer...
echo=

"%iscc_path%" /DMyAppVersion=%app_version% /DMyPublishDir=%publish_path% /DMyOutputDir=%output_path% /DMyIconPath=%icon_path% "%installer_script%"
if not %errorlevel%==0 (
    echo [!] Inno Setup build failed
    goto return
)

echo=
echo=
echo ---------------------------------------------------------------
echo [*] installer created!
echo ---------------------------------------------------------------
echo [*] output: %output_path%\QuickDesk-win-x64-setup.exe
echo=

set errno=0

:return
cd /d "%old_cd%"
exit /B %errno%

ENDLOCAL
