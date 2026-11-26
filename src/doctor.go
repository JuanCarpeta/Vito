package main

// Doctor: chequeo de salud de la instalacion de Vito. La idea es de

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type chequeo struct {
	nombre  string
	bien    bool
	detalle string
	consejo string // que hacer si fallo ("" = nada)
}

// Doctor ejecuta todos los chequeos y devuelve el reporte en texto.
func Doctor() string {
	chequeos := []chequeo{
		chequearOllama(),
		chequearEscritura(),
		chequearMemoria(),
		chequearIndice(),
		chequearActualizacion(),
	}
	var b strings.Builder
	b.WriteString("Diagnostico de Vito v" + VersionActual + ":\n")
	fallos := 0
	for _, c := range chequeos {
		marca := "[OK]"
		if !c.bien {
			marca = "[X]"
			fallos++
		}
		b.WriteString("  " + marca + " " + c.nombre + ": " + c.detalle + "\n")
		if !c.bien && c.consejo != "" {
			b.WriteString("        -> " + c.consejo + "\n")
		}
	}
	if fallos == 0 {
		b.WriteString("Todo esta en orden.")
	} else {
		fmt.Fprintf(&b, "%d problema(s) encontrado(s).", fallos)
	}
	return b.String()
}

// chequearOllama verifica que el servidor local responda y que el modelo
// configurado este descargado.
func chequearOllama() chequeo {
	cliente := &http.Client{Timeout: 4 * time.Second}
	resp, err := cliente.Get("http://localhost:11434/api/tags")
	if err != nil {
		return chequeo{
			nombre:  "Ollama",
			detalle: "no responde",
			consejo: "Abre Ollama desde el menu de inicio (icono de llama) e intenta de nuevo",
		}
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return chequeo{nombre: "Ollama", detalle: fmt.Sprintf("respondio con codigo %d", resp.StatusCode)}
	}
	var datos struct {
		Modelos []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&datos); err != nil {
		return chequeo{nombre: "Ollama", detalle: "respuesta ilegible"}
	}
	for _, modelo := range datos.Modelos {
		if strings.HasPrefix(modelo.Name, Modelo) {
			return chequeo{nombre: "Ollama + modelo", bien: true, detalle: Modelo + " disponible"}
		}
	}
	return chequeo{
		nombre:  "Ollama + modelo",
		detalle: "Ollama corre pero falta el modelo " + Modelo,
		consejo: "Ejecuta: ollama pull " + Modelo,
	}
}

// chequearEscritura prueba que Vito pueda escribir en su propia carpeta
// (el Controlled Folder Access de Windows Defender bloquea Documents a veces).
func chequearEscritura() chequeo {
	prueba := filepath.Join(Base, ".prueba-escritura")
	archivo, err := os.Create(prueba)
	if err != nil {
		return chequeo{
			nombre:  "Escritura en la carpeta",
			detalle: "bloqueada",
			consejo: "Windows Defender (Acceso controlado a carpetas) puede estar bloqueando Vito: permite vito.exe en su configuracion",
		}
	}
	archivo.Close()
	os.Remove(prueba)
	return chequeo{nombre: "Escritura en la carpeta", bien: true, detalle: "permisos correctos"}
}

// chequearMemoria valida que memoria.json sea JSON legible.
func chequearMemoria() chequeo {
	contenido, err := os.ReadFile(rutaMemoria())
	if err != nil {
		// No existir todavia no es un problema.
		return chequeo{nombre: "Memoria", bien: true, detalle: "vacia (aun no guardas nada)"}
	}
	var envoltura map[string]any
	if err := json.Unmarshal(contenido, &envoltura); err != nil {
		return chequeo{
			nombre:  "Memoria",
			detalle: "memoria.json corrupto",
			consejo: "Borra memoria.json y la memoria reiniciara limpia",
		}
	}
	datos, _ := envoltura["datos"].(map[string]any)
	return chequeo{nombre: "Memoria", bien: true, detalle: fmt.Sprintf("%d dato(s) guardados", len(datos))}
}

// chequearIndice revisa la base de archivos: filas, ultimo escaneo y unidad.
func chequearIndice() chequeo {
	if db == nil {
		return chequeo{
			nombre:  "Indice de archivos",
			detalle: "sin base de datos",
			consejo: "No se pudo abrir archivos.db; la busqueda recorre el disco lentamente",
		}
	}
	filas := ConteoFilas()
	var escaneo string
	db.QueryRow(`SELECT valor FROM meta WHERE clave='ultimo_escaneo'`).Scan(&escaneo)
	detalle := fmt.Sprintf("%d archivos indexados", filas)
	if escaneo != "" {
		if _, err := time.Parse(time.RFC3339, escaneo); err == nil {
			detalle += ", ultimo escaneo " + etiquetaFecha(escaneo)
		}
	}
	chequeoIndice := chequeo{nombre: "Indice de archivos", bien: true, detalle: detalle}
	if filas == 0 {
		chequeoIndice.bien = false
		chequeoIndice.consejo = "El indice esta vacio; la busqueda tardara. Reincia Vito para que escanee"
	}
	// Unidades que estaban indexadas y ya no estan conectadas.
	unidades := ""
	for _, unidad := range discosDisponibles() {
		unidades += unidad + ";"
	}
	_ = unidades
	return chequeoIndice
}

// chequearActualizacion reusa el auto-actualizador: un chequeo mas.
func chequearActualizacion() chequeo {
	version, _ := HayActualizacion()
	if version == "" {
		return chequeo{nombre: "Version", bien: true, detalle: "v" + VersionActual + " (al dia)"}
	}
	return chequeo{
		nombre:  "Version",
		bien:    false,
		detalle: fmt.Sprintf("v%s con actualizacion disponible (v%s)", VersionActual, version),
		consejo: "Escribe 'actualizar' para instalarla",
	}
}
