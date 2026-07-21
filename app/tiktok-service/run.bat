@echo off
setlocal
cd /d "%~dp0\..\.."
go run ./app/tiktok-service/cmd/main.go
