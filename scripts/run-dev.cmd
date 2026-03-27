@echo off
REM Used by Air (full_bin). Ensures cwd is project root and APP_ADDR has a default.
cd /d "%~dp0.."
if not defined APP_ADDR set APP_ADDR=:8051
bin\remi-server.exe
