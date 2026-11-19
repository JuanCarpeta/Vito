# Vito

Asistente de terminal en Go: un solo ejecutable que conversa por texto y actúa
sobre el computador usando Ollama (`llama3.1:8b`) con function calling.

## Qué incluye (columna vertebral)

- Cliente de la API `/api/chat` de Ollama con herramientas (`ollama.go`)
- 12 herramientas: buscar/listar/navegar/abrir/leer/crear archivos y carpetas,
  info, búsqueda web (DuckDuckGo, stdlib) y memoria persistente
- Memoria en `memoria.json` (sobrevive entre sesiones)
- Bucle de function calling con máximo 4 pasos y recorte de historial

## Compilar y ejecutar

```powershell
cd C:\Proyectos\vito
go build -o vito.exe .
.\vito.exe
```

Requisito: Ollama corriendo con `llama3.1:8b`. No necesita Python ni venv.

## Comandos de apagado

`apagar`, `salir`, `apagar vito`, `cerrar vito` (exactos, en minúsculas).

## Estado

Versión inicial: sin indexación SQLite (la búsqueda recorre Documents,
Desktop, Downloads y la carpeta actual), sin voz y sin GUI. El proyecto
Python completo vive en `C:\Proyectos\JarvisIA`.
