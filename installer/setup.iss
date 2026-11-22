; MediaFlow Inno Setup Installer Script
; Pinned Target: Windows 10 / Windows 11 (64-bit)

#define MyAppName "MediaFlow"
#define MyAppVersion "1.0.0"
#define MyAppPublisher "MediaFlow Open Source Project"
#define MyAppURL "https://github.com/mukkss/MediaFlow"
#define MyAppExeName "MediaFlow.exe"

[Setup]
AppId={{5D8B4F91-72A0-4C9D-9481-B136C701B058}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
OutputDir=..\bin
OutputBaseFilename=MediaFlow_Setup_{#MyAppVersion}
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
; Requirement 5: Desktop shortcut is strictly optional (unchecked by default)
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
; Requirement 6: Windows startup registration is optional (unchecked by default)
Name: "autostart"; Description: "Automatically launch MediaFlow on Windows startup"; GroupDescription: "System Integration:"; Flags: unchecked

[Files]
; Main Executable
Source: "..\bin\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion

; Bundled FFmpeg & FFprobe (LGPL static build)
Source: "..\resources\ffmpeg\ffmpeg.exe"; DestDir: "{app}\ffmpeg"; Flags: ignoreversion external; Flags: skipifsourcedoesntexist
Source: "..\resources\ffmpeg\ffprobe.exe"; DestDir: "{app}\ffmpeg"; Flags: ignoreversion external; Flags: skipifsourcedoesntexist

; Codec dependencies (LibRaw / libheif components)
Source: "..\resources\codecs\*"; DestDir: "{app}\codecs"; Flags: ignoreversion recursesubdirs createallsubdirs external; Flags: skipifsourcedoesntexist

; License documentation and notices
Source: "..\resources\licenses\*"; DestDir: "{app}\licenses"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Registry]
; Requirement 6: Optional startup registration with automatic uninstallation cleanup
Root: HKCU; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "{#MyAppName}"; ValueData: """{app}\{#MyAppExeName}"""; Flags: uninsdeletevalue; Tasks: autostart

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[Code]
// Requirement 4: Verify WebView2 Runtime availability on Windows 10/11
function IsWebView2Installed(): Boolean;
var
  Installed: Boolean;
  VersionStr: String;
begin
  Installed := RegQueryStringValue(HKLM, 'SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}', 'pv', VersionStr);
  if not Installed then
    Installed := RegQueryStringValue(HKCU, 'Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}', 'pv', VersionStr);
  Result := Installed and (VersionStr <> '');
end;

function InitializeSetup(): Boolean;
begin
  Result := True;
  if not IsWebView2Installed() then
  begin
    MsgBox('Microsoft Edge WebView2 Runtime was not detected on this system.' + #13#10 + #13#10 +
           'MediaFlow desktop interface requires WebView2. It is typically included with Windows 11 and modern Windows 10 updates.' + #13#10 +
           'If offline, please ensure the Evergreen Standalone WebView2 installer is installed.',
           mbInformation, MB_OK);
  end;
end;

// Requirement 7: Preserve %APPDATA%\MediaFlow\ during uninstall unless explicitly requested
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  AppDataPath: String;
begin
  if CurUninstallStep = usPostUninstall then
  begin
    AppDataPath := ExpandConstant('{userappdata}\MediaFlow');
    if DirExists(AppDataPath) then
    begin
      if MsgBox('Do you want to delete your MediaFlow settings, conversion logs, and history from AppData?' + #13#10#13#10 +
                'Select No to keep your configurations intact.',
                mbConfirmation, MB_YESNO or MB_DEFBUTTON2) = IDYES then
      begin
        DelTree(AppDataPath, True, True, True);
      end;
    end;
  end;
end;
