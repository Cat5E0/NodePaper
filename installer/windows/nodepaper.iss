; NodePaper Windows x64 Setup definition (Inno Setup 6, pinned by
; installer/windows/innosetup-toolchain.json).
;
; This file is an installation adapter only. It installs the already built and
; already verified release payload; it never compiles nodepaper.exe and never
; reimplements Project discovery, help, doctor, validate, build or JSON output.
;
; Required defines (passed by scripts/build-setup.ps1):
;   NodePaperVersion  release version, for example 0.1.0-rc.4
;   PayloadDir        directory holding the built release payload
;   ChecksumFile      payload checksum list (<sha256>|<relative path> per line)
;   PayloadExcludes   payload files Setup must not install, as an Inno
;                     [Files] Excludes list (see [Files] below)
;   OutputDir         directory for the generated Setup
;   OutputBaseName    Setup file name without .exe
;   SourceCommit      fixed source commit the payload was built from

#ifndef NodePaperVersion
  #error NodePaperVersion must be defined by the build script
#endif
#ifndef PayloadDir
  #error PayloadDir must be defined by the build script
#endif
#ifndef ChecksumFile
  #error ChecksumFile must be defined by the build script
#endif
#ifndef PayloadExcludes
  #error PayloadExcludes must be defined by the build script
#endif
#ifndef OutputDir
  #error OutputDir must be defined by the build script
#endif
#ifndef OutputBaseName
  #error OutputBaseName must be defined by the build script
#endif
#ifndef SourceCommit
  #define SourceCommit ""
#endif

#define NodePaperAppId "{6E1B5C6A-6C2F-4D4B-9A62-2C7E60C0A5F1}"
#define NodePaperName "NodePaper"
#define NodePaperURL "https://github.com/Cat5E0/NodePaper"

[Setup]
AppId={{#NodePaperAppId}
AppName={#NodePaperName}
AppVersion={#NodePaperVersion}
AppVerName={#NodePaperName} {#NodePaperVersion}
VersionInfoVersion=0.1.0
VersionInfoTextVersion={#NodePaperVersion}
VersionInfoProductTextVersion={#NodePaperVersion}
AppPublisher=NodePaper contributors
AppPublisherURL={#NodePaperURL}
AppSupportURL={#NodePaperURL}
AppUpdatesURL={#NodePaperURL}
; Current-user installation only: no administrator rights, no service, no
; machine-wide change.
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=
DefaultDirName={localappdata}\Programs\NodePaper
DefaultGroupName={#NodePaperName}
DisableProgramGroupPage=yes
AllowNoIcons=yes
UsePreviousAppDir=yes
LicenseFile={#PayloadDir}\LICENSE
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
OutputDir={#OutputDir}
OutputBaseFilename={#OutputBaseName}
UninstallDisplayName={#NodePaperName} {#NodePaperVersion}
UninstallDisplayIcon={app}\nodepaper.exe
UninstallFilesDir={app}
; The user Path is changed for the current user only; Windows is notified so
; new terminals see the change.
ChangesEnvironment=yes
; NodePaper never force-closes a running build.
CloseApplications=no
RestartApplications=no
SetupMutex=NodePaperSetup

[Languages]
Name: "chinesesimplified"; MessagesFile: "compiler:Languages\ChineseSimplified.isl"
Name: "english"; MessagesFile: "compiler:Default.isl"

[CustomMessages]
chinesesimplified.LaunchNodePaper=启动 NodePaper
chinesesimplified.UninstallShortcut=卸载 NodePaper
chinesesimplified.DesktopIcon=创建桌面快捷方式
chinesesimplified.NodePaperRunning=NodePaper 正在运行。请关闭 NodePaper 窗口或等待构建结束，然后重试。
chinesesimplified.PayloadBroken=安装文件校验失败：%1。安装已取消并回滚，未修改 PATH、快捷方式和已有安装。
chinesesimplified.VersionMismatch=安装后的程序版本校验失败（报告为“%1”，期望“nodepaper {#NodePaperVersion}”）。安装已取消并回滚。
chinesesimplified.DowngradeWarning=当前已安装 NodePaper %1，本安装包为较旧的 %2。继续将降级为 %2。是否继续？
chinesesimplified.ExistingHeading=现有安装：
chinesesimplified.ExistingReinstall=已安装 %1，将重新安装（修复）。
chinesesimplified.ExistingUpgrade=已安装 %1，将升级为 %2。
chinesesimplified.ExistingDowngrade=已安装 %1，将降级为 %2。
chinesesimplified.PortableHere=该目录当前登记为便携安装，安装后将由本程序接管。
chinesesimplified.PortableElsewhere=另有一份便携安装登记在 %1，它在 PATH 中可能优先于本次安装。
english.LaunchNodePaper=Launch NodePaper
english.UninstallShortcut=Uninstall NodePaper
english.DesktopIcon=Create a desktop shortcut
english.NodePaperRunning=NodePaper is running. Close the NodePaper window or wait for the build to finish, then try again.
english.PayloadBroken=Payload verification failed: %1. The installation was canceled and rolled back; Path, shortcuts and any previous installation were left unchanged.
english.VersionMismatch=The installed executable failed version verification (reported "%1", expected "nodepaper {#NodePaperVersion}"). The installation was canceled and rolled back.
english.DowngradeWarning=NodePaper %1 is currently installed and this Setup contains the older %2. Continuing downgrades the installation to %2. Continue?
english.ExistingHeading=Existing installation:
english.ExistingReinstall=NodePaper %1 is installed; it will be reinstalled (repair).
english.ExistingUpgrade=NodePaper %1 is installed; it will be upgraded to %2.
english.ExistingDowngrade=NodePaper %1 is installed; it will be downgraded to %2.
english.PortableHere=This directory is currently registered as a portable installation; this installation takes it over.
english.PortableElsewhere=Another portable installation is registered at %1; it may come before this installation on Path.

[Tasks]
Name: "desktopicon"; Description: "{cm:DesktopIcon}"; Flags: unchecked

[Files]
; The already verified release payload, minus the files PayloadExcludes names.
; Setup does not build or modify any payload file.
;
; The exclusions are the ZIP channel's own Install-NodePaper.ps1 and
; Uninstall-NodePaper.ps1. Installing them here put a script named
; "Uninstall-NodePaper" inside a Setup installation directory, where running it
; is the obvious thing to do and does the wrong thing: it takes the Path entry
; away and leaves the installation, its Start-menu entries and its entry in
; Settings behind. scripts\build-setup.ps1 owns the list and keeps the embedded
; checksum list in step, so an excluded file is not expected under {app} either.
Source: "{#PayloadDir}\*"; DestDir: "{app}"; Excludes: "{#PayloadExcludes}"; Flags: recursesubdirs createallsubdirs ignoreversion
; Verification data stays in the temporary directory and is never installed.
Source: "{#ChecksumFile}"; Flags: dontcopy

[Icons]
; A persistent command-line window: it runs the same nodepaper.exe onboarding
; and then keeps accepting commands.
Name: "{group}\{#NodePaperName}"; Filename: "{cmd}"; Parameters: "/K """"{app}\nodepaper.exe"""""; WorkingDir: "{userdocs}"; IconFilename: "{app}\nodepaper.exe"; Comment: "{#NodePaperName} {#NodePaperVersion}"
Name: "{group}\{cm:UninstallShortcut}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#NodePaperName}"; Filename: "{cmd}"; Parameters: "/K """"{app}\nodepaper.exe"""""; WorkingDir: "{userdocs}"; IconFilename: "{app}\nodepaper.exe"; Tasks: desktopicon

[Run]
; This window is a child of Setup, so it inherited the environment Setup
; started with -- the one from before the install directory was added to Path.
; Running nodepaper.exe by full path worked, but typing "nodepaper" in the
; window Setup had just opened to try it did not, which is the opposite of what
; the window is for. Setting Path for this shell makes it behave like the fresh
; terminal the user would otherwise have to open.
Filename: "{cmd}"; Parameters: "/K ""set PATH={app};%PATH% && ""{app}\nodepaper.exe"""""; WorkingDir: "{userdocs}"; Description: "{cm:LaunchNodePaper}"; Flags: postinstall nowait skipifsilent

[Code]
const
  EnvironmentKey = 'Environment';
  UninstallKey = 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{#NodePaperAppId}_is1';
  { The ZIP channel's own registration (Install-NodePaper.ps1 writes it, and
    Uninstall-NodePaper.ps1 reads it to know what to unregister). Setup only
    reads it, and only ever deletes it when it names Setup's own directory. }
  PortableKey = 'Software\NodePaper';
  PortableValueName = 'PortablePath';

var
  PathEntryAdded: Boolean;

{ ---------- user Path handling (exact entries only) ---------------------- }

function NormalizeEntry(const Value: String): String;
var
  Normalized: String;
begin
  Normalized := Trim(RemoveQuotes(Trim(Value)));
  while (Length(Normalized) > 1) and (Normalized[Length(Normalized)] = '\') do
    Normalized := Copy(Normalized, 1, Length(Normalized) - 1);
  Result := Lowercase(Normalized);
end;

function ReadUserPath(var Value: String): Boolean;
begin
  Result := RegQueryStringValue(HKEY_CURRENT_USER, EnvironmentKey, 'Path', Value);
  if not Result then
    Value := '';
end;

procedure WriteUserPath(const Value: String);
begin
  { REG_EXPAND_SZ is the Windows default type for the user Path; writing it
    keeps any %VARIABLE% entry of other software working. }
  RegWriteExpandStringValue(HKEY_CURRENT_USER, EnvironmentKey, 'Path', Value);
end;

{ Splitting must not use Pos: Inno's Pos() counts DBCS bytes, so a Path entry
  containing non-ASCII characters (a Chinese directory name of unrelated
  software) would shift every following separator and silently corrupt the
  user Path. TStringList parsing is character-correct. }
function SplitPathValue(const Value: String): TStringList;
var
  Text: String;
begin
  Text := Value;
  StringChangeEx(Text, ';', #13#10, False);
  Result := TStringList.Create;
  Result.Text := Text;
end;

function UserPathContains(const Directory: String): Boolean;
var
  Value: String;
  Entries: TStringList;
  Index: Integer;
begin
  Result := False;
  if not ReadUserPath(Value) then
    Exit;
  Entries := SplitPathValue(Value);
  try
    for Index := 0 to Entries.Count - 1 do
      if (Trim(Entries.Strings[Index]) <> '') and
         (NormalizeEntry(Entries.Strings[Index]) = NormalizeEntry(Directory)) then
      begin
        Result := True;
        Exit;
      end;
  finally
    Entries.Free;
  end;
end;

procedure AddUserPathEntry(const Directory: String);
var
  Value: String;
begin
  if UserPathContains(Directory) then
    Exit;
  ReadUserPath(Value);
  Value := Trim(Value);
  while (Value <> '') and (Value[Length(Value)] = ';') do
    Value := Copy(Value, 1, Length(Value) - 1);
  if Value = '' then
    Value := Directory
  else
    Value := Value + ';' + Directory;
  WriteUserPath(Value);
end;

procedure RemoveUserPathEntry(const Directory: String);
var
  Value, Rebuilt: String;
  Entries: TStringList;
  Index: Integer;
  Removed: Boolean;
begin
  if not ReadUserPath(Value) then
    Exit;
  Entries := SplitPathValue(Value);
  try
    Rebuilt := '';
    Removed := False;
    for Index := 0 to Entries.Count - 1 do
    begin
      if Trim(Entries.Strings[Index]) = '' then
        Continue;
      if NormalizeEntry(Entries.Strings[Index]) = NormalizeEntry(Directory) then
      begin
        Removed := True;
        Continue;
      end;
      if Rebuilt = '' then
        Rebuilt := Entries.Strings[Index]
      else
        Rebuilt := Rebuilt + ';' + Entries.Strings[Index];
    end;
  finally
    Entries.Free;
  end;
  { Only rewrite the value when this exact entry was present. }
  if Removed then
    WriteUserPath(Rebuilt);
end;

{ ---------- version comparison (semver-style, prerelease aware) ---------- }

function SplitOn(const Value: String; const Separator: Char): TArrayOfString;
var
  Remaining, Item: String;
  SeparatorPos, Count: Integer;
begin
  SetArrayLength(Result, 0);
  Count := 0;
  Remaining := Value;
  repeat
    SeparatorPos := Pos(Separator, Remaining);
    if SeparatorPos > 0 then
    begin
      Item := Copy(Remaining, 1, SeparatorPos - 1);
      Remaining := Copy(Remaining, SeparatorPos + 1, Length(Remaining) - SeparatorPos);
    end
    else
    begin
      Item := Remaining;
      Remaining := '';
    end;
    Count := Count + 1;
    SetArrayLength(Result, Count);
    Result[Count - 1] := Item;
  until Remaining = '';
end;

function IsAllDigits(const Value: String): Boolean;
var
  Index: Integer;
begin
  Result := Value <> '';
  for Index := 1 to Length(Value) do
    if (Value[Index] < '0') or (Value[Index] > '9') then
    begin
      Result := False;
      Exit;
    end;
end;

function CompareIdentifiers(const Left, Right: String): Integer;
begin
  if IsAllDigits(Left) and IsAllDigits(Right) then
  begin
    if StrToIntDef(Left, 0) < StrToIntDef(Right, 0) then
      Result := -1
    else if StrToIntDef(Left, 0) > StrToIntDef(Right, 0) then
      Result := 1
    else
      Result := 0;
  end
  else if IsAllDigits(Left) then
    Result := -1
  else if IsAllDigits(Right) then
    Result := 1
  else
    Result := CompareStr(Lowercase(Left), Lowercase(Right));
end;

function ComparePrerelease(const Left, Right: String): Integer;
var
  LeftParts, RightParts: TArrayOfString;
  Index, Shared, Compared: Integer;
begin
  if (Left = '') and (Right = '') then
  begin
    Result := 0;
    Exit;
  end;
  { A release version outranks any prerelease of the same numbers. }
  if Left = '' then
  begin
    Result := 1;
    Exit;
  end;
  if Right = '' then
  begin
    Result := -1;
    Exit;
  end;
  LeftParts := SplitOn(Left, '.');
  RightParts := SplitOn(Right, '.');
  Shared := GetArrayLength(LeftParts);
  if GetArrayLength(RightParts) < Shared then
    Shared := GetArrayLength(RightParts);
  for Index := 0 to Shared - 1 do
  begin
    Compared := CompareIdentifiers(LeftParts[Index], RightParts[Index]);
    if Compared <> 0 then
    begin
      Result := Compared;
      Exit;
    end;
  end;
  if GetArrayLength(LeftParts) < GetArrayLength(RightParts) then
    Result := -1
  else if GetArrayLength(LeftParts) > GetArrayLength(RightParts) then
    Result := 1
  else
    Result := 0;
end;

function CompareNodePaperVersions(const Left, Right: String): Integer;
var
  LeftMain, RightMain, LeftPre, RightPre: String;
  LeftNumbers, RightNumbers: TArrayOfString;
  Index, DashPos, LeftValue, RightValue: Integer;
begin
  LeftMain := Left;
  LeftPre := '';
  DashPos := Pos('-', LeftMain);
  if DashPos > 0 then
  begin
    LeftPre := Copy(LeftMain, DashPos + 1, Length(LeftMain) - DashPos);
    LeftMain := Copy(LeftMain, 1, DashPos - 1);
  end;
  RightMain := Right;
  RightPre := '';
  DashPos := Pos('-', RightMain);
  if DashPos > 0 then
  begin
    RightPre := Copy(RightMain, DashPos + 1, Length(RightMain) - DashPos);
    RightMain := Copy(RightMain, 1, DashPos - 1);
  end;
  LeftNumbers := SplitOn(LeftMain, '.');
  RightNumbers := SplitOn(RightMain, '.');
  for Index := 0 to 2 do
  begin
    LeftValue := 0;
    RightValue := 0;
    if Index < GetArrayLength(LeftNumbers) then
      LeftValue := StrToIntDef(LeftNumbers[Index], 0);
    if Index < GetArrayLength(RightNumbers) then
      RightValue := StrToIntDef(RightNumbers[Index], 0);
    if LeftValue < RightValue then
    begin
      Result := -1;
      Exit;
    end;
    if LeftValue > RightValue then
    begin
      Result := 1;
      Exit;
    end;
  end;
  Result := ComparePrerelease(LeftPre, RightPre);
end;

function InstalledNodePaperVersion: String;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, UninstallKey, 'DisplayVersion', Result) then
    Result := '';
end;

{ ---------- ZIP channel registration ------------------------------------ }

function RegisteredPortablePath: String;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, PortableKey, PortableValueName, Result) then
    Result := '';
  Result := Trim(Result);
end;

{ NormalizeEntry is the same comparison the Path code uses: quotes stripped,
  trailing backslashes dropped, case folded. Reusing it keeps "is this the
  registered directory" and "is this the Path entry" from ever disagreeing. }
function PortablePathIs(const Registered, Directory: String): Boolean;
begin
  Result := (Registered <> '') and (NormalizeEntry(Registered) = NormalizeEntry(Directory));
end;

{ Setup owns this directory from now on, so the ZIP channel's bookkeeping for
  it has to go: leaving it behind has both channels claiming one directory, and
  Uninstall-NodePaper.ps1 would later take the Path entry of a Setup
  installation away. A registration naming any other directory is a real
  portable installation of its own and is never touched. }
procedure ReleasePortableRegistration(const Directory: String);
begin
  if PortablePathIs(RegisteredPortablePath, Directory) then
    RegDeleteValue(HKEY_CURRENT_USER, PortableKey, PortableValueName);
end;

{ ---------- payload verification ---------------------------------------- }

function ExecutableIsInUse(const FileName: String): Boolean;
var
  Probe: String;
begin
  Result := False;
  if not FileExists(FileName) then
    Exit;
  Probe := FileName + '.inuse-probe';
  if FileExists(Probe) then
    DeleteFile(Probe);
  if RenameFile(FileName, Probe) then
    RenameFile(Probe, FileName)
  else
    Result := True;
end;

function VerifyInstalledPayload(var FailedFile: String): Boolean;
var
  Lines: TArrayOfString;
  Index, SeparatorPos: Integer;
  Line, Expected, Relative, FullPath: String;
begin
  Result := False;
  FailedFile := '';
  ExtractTemporaryFile(ExtractFileName('{#ChecksumFile}'));
  if not LoadStringsFromFile(ExpandConstant('{tmp}\') + ExtractFileName('{#ChecksumFile}'), Lines) then
  begin
    FailedFile := ExtractFileName('{#ChecksumFile}');
    Exit;
  end;
  for Index := 0 to GetArrayLength(Lines) - 1 do
  begin
    Line := Trim(Lines[Index]);
    if Line = '' then
      Continue;
    SeparatorPos := Pos('|', Line);
    if SeparatorPos <= 0 then
    begin
      FailedFile := Line;
      Exit;
    end;
    Expected := Lowercase(Trim(Copy(Line, 1, SeparatorPos - 1)));
    Relative := Trim(Copy(Line, SeparatorPos + 1, Length(Line) - SeparatorPos));
    StringChangeEx(Relative, '/', '\', False);
    FullPath := AddBackslash(ExpandConstant('{app}')) + Relative;
    if not FileExists(FullPath) then
    begin
      FailedFile := Relative;
      Exit;
    end;
    if Lowercase(GetSHA256OfFile(FullPath)) <> Expected then
    begin
      FailedFile := Relative;
      Exit;
    end;
  end;
  Result := True;
end;

function VerifyInstalledVersion(var Reported: String): Boolean;
var
  OutputFile, Command: String;
  Lines: TArrayOfString;
  ResultCode: Integer;
begin
  Result := False;
  Reported := '';
  OutputFile := ExpandConstant('{tmp}\nodepaper-version.txt');
  if FileExists(OutputFile) then
    DeleteFile(OutputFile);
  Command := '/C ""' + AddBackslash(ExpandConstant('{app}')) + 'nodepaper.exe" --version > "' + OutputFile + '""';
  if not Exec(ExpandConstant('{cmd}'), Command, ExpandConstant('{app}'), SW_HIDE, ewWaitUntilTerminated, ResultCode) then
    Exit;
  if ResultCode <> 0 then
    Exit;
  if not LoadStringsFromFile(OutputFile, Lines) then
    Exit;
  if GetArrayLength(Lines) < 1 then
    Exit;
  Reported := Trim(Lines[0]);
  Result := Reported = 'nodepaper {#NodePaperVersion}';
end;

procedure UndoFailedInstallation;
begin
  if PathEntryAdded then
  begin
    RemoveUserPathEntry(ExpandConstant('{app}'));
    PathEntryAdded := False;
  end;
  DeleteFile(ExpandConstant('{autodesktop}\{#NodePaperName}.lnk'));
  DelTree(ExpandConstant('{group}'), True, True, True);
  DelTree(ExpandConstant('{app}'), True, True, True);
  RegDeleteKeyIncludingSubkeys(HKEY_CURRENT_USER, UninstallKey);
end;

{ ---------- wizard flow ------------------------------------------------- }

function InitializeSetup: Boolean;
var
  Installed: String;
begin
  Result := True;
  PathEntryAdded := False;
  Installed := InstalledNodePaperVersion;
  if (Installed <> '') and (CompareNodePaperVersions(Installed, '{#NodePaperVersion}') > 0) then
    Result := MsgBox(FmtMessage(CustomMessage('DowngradeWarning'), [Installed, '{#NodePaperVersion}']),
      mbConfirmation, MB_YESNO) = IDYES;
end;

{ The Ready page is where Setup states what it is about to do, so the state of
  any existing installation belongs there rather than in one more dialog to
  click away. Purely informational: nothing here changes what Setup installs,
  and the downgrade confirmation stays in InitializeSetup. }
function UpdateReadyMemo(const Space, NewLine, MemoUserInfoInfo, MemoDirInfo,
  MemoTypeInfo, MemoComponentsInfo, MemoGroupInfo, MemoTasksInfo: String): String;
var
  Installed, Registered, Directory, Details: String;
  Compared: Integer;
begin
  { The stock memo first, unchanged. }
  Result := '';
  if MemoUserInfoInfo <> '' then
    Result := Result + MemoUserInfoInfo + NewLine + NewLine;
  if MemoDirInfo <> '' then
    Result := Result + MemoDirInfo + NewLine + NewLine;
  if MemoTypeInfo <> '' then
    Result := Result + MemoTypeInfo + NewLine + NewLine;
  if MemoComponentsInfo <> '' then
    Result := Result + MemoComponentsInfo + NewLine + NewLine;
  if MemoGroupInfo <> '' then
    Result := Result + MemoGroupInfo + NewLine + NewLine;
  if MemoTasksInfo <> '' then
    Result := Result + MemoTasksInfo + NewLine + NewLine;

  Directory := WizardDirValue;
  Details := '';

  Installed := InstalledNodePaperVersion;
  if Installed <> '' then
  begin
    Compared := CompareNodePaperVersions(Installed, '{#NodePaperVersion}');
    if Compared = 0 then
      Details := Details + Space +
        FmtMessage(CustomMessage('ExistingReinstall'), [Installed]) + NewLine
    else if Compared < 0 then
      Details := Details + Space +
        FmtMessage(CustomMessage('ExistingUpgrade'), [Installed, '{#NodePaperVersion}']) + NewLine
    else
      Details := Details + Space +
        FmtMessage(CustomMessage('ExistingDowngrade'), [Installed, '{#NodePaperVersion}']) + NewLine;
  end;

  Registered := RegisteredPortablePath;
  if Registered <> '' then
  begin
    if PortablePathIs(Registered, Directory) then
      Details := Details + Space + CustomMessage('PortableHere') + NewLine
    else
      Details := Details + Space +
        FmtMessage(CustomMessage('PortableElsewhere'), [Registered]) + NewLine;
  end;

  { Nothing detected: the memo stays exactly as it was. }
  if Details <> '' then
    Result := Result + CustomMessage('ExistingHeading') + NewLine + Details;
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
begin
  Result := '';
  NeedsRestart := False;
  if ExecutableIsInUse(AddBackslash(ExpandConstant('{app}')) + 'nodepaper.exe') then
    Result := CustomMessage('NodePaperRunning');
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  FailedFile, Reported: String;
begin
  if CurStep <> ssPostInstall then
    Exit;
  { Verify the installed payload and program version before touching the user
    Path. A failure removes everything this Setup created. }
  if not VerifyInstalledPayload(FailedFile) then
  begin
    UndoFailedInstallation;
    MsgBox(FmtMessage(CustomMessage('PayloadBroken'), [FailedFile]), mbCriticalError, MB_OK);
    Abort;
  end;
  if not VerifyInstalledVersion(Reported) then
  begin
    UndoFailedInstallation;
    MsgBox(FmtMessage(CustomMessage('VersionMismatch'), [Reported]), mbCriticalError, MB_OK);
    Abort;
  end;
  AddUserPathEntry(ExpandConstant('{app}'));
  PathEntryAdded := True;
  { Only after the Path entry exists: until then this installation is not yet
    reachable, and dropping the ZIP registration first would leave the
    directory unaccounted for by either channel if the step above failed. }
  ReleasePortableRegistration(ExpandConstant('{app}'));
end;

function InitializeUninstall: Boolean;
begin
  Result := True;
  if ExecutableIsInUse(AddBackslash(ExpandConstant('{app}')) + 'nodepaper.exe') then
  begin
    MsgBox(CustomMessage('NodePaperRunning'), mbError, MB_OK);
    Result := False;
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  { Remove only the exact installation directory entry. Projects, PDFs, TeX,
    Pandoc, Node.js, Git and unrelated Path entries are never touched. }
  if CurUninstallStep = usUninstall then
  begin
    RemoveUserPathEntry(ExpandConstant('{app}'));
    { A registration naming this directory can only be one this Setup took
      over, or one left pointing at a directory that is about to be emptied.
      Either way it would outlive the installation it describes. }
    ReleasePortableRegistration(ExpandConstant('{app}'));
  end;
end;

[UninstallDelete]
; Remove the installation directory itself once its installed files are gone.
Type: dirifempty; Name: "{app}\tools\windows-x64\sources"
Type: dirifempty; Name: "{app}\tools\windows-x64\pandoc-crossref"
Type: dirifempty; Name: "{app}\tools\windows-x64\pandoc"
Type: dirifempty; Name: "{app}\tools\windows-x64"
Type: dirifempty; Name: "{app}\tools"
Type: dirifempty; Name: "{app}"
