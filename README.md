# Vito

Asistente de terminal en Go: un solo ejecutable que conversa por texto y actúa
sobre el computador usando Ollama (`llama3.1:8b`) con function calling.
Arquitectura equivalente al proyecto Python original, sin voz.

## Arquitectura

| Archivo | Equivalente en Python | Qué hace |
|---|---|---|
| `config.go` | config.py | Toda la configuración |
| `ollama.go` | core/cerebro.py (parte) | Cliente de `/api/chat` con herramientas |
| `cerebro.go` | core/cerebro.py | Prompt, 12 herramientas, loop de function calling, intenciones directas |
| `herramientas.go` | core/archivos.py | Buscar/listar/navegar/abrir/leer/crear/info |
| `scanner.go` | core/scanner.py | Escaneo incremental con firma por carpeta (SHA-256), búsqueda en índice |
| `database.go` | core/database.py | SQLite (archivos.db, WAL), borrado/movimiento de subárboles |
| `vigilante.go` | core/watchdog_monitor.py | ReadDirectoryChangesW: un handle recursivo por unidad |
| `cola.go` | core/event_queue.py | Debounce, coalescencia de eventos, reconciliación periódica |
| `procesador.go` | watchdog_monitor `_aplicar` | Aplica eventos al índice (el disco es la verdad) |
| `inventario.go` | core/inventario.py | Resumen del equipo para el prompt + vito.log |
| `web.go` | core/web.py | Búsqueda web en DuckDuckGo (stdlib) |
| `memoria.go` | core/memoria.py | Memoria persistente en memoria.json |

## Compilar y ejecutar

```powershell
cd C:\Proyectos\vito
go build -o vito.exe .
.\vito.exe
```

Requisito: Ollama corriendo con `llama3.1:8b`. No necesita Python ni venv.

## Pruebas

```powershell
go test ./...
```

## Comandos de apagado

`apagar`, `salir`, `apagar vito`, `cerrar vito` (exactos, en minúsculas).

## Datos locales

`archivos.db`, `memoria.json` y `vito.log` se generan junto al ejecutable y no
se suben al repositorio.
