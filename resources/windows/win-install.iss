[Setup]
AppName=Ton Proxy
AppVersion=0.2.0
WizardStyle=modern
DefaultDirName={autopf}\Ton Proxy
DefaultGroupName=Ton Proxy
UninstallDisplayIcon={app}\TonProxy.exe
Compression=lzma2
OutputBaseFilename=TonProxy-setup
OutputDir=GUIWinSetup
SetupIconFile=ton_icon.ico
RestartIfNeededByRun=no

[Files]
Source: "tonutils-proxy-gui.exe"; DestDir: "{app}"
Source: "WebView2Loader.dll"; DestDir: "{app}"
Source: "MicrosoftEdgeWebview2Setup.exe"; DestDir: "{app}"; Flags: deleteafterinstall; Check: not IsWebView2Detected 


[Icons]
Name: "{userdesktop}\TonProxy"; Filename: "{app}\tonutils-proxy-gui.exe"; WorkingDir: "{app}"; Tasks: desktopicon

[Tasks]
Name: desktopicon; Description: "Create shortcut on desktop"; GroupDescription: "Additionally:"

[Code]
function IsWebView2Detected(): boolean;

var
    reg_key: string;
    success: boolean;
    successLocal: boolean;
    version: string;

begin

    reg_key := 'SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}';
    if RegValueExists(HKEY_LOCAL_MACHINE, reg_key, 'pv') then
    begin
        success := true;
        if RegQueryStringValue(HKEY_LOCAL_MACHINE, reg_key, 'pv', version) then
        begin
            if version = '0.0.0.0' then
                success := false;
            if version = 'null' then
                success := false;
            if version = '' then
                success := false;
        end;
        
    end else
        success := false;

    reg_key := 'Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}';
    if RegValueExists(HKEY_CURRENT_USER, reg_key, 'pv') then
    begin
      successLocal := true;  
        if RegQueryStringValue(HKEY_CURRENT_USER, reg_key, 'pv', version) then
        begin
            if version = '0.0.0.0' then
                successLocal := false;
            if version = 'null' then
                successLocal := false;
            if version = '' then
                successLocal := false;
        end;
    end else
        successLocal := false;



    if success = true or successLocal = true then
        begin
            Result := true;
        end else
            Result := false;

    end;

function InitializeSetup(): boolean;
begin

  if not IsWebView2Detected() then
    begin
      MsgBox('Microsoft WebView2 is required for the program to work. We will install WebView2 for you. Make sure you have internet connection :)', mbInformation, MB_OK)
    end;   

  result := true;
end;


[Run]
Filename: "{app}\MicrosoftEdgeWebview2Setup.exe"; Check: not IsWebView2Detected; StatusMsg: "installation Microsoft WebView2"
Filename: "{app}\tonutils-proxy-gui.EXE"; Description: "Launch application?"; Flags: postinstall






