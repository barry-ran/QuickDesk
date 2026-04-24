@echo off
REM ---------------------------------------------------------------
REM Build QuickDesk Virtual Display Driver (IDD)
REM Requires: WDK 11, VS 2022, EWDK (optional)
REM Usage: build_vdd_win.bat [Release|Debug]
REM ---------------------------------------------------------------

echo=
echo ---------------------------------------------------------------
echo Build QuickDesk Virtual Display Driver
echo ---------------------------------------------------------------

:: get script absolute path
set script_path=%~dp0
set old_cd=%cd%
cd /d %~dp0

SETLOCAL EnableDelayedExpansion
set build_mode=Release
set errno=1

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

set vdd_project=%script_path%..\quickdesk-virtual-display
set vdd_sln=%vdd_project%\quickdesk-virtual-display.sln
set output_path=%script_path%..\output\x64\%build_mode%\drivers\vdd

echo [*] project path: %vdd_project%
echo [*] solution: %vdd_sln%
echo [*] output path: %output_path%
echo=

:: check if solution exists
if not exist "%vdd_sln%" (
    echo [!] error: solution file not found: %vdd_sln%
    echo [!] the IDD driver project has not been set up yet.
    echo [!] please fork RustDeskIddDriver into quickdesk-virtual-display/ first.
    goto return
)

echo=
echo ---------------------------------------------------------------
echo build driver
echo ---------------------------------------------------------------

msbuild "%vdd_sln%" /p:Configuration=%build_mode% /p:Platform=x64 /m

if %errorlevel% neq 0 (
    echo [!] driver build failed with error %errorlevel%
    goto return
)

echo=
echo ---------------------------------------------------------------
echo copy artifacts
echo ---------------------------------------------------------------

if not exist "%output_path%" (
    echo [*] creating output dir: %output_path%
    mkdir "%output_path%"
)

:: copy driver files (UMDF IDD driver outputs .dll, not .sys)
echo [*] copying driver files...
set driver_out=%vdd_project%\driver\x64\%build_mode%\quickdesk_display
if exist "%driver_out%\quickdesk_display.dll" (
    copy /y "%driver_out%\quickdesk_display.dll" "%output_path%\" >nul
) else (
    echo [!] warning: quickdesk_display.dll not found in %driver_out%
)
if exist "%driver_out%\quickdesk_display.inf" (
    copy /y "%driver_out%\quickdesk_display.inf" "%output_path%\" >nul
)
if exist "%driver_out%\quickdesk_display.cat" (
    copy /y "%driver_out%\quickdesk_display.cat" "%output_path%\" >nul
)

:: copy nefconw tool
if exist "%vdd_project%\tools\nefconw.exe" (
    echo [*] copying nefconw.exe...
    copy /y "%vdd_project%\tools\nefconw.exe" "%output_path%\" >nul
)

echo=
echo [*] driver build completed successfully
echo [*] output: %output_path%
set errno=0

:return
cd /d %old_cd%
exit /b %errno%
