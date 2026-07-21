@echo off
setlocal
cd /d "%~dp0"

if not exist "mvnw.cmd" (
  echo [ERROR] mvnw.cmd not found. Run this script from the gateway folder.
  exit /b 1
)

if exist ".env" (
  for /f "usebackq eol=# tokens=1,* delims==" %%A in (".env") do (
    if not "%%A"=="" set "%%A=%%B"
  )
)

if not defined GATEWAY_PORT set GATEWAY_PORT=8099
if not defined GATEWAY_SECRET set GATEWAY_SECRET=hello
if not defined GATEWAY_JWT_HMAC_SECRET set GATEWAY_JWT_HMAC_SECRET=comms-local-dev-jwt-hmac-secret-ok!
if not defined GATEWAY_CORS_ALLOWED_ORIGINS set GATEWAY_CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
if not defined FACEBOOK_SERVICE_URI set FACEBOOK_SERVICE_URI=http://localhost:8081
if not defined LINE_SERVICE_URI set LINE_SERVICE_URI=http://localhost:8083
if not defined LAZADA_SERVICE_URI set LAZADA_SERVICE_URI=http://localhost:8084
if not defined SHOPEE_SERVICE_URI set SHOPEE_SERVICE_URI=http://localhost:8085
if not defined TIKTOK_SERVICE_URI set TIKTOK_SERVICE_URI=http://localhost:8086

echo [INFO] Starting communication-gateway on port %GATEWAY_PORT%...
echo [INFO] Health: http://localhost:%GATEWAY_PORT%/actuator/health
echo [INFO] Lazada proxy: http://localhost:%GATEWAY_PORT%/comms/lazada/api/v1/healthz
echo.

call mvnw.cmd spring-boot:run %*
exit /b %errorlevel%
