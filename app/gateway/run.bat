@echo off
setlocal
cd /d "%~dp0"

where mvn >nul 2>&1
if errorlevel 1 (
  echo [ERROR] Maven not found. Install Maven or add mvnw.cmd to this folder.
  exit /b 1
)

set GATEWAY_PORT=8080
set GATEWAY_SECRET=hello
set GATEWAY_JWT_HMAC_SECRET=comms-local-dev-jwt-hmac-secret-ok!
set GATEWAY_CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
set FACEBOOK_SERVICE_URI=http://localhost:8081
set LINE_SERVICE_URI=http://localhost:8083
set LAZADA_SERVICE_URI=http://localhost:8084
set SHOPEE_SERVICE_URI=http://localhost:8085
set TIKTOK_SERVICE_URI=http://localhost:8086

echo [INFO] Starting communication-gateway on port %GATEWAY_PORT%...
echo [INFO] Health: http://localhost:%GATEWAY_PORT%/actuator/health
echo [INFO] Lazada proxy: http://localhost:%GATEWAY_PORT%/comms/lazada/api/v1/healthz
echo.

mvn spring-boot:run %*
exit /b %errorlevel%
