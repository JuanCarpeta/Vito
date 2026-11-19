package main

// Herramientas de archivos: buscar, listar, navegar, abrir, leer, crear e info.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

var directorioActual = directorioUsuario()

func directorioUsuario() string {
	cwd, err := os.Getwd()
	if err == nil {
		return cwd
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}

// discosDisponibles devuelve las unidades que existen (C:\, D:\, ...).
func discosDisponibles() []string {
	discos := []string{}
	for c := 'A'; c <= 'Z'; c++ {
		ruta := string(c) + `:\`
		if _, err := os.Stat(ruta); err == nil {
			discos = append(discos, ruta)
		}
	}
	return discos
}

var carpetasRuido = map[string]bool{
	"appdata": true, "node_modules": true, ".git": true, "__pycache__": true,
	"venv": true, ".venv": true, "windows": true, "program files": true,
	"program files (x86)": true, "programdata": true, "$recycle.bin": true,
	"xboxgames": true, "windows.old": true,
	"system volume information": true, "perflogs": true,
	"onedrivetemp": true, "$sysreset": true,
}

// buscarEnDisco recorre TODAS las unidades buscando archivos cuyo nombre
// contenga todas las palabras de la consulta (limite de resultados y tiempo).
func buscarEnDisco(consulta string, maximo int) []string {
	palabras := strings.Fields(strings.ToLower(consulta))
	if len(palabras) == 0 {
		return nil
	}
	encontrados := []string{}
	vistas := map[string]bool{}
	limite := time.Now().Add(20 * time.Second)
	for _, raiz := range discosDisponibles() {
		vistas[raiz] = true
		filepath.WalkDir(raiz, func(ruta string, d os.DirEntry, err error) error {
			if err != nil {
				return filepath.SkipDir
			}
			if time.Now().After(limite) || len(encontrados) >= maximo {
				return filepath.SkipAll
			}
			nombre := strings.ToLower(d.Name())
			if d.IsDir() {
				if carpetasRuido[nombre] || strings.HasPrefix(nombre, "$") || strings.HasPrefix(nombre, ".") {
					return filepath.SkipDir
				}
				return nil
			}
			coincide := true
			for _, p := range palabras {
				if !strings.Contains(nombre, p) {
					coincide = false
					break
				}
			}
			if coincide {
				encontrados = append(encontrados, ruta)
			}
			return nil
		})
		if len(encontrados) >= maximo {
			break
		}
	}
	return encontrados
}

func formatearBusqueda(consulta string) string {
	if strings.TrimSpace(consulta) == "" {
		return "Falta la consulta de busqueda."
	}
	rutas := buscarEnDisco(consulta, 5)
	if len(rutas) == 0 {
		return "No encontre ningun archivo con ese nombre."
	}
	lineas := []string{fmt.Sprintf("Encontre %d archivo(s):", len(rutas))}
	for i, r := range rutas {
		lineas = append(lineas, fmt.Sprintf("%d. %s", i+1, r))
	}
	return strings.Join(lineas, "\n")
}

func listar(ruta string) string {
	if ruta == "." || ruta == "" {
		ruta = directorioActual
	}
	entradas, err := os.ReadDir(ruta)
	if err != nil {
		return fmt.Sprintf("No puedo leer %s: %v", ruta, err)
	}
	nombres := []string{}
	for _, e := range entradas {
		nombre := e.Name()
		if e.IsDir() {
			nombre += "/"
		}
		nombres = append(nombres, nombre)
	}
	sort.Strings(nombres)
	if len(nombres) == 0 {
		return fmt.Sprintf("%s esta vacia.", ruta)
	}
	return fmt.Sprintf("Contenido de %s:\n%s", ruta, strings.Join(nombres, "\n"))
}

func navegar(ruta string) string {
	info, err := os.Stat(ruta)
	if err != nil {
		return fmt.Sprintf("No existe: %s", ruta)
	}
	if !info.IsDir() {
		ruta = filepath.Dir(ruta)
	}
	abs, err := filepath.Abs(ruta)
	if err != nil {
		return fmt.Sprintf("Ruta invalida: %s", ruta)
	}
	directorioActual = abs
	return "Ahora estoy en " + abs
}

func abrir(ruta string) string {
	if _, err := os.Stat(ruta); err == nil {
		return abrirReal(ruta)
	}
	// Si el destino no existe y parece un nombre de aplicacion, se resuelve.
	if !strings.ContainsAny(ruta, `\/`) || filepath.Ext(ruta) == ".exe" {
		if resuelta := resolverApp(filepath.Base(ruta)); resuelta != "" {
			return abrirReal(resuelta)
		}
	}
	return fmt.Sprintf("No existe: %s", ruta)
}

func abrirReal(ruta string) string {
	cmd := exec.Command("cmd", "/c", "start", "", ruta)
	cmd.SysProcAttr = sysProcOculto()
	if err := cmd.Start(); err != nil {
		return fmt.Sprintf("No pude abrir %s: %v", ruta, err)
	}
	return fmt.Sprintf("Abri %s", ruta)
}

// normalizarNombre quita espacios, guiones y acentos para comparar nombres.
func normalizarNombre(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer(" ", "", "-", "", "_", "", "á", "a", "é", "e",
		"í", "i", "ó", "o", "ú", "u").Replace(s)
	return s
}

// resolverApp busca una aplicacion instalada por nombre y devuelve su ruta
// o acceso directo. Orden: registro App Paths, Menu Inicio, indice de archivos.
func resolverApp(nombre string) string {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return ""
	}
	limpio := normalizarNombre(nombre)
	limpio = strings.TrimSuffix(limpio, ".exe")

	// 1) Claves App Paths del registro (el metodo canonico de Windows).
	raices := []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER}
	for _, raiz := range raices {
		clave, err := registry.OpenKey(raiz,
			fmt.Sprintf(`SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\%s.exe`, limpio),
			registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		ruta, _, err := clave.GetStringValue("")
		clave.Close()
		if err == nil && ruta != "" {
			if _, err := os.Stat(ruta); err == nil {
				return ruta
			}
		}
	}

	// 2) Accesos directos del Menu Inicio (programas y juegos instalan ahi).
	candidatos := []string{}
	carpetas := []string{
		`C:\ProgramData\Microsoft\Windows\Start Menu\Programs`,
		filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs"),
	}
	for _, carpeta := range carpetas {
		filepath.WalkDir(carpeta, func(ruta string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.EqualFold(filepath.Ext(d.Name()), ".lnk") {
				return nil
			}
			if strings.Contains(normalizarNombre(d.Name()), limpio) {
				candidatos = append(candidatos, ruta)
			}
			return nil
		})
	}
	if len(candidatos) > 0 {
		// El de nombre mas corto suele ser el lanzador principal.
		menor := candidatos[0]
		for _, c := range candidatos[1:] {
			if len(c) < len(menor) {
				menor = c
			}
		}
		return menor
	}

	// 3) Ultimo recurso: ejecutables con ese nombre en el indice.
	if db != nil {
		for _, ruta := range BuscarIndice(limpio, 10) {
			if strings.EqualFold(filepath.Ext(ruta), ".exe") &&
				strings.Contains(normalizarNombre(filepath.Base(ruta)), limpio) {
				return ruta
			}
		}
	}
	return ""
}

func leerArchivo(ruta string) string {
	contenido, err := os.ReadFile(ruta)
	if err != nil {
		return fmt.Sprintf("No pude leer %s: %v", ruta, err)
	}
	texto := string(contenido)
	if len(texto) > 3000 {
		return fmt.Sprintf("Contenido de %s (truncado, usa offset para seguir): %s", ruta, texto[:3000])
	}
	return fmt.Sprintf("Contenido de %s: %s", ruta, texto)
}

func crearCarpeta(ruta string) string {
	if err := os.MkdirAll(ruta, 0o755); err != nil {
		return fmt.Sprintf("No pude crear la carpeta: %v", err)
	}
	return "Carpeta creada: " + ruta
}

func crearArchivo(ruta, contenido string) string {
	if strings.TrimSpace(ruta) == "" {
		return "Falta el nombre del archivo."
	}
	if !filepath.IsAbs(ruta) {
		ruta = filepath.Join(directorioActual, ruta)
	}
	if contenido == "" {
		contenido = ""
	}
	if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
		return fmt.Sprintf("No pude crear el archivo: %v", err)
	}
	return "Archivo creado: " + ruta
}

func infoArchivo(ruta string) string {
	info, err := os.Stat(ruta)
	if err != nil {
		return fmt.Sprintf("No existe: %s", ruta)
	}
	tipo := "archivo"
	if info.IsDir() {
		tipo = "carpeta"
	}
	return fmt.Sprintf("%s es un %s de %.1f KB, modificado el %s.",
		ruta, tipo, float64(info.Size())/1024, info.ModTime().Format("2006-01-02 15:04"))
}

// EjecutarHerramienta despacha una llamada del modelo y devuelve el resultado.
func EjecutarHerramienta(nombre string, argsJSON json.RawMessage) string {
	var args map[string]any
	if len(argsJSON) > 0 {
		json.Unmarshal(argsJSON, &args)
	}
	texto := func(clave string) string {
		v, _ := args[clave].(string)
		return strings.TrimSpace(v)
	}
	switch nombre {
	case "buscar_archivos":
		// Con indice SQLite es instantaneo; sin indice, recorre el disco.
		if db != nil {
			return formatearResultados(texto("consulta"))
		}
		return formatearBusqueda(texto("consulta"))
	case "navegar_carpeta":
		return navegar(texto("ruta"))
	case "abrir_archivo":
		return abrir(texto("ruta"))
	case "leer_archivo":
		return leerArchivo(texto("ruta"))
	case "crear_carpeta":
		return crearCarpeta(texto("ruta"))
	case "crear_archivo":
		return crearArchivo(texto("ruta"), texto("contenido"))
	case "listar_carpeta":
		return listar(texto("ruta"))
	case "info_archivo":
		return infoArchivo(texto("ruta"))
	case "buscar_web":
		return BuscarWeb(texto("consulta"))
	case "guardar_memoria":
		return GuardarMemoria(texto("clave"), texto("valor"))
	case "consultar_memoria":
		return ConsultarMemoria(texto("clave"))
	case "olvidar_memoria":
		return OlvidarMemoria(texto("clave"))
	default:
		return fmt.Sprintf("Herramienta desconocida: %s", nombre)
	}
}
