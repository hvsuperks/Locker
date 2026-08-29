@echo off
setlocal

cd /d Master
for /f "tokens=2 delims=`" %%i in ('findstr "Ver" .\config\var.go') do (
    set VERSION=%%~i
)

set VERSION=%VERSION:"=%
set EXENAME=Master_%VERSION%.exe
echo Version: %VERSION%

go build ^
-ldflags="-s -w -H windowsgui" ^
-o "D:\Code\App_Dac_Tinh\Build\%EXENAME%"
echo %EXENAME%
cd /d "D:\Code\App_Dac_Tinh\Build"
curl -X POST "http://172.16.219.251:50001/api/update" -F "myFile=@%EXENAME%"
pause

