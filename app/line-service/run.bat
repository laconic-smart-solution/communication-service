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

if not defined LINE_SERVICE_PORT set LINE_SERVICE_PORT=8083
if not defined GATEWAY_SECRET set GATEWAY_SECRET=hello
if not defined LOG_LEVEL set LOG_LEVEL=info

cd /d "%~dp0..\.."

echo [INFO] Starting line-service on port %LINE_SERVICE_PORT%...
echo [INFO] Health: http://localhost:%LINE_SERVICE_PORT%/api/v1/healthz
echo [INFO] Stateless proxy — pass X-Line-Channel-Access-Token from calling apps.
echo.

go run ./app/line-service/cmd %*
exit /b %errorlevel%
