# Vito

Asistente de terminal que responde por texto y puede buscar, abrir y crear
archivos en tu computador. Todo corre local: construido en **Go** e integrado
con un modelo de lenguaje local vía **Ollama**, sin nube ni cuentas.

## Stack tecnológico

| Tecnología | Versión | Descripción |
|---|---|---|
| [Go](https://go.dev) | 1.26 | Lenguaje principal — compila a un solo ejecutable |
| [Ollama](https://ollama.com) | — | Servidor local del modelo `llama3.1:8b` |
| [SQLite](https://modernc.org/sqlite) | — | Índice de archivos (driver puro Go, sin CGo) |
| [ReadDirectoryChangesW](https://learn.microsoft.com/windows/win32/api/winuser/) | — | API de Windows para vigilancia en tiempo real |
| [DuckDuckGo](https://duckduckgo.com) | — | Búsqueda web (sin dependencias externas) |

## Requisitos previos

- **Windows 10 u 11** (64 bits)
- **Ollama** corriendo con el modelo `llama3.1:8b`
- 8 GB de RAM recomendados para que el modelo responda con fluidez

Para instalar Ollama y el modelo automáticamente, usa el instalador incluido
(paso 2 de la instalación).

## Instalación

### Opción A — Solo usar Vito (recomendada)

1. **Descarga el paquete** `vito.zip` desde la sección
   [Releases](https://github.com/JuanCarpeta/Vito/releases) del repositorio
   y descomprímelo donde quieras (por ejemplo `C:\Vito`).
2. **Doble clic sobre `instalar.bat`** (no sobre el `.ps1`).
   El instalador deja todo listo: instala Ollama si falta, descarga el
   modelo (~4.9 GB, una sola vez) y lo deja corriendo en segundo plano.
3. **Doble clic en `vito.exe`**. Listo.

### Opción B — Compilar desde el código

1. **Clonar el repositorio**

```powershell
git clone https://github.com/JuanCarpeta/Vito.git
cd Vito
```

2. **Compilar el ejecutable**

```powershell
go build -o vito.exe ./src
```

3. **Preparar Ollama**

```powershell
winget install --id Ollama.Ollama -e
ollama pull llama3.1:8b
```

4. **Ejecutar**

```powershell
.\vito.exe
```

## Cómo usarlo

Escribe tus órdenes y presiona Enter. Ejemplos:

| Orden | Qué hace |
|---|---|
| `que hay en la carpeta actual` | Lista el contenido real del directorio |
| `busca el archivo de calculo` | Busca archivos por nombre en todo el disco |
| `ve a C:\Users\PC\Documents` | Se ubica en una carpeta (sin abrir ventanas) |
| `abre steam` | Abre un archivo, carpeta o aplicación instalada |
| `cierra steam` | Cierra una aplicación (cierre limpio, luego forzado) |
| `lee notas.txt` | Muestra el contenido de un archivo |
| `hazme un resumen de informe.docx` | Resume un .txt o .docx con el modelo |
| `crea un word llamado informe.docx con el texto...` | Crea Word, Excel o PDF reales |
| `crea un archivo llamado notas.txt con el texto hola` | Crea archivos de texto |
| `recuerda que estudio a las 7` | Guarda el dato para siempre |
| `que recuerdas de mi` | Consulta la memoria persistente |
| `busca en internet el clima de hoy` | Búsqueda web en DuckDuckGo |
| `apagar` | Cierra Vito (escrito exacto y solo) |

> **Nota:** `ve a` solo mueve a Vito internamente; `abre` sí abre ventanas
> y programas.

## Arquitectura

| Archivo | Equivalente en Python | Qué hace |
|---|---|---|
| `src/config.go` | config.py | Toda la configuración |
| `src/ollama.go` | core/cerebro.py (parte) | Cliente de `/api/chat` con herramientas |
| `src/cerebro.go` | core/cerebro.py | Prompt, 12 herramientas, loop de function calling, intenciones directas |
| `src/herramientas.go` | core/archivos.py | Buscar/listar/navegar/abrir/leer/crear/info |
| `src/scanner.go` | core/scanner.py | Escaneo incremental con firma por carpeta (SHA-256), búsqueda en índice |
| `src/database.go` | core/database.py | SQLite (archivos.db, WAL), borrado/movimiento de subárboles |
| `src/vigilante.go` | core/watchdog_monitor.py | ReadDirectoryChangesW: un handle recursivo por unidad |
| `src/cola.go` | core/event_queue.py | Debounce, coalescencia de eventos, reconciliación periódica |
| `src/procesador.go` | watchdog_monitor `_aplicar` | Aplica eventos al índice (el disco es la verdad) |
| `src/inventario.go` | core/inventario.py | Resumen del equipo para el prompt + vito.log |
| `src/web.go` | core/web.py | Búsqueda web en DuckDuckGo (stdlib) |
| `src/memoria.go` | core/memoria.py | Memoria persistente en memoria.json |
| `scripts/instalar.ps1` | — | Instalador para usuarios finales |
| `scripts/empaquetar.ps1` | — | Genera el paquete de distribución |
| `docs/COMO-INSTALAR.md` | — | Guía resumida dentro del paquete |

## Scripts disponibles

| Comando | Descripción |
|---|---|
| `.\vito.exe` | Inicia el asistente |
| `go build -o vito.exe ./src` | Compila el ejecutable |
| `go test ./...` | Ejecuta las pruebas del índice y la búsqueda |
| `powershell -File scripts/empaquetar.ps1` | Genera `vito.zip` listo para repartir |

## Compilación para distribución

```powershell
powershell -ExecutionPolicy Bypass -File scripts/empaquetar.ps1
```

Genera `vito.zip` con el ejecutable en modo release, el instalador y las
instrucciones: ese es el único archivo que tus compañeros necesitan.

## Solución de problemas

- **"No pude conectar con Ollama"** — Ollama no está corriendo. Ábrelo desde
  el menú de inicio (ícono de llama) o reinicia el PC.
- **Respuestas lentas la primera vez** — el modelo se está cargando en RAM;
  las siguientes salen más rápido.
- **`vito.exe~` aparece junto al ejecutable** — es un respaldo que deja
  Windows al recompilar con Vito abierto. Se puede borrar sin problema.

## Datos locales

`archivos.db`, `memoria.json` y `vito.log` se generan junto al ejecutable y
no se suben al repositorio.
