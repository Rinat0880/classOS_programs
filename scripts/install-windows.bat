@echo off
setlocal enabledelayedexpansion

:: Проверяем права администратора
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo ОШИБКА: Для установки требуются права администратора
    echo Запустите скрипт от имени администратора
    pause
    exit /b 1
)

echo ============================================
echo     Установка School Agent Service
echo ============================================

:: Настройки установки
set SERVICE_NAME=SchoolAgent
set SERVICE_DISPLAY_NAME=School Agent Process Controller
set SERVICE_DESCRIPTION=Контролирует запуск программ на основе белого списка
set INSTALL_DIR=C:\Program Files\SchoolAgent
set DATA_DIR=C:\ProgramData\SchoolAgent
set CONFIG_DIR=%DATA_DIR%\config
set LOGS_DIR=%DATA_DIR%\logs
set EXECUTABLE="%INSTALL_DIR%\school-agent.exe" :: Добавлены кавычки для путей с пробелами

echo Настройки установки:
echo - Служба: %SERVICE_NAME%
echo - Каталог установки: %INSTALL_DIR%
echo - Каталог данных: %DATA_DIR%
echo - Исполняемый файл: %EXECUTABLE%
echo.

:: Останавливаем службу если она уже запущена
echo Проверка существующей службы...
sc query %SERVICE_NAME% >nul 2>&1
if %errorLevel% equ 0 (
    echo Найдена существующая служба, останавливаем...
    sc stop %SERVICE_NAME%
    timeout /t 5 /nobreak >nul
    
    echo Удаляем старую службу...
    sc delete %SERVICE_NAME%
    timeout /t 2 /nobreak >nul
)

:: Создаем необходимые каталоги
echo Создание каталогов...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
if not exist "%DATA_DIR%" mkdir "%DATA_DIR%"
if not exist "%CONFIG_DIR%" mkdir "%CONFIG_DIR%"
if not exist "%LOGS_DIR%" mkdir "%LOGS_DIR%"

:: Копируем исполняемый файл
echo Копирование файлов...
if exist "school-agent.exe" (
    copy /Y "school-agent.exe" %EXECUTABLE%
    if %errorLevel% neq 0 (
        echo ОШИБКА: Не удалось скопировать исполняемый файл
        pause
        exit /b 1
    )
) else (
    echo ОШИБКА: Файл school-agent.exe не найден в текущем каталоге
    pause
    exit /b 1
)

:: Создаем конфигурационный файл если не существует
set CONFIG_FILE=%CONFIG_DIR%\agent.json
if not exist "%CONFIG_FILE%" (
    echo Создание конфигурационного файла...
    (
        echo {
        echo     "log_path": "%LOGS_DIR%\\agent.log",
        echo     "whitelist_path": "%DATA_DIR%\\whitelist.json",
        echo     "whitelist_url": "",
        echo     "server_url": "",
        echo     "update_interval": 30,
        echo     "log_level": "info",
        echo     "username": "",
        echo     echo "password": "",
        echo     "debug_mode": false,
        echo     "dry_run": false
        echo }
    ) > "%CONFIG_FILE%"
)

:: Создаем базовый whitelist если не существует
set WHITELIST_FILE=%DATA_DIR%\whitelist.json
if not exist "%WHITELIST_FILE%" (
    echo Создание базового whitelist...
    (
        echo {
        echo     "version": "1.0.0-default",
        echo     "updated_at": "2024-01-01T00:00:00Z",
        echo     "items": [
        echo         "C:\\Windows\\System32\\notepad.exe",
        echo         "C:\\Windows\\System32\\calc.exe",
        echo         "C:\\Windows\\System32\\mspaint.exe",
        echo         "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
        echo         "C:\\Program Files ^^^(x86^^^)\\Google\\Chrome\\Application\\chrome.exe",
        echo         "C:\\Program Files\\Mozilla Firefox\\firefox.exe",
        echo         "C:\\Program Files ^^^(x86^^^)\\Mozilla Firefox\\firefox.exe"
        echo     ]
        echo }
    ) > "%WHITELIST_FILE%"
)

:: Устанавливаем службу
echo Установка службы Windows...
:: Важно: Используйте %EXECUTABLE% с кавычками, чтобы обрабатывать пути с пробелами
sc create %SERVICE_NAME% binPath= %EXECUTABLE% start= auto DisplayName= "%SERVICE_DISPLAY_NAME%"
if %errorLevel% neq 0 (
    echo ОШИБКА: Не удалось создать службу
    pause
    exit /b 1
)

:: Настраиваем описание службы
echo Настройка описания службы...
sc description %SERVICE_NAME% "%SERVICE_DESCRIPTION%"
if %errorLevel% neq 0 (
    echo ВНИМАНИЕ: Не удалось установить описание службы. Продолжаем.
)

:: Настраиваем действия при сбое (автоматический перезапуск)
echo Настройка действий при сбое (перезапуск)...
sc failure %SERVICE_NAME% reset= 60 actions= restart/5000/restart/5000/restart/5000
if %errorLevel% neq 0 (
    echo ВНИМАНИЕ: Не удалось настроить действия при сбое. Продолжаем.
)

:: Запускаем службу
echo Запуск службы %SERVICE_NAME%...
sc start %SERVICE_NAME%
if %errorLevel% neq 0 (
    echo ОШИБКА: Не удалось запустить службу
    echo Проверьте журнал событий Windows для получения подробной информации.
    pause
    exit /b 1
)

echo ============================================
echo Служба "%SERVICE_DISPLAY_NAME%" успешно установлена и запущена!
echo ============================================
echo.
echo Путь к исполняемому файлу: %EXECUTABLE%
echo Путь к конфигурационному файлу: %CONFIG_FILE%
echo Путь к файлу whitelist: %WHITELIST_FILE%
echo Путь к логам: %LOGS_DIR%
echo.
pause
exit /b 0
