; QuickDesk Inno Setup Script
; Version placeholders are replaced by package_qd_win.bat before compilation

#define MyAppName "QuickDesk"
#define MyAppPublisher "QuickCoder"
#define MyAppURL "https://github.com/user/QuickDesk"
#define MyAppExeName "QuickDesk.exe"
#define MyAppCopyright "Copyright (C) QuickCoder 2018-2038. All rights reserved."

; These are set via /D command line options from package_qd_win.bat
#ifndef MyAppVersion
  #define MyAppVersion "1.0.0"
#endif
#ifndef MyPublishDir
  #define MyPublishDir "..\..\publish\Release"
#endif
#ifndef MyOutputDir
  #define MyOutputDir "..\..\publish"
#endif
#ifndef MyIconPath
  #define MyIconPath "..\..\QuickDesk\res\QuickDesk.ico"
#endif

[Setup]
AppId={{B7E3F2A1-8C4D-4E5F-9A6B-1D2E3F4A5B6C}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppCopyright={#MyAppCopyright}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=no
OutputDir={#MyOutputDir}
OutputBaseFilename=QuickDesk-win-x64-setup
SetupIconFile={#MyIconPath}
UninstallDisplayIcon={app}\{#MyAppExeName}
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=admin
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
VersionInfoVersion={#MyAppVersion}
ShowLanguageDialog=auto
CloseApplications=force
CloseApplicationsFilter=*.exe

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "chinesesimplified"; MessagesFile: "ChineseSimplified.isl"

[Messages]
english.WelcomeLabel2=This will install [name/ver] on your computer.%n%nPlease read the following important information before continuing.
chinesesimplified.WelcomeLabel2=即将在您的计算机上安装 [name/ver]。%n%n请在继续之前阅读以下重要信息。

[CustomMessages]
english.CreateStartMenuShortcut=Create a Start Menu shortcut
chinesesimplified.CreateStartMenuShortcut=创建开始菜单快捷方式
english.VirtualDisplayDriver=Virtual Display Driver:
english.InstallVirtualDisplayDriver=Install virtual display driver (required for virtual display feature)
chinesesimplified.VirtualDisplayDriver=虚拟显示器驱动：
chinesesimplified.InstallVirtualDisplayDriver=安装虚拟显示器驱动（虚拟屏功能需要）

[Code]
var
  DisclaimerPage: TOutputMsgMemoWizardPage;

function GetDisclaimerTitle: String;
begin
  if ActiveLanguage = 'chinesesimplified' then
    Result := '免责声明'
  else
    Result := 'Disclaimer';
end;

function GetDisclaimerDescription: String;
begin
  if ActiveLanguage = 'chinesesimplified' then
    Result := '请在继续之前仔细阅读以下免责声明。'
  else
    Result := 'Please read the following disclaimer carefully before proceeding.';
end;

function GetDisclaimerSubCaption: String;
begin
  if ActiveLanguage = 'chinesesimplified' then
    Result := '点击"下一步"即表示您已阅读并同意以下内容：'
  else
    Result := 'By clicking "Next", you acknowledge that you have read and agree to the following:';
end;

function GetDisclaimerBody: String;
begin
  if ActiveLanguage = 'chinesesimplified' then
    Result :=
      '免责声明'#13#10 +
      '========'#13#10#13#10 +
      '1. 本软件是一款远程桌面工具，仅供合法用途使用，'#13#10 +
      '   包括但不限于远程技术支持、远程办公和个人设备管理。'#13#10#13#10 +
      '2. 在发起远程连接之前，您必须获得设备所有者的明确授权。'#13#10 +
      '   未经授权访问计算机系统可能违反相关法律法规。'#13#10#13#10 +
      '3. 开发者不对本软件的任何滥用行为承担任何责任，'#13#10 +
      '   包括但不限于未经授权的访问、数据窃取、隐私侵犯'#13#10 +
      '   或使用本软件进行的任何违法活动。'#13#10#13#10 +
      '4. 您同意在使用本软件时遵守所有适用的地方、国家'#13#10 +
      '   和国际法律法规。'#13#10#13#10 +
      '5. 本软件按"原样"提供，不提供任何明示或暗示的保证。'#13#10 +
      '   使用风险由您自行承担。'#13#10#13#10 +
      '6. 继续安装即表示您已阅读、理解并同意本免责声明。'
  else
    Result :=
      'DISCLAIMER'#13#10 +
      '=========='#13#10#13#10 +
      '1. This software is a remote desktop tool designed for lawful purposes only, '#13#10 +
      '   including but not limited to remote technical support, remote work, and '#13#10 +
      '   personal device management.'#13#10#13#10 +
      '2. You must obtain explicit authorization from the owner of any device before '#13#10 +
      '   initiating a remote connection. Unauthorized access to computer systems '#13#10 +
      '   may violate applicable laws and regulations.'#13#10#13#10 +
      '3. The developer assumes no responsibility or liability for any misuse of '#13#10 +
      '   this software, including but not limited to unauthorized access, data '#13#10 +
      '   theft, privacy violations, or any illegal activities conducted using '#13#10 +
      '   this software.'#13#10#13#10 +
      '4. You agree to comply with all applicable local, national, and international '#13#10 +
      '   laws and regulations when using this software.'#13#10#13#10 +
      '5. This software is provided "AS IS" without warranty of any kind, express '#13#10 +
      '   or implied. Use at your own risk.'#13#10#13#10 +
      '6. By proceeding with the installation, you acknowledge that you have read, '#13#10 +
      '   understood, and agreed to this disclaimer.';
end;

procedure InitializeWizard;
begin
  DisclaimerPage := CreateOutputMsgMemoPage(wpWelcome,
    GetDisclaimerTitle, GetDisclaimerDescription,
    GetDisclaimerSubCaption, GetDisclaimerBody);
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  ResultCode: Integer;
begin
  // Stop the QuickDeskHost service if it exists (upgrade scenario).
  Exec('sc.exe', 'stop QuickDeskHost', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  // Give the service time to stop.
  Sleep(2000);
  // Uninstall old service registration so files can be replaced.
  if FileExists(ExpandConstant('{app}\quickdesk_host.exe')) then
    Exec(ExpandConstant('{app}\quickdesk_host.exe'), '--uninstall-service', '',
         SW_HIDE, ewWaitUntilTerminated, ResultCode);

  // Remove old virtual display driver so the new version can be installed cleanly.
  if FileExists(ExpandConstant('{app}\drivers\vdd\nefconw.exe')) then
    Exec(ExpandConstant('{app}\drivers\vdd\nefconw.exe'),
         '--remove-device-node --hardware-id Root\QuickDesk\VDD --class-guid "{4D36E968-E325-11CE-BFC1-08002BE10318}"',
         '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  Result := '';
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  ResultCode: Integer;
begin
  if CurUninstallStep = usUninstall then
  begin
    // Stop and uninstall the service before removing files.
    Exec('sc.exe', 'stop QuickDeskHost', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
    Sleep(2000);
    if FileExists(ExpandConstant('{app}\quickdesk_host.exe')) then
      Exec(ExpandConstant('{app}\quickdesk_host.exe'), '--uninstall-service', '',
           SW_HIDE, ewWaitUntilTerminated, ResultCode);

    // Remove virtual display driver (cleanup regardless of install-time choice).
    if FileExists(ExpandConstant('{app}\drivers\vdd\nefconw.exe')) then
      Exec(ExpandConstant('{app}\drivers\vdd\nefconw.exe'),
           '--remove-device-node --hardware-id Root\QuickDesk\VDD --class-guid "{4D36E968-E325-11CE-BFC1-08002BE10318}"',
           '', SW_HIDE, ewWaitUntilTerminated, ResultCode);
  end;
end;

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: checkedonce
Name: "startmenuicon"; Description: "{cm:CreateStartMenuShortcut}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: checkedonce
Name: "vdd"; Description: "{cm:InstallVirtualDisplayDriver}"; GroupDescription: "{cm:VirtualDisplayDriver}"; Flags: checkedonce

[Files]
Source: "{#MyPublishDir}\*"; DestDir: "{app}"; Excludes: "\drivers\vdd\*"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#MyPublishDir}\drivers\vdd\*"; DestDir: "{app}\drivers\vdd"; Flags: ignoreversion recursesubdirs createallsubdirs; Tasks: vdd

[Icons]
Name: "{group}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: startmenuicon
Name: "{group}\Uninstall {#MyAppName}"; Filename: "{uninstallexe}"; Tasks: startmenuicon
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
; Virtual display driver installation (only if user selected the task).
Filename: "{app}\drivers\vdd\nefconw.exe"; Parameters: "--remove-device-node --hardware-id Root\QuickDesk\VDD --class-guid ""{{4D36E968-E325-11CE-BFC1-08002BE10318}}"""; StatusMsg: "Removing old virtual display driver..."; Flags: runhidden waituntilterminated; Tasks: vdd
Filename: "{app}\drivers\vdd\nefconw.exe"; Parameters: "--create-device-node --class-name Display --class-guid ""{{4D36E968-E325-11CE-BFC1-08002BE10318}}"" --hardware-id Root\QuickDesk\VDD --no-duplicates"; StatusMsg: "Creating virtual display device..."; Flags: runhidden waituntilterminated; Tasks: vdd
Filename: "{app}\drivers\vdd\nefconw.exe"; Parameters: "--install-driver --inf-path ""{app}\drivers\vdd\quickdesk_display.inf"""; StatusMsg: "Installing virtual display driver..."; Flags: runhidden waituntilterminated; Tasks: vdd
; Install the QuickDeskHost Windows service (runs as SYSTEM).
; Pass --log-dir and --config-dir so that the service worker stores logs and
; config in the same location as the Qt child-process mode.
Filename: "{app}\quickdesk_host.exe"; Parameters: "--install-service --log-dir=""{userappdata}\QuickDesk\logs"" --config-dir=""{userappdata}\QuickDesk"""; StatusMsg: "Installing QuickDesk Host Service..."; Flags: runhidden waituntilterminated
; Start the service.
Filename: "sc.exe"; Parameters: "start QuickDeskHost"; StatusMsg: "Starting QuickDesk Host Service..."; Flags: runhidden waituntilterminated
; Launch the application.
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent
