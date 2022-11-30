[Setup]
AppName=Ton Proxy
AppVersion=0.1.5
WizardStyle=modern
DefaultDirName={autopf}\Ton Proxy
DefaultGroupName=Ton Proxy
UninstallDisplayIcon={app}\TonProxy.exe
Compression=lzma2
OutputBaseFilename=TonProxy-setup
OutputDir=GUIWinSetup
SetupIconFile=ton_icon.ico

[Files]
Source: "tonutils-proxy-gui.exe"; DestDir: "{app}"
Source: "WebView2Loader.dll"; DestDir: "{app}"

[Icons]
Name: "{userdesktop}\TonProxy"; Filename: "{app}\tonutils-proxy-gui.exe"; WorkingDir: "{app}"; Tasks: desktopicon

[Tasks]
Name: desktopicon; Description: "Создать ярлык на Рабочем столе"; GroupDescription: "Дополнительно:"



