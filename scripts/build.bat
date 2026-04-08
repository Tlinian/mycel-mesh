@echo off
REM Mycel Mesh Windows 编译打包脚本
REM 使用方法：scripts\build.bat

setlocal enabledelayedexpansion

set VERSION=%VERSION:~0,5%
if "%VERSION%"=="" set VERSION=1.0.0
set OUTPUT_DIR=dist

echo ========================================
echo Mycel Mesh v%VERSION% 编译打包 (Windows)
echo ========================================

REM 创建输出目录
if not exist "%OUTPUT_DIR%" mkdir "%OUTPUT_DIR%"

set ARTIFACT_NAME=mycel-win-amd64
set OUTPUT_PATH=%OUTPUT_DIR%\%ARTIFACT_NAME%

if not exist "%OUTPUT_PATH%" mkdir "%OUTPUT_PATH%"

echo.
echo 编译中...

REM 编译 Coordinator
echo   - 编译 coordinator.exe
go build -ldflags "-X main.Version=%VERSION%" -o "%OUTPUT_PATH%\coordinator.exe" ./cmd/coordinator

REM 编译 Agent
echo   - 编译 agent.exe
go build -ldflags "-X main.Version=%VERSION%" -o "%OUTPUT_PATH%\agent.exe" ./cmd/agent

REM 编译 CLI
echo   - 编译 mycelctl.exe
go build -ldflags "-X main.Version=%VERSION%" -o "%OUTPUT_PATH%\mycelctl.exe" ./cmd/mycelctl

REM 复制文档
copy /Y docs\quickstart.md "%OUTPUT_PATH%\QUICKSTART.md" >nul 2>&1 || echo 注意：快速开始文档未找到
copy /Y README.md "%OUTPUT_PATH%\README.md" >nul 2>&1 || echo 注意：README 未找到

echo.
echo ========================================
echo 编译完成！
echo ========================================
echo.
echo 输出目录：%OUTPUT_DIR%\%ARTIFACT_NAME%\
dir /b "%OUTPUT_DIR%\%ARTIFACT_NAME%"
echo.
echo 版本：v%VERSION%
