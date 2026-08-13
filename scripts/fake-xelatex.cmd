@echo off
rem Dispatcher for the fake XeLaTeX (scripts/fake-xelatex.ps1). It exists so
rem Build-Paper.ps1 can resolve a "xelatex" command on PATH in the convergence
rem tests without a real TeX installation.
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0fake-xelatex.ps1" %*
exit /b %ERRORLEVEL%
