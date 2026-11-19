@echo off
rem Lanzador del instalador de Vito.
rem El .bat evita el bloqueo de la politica de ejecucion de PowerShell
rem y mantiene la ventana abierta si algo falla.

powershell -NoProfile -ExecutionPolicy Bypass -File "%~dp0instalar.ps1"
if errorlevel 1 (
    echo.
    echo Hubo un problema durante la instalacion. Revisa el mensaje de arriba.
)
pause
