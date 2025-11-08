#define MyAppName "ClassOS_scripts"
#define MyAppVersion "1.0"
#define MyAppPublisher "ClassOS-prod"

[Setup]
AppId={{444C0650-6980-41DF-B507-0C7033D990B0}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\ClassOS
DefaultGroupName={#MyAppName}
SetupIconFile=C:\Users\Rinat\Pictures\ico\favicon.ico
UninstallDisplayIcon={app}\CustomShell.exe
OutputBaseFilename=ClassOS-setup
SolidCompression=yes
WizardStyle=modern

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "japanese"; MessagesFile: "compiler:Languages\Japanese.isl"
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"

[Files]
Source: "..\CustomShell\CustomShell\bin\Debug\net6.0-windows\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "..\dist\CustomShell.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\School_agent.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"

[Tasks]
Name: "autostart"; Description: "Добавить в автозапуск Windows"; GroupDescription: "Дополнительные параметры:"; Flags: checkedonce

[Registry]
Root: HKLM; Subkey: "SOFTWARE\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "CustomShell"; ValueData: """{app}\CustomShell.exe"""; Flags: uninsdeletevalue; Tasks: autostart
Root: HKLM; Subkey: "SOFTWARE\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "SchoolAgent"; ValueData: """{app}\school_agent.exe"""; Flags: uninsdeletevalue; Tasks: autostart

