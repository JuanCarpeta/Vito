# Empaqueta Vito para distribucion: compila el exe en modo release y
# crea vito.zip con solo lo necesario para instalar en otra maquina.
# Uso: powershell -ExecutionPolicy Bypass -File empaquetar.ps1

$ErrorActionPreference = "Stop"
# $raiz = la carpeta raiz del proyecto (un nivel arriba de scripts/).
$raiz = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $raiz

Write-Host "Compilando vito.exe (modo release)..." -ForegroundColor Cyan
go build -ldflags "-s -w" -o vito.exe ./src
if ($LASTEXITCODE -ne 0) { throw "La compilacion fallo." }

Write-Host "Creando vito.zip..." -ForegroundColor Cyan
$staging = Join-Path $env:TEMP ("vito-paquete-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $staging | Out-Null
Copy-Item (Join-Path $raiz "vito.exe") $staging
Copy-Item (Join-Path $raiz "scripts\instalar.ps1") $staging
Copy-Item (Join-Path $raiz "docs\COMO-INSTALAR.md") $staging

$zip = Join-Path $raiz "vito.zip"
if (Test-Path $zip) { Remove-Item $zip }
Compress-Archive -Path (Join-Path $staging "*") -DestinationPath $zip
Remove-Item $staging -Recurse -Force

Write-Host "Listo: $zip" -ForegroundColor Green
Write-Host "Ese zip es lo unico que tus compañeros necesitan descargar."
