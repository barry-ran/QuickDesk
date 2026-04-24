@echo off

echo=
echo=
echo ---------------------------------------------------------------
echo check ENV
echo ---------------------------------------------------------------

:: example: C:\QtPro\6.8.4
if "%ENV_QT_PATH%"=="" set ENV_QT_PATH=C:\QtPro\6.8.4
:: example: C:\Program Files\Microsoft Visual Studio\2022\Community
if "%ENV_VS_INSTALL%"=="" set "ENV_VS_INSTALL=C:\Program Files\Microsoft Visual Studio\2022\Community"
if "%ENV_VCVARSALL%"=="" set "ENV_VCVARSALL=%ENV_VS_INSTALL%\VC\Auxiliary\Build\vcvarsall.bat"
:: VC Runtime DLL version
if "%ENV_VCRUNTIME_VERSION%"=="" set ENV_VCRUNTIME_VERSION=14.42.34433

echo ENV_VS_INSTALL %ENV_VS_INSTALL%
echo ENV_VCVARSALL %ENV_VCVARSALL%
echo ENV_QT_PATH %ENV_QT_PATH%
echo ENV_VCRUNTIME_VERSION %ENV_VCRUNTIME_VERSION%

:: get script absolute path
set script_path=%~dp0
:: enter script directory, as it affects the working directory of programs executed in the script
set old_cd=%cd%
cd /d %~dp0

:: declare startup parameters and default values
SETLOCAL EnableDelayedExpansion
set cpu_mode=x64
set build_mode=Release
set errno=1

echo=
echo=
echo ---------------------------------------------------------------
echo parse arguments
echo ---------------------------------------------------------------

:: iterate all arguments
:parse_args
if "%1"=="" goto args_done

REM check build type (case insensitive)
if /i "%1"=="debug" set build_mode=Debug
if /i "%1"=="release" set build_mode=Release

shift
goto parse_args
:args_done

echo [*] arch: %cpu_mode%
echo [*] build mode: %build_mode%
echo=

:: set paths
set qt_msvc_path=%ENV_QT_PATH%\msvc2022_64\bin
set publish_path=%script_path%..\publish\%build_mode%\
set release_path=%script_path%..\output\x64\%build_mode%
set src_out_path=%script_path%..\..\src\out\%build_mode%
set vcvarsall="%ENV_VCVARSALL%"
set "vcruntime_path=%ENV_VS_INSTALL%\VC\Redist\MSVC\%ENV_VCRUNTIME_VERSION%\x64\Microsoft.VC143.CRT"

echo [*] Qt MSVC path: %qt_msvc_path%
echo [*] publish path: %publish_path%
echo [*] output path: %release_path%
echo [*] src/out path: %src_out_path%
echo [*] VCRuntime path: %vcruntime_path%
echo=

set PATH=%qt_msvc_path%;%PATH%

:: register VC environment
call %vcvarsall% x64

echo=
echo=
echo ---------------------------------------------------------------
echo begin publish
echo ---------------------------------------------------------------

:: check if output path exists
if not exist "%release_path%" (
    echo [!] error: output path does not exist: %release_path%
    echo [!] please run build_qd_win.bat %build_mode% first
    goto return
)

:: clean and create publish directory
if exist "%publish_path%" (
    echo [*] cleaning old publish dir...
    rmdir /s/q "%publish_path%"
)
echo [*] creating publish dir: %publish_path%
mkdir "%publish_path%"

:: copy program files to publish
echo [*] copying program files...
xcopy "%release_path%" "%publish_path%" /E /Y

:: copy host and client (priority: src/out > 3rdparty)
set thirdparty_path=%script_path%..\QuickDesk\3rdparty\quickdesk-remoting\x64
echo [*] 3rdparty path: %thirdparty_path%
echo [*] copying host and client...

if exist "%src_out_path%\quickdesk_core.dll" (
    copy /Y "%src_out_path%\quickdesk_core.dll" "%publish_path%\" >nul
    echo [*] copied quickdesk_core.dll from src/out
) else if exist "%thirdparty_path%\quickdesk_core.dll" (
    copy /Y "%thirdparty_path%\quickdesk_core.dll" "%publish_path%\" >nul
    echo [*] copied quickdesk_core.dll from 3rdparty
) else (
    echo [!] warning: quickdesk_core.dll not found
)

if exist "%src_out_path%\quickdesk_host.exe" (
    copy /Y "%src_out_path%\quickdesk_host.exe" "%publish_path%\" >nul
    echo [*] copied quickdesk_host.exe from src/out
) else if exist "%thirdparty_path%\quickdesk_host.exe" (
    copy /Y "%thirdparty_path%\quickdesk_host.exe" "%publish_path%\" >nul
    echo [*] copied quickdesk_host.exe from 3rdparty
) else (
    echo [!] warning: quickdesk_host.exe not found
)

if exist "%src_out_path%\quickdesk_client.exe" (
    copy /Y "%src_out_path%\quickdesk_client.exe" "%publish_path%\" >nul
    echo [*] copied quickdesk_client.exe from src/out
) else if exist "%thirdparty_path%\quickdesk_client.exe" (
    copy /Y "%thirdparty_path%\quickdesk_client.exe" "%publish_path%\" >nul
    echo [*] copied quickdesk_client.exe from 3rdparty
) else (
    echo [!] warning: quickdesk_client.exe not found
)

if exist "%src_out_path%\icudtl.dat" (
    copy /Y "%src_out_path%\icudtl.dat" "%publish_path%\" >nul
    echo [*] copied icudtl.dat from src/out
) else if exist "%thirdparty_path%\icudtl.dat" (
    copy /Y "%thirdparty_path%\icudtl.dat" "%publish_path%\" >nul
    echo [*] copied icudtl.dat from 3rdparty
) else (
    echo [!] warning: icudtl.dat not found
)

:: copy MCP bridge
echo [*] copying quickdesk-mcp...
if exist "%release_path%\quickdesk-mcp.exe" (
    copy /Y "%release_path%\quickdesk-mcp.exe" "%publish_path%\" >nul
    echo [*] copied quickdesk-mcp.exe from output
) else (
    echo [!] warning: quickdesk-mcp.exe not found (run build_mcp_win.bat first)
)
echo=

:: copy agent and built-in skills
echo [*] copying quickdesk-agent...
if exist "%release_path%\quickdesk-agent.exe" (
    copy /Y "%release_path%\quickdesk-agent.exe" "%publish_path%\" >nul
    echo [*] copied quickdesk-agent.exe from output
) else (
    echo [!] warning: quickdesk-agent.exe not found (run build_agent_win.bat first)
)

echo [*] copying built-in skills...
if exist "%release_path%\skills" (
    if not exist "%publish_path%\skills" mkdir "%publish_path%\skills"
    xcopy "%release_path%\skills" "%publish_path%\skills" /E /Y /Q >nul
    echo [*] copied skills directory
) else (
    echo [!] warning: skills directory not found (run build_agent_win.bat first)
)
echo=

:: copy virtual display driver files (from build_vdd_win.bat output)
set vdd_output=%script_path%..\output\x64\%build_mode%\drivers\vdd
if exist "%vdd_output%\quickdesk_display.dll" (
    echo [*] copying virtual display driver...
    if not exist "%publish_path%drivers\vdd\" mkdir "%publish_path%drivers\vdd\"
    xcopy "%vdd_output%\*" "%publish_path%drivers\vdd\" /E /Y /Q >nul
    echo [*] virtual display driver copied
) else (
    echo [!] warning: virtual display driver not found in %vdd_output%
    echo [!] skipping VDD ^(run build_vdd_win.bat first if needed^)
)
echo=

:: add Qt dependencies (specify qml path)
echo [*] running windeployqt to add Qt dependencies...
windeployqt --qmldir "%script_path%..\QuickDesk\qml" "%publish_path%\QuickDesk.exe"

:: remove unnecessary Qt dependencies
echo [*] cleaning unnecessary Qt dependencies...
if exist "%publish_path%\iconengines" (
    rmdir /s/q "%publish_path%\iconengines"
)
if exist "%publish_path%\translations" (
    rmdir /s/q "%publish_path%\translations"
)
if exist "%publish_path%\generic" (
    rmdir /s/q "%publish_path%\generic"
)
if exist "%publish_path%\logs" (
    rmdir /s/q "%publish_path%\logs"
)
if exist "%publish_path%\db" (
    rmdir /s/q "%publish_path%\db"
)
if exist "%publish_path%\platforminputcontexts" (
    rmdir /s/q "%publish_path%\platforminputcontexts"
)
if exist "%publish_path%\qmltooling" (
    rmdir /s/q "%publish_path%\qmltooling"
)

:: clean imageformats, keep only needed dlls
if exist "%publish_path%\imageformats" (
    echo [*] cleaning imageformats...
    del /q "%publish_path%\imageformats\qgif.dll" 2>nul
    del /q "%publish_path%\imageformats\qicns.dll" 2>nul
    del /q "%publish_path%\imageformats\qico.dll" 2>nul
    del /q "%publish_path%\imageformats\qsvg.dll" 2>nul
    del /q "%publish_path%\imageformats\qtga.dll" 2>nul
    del /q "%publish_path%\imageformats\qtiff.dll" 2>nul
    del /q "%publish_path%\imageformats\qwbmp.dll" 2>nul
    del /q "%publish_path%\imageformats\qwebp.dll" 2>nul
)

:: clean sqldrivers, keep only sqlite
if exist "%publish_path%\sqldrivers" (
    echo [*] cleaning sqldrivers ^(keep sqlite^)...
    for %%f in ("%publish_path%\sqldrivers\*.dll") do (
        echo %%~nxf | findstr /i "sqlite" >nul
        if errorlevel 1 (
            del /q "%%f" 2>nul
        )
    )
)

:: remove unnecessary dlls and files
echo [*] removing unnecessary files...
del /q "%publish_path%\Qt6VirtualKeyboard.dll" 2>nul
del /q "%publish_path%\QuickDesk.exe.manifest" 2>nul
del /q "%publish_path%\*.exp" 2>nul
del /q "%publish_path%\*.lib" 2>nul

:: remove unnecessary Qt6 dlls
del /q "%publish_path%\dxcompiler.dll" 2>nul
del /q "%publish_path%\opengl32sw.dll" 2>nul
del /q "%publish_path%\Qt6QuickControls2FluentWinUI3StyleImpl.dll" 2>nul
del /q "%publish_path%\Qt6QuickControls2Fusion.dll" 2>nul
del /q "%publish_path%\Qt6QuickControls2FusionStyleImpl.dll" 2>nul
del /q "%publish_path%\Qt6QuickControls2Imagine.dll" 2>nul
del /q "%publish_path%\Qt6QuickControls2ImagineStyleImpl.dll" 2>nul
del /q "%publish_path%\Qt6QuickControls2Material.dll" 2>nul
del /q "%publish_path%\Qt6QuickControls2MaterialStyleImpl.dll" 2>nul
del /q "%publish_path%\Qt6QuickControls2Universal.dll" 2>nul
del /q "%publish_path%\Qt6QuickControls2UniversalStyleImpl.dll" 2>nul
del /q "%publish_path%\Qt6QuickControls2WindowsStyleImpl.dll" 2>nul

:: remove vc_redist installer, copy vcruntime dlls instead
echo [*] removing vc_redist installer...
del /q "%publish_path%\vc_redist.x64.exe" 2>nul

:: copy vcruntime dll from VC Redist directory
echo [*] copying VCRuntime DLLs...
if not exist "%vcruntime_path%" (
    echo [!] warning: VCRuntime path does not exist: %vcruntime_path%
    echo [!] please check if ENV_VCRUNTIME_VERSION is correct
) else (
    copy /Y "%vcruntime_path%\msvcp140.dll" "%publish_path%\" >nul
    copy /Y "%vcruntime_path%\msvcp140_1.dll" "%publish_path%\" >nul
    copy /Y "%vcruntime_path%\msvcp140_2.dll" "%publish_path%\" >nul
    copy /Y "%vcruntime_path%\vcruntime140.dll" "%publish_path%\" >nul
    copy /Y "%vcruntime_path%\vcruntime140_1.dll" "%publish_path%\" >nul
    echo [*] VCRuntime DLLs copied
)

echo=
echo=
echo ---------------------------------------------------------------
echo [*] publish finished!
echo ---------------------------------------------------------------
echo [*] publish dir: %publish_path%
echo=

set errno=0

:return
cd /d "%old_cd%"
exit /B %errno%

ENDLOCAL
