@echo off
setlocal enabledelayedexpansion

mkdir bin 2>nul

set platforms=windows/amd64 linux/amd64 darwin/amd64 darwin/arm64

for %%p in (%platforms%) do (
    for /f "tokens=1,2 delims=/" %%a in ("%%p") do (
        set GOOS=%%a
        set GOARCH=%%b
        set output=bin\webscan-%%a-%%b
        if "%%a"=="windows" set output=!output!.exe
        
        echo Building !output!...
        set CGO_ENABLED=0
        set GOOS=%%a
        set GOARCH=%%b
        go build -o !output! .
    )
)

