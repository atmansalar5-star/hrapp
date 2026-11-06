@echo off
rem =====================================================================
rem  HR App promo site - launcher (ASCII only: cmd codepage safety)
rem  Runs serve-site.ps1 which starts a local static server and opens
rem  the site in the default browser. Close this window to stop it.
rem =====================================================================
setlocal
title HR App - promo site server
chcp 65001 >nul 2>nul
set "PS=%SystemRoot%\System32\WindowsPowerShell\v1.0\powershell.exe"
if not exist "%PS%" set "PS=powershell.exe"
"%PS%" -NoProfile -ExecutionPolicy Bypass -File "%~dp0serve-site.ps1" %*
if errorlevel 1 (
  echo.
  echo   Could not start the local server. Opening the page directly...
  echo.
  start "" "%~dp0index.html"
)
endlocal
