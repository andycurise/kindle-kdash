@echo off
setlocal
chcp 65001 >nul
cd /d "%~dp0"

echo Requesting graceful shutdown for this kdash installation only.
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0uninstall_task.ps1" -StopOnly %*
set "KDASH_EXIT=%ERRORLEVEL%"
if not "%KDASH_EXIT%"=="0" echo [ERROR] Stop failed with code %KDASH_EXIT%.
pause
exit /b %KDASH_EXIT%
