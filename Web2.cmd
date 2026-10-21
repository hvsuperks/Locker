@echo off

setlocal EnableDelayedExpansion
cd /d Web\build



set ROOT=%cd%

for /r %%f in (*) do (
    set "p=%%f"
    call set "p=%%p:%ROOT%\=%%"
    set "dir=web\!p!"
    for %%a in ("!dir!") do set "dir=!dir:\%%~nxa=!"

    echo curl -X POST http://172.16.219.251:50001/api/uploadfile ^
        -F "myFile=@%%f" ^
        -F "folder=!dir!" ^
        -F "mode=free"
    curl -X POST http://172.16.219.251:50001/api/uploadfile ^
        -F "myFile=@%%f" ^
        -F "folder=!dir!" ^
        -F "mode=free"
)

endlocal