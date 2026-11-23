package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBuscarEnDiscoTodoElDisco planta un archivo en una carpeta profunda y
// comprueba que la busqueda por nombre lo encuentre recorriendo las unidades.
func TestBuscarEnDiscoTodoElDisco(t *testing.T) {
	home, _ := os.UserHomeDir()
	directorio := filepath.Join(home, "vito-prueba-busqueda", "SubNivel")
	if err := os.MkdirAll(directorio, 0o755); err != nil {
		t.Fatalf("no pude crear la carpeta de prueba: %v", err)
	}
	rutaObjetivo := filepath.Join(directorio, "tesoro-perdido.txt")
	if err := os.WriteFile(rutaObjetivo, []byte("prueba"), 0o644); err != nil {
		t.Fatalf("no pude crear el archivo de prueba: %v", err)
	}
	defer os.RemoveAll(filepath.Join(home, "vito-prueba-busqueda"))

	inicio := time.Now()
	encontrados := buscarEnDisco("tesoro-perdido", 5)
	tardo := time.Since(inicio)

	if len(encontrados) == 0 {
		t.Fatalf("la busqueda no encontro %s", rutaObjetivo)
	}
	if !strings.HasSuffix(encontrados[0], "tesoro-perdido.txt") {
		t.Errorf("resultado inesperado: %s", encontrados[0])
	}
	if tardo > 25*time.Second {
		t.Errorf("la busqueda tardo demasiado: %v", tardo)
	}
}

// TestDiscosDisponibles comprueba que al menos C: se detecte.
func TestDiscosDisponibles(t *testing.T) {
	discos := discosDisponibles()
	if len(discos) == 0 {
		t.Fatal("no se detecto ninguna unidad")
	}
	encontroC := false
	for _, d := range discos {
		if d == `C:\` {
			encontroC = true
		}
	}
	if !encontroC {
		t.Errorf("no se detecto C:\\; unidades: %v", discos)
	}
}

func TestBorrarArchivoConfirmacion(t *testing.T) {
	anterior := directorioActual
	directorioActual = t.TempDir()
	defer func() { directorioActual = anterior }()

	ruta := filepath.Join(directorioActual, "paso-dos.docx")
	if err := os.WriteFile(ruta, []byte("prueba"), 0o644); err != nil {
		t.Fatalf("no pude crear el archivo de prueba: %v", err)
	}

	respuesta := borrarArchivo("paso-dos.docx", func(string) bool { return false })
	if !strings.Contains(respuesta, "No borre nada") {
		t.Errorf("respuesta inesperada al cancelar: %s", respuesta)
	}
	if _, err := os.Stat(ruta); err != nil {
		t.Fatalf("borro el archivo sin confirmacion: %v", err)
	}

	respuesta = borrarArchivo("paso-dos.docx", func(string) bool { return true })
	if !strings.Contains(respuesta, "Borre") {
		t.Errorf("respuesta inesperada al confirmar: %s", respuesta)
	}
	if _, err := os.Stat(ruta); !os.IsNotExist(err) {
		t.Fatalf("el archivo sigue ahi despues de confirmar: %v", err)
	}
}

func TestNombreYPistas(t *testing.T) {
	nombre, pistas := nombreYPistas("el paso-dos.docx de Ejemplo")
	if nombre != "paso-dos.docx" {
		t.Errorf("nombre inesperado: %s", nombre)
	}
	if len(pistas) != 1 || pistas[0] != "ejemplo" {
		t.Errorf("pistas inesperadas: %v", pistas)
	}
}

func TestRutaProtegida(t *testing.T) {
	if !rutaProtegida(filepath.Join(dirDatos(), "vito.exe")) {
		t.Error("vito.exe deberia estar protegido")
	}
}

func TestCopiarArchivo(t *testing.T) {
	anterior := directorioActual
	directorioActual = t.TempDir()
	defer func() { directorioActual = anterior }()

	origen := filepath.Join(directorioActual, "origen.txt")
	if err := os.WriteFile(origen, []byte("contenido"), 0o644); err != nil {
		t.Fatalf("no pude crear el origen: %v", err)
	}
	destino := filepath.Join(directorioActual, "destino")
	if err := os.MkdirAll(destino, 0o755); err != nil {
		t.Fatalf("no pude crear el destino: %v", err)
	}

	respuesta := moverOCopiar("origen.txt", destino, false, nil)
	if !strings.Contains(respuesta, "Copie") {
		t.Errorf("respuesta inesperada: %s", respuesta)
	}
	if _, err := os.Stat(origen); err != nil {
		t.Fatalf("la copia borro el origen: %v", err)
	}
	datos, err := os.ReadFile(filepath.Join(destino, "origen.txt"))
	if err != nil || string(datos) != "contenido" {
		t.Fatalf("copia incorrecta: %q, %v", datos, err)
	}
}

func TestMoverArchivo(t *testing.T) {
	anterior := directorioActual
	directorioActual = t.TempDir()
	defer func() { directorioActual = anterior }()

	origen := filepath.Join(directorioActual, "origen.txt")
	if err := os.WriteFile(origen, []byte("contenido"), 0o644); err != nil {
		t.Fatalf("no pude crear el origen: %v", err)
	}
	destino := filepath.Join(directorioActual, "destino")
	if err := os.MkdirAll(destino, 0o755); err != nil {
		t.Fatalf("no pude crear el destino: %v", err)
	}

	respuesta := moverOCopiar("origen.txt", destino, true, nil)
	if !strings.Contains(respuesta, "Move") {
		t.Errorf("respuesta inesperada: %s", respuesta)
	}
	if _, err := os.Stat(origen); !os.IsNotExist(err) {
		t.Fatalf("el origen sigue ahi: %v", err)
	}
	if _, err := os.Stat(filepath.Join(destino, "origen.txt")); err != nil {
		t.Fatalf("no llego al destino: %v", err)
	}
}

func TestSobrescribirRequiereConfirmacion(t *testing.T) {
	anterior := directorioActual
	directorioActual = t.TempDir()
	defer func() { directorioActual = anterior }()

	origen := filepath.Join(directorioActual, "origen.txt")
	if err := os.WriteFile(origen, []byte("nuevo"), 0o644); err != nil {
		t.Fatalf("no pude crear el origen: %v", err)
	}
	destino := filepath.Join(directorioActual, "destino")
	if err := os.MkdirAll(destino, 0o755); err != nil {
		t.Fatalf("no pude crear el destino: %v", err)
	}
	final := filepath.Join(destino, "origen.txt")
	if err := os.WriteFile(final, []byte("viejo"), 0o644); err != nil {
		t.Fatalf("no pude crear el destino existente: %v", err)
	}

	respuesta := moverOCopiar("origen.txt", destino, true, func(string) bool { return false })
	if !strings.Contains(respuesta, "No hice nada") {
		t.Errorf("respuesta inesperada al cancelar: %s", respuesta)
	}
	datos, err := os.ReadFile(final)
	if err != nil || string(datos) != "viejo" {
		t.Fatalf("sobrescribio sin confirmacion: %q, %v", datos, err)
	}
	if _, err := os.Stat(origen); err != nil {
		t.Fatalf("borro el origen sin confirmacion: %v", err)
	}

	respuesta = moverOCopiar("origen.txt", destino, true, func(string) bool { return true })
	if !strings.Contains(respuesta, "Move") {
		t.Errorf("respuesta inesperada al confirmar: %s", respuesta)
	}
	datos, err = os.ReadFile(final)
	if err != nil || string(datos) != "nuevo" {
		t.Fatalf("no sobrescribio tras confirmar: %q, %v", datos, err)
	}
}

func TestMoverConPronombre(t *testing.T) {
	anterior := directorioActual
	anteriorUltimo := ultimoArchivo
	directorioActual = t.TempDir()
	defer func() {
		directorioActual = anterior
		ultimoArchivo = anteriorUltimo
	}()

	origen := filepath.Join(directorioActual, "origen.txt")
	if err := os.WriteFile(origen, []byte("contenido"), 0o644); err != nil {
		t.Fatalf("no pude crear el origen: %v", err)
	}
	primero := filepath.Join(directorioActual, "uno")
	segundo := filepath.Join(directorioActual, "dos")
	for _, carpeta := range []string{primero, segundo} {
		if err := os.MkdirAll(carpeta, 0o755); err != nil {
			t.Fatalf("no pude crear el destino: %v", err)
		}
	}

	moverOCopiar("origen.txt", primero, true, nil)
	respuesta := moverOCopiar("esto", segundo, true, nil)
	if !strings.Contains(respuesta, "Move") {
		t.Errorf("respuesta inesperada: %s", respuesta)
	}
	if _, err := os.Stat(filepath.Join(segundo, "origen.txt")); err != nil {
		t.Fatalf("'esto' no movio el ultimo archivo: %v", err)
	}
}

func TestResolverCarpetaDestinoAlias(t *testing.T) {
	ruta, mensaje := resolverCarpetaDestino("descargas")
	if mensaje != "" {
		t.Fatalf("no resolvio descargas: %s", mensaje)
	}
	nombre := strings.ToLower(filepath.Base(ruta))
	if !strings.Contains(nombre, "download") && !strings.Contains(nombre, "descarg") {
		t.Errorf("destino inesperado: %s", ruta)
	}
}
