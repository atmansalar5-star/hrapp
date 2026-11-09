@echo off
rem ---------------------------------------------------------------------------
rem HRMS - install the freshly built server binary and restart the serve task.
rem Run this file as Administrator (right click > Run as administrator).
rem   BIN\hrmsd-new2.exe  ->  BIN\hrmsd.exe   (old file kept as hrmsd-prev.exe)
rem ---------------------------------------------------------------------------
setlocal
set "BIN=%~dp0server\bin"

if not exist "%BIN%\hrmsd-new2.exe" (
  echo [HRMS] %BIN%\hrmsd-new2.exe not found. Build it first:
  echo        cd server ^&^& go build -trimpath -ldflags "-s -w" -o bin\hrmsd-new2.exe .\cmd\hrmsd
  pause
  exit /b 1
)

echo [HRMS] stopping running server ...
taskkill /F /IM hrmsd.exe >nul 2>&1
schtasks /End /TN "HRMS Server" >nul 2>&1
timeout /t 3 /nobreak >nul

echo [HRMS] installing new binary ...
if not exist "%BIN%\hrmsd-prev.exe" copy /Y "%BIN%\hrmsd.exe" "%BIN%\hrmsd-prev.exe" >nul 2>&1
copy /Y "%BIN%\hrmsd-new2.exe" "%BIN%\hrmsd.exe" >nul
if errorlevel 1 (
  echo [HRMS] copy failed - is the server still running?
  pause
  exit /b 1
)

echo [HRMS] starting task "HRMS Server" ...
schtasks /Run /TN "HRMS Server" >nul 2>&1
timeout /t 5 /nobreak >nul

echo [HRMS] listening sockets on 8099:
netstat -ano | findstr :8099
echo [HRMS] done.
endlocal
pause
