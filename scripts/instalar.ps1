# Instalador de Vito para Windows 10/11.
# Uso: clic derecho sobre este archivo > "Ejecutar con PowerShell"
# (o desde una terminal: powershell -ExecutionPolicy Bypass -File instalar.ps1)

Write-Host "=== Instalador de Vito ===" -ForegroundColor Cyan

# 1. Verificar que vito.exe este junto a este script.
$raiz = Split-Path -Parent $MyInvocation.MyCommand.Path
if (-not (Test-Path (Join-Path $raiz "vito.exe"))) {
    Write-Host "ERROR: no encuentro vito.exe junto a este instalador." -ForegroundColor Red
    Read-Host "Presiona Enter para salir"
    exit 1
}

# 2. Instalar Ollama si no existe (via winget, incluido en Windows 10/11).
$ollama = Get-Command ollama -ErrorAction SilentlyContinue
if (-not $ollama) {
    Write-Host "Ollama no esta instalado. Instalandolo con winget..." -ForegroundColor Yellow
    winget install --id Ollama.Ollama -e --accept-source-agreements --accept-package-agreements
    if ($LASTEXITCODE -ne 0) {
        Write-Host "No pude instalar Ollama automaticamente. Descargalo de https://ollama.com/download y vuelve a ejecutar este instalador." -ForegroundColor Red
        Read-Host "Presiona Enter para salir"
        exit 1
    }
    # Refrescar el PATH de esta sesion.
    $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
} else {
    Write-Host "Ollama ya esta instalado." -ForegroundColor Green
}

# 3. Descargar el modelo que Vito usa (solo la primera vez, ~4.9 GB).
$modelo = & ollama list 2>$null | Select-String "llama3.1:8b"
if (-not $modelo) {
    Write-Host "Descargando el modelo llama3.1:8b (~4.9 GB, tomara un rato)..." -ForegroundColor Yellow
    ollama pull llama3.1:8b
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Fallo la descarga del modelo. Revisa tu internet y ejecuta 'ollama pull llama3.1:8b' a mano." -ForegroundColor Red
        Read-Host "Presiona Enter para salir"
        exit 1
    }
} else {
    Write-Host "El modelo llama3.1:8b ya esta descargado." -ForegroundColor Green
}

# 4. Asegurar que Ollama este corriendo (arranca solo al instalar; si no, lo lanzamos).
$servicio = Get-Process ollama* -ErrorAction SilentlyContinue
if (-not $servicio) {
    Write-Host "Iniciando Ollama en segundo plano..." -ForegroundColor Yellow
    Start-Process "ollama" -ArgumentList "serve" -WindowStyle Hidden
    Start-Sleep -Seconds 3
}

Write-Host ""
Write-Host "Todo listo. Ejecuta vito.exe para empezar." -ForegroundColor Green
Write-Host "Dentro de Vito escribe 'apagar' para salir." -ForegroundColor Green
Read-Host "Presiona Enter para salir"
