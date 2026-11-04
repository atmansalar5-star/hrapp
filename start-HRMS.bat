@echo off
rem ============================================================================
rem  HRMS - start the local server and open the app in the browser (Windows).
rem
rem  Double-click this file, or run it from a terminal.
rem
rem  MODES (default: build from source and run in this window)
rem    start-HRMS.bat            stop the boot task, build from source, run here
rem    start-HRMS.bat nobuild    run the existing binary, no build
rem    start-HRMS.bat rebuild    delete the binary and do a full build
rem    start-HRMS.bat status     report: Go, binary, database, boot task, server health
rem    start-HRMS.bat task       hand the server back to the "HRMS Server" task
rem    start-HRMS.bat help       this help
rem
rem  WHY THE MESSAGES ARE ASCII: cmd.exe reads this file byte-wise in the console
rem  code page; UTF-8 Persian text in a .bat file is split mid-character and gets
rem  executed as commands. Data, logs and the web app are Persian; this file is not.
rem
rem  THIS MACHINE
rem   - PostgreSQL listens on 127.0.0.1:5432 and comes up either through the
rem     Windows service "HRMS-PostgreSQL" (preferred) or, when the service is not
rem     registered, through pg_ctl on the local cluster in %USERPROFILE%\tools\pg.
rem   - The scheduled task "HRMS Server" starts server\bin\hrmsd.exe on
rem     127.0.0.1:8099 at every boot. For development this file stops that task so
rem     two servers never share the port; when you are done run
rem     "start-HRMS.bat task" to hand the server back to the task.
rem   - Secrets come from server\.env (HRMS_JWT_SECRET, HRMS_DATABASE_URL,
rem     HRMS_SEED_DEMO); with the Infisical CLI installed they come from it instead.
rem   - Server data lives in server\data (company logos, personnel photos, data
rem     exchange files). It is not code and is backed up separately.
rem   - Active company: bandar_imam. Company list: GET /api/v1/orgs
rem ============================================================================
setlocal EnableExtensions EnableDelayedExpansion
rem UTF-8 console: the server prints Persian logs, which would otherwise show as
rem question marks in this window.
chcp 65001 >nul 2>nul
title HRMS local server (127.0.0.1:8099)
cd /d "%~dp0"

set "ADDR=127.0.0.1:8099"
set "ORG=bandar_imam"
set "URL=http://%ADDR%/"
set "EXE=%~dp0server\bin\hrmsd.exe"
set "DATADIR=%~dp0server\data"
set "PGROOT=%USERPROFILE%\tools\pg"
set "PGDATA=%PGROOT%\data"
set "PGLOG=%PGROOT%\pg.log"
set "PGSVC=HRMS-PostgreSQL"
set "TASK=HRMS Server"
set "MODE=%~1"
if "%MODE%"=="" set "MODE=run"
set "NOPAUSE="
if /i "%~2"=="nopause" set "NOPAUSE=1"

if /i "%MODE%"=="help" goto help
if /i "%MODE%"=="-h" goto help
if /i "%MODE%"=="status" goto status
if /i "%MODE%"=="task" goto task
if /i "%MODE%"=="run" goto prepare
if /i "%MODE%"=="nobuild" goto prepare
if /i "%MODE%"=="rebuild" goto prepare

echo [warn] Unknown mode: %MODE%
echo.
goto help

:prepare
rem --- Server data folder (company logos, uploaded files) ---------------------
call :ensure_data

rem --- Database: Windows service first, local pg_ctl cluster otherwise --------
call :ensure_pg

rem --- Locate Go (PATH first, then the usual install paths) -------------------
call :find_go

rem --- Secrets: Infisical when installed, otherwise server\.env --------------
set "USE_INFISICAL="
where infisical >nul 2>nul
if not errorlevel 1 set "USE_INFISICAL=1"
if not defined USE_INFISICAL if not exist "%~dp0server\.env" (
  echo [ERROR] No secrets source was found.
  echo         Create server\.env with HRMS_JWT_SECRET and HRMS_DATABASE_URL,
  echo         or install the Infisical CLI and run it from the server folder.
  echo.
  call :maybe_pause
  exit /b 1
)

rem --- Stop the boot task so the dev server can own the port ----------------
call :stop_task
if "%TASKSTOPPED%"=="0" (
  echo [warn] The task did not stop; if its server is still up, port %ADDR% stays busy.
)

rem --- Build the server from the current source before every start ----------
rem  Why: a stale hrmsd.exe silently serves yesterday's rules while the sources
rem  have moved on. Go build is incremental, so this costs a second or two.
set "SKIPBUILD="
if /i "%MODE%"=="nobuild" set "SKIPBUILD=1"

if defined SKIPBUILD goto run
if not defined GOEXE if not exist "%EXE%" (
  echo [ERROR] Go was not found and no existing binary is available.
  echo         Install Go from https://go.dev/dl/ or make sure
  echo         %USERPROFILE%\tools\go\bin\go.exe exists.
  echo.
  call :maybe_pause
  exit /b 1
)
if not defined GOEXE (
  echo [warn] Go was not found; running the existing binary - it may be out of date.
  goto run
)

if /i "%MODE%"=="rebuild" (
  echo [*] Removing the binary for a full rebuild ...
  del /q "%EXE%" 2>nul
)
echo [*] Building the HRMS server from the current source ...
pushd "%~dp0server"
"%GOEXE%" build -trimpath -o "bin\hrmsd.exe" ".\cmd\hrmsd"
if errorlevel 1 (
  popd
  if exist "%EXE%" (
    echo [warn] Build failed; starting the previous binary instead.
    goto run
  )
  echo [ERROR] Build failed. The messages above show why.
  echo.
  call :maybe_pause
  exit /b 1
)
popd
echo [*] Build finished.

rem --- If a server is already answering, just open the browser --------------
call :readyz
if "%READY%"=="1" (
  echo [*] HRMS is already running on %ADDR%.
  echo     Opening %URL%
  start "" %URL%
  echo.
  echo     To restart it, close that HRMS window or run "start-HRMS.bat task"
  echo     and then start this file again.
  call :maybe_pause
  exit /b 0
)

:run
echo ============================================================================
echo   HRMS is starting on %ADDR%
echo   Open in your browser:  %URL%
echo   Use 127.0.0.1, not localhost, if localhost does not respond.
echo   Keep this window open. Press Ctrl+C or close it to stop the server.
echo   Back to the boot task when you are done:  start-HRMS.bat task
echo ============================================================================
echo.

start "" cmd /c "timeout /t 2 /nobreak >nul & explorer %URL%"

pushd "%~dp0server"
if defined USE_INFISICAL (
  infisical run --env=dev -- "bin\hrmsd.exe" -addr %ADDR% -org %ORG% -web "..\design\prototype" -root . -env ".env" -data "data"
) else (
  "bin\hrmsd.exe" -addr %ADDR% -org %ORG% -web "..\design\prototype" -root . -env ".env" -data "data"
)
set "SERVEREXIT=%ERRORLEVEL%"
popd

echo.
if not "%SERVEREXIT%"=="0" echo [warn] The server exited with code %SERVEREXIT%.
echo [*] Server stopped.
call :task_state
if not "%TASKSTATE%"=="" echo     Hand the server back to the boot task with:  start-HRMS.bat task
echo.
call :maybe_pause
exit /b 0

:task
rem --- Hand the server back to the scheduled task (no window) ---------------
call :task_exists
if "%TASKEXISTS%"=="0" (
  echo [warn] The scheduled task "%TASK%" is not registered on this machine.
  echo     Setup steps - needs administrator - are in AGENTS.md, section 2.
  echo.
  call :maybe_pause
  exit /b 1
)
call :task_state
if /i "%TASKSTATE%"=="Running" (
  call :readyz
  if "%READY%"=="1" (
    echo [*] Task "%TASK%" is already running: %URL%
  ) else (
    echo [warn] Task "%TASK%" is running but the server does not answer; see server\server.log
  )
  echo.
  call :maybe_pause
  exit /b 0
)
echo [*] Starting task "%TASK%" ...
schtasks /Run /TN "%TASK%" >nul 2>nul
echo [*] Waiting for the server on %ADDR% ...
for /l %%i in (1,1,20) do (
  call :readyz
  if "!READY!"=="1" goto task_ready
  call :sleep 2
)
echo [warn] No answer after about 40 seconds; check server\server.log
echo.
call :maybe_pause
exit /b 1

:task_ready
echo [*] Task server is up: %URL%
echo.
call :maybe_pause
exit /b 0

:status
echo ============================================================================
echo   HRMS status report
echo ============================================================================
echo.
echo -- Environment ------------------------------------------------------------
call :find_go
if defined GOEXE (echo    Go:             %GOEXE%) else (echo    Go:             not found)
echo    Server binary:  %EXE%
if exist "%EXE%" (
  for %%f in ("%EXE%") do echo    Built at:       %%~tf  [%%~zf bytes]
) else (
  echo    Built at:       binary missing
)
echo    Active company: %ORG%
echo.
echo -- Database ---------------------------------------------------------------
call :pg_state
echo    Service %PGSVC%: %PGSVCFA%
call :pg_port
echo    Port 5432:      %PGPORTFA%
if exist "%PGDATA%\postgresql.conf" (echo    Local cluster:  %PGDATA%) else (echo    Local cluster:  not created)
echo.
echo -- Boot task "%TASK%" -----------------------------------------------------
call :task_exists
call :task_state
if "%TASKEXISTS%"=="0" (echo    State:          not registered) else (echo    State:          %TASKSTATE%)
echo.
echo -- Server on %ADDR% ------------------------------------------------------
call :readyz
if "%READY%"=="1" (
  echo    /readyz:        %READYTEXT%
) else (
  echo    /readyz:        no answer
)
echo.
echo -- URLs -------------------------------------------------------------------
echo    App:            %URL%
echo    Health:         %URL%readyz
echo    Companies:      %URL%api/v1/orgs
echo.
call :maybe_pause
exit /b 0

:help
echo ============================================================================
echo   HRMS local server - usage
echo ============================================================================
echo.
echo    start-HRMS.bat            stop the boot task, build from source, run here
echo    start-HRMS.bat nobuild    run the existing binary, no build
echo    start-HRMS.bat rebuild    full rebuild - deletes the binary first
echo    start-HRMS.bat status     Go, binary, database, boot task, server health
echo    start-HRMS.bat task       hand the server back to the "HRMS Server" task
echo    start-HRMS.bat help       this help
echo.
echo    At every boot the task "HRMS Server" and the service "HRMS-PostgreSQL"
echo    start the stack automatically; this file is for development runs.
echo.
call :maybe_pause
exit /b 0

rem ============================================================================
rem  Subroutines
rem ============================================================================

:ensure_data
if not exist "%DATADIR%" mkdir "%DATADIR%" 2>nul
if not exist "%DATADIR%\branding" mkdir "%DATADIR%\branding" 2>nul
if not exist "%DATADIR%\uploads" mkdir "%DATADIR%\uploads" 2>nul
if not exist "%DATADIR%\photos" mkdir "%DATADIR%\photos" 2>nul
exit /b 0

:ensure_pg
rem A registered Windows service is the standard path (no logon needed).
call :pg_state
if /i "%PGSVCFA%"=="Running" exit /b 0
if /i not "%PGSVCFA%"=="not registered" (
  echo [*] Starting the database service "%PGSVC%" ...
  net start "%PGSVC%" >nul 2>nul
  if errorlevel 1 (
    echo [warn] The service did not start - service control needs administrator.
    echo     Harmless when the database is already up; otherwise run this window
    echo     as administrator.
  )
  exit /b 0
)
rem No service: bring up the local cluster with pg_ctl.
if exist "%PGROOT%\bin\pg_ctl.exe" if exist "%PGDATA%\postgresql.conf" (
  "%PGROOT%\bin\pg_ctl.exe" -D "%PGDATA%" status >nul 2>nul
  if errorlevel 1 (
    echo [*] Starting the local PostgreSQL cluster ...
    "%PGROOT%\bin\pg_ctl.exe" -D "%PGDATA%" -l "%PGLOG%" -o "-p 5432 -c listen_addresses=127.0.0.1" -w start
  )
)
exit /b 0
:stop_task
rem Stops the boot task and waits until the scheduler reports it left Running.
set "TASKSTOPPED=1"
call :task_state
if /i not "%TASKSTATE%"=="Running" exit /b 0
echo [*] Stopping the scheduled task "%TASK%" so the dev server can bind %ADDR% ...
schtasks /End /TN "%TASK%" >nul 2>nul
for /l %%i in (1,1,6) do (
  call :sleep 2
  call :task_state
  if /i "!TASKSTATE!"=="Running" (
    set "TASKSTOPPED=0"
  ) else (
    set "TASKSTOPPED=1"
    exit /b 0
  )
)
exit /b 0

:sleep
rem Sleep independent of an interactive console (timeout fails when stdin is
rem redirected, which happens when output is captured to a file or a job).
set "SECS=%~1"
if "%SECS%"=="" set "SECS=2"
set /a "PINGS=%SECS%+1"
ping -n %PINGS% 127.0.0.1 >nul 2>nul
exit /b 0

:find_go
set "GOEXE="
where go >nul 2>nul && set "GOEXE=go"
if not defined GOEXE if exist "%USERPROFILE%\tools\go\bin\go.exe" set "GOEXE=%USERPROFILE%\tools\go\bin\go.exe"
if not defined GOEXE if exist "C:\Go\bin\go.exe" set "GOEXE=C:\Go\bin\go.exe"
exit /b 0

:task_exists
set "TASKEXISTS=0"
powershell -NoProfile -Command "if (Get-ScheduledTask -TaskName '%TASK%' -ErrorAction SilentlyContinue) { exit 0 } else { exit 1 }" >nul 2>nul
if not errorlevel 1 set "TASKEXISTS=1"
exit /b 0

:task_state
set "TASKSTATE="
for /f "usebackq delims=" %%s in (`powershell -NoProfile -Command "(Get-ScheduledTask -TaskName '%TASK%' -ErrorAction SilentlyContinue).State"`) do set "TASKSTATE=%%s"
if not defined TASKSTATE set "TASKSTATE=unknown"
exit /b 0

:pg_state
set "PGSVCFA=not registered"
for /f "usebackq delims=" %%s in (`powershell -NoProfile -Command "(Get-Service -Name '%PGSVC%' -ErrorAction SilentlyContinue).Status"`) do set "PGSVCFA=%%s"
exit /b 0

:pg_port
set "PGPORTFA=closed"
powershell -NoProfile -Command "if (Test-NetConnection -ComputerName 127.0.0.1 -Port 5432 -InformationLevel Quiet) { exit 0 } else { exit 1 }" >nul 2>nul
if not errorlevel 1 set "PGPORTFA=open"
exit /b 0

:readyz
set "READY=0"
set "READYTEXT="
for /f "usebackq delims=" %%r in (`curl -s --max-time 3 %URL%readyz 2^>nul`) do (
  set "READYTEXT=%%r"
  set "READY=1"
)
exit /b 0

:maybe_pause
if defined NOPAUSE exit /b 0
if not defined HRMS_NO_PAUSE pause
exit /b 0