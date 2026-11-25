package main

// Auto-actualizacion: consulta los Releases de GitHub, descarga el nuevo
// vito.exe del paquete y lo instala al salir (Windows no permite
// sobrescribir un ejecutable mientras corre, asi que el cambio lo hace un
// script que arranca justo antes de salir).

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// VersionActual es la version de este ejecutable. Debe coincidir con la
// etiqueta del release cuando se publique.
const VersionActual = "1.2.0"

const urlReleases = "https://api.github.com/repos/JuanCarpeta/Vito/releases/latest"

// reiniciarTrasActualizar la activa el comando 'actualizar' y main la
// consulta despues de cada respuesta para salir y aplicar el cambio.
var reiniciarTrasActualizar atomic.Bool

type lanzamiento struct {
	TagName string         `json:"tag_name"`
	Assets  []activoLanzam `json:"assets"`
}

type activoLanzam struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// versionNumerica quita la "v" inicial de una etiqueta ("v1.1" -> "1.1").
func versionNumerica(etiqueta string) string {
	return strings.TrimPrefix(strings.TrimSpace(etiqueta), "v")
}

// esVersionNueva compara "a.b[.c]" con la version actual campo por campo.
func esVersionNueva(otra string) bool {
	camposOtra := strings.Split(versionNumerica(otra), ".")
	camposActual := strings.Split(VersionActual, ".")
	for i := 0; i < len(camposOtra) || i < len(camposActual); i++ {
		numOtra, numActual := 0, 0
		if i < len(camposOtra) {
			fmt.Sscanf(camposOtra[i], "%d", &numOtra)
		}
		if i < len(camposActual) {
			fmt.Sscanf(camposActual[i], "%d", &numActual)
		}
		if numOtra != numActual {
			return numOtra > numActual
		}
	}
	return false
}

// HayActualizacion consulta GitHub y devuelve (version nueva, url del zip).
// Sin internet o sin release devuelve cadenas vacias: nunca es un error fatal.
func HayActualizacion() (string, string) {
	cliente := &http.Client{Timeout: 8 * time.Second}
	peticion, err := http.NewRequest("GET", urlReleases, nil)
	if err != nil {
		return "", ""
	}
	peticion.Header.Set("User-Agent", "vito-updater")
	resp, err := cliente.Do(peticion)
	if err != nil {
		return "", ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", ""
	}
	var lanzamiento lanzamiento
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&lanzamiento); err != nil {
		return "", ""
	}
	if !esVersionNueva(lanzamiento.TagName) {
		return "", ""
	}
	for _, activo := range lanzamiento.Assets {
		if strings.EqualFold(activo.Name, "vito.zip") {
			return versionNumerica(lanzamiento.TagName), activo.URL
		}
	}
	return "", ""
}

// DescargarNuevoExe baja el zip del release y extrae vito.exe a
// "vito.exe.nuevo" junto al ejecutable actual.
func DescargarNuevoExe(urlZip string) error {
	cliente := &http.Client{Timeout: 120 * time.Second}
	resp, err := cliente.Get(urlZip)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("el servidor respondio con codigo %d", resp.StatusCode)
	}
	temporal, err := os.CreateTemp("", "vito-*.zip")
	if err != nil {
		return err
	}
	rutaZip := temporal.Name()
	defer os.Remove(rutaZip)
	if _, err := io.Copy(temporal, io.LimitReader(resp.Body, 200<<20)); err != nil {
		temporal.Close()
		return err
	}
	temporal.Close()

	return ExeDeZip(rutaZip, filepath.Join(Base, "vito.exe.nuevo"))
}

// ExeDeZip extrae vito.exe de un paquete zip a la ruta destino.
func ExeDeZip(rutaZip, destino string) error {
	lector, err := zip.OpenReader(rutaZip)
	if err != nil {
		return err
	}
	defer lector.Close()
	for _, entrada := range lector.File {
		if !strings.EqualFold(entrada.Name, "vito.exe") {
			continue
		}
		abierta, err := entrada.Open()
		if err != nil {
			return err
		}
		salida, err := os.Create(destino)
		if err != nil {
			abierta.Close()
			return err
		}
		_, err = io.Copy(salida, abierta)
		abierta.Close()
		salida.Close()
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("el paquete no contiene vito.exe")
}

// iniciarProceso lanza un comando sin ventana de consola.
func iniciarProceso(ruta string) *exec.Cmd {
	comando := exec.Command(ruta)
	comando.SysProcAttr = sysProcOculto()
	return comando
}

// InstalarActualizacion escribe actualizar.bat (que reemplaza el exe y
// reinicia Vito) y lo lanza. Vito debe salir inmediatamente despues.
func InstalarActualizacion(version string) error {
	script := fmt.Sprintf(`@echo off
timeout /t 2 /nobreak >nul
if exist "%s\vito.exe" move /y "%s\vito.exe" "%s\vito.exe.old" >nul
move /y "%s\vito.exe.nuevo" "%s\vito.exe" >nul
if exist "%s\vito.exe.old" del "%s\vito.exe.old" >nul
start "" "%s\vito.exe"
del "%%~f0" >nul
`, Base, Base, Base, Base, Base, Base, Base, Base)
	rutaScript := filepath.Join(Base, "actualizar.bat")
	if err := os.WriteFile(rutaScript, []byte(script), 0o644); err != nil {
		return err
	}
	return iniciarProceso(rutaScript).Start()
}

// ActualizarVito ejecuta el flujo completo del comando 'actualizar'.
func ActualizarVito() string {
	version, url := HayActualizacion()
	if version == "" {
		return fmt.Sprintf("Ya tienes la ultima version (v%s).", VersionActual)
	}
	if err := DescargarNuevoExe(url); err != nil {
		return fmt.Sprintf("No pude descargar la actualizacion: %v", err)
	}
	if err := InstalarActualizacion(version); err != nil {
		return fmt.Sprintf("Descargue la v%s pero no pude preparar el cambio: %v", version, err)
	}
	reiniciarTrasActualizar.Store(true)
	return fmt.Sprintf("Descargue la v%s. Vito se reiniciara en unos segundos con la version nueva.", version)
}

// VerificarActualizacionAlArrancar avisa si hay version nueva (una linea).
func VerificarActualizacionAlArrancar() string {
	version, _ := HayActualizacion()
	if version == "" {
		return ""
	}
	return fmt.Sprintf("Hay una actualizacion disponible (v%s). Escribe 'actualizar' para instalarla.", version)
}
