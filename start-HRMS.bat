@echo off
rem ============================================================================
rem  HRMS - run the local server and open the app in the browser (Windows).
rem  Double-click this file, or run it from a terminal.
rem
rem  - First run builds server\bin\hrmsd.exe (needs Go on PATH or in
rem    %USERPROFILE%\tools\go\bin). Later runs start instantly.
rem  - To force a rebuild after code changes:  start-HRMS.bat rebuild
rem  - Data lives in PostgreSQL. If a local cluster exists at
rem    %USERPROFILE%\tools\pg it is started automatically before the server.
rem  - Secrets come from Infisical when its CLI is available; otherwise from
rem    server\.env (HRMS_JWT_SECRET, HRMS_DATABASE_URL).
rem  - The active company is bandar_imam (change %%ORG%% below for another one);
rem    the list of companies is at GET /api/v1/orgs.
rem  - Keep this window open while you use the app; closing it stops the server.
rem ============================================================================
setlocal
title HRMS local server (127.0.0.1:8080)
cd /d "%~dp0"

set "ADDR=127.0.0.1:8080"
set "ORG=bandar_imam"
set "URL=http://%ADDR%/"
set "EXE=%~dp0server\bin\hrmsd.exe"
set "PGROOT=%USERPROFILE%\tools\pg"
set "PGDATA=%PGROOT%\data"
set "PGLOG=%PGROOT%\pg.log"

if /i "%~1"=="rebuild" del /q "%EXE%" 2>nul

rem --- If the server is already answering, just open the browser. -------------
curl -s -o nul --max-time 3 %URL%readyz >nul 2>nul
if not errorlevel 1 (
  echo [*] HRMS is already running on %ADDR%.
  echo     Opening %URL%
  start "" %URL%
  echo.
  echo     To restart it, close the other HRMS window first.
  pause
  exit /b 0
)

rem --- Start the local PostgreSQL cluster if installed and not running. -------
if exist "%PGROOT%\bin\pg_ctl.exe" if exist "%PGDATA%\postgresql.conf" (
  "%PGROOT%\bin\pg_ctl.exe" -D "%PGDATA%" status >nul 2>nul
  if errorlevel 1 (
    echo [*] Starting the local PostgreSQL database ...
    "%PGROOT%\bin\pg_ctl.exe" -D "%PGDATA%" -l "%PGLOG%" -o "-p 5432 -c listen_addresses=127.0.0.1" -w start
  )
)

rem --- Locate Go (PATH first, then common install paths). ---------------------
set "GOEXE="
where go >nul 2>nul && set "GOEXE=go"
if not defined GOEXE if exist "%USERPROFILE%\tools\go\bin\go.exe" set "GOEXE=%USERPROFILE%\tools\go\bin\go.exe"
if not defined GOEXE if exist "C:\Go\bin\go.exe" set "GOEXE=C:\Go\bin\go.exe"

rem --- Secrets: Infisical when available, otherwise the local server\.env. ----
set "USE_INFISICAL="
where infisical >nul 2>nul
if not errorlevel 1 set "USE_INFISICAL=1"
if not defined USE_INFISICAL if not exist "%~dp0server\.env" (
  echo [ERROR] No secrets source was found.
  echo         Either install the Infisical CLI ^(https://infisical.com/docs/cli/overview^)
  echo         and run "infisical login" and "infisical init" in the server folder,
  echo         or create server\.env with HRMS_JWT_SECRET and HRMS_DATABASE_URL.
  echo.
  pause
  exit /b 1
)

rem --- Build once if the binary is missing. ----------------------------------
if exist "%EXE%" goto run
if defined GOEXE goto build
echo [ERROR] Go was not found on this computer.
echo         Install Go from https://go.dev/dl/ then run this file again,
echo         or make sure %USERPROFILE%\tools\go\bin\go.exe exists.
echo.
pause
exit /b 1

:build
echo [*] Building the HRMS server (first run only)...
pushd "%~dp0server"
"%GOEXE%" build -o "bin\hrmsd.exe" ".\cmd\hrmsd"
if errorlevel 1 goto buildfail
popd
echo [*] Build finished.
echo.

:run
echo ============================================================================
echo   HRMS is starting on %ADDR%
echo   Open in your browser:  %URL%
echo   (Use 127.0.0.1, not localhost, if localhost does not respond.)
echo   Keep this window open. Press Ctrl+C or close it to stop the server.
echo ============================================================================
echo.

rem Open the default browser a couple of seconds after the server starts.
start "" cmd /c "timeout /t 2 /nobreak >nul & explorer %URL%"

pushd "%~dp0server"
if defined USE_INFISICAL (
  infisical run --env=dev -- "bin\hrmsd.exe" -addr %ADDR% -org %ORG% -web "..\design\prototype" -root . -env ".env"
) else (
  "bin\hrmsd.exe" -addr %ADDR% -org %ORG% -web "..\design\prototype" -root . -env ".env"
)
popd

echo.
echo [*] Server stopped.
pause
exit /b 0

:buildfail
popd
echo [ERROR] Build failed. The messages above show why.
echo.
pause
exit /b 1
