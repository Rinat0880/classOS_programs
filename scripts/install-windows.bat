@echo off
setlocal enabledelayedexpansion

REM Check for administrator privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo ERROR: Administrator privileges required for installation
    echo Please run this script as administrator
    pause
    exit /b 1
)

echo ============================================
echo     School Agent Service Installation
echo ============================================

REM Installation settings
set SERVICE_NAME=SchoolAgent
set SERVICE_DISPLAY_NAME=School Agent Process Controller
set SERVICE_DESCRIPTION=Controls program execution based on whitelist
set INSTALL_DIR=C:\Program Files\SchoolAgent
set DATA_DIR=C:\ProgramData\SchoolAgent
set CONFIG_DIR=%DATA_DIR%\config
set LOGS_DIR=%DATA_DIR%\logs
set EXECUTABLE="%INSTALL_DIR%\school_agent.exe"

echo Installation settings:
echo - Service: %SERVICE_NAME%
echo - Installation directory: %INSTALL_DIR%
echo - Data directory: %DATA_DIR%
echo - Executable: %EXECUTABLE%
echo.

REM Stop service if already running
echo Checking for existing service...
sc query %SERVICE_NAME% >nul 2>&1
if %errorLevel% equ 0 (
    echo Found existing service, stopping...
    sc stop %SERVICE_NAME%
    timeout /t 5 /nobreak >nul
    
    echo Removing old service...
    sc delete %SERVICE_NAME%
    timeout /t 2 /nobreak >nul
)

REM Create necessary directories
echo Creating directories...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
if not exist "%DATA_DIR%" mkdir "%DATA_DIR%"
if not exist "%CONFIG_DIR%" mkdir "%CONFIG_DIR%"
if not exist "%LOGS_DIR%" mkdir "%LOGS_DIR%"

REM Debug: Show current directory contents
echo Current directory contents:
dir *.exe

REM Copy executable file - check multiple possible names
echo Copying files...
set FOUND_FILE=0

if exist "school_agent.exe" (
    echo Found school_agent.exe
    copy /Y "school_agent.exe" "%INSTALL_DIR%\school_agent.exe"
    set FOUND_FILE=1
) else if exist "school-agent.exe" (
    echo Found school-agent.exe
    copy /Y "school-agent.exe" "%INSTALL_DIR%\school_agent.exe"
    set FOUND_FILE=1
) else if exist "SchoolAgent.exe" (
    echo Found SchoolAgent.exe
    copy /Y "SchoolAgent.exe" "%INSTALL_DIR%\school_agent.exe"
    set FOUND_FILE=1
)

if %FOUND_FILE% equ 0 (
    echo ERROR: No executable file found in current directory
    echo Looking for one of: school_agent.exe, school-agent.exe, SchoolAgent.exe
    echo Current directory: %CD%
    echo Available exe files:
    dir *.exe
    pause
    exit /b 1
)

REM Check if copy was successful
if %errorLevel% neq 0 (
    echo ERROR: Failed to copy executable file
    pause
    exit /b 1
)

echo File copied successfully!

REM Create configuration file if it doesn't exist
set CONFIG_FILE=%CONFIG_DIR%\agent.json
if not exist "%CONFIG_FILE%" (
    echo Creating configuration file...
    (
        echo {
        echo     "log_path": "%LOGS_DIR%\\agent.log",
        echo     "whitelist_path": "%DATA_DIR%\\whitelist.json",
        echo     "whitelist_url": "",
        echo     "server_url": "",
        echo     "update_interval": 30,
        echo     "log_level": "info",
        echo     "username": "",
        echo     "password": "",
        echo     "debug_mode": false,
        echo     "dry_run": false
        echo }
    ) > "%CONFIG_FILE%"
)

REM Create basic whitelist if it doesn't exist
set WHITELIST_FILE=%DATA_DIR%\whitelist.json
if not exist "%WHITELIST_FILE%" (
    echo Creating basic whitelist...
    (
        echo {
        echo     "version": "1.0.0-default",
        echo     "updated_at": "2024-01-01T00:00:00Z",
        echo     "items": [
        echo         "C:\\Windows\\System32\\notepad.exe",
        echo         "C:\\Windows\\System32\\calc.exe",
        echo         "C:\\Windows\\System32\\mspaint.exe",
        echo         "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
        echo         "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
        echo         "C:\\Program Files\\Mozilla Firefox\\firefox.exe",
        echo         "C:\\Program Files (x86)\\Mozilla Firefox\\firefox.exe"
        echo     ]
        echo }
    ) > "%WHITELIST_FILE%"
)

REM Install Windows service
echo Installing Windows service...
sc create %SERVICE_NAME% binPath= %EXECUTABLE% start= auto DisplayName= "%SERVICE_DISPLAY_NAME%"
if %errorLevel% neq 0 (
    echo ERROR: Failed to create service
    pause
    exit /b 1
)

REM Configure service description
echo Configuring service description...
sc description %SERVICE_NAME% "%SERVICE_DESCRIPTION%"
if %errorLevel% neq 0 (
    echo WARNING: Failed to set service description. Continuing.
)

REM Configure failure actions (automatic restart)
echo Configuring failure actions (restart on failure)...
sc failure %SERVICE_NAME% reset= 60 actions= restart/5000/restart/5000/restart/5000
if %errorLevel% neq 0 (
    echo WARNING: Failed to configure failure actions. Continuing.
)

REM Start the service
echo Starting service %SERVICE_NAME%...
sc start %SERVICE_NAME%
if %errorLevel% neq 0 (
    echo ERROR: Failed to start service
    echo Check Windows Event Log for detailed information.
    pause
    exit /b 1
)

echo ============================================
echo Service "%SERVICE_DISPLAY_NAME%" successfully installed and started!
echo ============================================
echo.
echo Executable path: %EXECUTABLE%
echo Configuration file: %CONFIG_FILE%
echo Whitelist file: %WHITELIST_FILE%
echo Logs directory: %LOGS_DIR%
echo.
echo Service Management Commands:
echo - Start service: sc start %SERVICE_NAME%
echo - Stop service: sc stop %SERVICE_NAME%
echo - Check status: sc query %SERVICE_NAME%
echo - Uninstall: sc delete %SERVICE_NAME%
echo.
pause
exit /b 0