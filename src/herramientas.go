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

var ultimoArchivo string

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

// CerrarApp cierra una aplicacion por nombre: intento de cierre limpio y,
// si no responde, forzado. Devuelve un mensaje para el usuario.
func CerrarApp(nombre string) string {
	nombre = strings.TrimSpace(nombre)
	if nombre == "" {
		return "Dime que aplicacion quieres que cierre."
	}
	proceso := resolverProceso(nombre)
	if proceso == "" {
		return fmt.Sprintf("No encontre una aplicacion llamada %s.", nombre)
	}
	if !procesoCorriendo(proceso) {
		return fmt.Sprintf("%s no esta corriendo ahora mismo.", proceso)
	}
	// 1) Cierre elegante: el programa puede guardar su trabajo.
	cmd := exec.Command("taskkill", "/IM", proceso)
	cmd.SysProcAttr = sysProcOculto()
	if err := cmd.Run(); err == nil {
		return fmt.Sprintf("Cerre %s.", proceso)
	}
	// 2) No respondio: cierre forzado (como Ctrl+Alt+Supr > finalizar tarea).
	cmd = exec.Command("taskkill", "/F", "/IM", proceso)
	cmd.SysProcAttr = sysProcOculto()
	if err := cmd.Run(); err == nil {
		return fmt.Sprintf("%s no respondio al cierre normal, asi que lo cerre a la fuerza.", proceso)
	}
	return fmt.Sprintf("No pude cerrar %s; puede que necesite permisos de administrador.", proceso)
}

// resolverProceso convierte un nombre de aplicacion en el nombre de su proceso
// (por ejemplo 'steam' -> 'steam.exe'), resolviendo accesos directos.
func resolverProceso(nombre string) string {
	nombre = strings.TrimSuffix(strings.TrimSpace(nombre), ".exe")
	if nombre == "" {
		return ""
	}
	// 1) App Paths del registro da el ejecutable directo.
	limpio := normalizarNombre(nombre)
	for _, raiz := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
		clave, err := registry.OpenKey(raiz,
			fmt.Sprintf(`SOFTWARE\Microsoft\Windows\CurrentVersion\App Paths\%s.exe`, limpio),
			registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		ruta, _, err := clave.GetStringValue("")
		clave.Close()
		if err == nil && ruta != "" {
			return strings.ToLower(filepath.Base(ruta))
		}
	}
	// 2) Si viene de un acceso directo del Menu Inicio, resolver su destino.
	atajo := resolverApp(nombre)
	if strings.EqualFold(filepath.Ext(atajo), ".lnk") {
		salida, err := exec.Command("powershell", "-NoProfile", "-Command",
			fmt.Sprintf("(New-Object -ComObject WScript.Shell).CreateShortcut('%s').TargetPath",
				strings.ReplaceAll(atajo, "'", "''"))).Output()
		if err == nil {
			destino := strings.TrimSpace(string(salida))
			if destino != "" {
				return strings.ToLower(filepath.Base(destino))
			}
		}
	}
	if atajo != "" && strings.EqualFold(filepath.Ext(atajo), ".exe") {
		return strings.ToLower(filepath.Base(atajo))
	}
	// 3) Convencion comun: el nombre mismo es el proceso.
	return limpio + ".exe"
}

// procesoCorriendo comprueba en tasklist si el proceso esta activo.
func procesoCorriendo(proceso string) bool {
	salida, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return true // si no podemos comprobarlo, intentamos cerrarlo igual
	}
	return strings.Contains(strings.ToLower(string(salida)), strings.ToLower(proceso))
}

// extraerRutaCreada saca la ruta del mensaje 'Archivo creado: <ruta>'.
func extraerRutaCreada(respuesta string) string {
	const prefijo = "Archivo creado: "
	if indice := strings.Index(respuesta, prefijo); indice >= 0 {
		ruta := strings.TrimSpace(respuesta[indice+len(prefijo):])
		if fin := strings.Index(ruta, "\n"); fin >= 0 {
			ruta = ruta[:fin]
		}
		return ruta
	}
	return respuesta
}

func leerArchivo(ruta string) string {
	// Documentos con formato: usar los lectores dedicados.
	ext := strings.ToLower(filepath.Ext(ruta))
	if ext == ".pdf" || ext == ".docx" || ext == ".xlsx" {
		texto, err := LeerTextoDocumento(ruta)
		if err != nil {
			return fmt.Sprintf("No pude leer %s: %v", ruta, err)
		}
		if len(texto) > 3000 {
			return fmt.Sprintf("Contenido de %s (truncado): %s", ruta, texto[:3000])
		}
		return fmt.Sprintf("Contenido de %s: %s", ruta, texto)
	}
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
		tipo = "una carpeta"
	} else {
		tipo = "un archivo"
	}
	return fmt.Sprintf("%s es %s de %.1f KB, modificado el %s.",
		ruta, tipo, float64(info.Size())/1024, info.ModTime().Format("2006-01-02 15:04"))
}

func resolverArchivo(consulta string) (string, string) {
	consulta = strings.Trim(strings.TrimSpace(consulta), `"'`)
	if consulta == "" {
		return "", "Falta el nombre del archivo."
	}
	if esPronombreArchivo(consulta) {
		if ultimoArchivo == "" {
			return "", "No se a que archivo te refieres."
		}
		if _, err := os.Stat(ultimoArchivo); err != nil {
			return "", "No se a que archivo te refieres."
		}
		return ultimoArchivo, ""
	}

	ruta := ""
	if filepath.IsAbs(consulta) {
		ruta = filepath.Clean(consulta)
	} else {
		candidata := filepath.Join(directorioActual, consulta)
		if _, err := os.Stat(candidata); err == nil {
			ruta = candidata
		}
	}

	if ruta == "" {
		nombre, pistas := nombreYPistas(consulta)
		encontradas := []string{}
		if db != nil {
			for _, candidata := range BuscarIndice(nombre, 20) {
				if _, err := os.Stat(candidata); err != nil {
					continue
				}
				coincide := true
				for _, pista := range pistas {
					if !strings.Contains(normalizarNombre(filepath.Dir(candidata)), normalizarNombre(pista)) {
						coincide = false
						break
					}
				}
				if coincide {
					encontradas = append(encontradas, candidata)
				}
			}
		}
		if len(encontradas) == 0 {
			return "", "No encontre ningun archivo con ese nombre."
		}
		if len(encontradas) > 1 {
			lineas := []string{"Encontre varios archivos; dime cual (nombre completo o ruta):"}
			for i, r := range encontradas {
				lineas = append(lineas, fmt.Sprintf("%d. %s", i+1, r))
			}
			return "", strings.Join(lineas, "\n")
		}
		ruta = encontradas[0]
	}

	ultimoArchivo = ruta
	return ruta, ""
}

func esPronombreArchivo(texto string) bool {
	switch strings.ToLower(strings.TrimSpace(texto)) {
	case "esto", "este", "esta", "eso", "esa", "ese", "ello":
		return true
	}
	return false
}

func borrarArchivo(consulta string, confirmar func(string) bool) string {
	ruta, mensaje := resolverArchivo(consulta)
	if mensaje != "" {
		return mensaje
	}
	info, err := os.Stat(ruta)
	if err != nil {
		return fmt.Sprintf("No existe: %s", ruta)
	}
	if info.IsDir() {
		return "No borro carpetas, solo archivos."
	}
	if rutaProtegida(ruta) {
		return "No puedo borrar ese archivo: es parte de Vito o del sistema."
	}
	if confirmar == nil || !confirmar(fmt.Sprintf("¿Borrar %s? (s/N): ", ruta)) {
		return "No borre nada."
	}
	if err := os.Remove(ruta); err != nil {
		return fmt.Sprintf("No pude borrar %s: %v", ruta, err)
	}
	if db != nil {
		BorrarArchivo(ruta)
	}
	escribirLog("borrado: %s", ruta)
	ultimoArchivo = ""
	return fmt.Sprintf("Borre %s.", ruta)
}

func moverOCopiar(consulta, destino string, mover bool, confirmar func(string) bool) string {
	origen, mensaje := resolverArchivo(consulta)
	if mensaje != "" {
		return mensaje
	}
	info, err := os.Stat(origen)
	if err != nil {
		return fmt.Sprintf("No existe: %s", origen)
	}
	if rutaProtegida(origen) {
		return "No puedo usar ese archivo: es parte de Vito o del sistema."
	}

	carpeta, mensaje := resolverCarpetaDestino(destino)
	if mensaje != "" {
		return mensaje
	}
	final := filepath.Join(carpeta, filepath.Base(origen))
	if strings.EqualFold(origen, final) {
		return "El archivo ya esta en esa carpeta."
	}
	if info.IsDir() && strings.HasPrefix(strings.ToLower(final), strings.ToLower(origen)+string(os.PathSeparator)) {
		return "No puedo mover una carpeta dentro de si misma."
	}
	if _, err := os.Stat(final); err == nil {
		if rutaProtegida(final) {
			return "No puedo sobrescribir ese destino: es parte de Vito o del sistema."
		}
		if confirmar == nil || !confirmar(fmt.Sprintf("Ya existe %s. ¿Sobrescribir? (s/N): ", final)) {
			return "No hice nada."
		}
		if err := os.RemoveAll(final); err != nil {
			return fmt.Sprintf("No pude sobrescribir %s: %v", final, err)
		}
	}

	accion := "Copie"
	if mover {
		accion = "Moví"
		if err := os.Rename(origen, final); err != nil {
			if err := copiarRuta(origen, final, info.IsDir()); err != nil {
				return fmt.Sprintf("No pude mover %s: %v", origen, err)
			}
			if err := os.RemoveAll(origen); err != nil {
				return fmt.Sprintf("Copie %s pero no pude borrar el original: %v", origen, err)
			}
		}
		if db != nil {
			BorrarArchivo(origen)
		}
	} else {
		if err := copiarRuta(origen, final, info.IsDir()); err != nil {
			return fmt.Sprintf("No pude copiar %s: %v", origen, err)
		}
	}

	if db != nil {
		if info.IsDir() {
			escanearSubarbol(final, time.Now().Add(2*time.Minute))
		} else {
			indexarArchivoSuelto(final)
		}
	}
	escribirLog("%s %s a %s", strings.ToLower(accion), origen, final)
	ultimoArchivo = final
	return fmt.Sprintf("%s %s a %s.", accion, origen, final)
}

func resolverCarpetaDestino(destino string) (string, string) {
	destino = strings.Trim(strings.TrimSpace(destino), `"'`)
	if destino == "" {
		return "", "Falta la carpeta de destino."
	}
	if ruta, ok := rutaConocida(destino); ok {
		if ruta == "" {
			return "", "No encontre la carpeta " + destino
		}
		return ruta, ""
	}
	if ruta, _ := resolverDestino(destino); ruta != "" {
		return ruta, ""
	}
	nueva := destino
	if !filepath.IsAbs(nueva) {
		nueva = filepath.Join(directorioActual, nueva)
	}
	if err := os.MkdirAll(nueva, 0o755); err != nil {
		return "", fmt.Sprintf("No pude crear la carpeta %s: %v", nueva, err)
	}
	return nueva, ""
}

func copiarRuta(origen, destino string, esCarpeta bool) error {
	if !esCarpeta {
		info, err := os.Stat(origen)
		if err != nil {
			return err
		}
		datos, err := os.ReadFile(origen)
		if err != nil {
			return err
		}
		return os.WriteFile(destino, datos, info.Mode().Perm())
	}
	return filepath.WalkDir(origen, func(ruta string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relativa, err := filepath.Rel(origen, ruta)
		if err != nil {
			return err
		}
		nueva := filepath.Join(destino, relativa)
		if d.IsDir() {
			return os.MkdirAll(nueva, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		datos, err := os.ReadFile(ruta)
		if err != nil {
			return err
		}
		return os.WriteFile(nueva, datos, info.Mode().Perm())
	})
}

func nombreYPistas(consulta string) (string, []string) {
	palabrasParar := map[string]bool{
		"el": true, "la": true, "los": true, "las": true, "un": true, "una": true,
		"de": true, "del": true, "en": true, "para": true, "archivo": true,
		"documento": true, "carpeta": true,
	}
	nombre := ""
	pistas := []string{}
	for _, palabra := range strings.Fields(strings.ToLower(consulta)) {
		if palabrasParar[palabra] {
			continue
		}
		if nombre == "" && strings.Contains(palabra, ".") {
			nombre = palabra
			continue
		}
		pistas = append(pistas, palabra)
	}
	if nombre == "" {
		nombre = strings.Join(pistas, "")
		pistas = nil
	}
	return nombre, pistas
}

func rutaProtegida(ruta string) bool {
	abs, err := filepath.Abs(ruta)
	if err != nil {
		return true
	}
	if filepath.Dir(abs) == abs {
		return true
	}
	if home, err := os.UserHomeDir(); err == nil && strings.EqualFold(abs, home) {
		return true
	}
	if strings.EqualFold(abs, dirDatos()) {
		return true
	}
	base := strings.ToLower(filepath.Base(abs))
	return ArchivosPropios[base] || Ignoradas[base]
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
		respuesta := crearArchivo(texto("ruta"), texto("contenido"))
		// Documentos con formato real segun la extension pedida.
		ext := strings.ToLower(filepath.Ext(texto("ruta")))
		rutaFinal := extraerRutaCreada(respuesta)
		switch ext {
		case ".docx", ".doc":
			if err := CrearDOCX(rutaFinal, strings.TrimSpace(texto("titulo")), texto("contenido")); err == nil {
				return "Documento Word creado: " + rutaFinal
			}
		case ".xlsx", ".xls":
			if err := CrearXLSX(rutaFinal, texto("contenido")); err == nil {
				return "Hoja de calculo creada: " + rutaFinal
			}
		case ".pdf":
			if err := CrearPDF(rutaFinal, strings.TrimSpace(texto("titulo")), texto("contenido")); err == nil {
				return "PDF creado: " + rutaFinal
			}
		}
		return respuesta
	case "cerrar_app":
		return CerrarApp(texto("nombre"))
	case "resumir_documento":
		return ResumirDocumento(texto("ruta"))
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
