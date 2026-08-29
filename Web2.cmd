@echo off
setlocal

cd /d Web
robocopy .\build D:\Code\App_Dac_Tinh\Build\Web /E /IS /IT
