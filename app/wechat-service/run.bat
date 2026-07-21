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

if not defined WECHAT_SERVICE_PORT set WECHAT_SERVICE_PORT=8090
if not defined GATEWAY_SECRET set GATEWAY_SECRET=hello
if not defined LOG_LEVEL set LOG_LEVEL=info

cd /d "%~dp0..\.."

echo [INFO] Starting wechat-service on port %WECHAT_SERVICE_PORT%...
echo [INFO] Health: http://localhost:%WECHAT_SERVICE_PORT%/api/v1/healthz
echo [INFO] Stateless proxy — pass X-WeChat-Access-Token from calling apps.
echo.

go run ./app/wechat-service/cmd %*
exit /b %errorlevel%
