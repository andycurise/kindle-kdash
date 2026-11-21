@echo off
setlocal
chcp 65001 >nul
cd /d "%~dp0"

echo ========================================================
echo   Kindle Dashboard rendering server (foreground)
echo ========================================================

if not exist "%~dp0kdash.exe" (
    echo [ERROR] kdash.exe was not found in %~dp0
    pause
    exit /b 1
)
if not exist "%~dp0kdash.json" (
    echo [ERROR] kdash.json was not found in %~dp0
    pause
    exit /b 1
)

echo.
echo The listen address is read only from kdash.json.
echo Press Ctrl+C to stop this foreground process.
echo.

"%~dp0kdash.exe" -config "%~dp0kdash.json"
set "KDASH_EXIT=%ERRORLEVEL%"
if not "%KDASH_EXIT%"=="0" echo [ERROR] kdash.exe exited with code %KDASH_EXIT%.
pause
exit /b %KDASH_EXIT%
