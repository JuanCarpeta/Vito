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
