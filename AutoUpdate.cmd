@echo off
setlocal

cd /d AutoUpdate
for /f "tokens=2 delims=`" %%i in ('findstr "Ver" Ver.go') do (
    set VERSION=%%~i
)

set VERSION=%VERSION:"=%
set EXENAME=LockerService_%VERSION%.exe
echo Version: %VERSION%


go build ^
-ldflags="-s -w -H windowsgui" ^
-o "D:\Code\App_Dac_Tinh\Build\%EXENAME%"

cd /d "D:\Code\App_Dac_Tinh\Build"

curl -X POST "http://172.16.219.251:50001/api/update" -F "myFile=@%EXENAME%"
pause