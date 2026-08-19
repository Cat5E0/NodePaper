; NodePaper Windows x64 Setup definition (Inno Setup 6, pinned by
; packaging/windows/innosetup-toolchain.json).
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
;
; Command-line switches of this Setup, in addition to Inno's own:
;   /ALLOWDOWNGRADE   install this version over a newer installed one without
;                     asking. Without it the downgrade confirmation appears,
;                     and a silent run (/SILENT or /VERYSILENT, with or
;                     without /SUPPRESSMSGBOXES) refuses the downgrade and
;                     leaves the installed version alone, because that
;                     confirmation defaults to No. /ALLOWDOWNGRADE=0 declines
;                     just as clearly as leaving the switch out.

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
chinesesimplified.PortableHere=该目录当前是一份便携安装（解压式），安装后将由本程序接管。
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
english.PortableHere=This directory currently holds a portable (extracted ZIP) installation; this installation takes it over.

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
  { Setup's own uninstaller. UninstallFilesDir puts it in the installation
    directory, so its absence beside a nodepaper.exe is what marks a directory
    as the ZIP channel's; see PortableInstallationAt. }
  SetupUninstallerName = 'unins000.exe';

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

{ ---------- ZIP channel detection --------------------------------------- }

{ Whether the directory currently holds a portable (extracted ZIP) NodePaper.
  A portable installation is described by its own directory and is recorded
  nowhere else: it has a nodepaper.exe and no unins000.exe. Setup used to read a
  global registry value instead (HKCU\Software\NodePaper\PortablePath) and had
  to delete it when it took a directory over; there is nothing to take over any
  more, because installing here writes Setup's own uninstaller into the
  directory and that is precisely what stops the ZIP channel's scripts from
  claiming it afterwards. }
function PortableInstallationAt(const Directory: String): Boolean;
var
  Base: String;
begin
  Result := False;
  if Trim(Directory) = '' then
    Exit;
  Base := AddBackslash(Directory);
  Result := FileExists(Base + 'nodepaper.exe') and not FileExists(Base + SetupUninstallerName);
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

{ ---------- command line ------------------------------------------------- }

{ Inno has no "was this switch given" function, so the parameters are scanned
  here. Both /ALLOWDOWNGRADE and /ALLOWDOWNGRADE=VALUE are accepted: the bare
  form is what someone types, and the =VALUE form is what a wrapper script
  builds from a variable, where =0 has to mean no rather than "the switch is
  present, so yes".

  ParamStr is read rather than Pos: the switch name is ASCII and is matched by
  its exact length, so a Path or directory argument holding non-ASCII
  characters cannot shift the comparison. }
function AllowDowngradeRequested: Boolean;
var
  Index: Integer;
  Param, Remainder, Value: String;
begin
  Result := False;
  for Index := 1 to ParamCount do
  begin
    Param := Trim(ParamStr(Index));
    if Lowercase(Copy(Param, 1, Length('/ALLOWDOWNGRADE'))) <> '/allowdowngrade' then
      Continue;
    Remainder := Copy(Param, Length('/ALLOWDOWNGRADE') + 1, Length(Param));
    if Remainder = '' then
    begin
      Result := True;
      Exit;
    end;
    if Copy(Remainder, 1, 1) = '=' then
    begin
      Value := Lowercase(Trim(Copy(Remainder, 2, Length(Remainder))));
      Result := (Value = '1') or (Value = 'yes') or (Value = 'true');
      Exit;
    end;
    { Anything else is a different switch that merely starts the same way. }
  end;
end;

{ ---------- wizard flow ------------------------------------------------- }

function InitializeSetup: Boolean;
var
  Installed: String;
begin
  Result := True;
  PathEntryAdded := False;
  Installed := InstalledNodePaperVersion;
  { A first installation, an upgrade and a repeat of the same version all pass
    straight through. Eight of nine surveyed Windows installers ask on none of
    them - 7-Zip, IrfanView, Notepad++, electron-builder, winget, Git for
    Windows and VS Code among them, the odd one out being JetBrains, which asks
    only when reinstalling an identical version - and Inno itself turns
    DirExistsWarning off when the directory belongs to the same application
    being upgraded. Only going backwards is worth a question. }
  if (Installed = '') or (CompareNodePaperVersions(Installed, '{#NodePaperVersion}') <= 0) then
    Exit;

  { /ALLOWDOWNGRADE is the caller stating the intent up front, which is what a
    script has instead of an answer to a dialog. }
  if AllowDowngradeRequested then
    Exit;

  { SuppressibleMsgBox, not MsgBox: /SUPPRESSMSGBOXES does not reach a MsgBox
    raised from [Code], so what a silent downgrade did here was decided by
    whatever happened next rather than by this line. Observed with the rc.8
    Setup over an installed rc.9: exit 1, no hang, no downgrade - but which
    step produced that exit was never established, so this change is about
    stating the outcome, not about repairing a diagnosed defect.

    The default is IDNO, matching both that observation and Git for Windows,
    which refuses a silent downgrade unless /ALLOWDOWNGRADE says otherwise.
    MB_DEFBUTTON2 puts the same answer under the Enter key in the dialog. }
  Result := SuppressibleMsgBox(FmtMessage(CustomMessage('DowngradeWarning'), [Installed, '{#NodePaperVersion}']),
    mbConfirmation, MB_YESNO or MB_DEFBUTTON2, IDNO) = IDYES;
end;

{ The Ready page is where Setup states what it is about to do, so the state of
  any existing installation belongs there rather than in one more dialog to
  click away. Purely informational: nothing here changes what Setup installs,
  and the downgrade confirmation stays in InitializeSetup. }
function UpdateReadyMemo(const Space, NewLine, MemoUserInfoInfo, MemoDirInfo,
  MemoTypeInfo, MemoComponentsInfo, MemoGroupInfo, MemoTasksInfo: String): String;
var
  Installed, Directory, Details: String;
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

  { Only what this directory shows for itself. A portable installation
    elsewhere on the machine used to be reported here from the global registry
    value; nothing records one any more, and there is nothing to read. }
  if PortableInstallationAt(Directory) then
    Details := Details + Space + CustomMessage('PortableHere') + NewLine;

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
  { SuppressibleMsgBox reports the failure when there is someone to read it and
    returns IDOK on its own when there is not. The rollback and the Abort are
    outside the box on purpose: a silent run that shows no dialog must still
    undo everything and still fail. }
  if not VerifyInstalledPayload(FailedFile) then
  begin
    UndoFailedInstallation;
    SuppressibleMsgBox(FmtMessage(CustomMessage('PayloadBroken'), [FailedFile]), mbCriticalError, MB_OK, IDOK);
    Abort;
  end;
  if not VerifyInstalledVersion(Reported) then
  begin
    UndoFailedInstallation;
    SuppressibleMsgBox(FmtMessage(CustomMessage('VersionMismatch'), [Reported]), mbCriticalError, MB_OK, IDOK);
    Abort;
  end;
  AddUserPathEntry(ExpandConstant('{app}'));
  PathEntryAdded := True;
  { Nothing else to release: a portable installation in this directory is
    described by the directory alone, and this Setup has already written its own
    unins000.exe into it, which is what tells Install-NodePaper.ps1 and
    Uninstall-NodePaper.ps1 that the directory is no longer theirs. }
end;

function InitializeUninstall: Boolean;
begin
  Result := True;
  if ExecutableIsInUse(AddBackslash(ExpandConstant('{app}')) + 'nodepaper.exe') then
  begin
    { As above: the answer to the box is irrelevant, Result stays False, so a
      silent uninstall of a running NodePaper still refuses rather than
      deleting files out from under it. }
    SuppressibleMsgBox(CustomMessage('NodePaperRunning'), mbError, MB_OK, IDOK);
    Result := False;
  end;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  { Remove only the exact installation directory entry. Projects, PDFs, TeX,
    Pandoc, Node.js, Git and unrelated Path entries are never touched. }
  if CurUninstallStep = usUninstall then
    RemoveUserPathEntry(ExpandConstant('{app}'));
end;

[UninstallDelete]
; Remove the installation directory itself once its installed files are gone.
Type: dirifempty; Name: "{app}\tools\windows-x64\sources"
Type: dirifempty; Name: "{app}\tools\windows-x64\pandoc-crossref"
Type: dirifempty; Name: "{app}\tools\windows-x64\pandoc"
Type: dirifempty; Name: "{app}\tools\windows-x64"
Type: dirifempty; Name: "{app}\tools"
Type: dirifempty; Name: "{app}"
