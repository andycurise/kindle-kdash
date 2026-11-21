@echo off
setlocal
chcp 65001 >nul
cd /d "%~dp0"

echo Registering a Limited scheduled task for the current user's logon.
echo This is not an SCM service and does not run while logged off.
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0install_task.ps1" %*
set "KDASH_EXIT=%ERRORLEVEL%"
if not "%KDASH_EXIT%"=="0" echo [ERROR] Task installation failed with code %KDASH_EXIT%.
pause
exit /b %KDASH_EXIT%
