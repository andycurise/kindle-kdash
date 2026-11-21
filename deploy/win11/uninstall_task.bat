@echo off
setlocal
chcp 65001 >nul
cd /d "%~dp0"

echo Stopping this installation and removing only its current-user task.
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0uninstall_task.ps1" %*
set "KDASH_EXIT=%ERRORLEVEL%"
if not "%KDASH_EXIT%"=="0" echo [ERROR] Task removal failed with code %KDASH_EXIT%.
pause
exit /b %KDASH_EXIT%
