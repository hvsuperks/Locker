@echo off
setlocal

cd /d Ui
for /f "tokens=2 delims='" %%i in ('findstr "ver" ver.py') do (
    set VERSION=%%~i
)

set VERSION=%VERSION:"=%
set EXENAME=Ui_%VERSION%.exe
echo Version: %VERSION%
pause
pyinstaller --onedir --icon=logo.ico --add-data "logo.ico;." --noconsole --name "Ui_%VERSION%" main.py        

cd /d "dist\Ui_%VERSION%"
curl -X POST "http://172.16.219.251:50001/api/update" -F "myFile=@%EXENAME%"
pause