package main

// Escaneo e indexacion incremental de las unidades (espeja core/scanner.py).

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

var carpetasEscaneadas atomic.Int64

// Entrada es un archivo o carpeta listo para el indice.
type Entrada struct {
	Ruta      string
	Nombre    string
	Tamano    int64
	Mtime     time.Time
	EsCarpeta bool
	Padre     string
}

// Ignoradas son las carpetas que no se indexan (misma lista que usa el vigilante).
var Ignoradas = map[string]bool{
	"appdata": true, "node_modules": true, ".git": true, "__pycache__": true,
	"venv": true, ".venv": true, "windows": true, "program files": true,
	"program files (x86)": true, "programdata": true, "$recycle.bin": true,
	"xboxgames": true, "windows.old": true,
	"system volume information": true, "perflogs": true,
	"onedrivetemp": true, "$sysreset": true,
}

// ArchivosPropios nunca se indexan (bucle de retroalimentacion).
var ArchivosPropios = map[string]bool{
	"archivos.db": true, "archivos.db-wal": true, "archivos.db-shm": true,
	"archivos.db-journal": true, "memoria.json": true, "vito.exe": true,
	"vito.log": true, "inventario.json": true,
}

// IgnorarRuta dice si una ruta es ruido que no debe llegar al indice.
func IgnorarRuta(ruta string, esCarpeta bool) bool {
	nombre := strings.ToLower(filepath.Base(ruta))
	if !esCarpeta && ArchivosPropios[nombre] {
		return true
	}
	if !esCarpeta && (strings.HasSuffix(nombre, ".tmp") || strings.HasPrefix(nombre, "~$") ||
		strings.HasPrefix(nombre, ".~lock.")) {
		return true
	}
	if esCarpeta && (Ignoradas[nombre] || strings.HasPrefix(nombre, "$") ||
		strings.HasPrefix(nombre, ".")) {
		return true
	}
	return false
}

// firmaCarpeta resume el contenido de una carpeta (nombres + mtamanos + mtimes).
func firmaCarpeta(ruta string, entradas []os.DirEntry) string {
	nombres := make([]string, 0, len(entradas))
	for _, e := range entradas {
		info, err := e.Info()
		if err != nil {
			continue
		}
		nombres = append(nombres, fmt.Sprintf("%s|%d|%d",
			strings.ToLower(e.Name()), info.Size(), info.ModTime().Unix()))
	}
	sort.Strings(nombres)
	resumen := sha256.Sum256([]byte(strings.Join(nombres, "\n")))
	return hex.EncodeToString(resumen[:16])
}

// escanearCarpeta indexa una carpeta y devuelve su firma.
func escanearCarpeta(ruta string) (string, error) {
	entradas, err := os.ReadDir(ruta)
	if err != nil {
		return "", err
	}
	carpetasEscaneadas.Add(1)
	firma := firmaCarpeta(ruta, entradas)
	// Si nada cambio, no reescribimos las filas.
	if previa := LeerFirma(ruta); previa == firma {
		return firma, nil
	}
	for _, e := range entradas {
		completa := filepath.Join(ruta, e.Name())
		if IgnorarRuta(completa, e.IsDir()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		GuardarArchivo(Entrada{
			Ruta: completa, Nombre: e.Name(), Tamano: info.Size(),
			Mtime: info.ModTime(), EsCarpeta: e.IsDir(), Padre: ruta,
		})
	}
	GuardarFirma(ruta, firma, time.Now())
	return firma, nil
}

// escanearSubarbol recorre en profundidad (para reconstruir tras un movimiento).
func escanearSubarbol(raiz string, limite time.Time) {
	if time.Now().After(limite) {
		return
	}
	if IgnorarRuta(raiz, true) {
		return
	}
	escanearCarpeta(raiz)
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		return
	}
	for _, e := range entradas {
		if e.IsDir() {
			escanearSubarbol(filepath.Join(raiz, e.Name()), limite)
		}
	}
}

// IndexarUnidades escanea las unidades con indexacion incremental: si la firma
// de una carpeta no cambio, no entra. forzar=true re-escanea todo.
func IndexarUnidades(forzar bool) int {
	carpetasEscaneadas.Store(0)
	inicio := time.Now()
	total := ConteoFilas()
	for _, unidad := range discosDisponibles() {
		recorrerUnidad(unidad, forzar, inicio.Add(10*time.Minute))
	}
	MarcarEscaneo(time.Now())
	return ConteoFilas() - total
}

func recorrerUnidad(raiz string, forzar bool, limite time.Time) {
	if IgnorarRuta(raiz, true) {
		return
	}
	escanearCarpeta(raiz)
	entradas, err := os.ReadDir(raiz)
	if err != nil {
		return
	}
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		completa := filepath.Join(raiz, e.Name())
		if IgnorarRuta(completa, true) {
			continue
		}
		if !forzar {
			// Escaneo incremental: primero solo la firma de primer nivel.
			entradasInternas, err := os.ReadDir(completa)
			if err != nil {
				continue
			}
			firma := firmaCarpeta(completa, entradasInternas)
			if previa := LeerFirma(completa); previa == firma {
				carpetasEscaneadas.Add(1)
				GuardarFirma(completa, firma, time.Now())
				continue
			}
		}
		escanearSubarbol(completa, limite)
	}
}

// BuscarIndice busca archivos por nombre en el indice SQLite.
func BuscarIndice(consulta string, maximo int) []string {
	palabras := strings.Fields(strings.ToLower(consulta))
	if len(palabras) == 0 || db == nil {
		return nil
	}
	condiciones := make([]string, len(palabras))
	valores := make([]any, 0, len(palabras))
	for i, p := range palabras {
		condiciones[i] = ` AND normalizado LIKE ? ESCAPE '\'`
		valores = append(valores, "%"+escaparLike(p)+"%")
	}
	consultaSQL := `SELECT ruta, tamano, es_carpeta FROM archivos
		WHERE 1=1` + strings.Join(condiciones, "") + `
		ORDER BY es_carpeta, length(ruta) LIMIT ?`
	valores = append(valores, maximo*3)
	filas, err := db.Query(consultaSQL, valores...)
	if err != nil {
		return nil
	}
	defer filas.Close()
	resultados := []string{}
	for filas.Next() {
		var ruta string
		var tamano int64
		var esCarpeta int
		if filas.Scan(&ruta, &tamano, &esCarpeta) == nil {
			if _, err := os.Stat(ruta); err == nil { // el disco es la verdad
				resultados = append(resultados, ruta)
			}
			if len(resultados) >= maximo {
				break
			}
		}
	}
	return resultados
}

func escaparLike(s string) string {
	reemplazo := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return reemplazo.Replace(s)
}

// formatearResultados conviene la lista de rutas en texto legible.
func formatearResultados(consulta string) string {
	if strings.TrimSpace(consulta) == "" {
		return "Falta la consulta de busqueda."
	}
	rutas := BuscarIndice(consulta, MaxResultadosBusqueda)
	if len(rutas) == 0 {
		return "No encontre ningun archivo con ese nombre."
	}
	lineas := []string{fmt.Sprintf("Encontre %d archivo(s):", len(rutas))}
	for i, r := range rutas {
		lineas = append(lineas, fmt.Sprintf("%d. %s", i+1, r))
	}
	return strings.Join(lineas, "\n")
}
