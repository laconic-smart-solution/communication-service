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

if not defined WHATSAPP_SERVICE_PORT set WHATSAPP_SERVICE_PORT=8089
if not defined GATEWAY_SECRET set GATEWAY_SECRET=hello
if not defined LOG_LEVEL set LOG_LEVEL=info

cd /d "%~dp0..\.."

echo [INFO] Starting whatsapp-service on port %WHATSAPP_SERVICE_PORT%...
echo [INFO] Health: http://localhost:%WHATSAPP_SERVICE_PORT%/api/v1/healthz
echo [INFO] Stateless proxy — pass X-WhatsApp-Access-Token + X-WhatsApp-Phone-Number-Id from calling apps.
echo.

go run ./app/whatsapp-service/cmd %*
exit /b %errorlevel%
