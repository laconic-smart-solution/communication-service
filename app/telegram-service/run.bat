@echo off
setlocal
cd /d "%~dp0"

where go >nul 2>&1
if errorlevel 1 (
  echo [ERROR] Go not found. Install Go 1.22+.
  exit /b 1
)

if exist ".env" (
  for /f "usebackq eol=# tokens=1,* delims==" %%A in (".env") do (
    if not "%%A"=="" set "%%A=%%B"
  )
) else if exist "..\..\.env" (
  for /f "usebackq eol=# tokens=1,* delims==" %%A in ("..\..\.env") do (
    if not "%%A"=="" set "%%A=%%B"
  )
) else if exist ".env.example" (
  echo [WARN] No .env found. Copy .env.example to .env and set GATEWAY_SECRET.
)

if not defined TELEGRAM_SERVICE_PORT set TELEGRAM_SERVICE_PORT=8087
if not defined GATEWAY_SECRET set GATEWAY_SECRET=hello
if not defined LOG_LEVEL set LOG_LEVEL=info

cd /d "%~dp0..\.."

echo [INFO] Starting telegram-service on port %TELEGRAM_SERVICE_PORT%...
echo [INFO] Health: http://localhost:%TELEGRAM_SERVICE_PORT%/api/v1/healthz
echo [INFO] Stateless proxy — pass X-Telegram-Bot-Token from calling apps.
echo.

go run ./app/telegram-service/cmd %*
exit /b %errorlevel%
