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
PrivilegesRequired=admin

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
Name: "japanese"; MessagesFile: "compiler:Languages\Japanese.isl"
Name: "russian"; MessagesFile: "compiler:Languages\Russian.isl"

[Files]
Source: "..\CustomShell\CustomShell\bin\Debug\net6.0-windows\*"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "..\dist\CustomShell.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\School_agent.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "dummy.txt"; DestDir: "C:\ProgramData\SchoolAgent"; DestName: "dummy.txt"; Flags: createallsubdirs deleteafterinstall

[Icons]
Name: "{group}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"

[Tasks]
Name: "autostart"; Description: "Добавить CustomShell в автозапуск Windows"; GroupDescription: "Дополнительные параметры:"; Flags: checkedonce

[Registry]
Root: HKLM; Subkey: "SOFTWARE\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; ValueName: "CustomShell"; ValueData: """{app}\CustomShell.exe"""; Flags: uninsdeletevalue; Tasks: autostart

[Run]
Filename: "cmd.exe"; Parameters: "/C setx AGENT_SERVER_URL ""ws://0.0.0.0:8000/ws"" /M"; StatusMsg: "Установка переменной окружения AGENT_SERVER_URL..."; Flags: runhidden

; ------------------------- CODE -----------------------

[Code]

var
  DeviceNamePage: TInputQueryWizardPage;
  UserDeviceName: string;

procedure InitializeWizard;
begin
  DeviceNamePage :=
    CreateInputQueryPage(
      wpSelectDir,
      'Название устройства',
      'Введите имя компьютера',
      'Это имя будет использоваться как идентификатор агента (пример: computer-12).'
    );

  DeviceNamePage.Add('Имя устройства:', False);
end;

function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;

  if CurPageID = DeviceNamePage.ID then begin
    UserDeviceName := Trim(DeviceNamePage.Values[0]);

    if UserDeviceName = '' then begin
      MsgBox('Введите имя устройства.', mbError, MB_OK);
      Result := False;
      exit;
    end;
  end;
end;

procedure InstallAndStartService(ExePath: string);
var
  ResultCode: Integer;
begin
  if not Exec(ExePath, 'install', '', SW_HIDE, ewWaitUntilTerminated, ResultCode) then
    Log('Ошибка установки службы: ' + IntToStr(ResultCode));

  if not Exec(ExePath, 'start', '', SW_HIDE, ewNoWait, ResultCode) then
    Log('Ошибка запуска службы: ' + IntToStr(ResultCode));
end;


procedure CurStepFinished(CurStep: TSetupStep);
var
    ConfigFile: TStringList;
    ConfigPath: string;
    AgentExePath: string;
begin
    if CurStep = ssPostInstall then begin

        ConfigPath := 'C:\ProgramData\SchoolAgent\config.json';
        AgentExePath := ExpandConstant('{app}\School_agent.exe');

        ConfigFile := TStringList.Create;
        try
            ConfigFile.Add('{');
            ConfigFile.Add('  "server_url": "ws://dummy-server:8080/ws",');
            ConfigFile.Add('  "device_token": "' + UserDeviceName + '-TOKEN",');
            ConfigFile.Add('  "hostname": "' + UserDeviceName + '",');
            ConfigFile.Add('  "log_dir": "C:\\ProgramData\\SchoolAgent\\Logs",');
            ConfigFile.Add('  "project_base": "D:\\UserProjects"');
            ConfigFile.Add('}');

            ConfigFile.SaveToFile(ConfigPath);
            Log('Конфиг создан: ' + ConfigPath);

        finally
            ConfigFile.Free;
        end;

        InstallAndStartService(AgentExePath);
    end;
end;

[UninstallRun]
Filename: "{app}\School_agent.exe"; Parameters: "stop"; Flags: runhidden
Filename: "{app}\School_agent.exe"; Parameters: "uninstall"; Flags: runhidden
