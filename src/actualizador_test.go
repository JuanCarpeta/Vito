package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEsVersionNueva(t *testing.T) {
	casos := []struct {
		otra  string
		nueva bool
	}{
		{"v1.0", false},
		{"v1.2.0", false},
		{"v1.3", true},
		{"v2.0", true},
		{"1.9", true},
		{"", false},
	}
	for _, c := range casos {
		if got := esVersionNueva(c.otra); got != c.nueva {
			t.Errorf("esVersionNueva(%q) = %v, esperaba %v", c.otra, got, c.nueva)
		}
	}
}

func TestHayActualizacionConsultaGitHub(t *testing.T) {
	version, url := HayActualizacion()
	t.Logf("HayActualizacion -> version=%q url=%q", version, url)
	// Coherencia: version y url van siempre juntas.
	if (version == "") != (url == "") {
		t.Errorf("inconsistente: version=%q url=%q", version, url)
	}
}

func TestExeDeZip(t *testing.T) {
	carpeta := t.TempDir()
	paquete := filepath.Join(carpeta, "paquete.zip")
	if err := escribirZip(paquete, map[string]string{
		"vito.exe":         "contenido-del-exe",
		"instalar.bat":     "@echo off",
		"COMO-INSTALAR.md": "# guia",
	}); err != nil {
		t.Fatalf("no pude crear el zip de prueba: %v", err)
	}
	destino := filepath.Join(carpeta, "vito.exe.nuevo")
	if err := ExeDeZip(paquete, destino); err != nil {
		t.Fatalf("extraccion fallo: %v", err)
	}
	contenido, err := os.ReadFile(destino)
	if err != nil {
		t.Fatalf("no pude leer el exe extraido: %v", err)
	}
	if !strings.Contains(string(contenido), "contenido-del-exe") {
		t.Errorf("contenido inesperado: %q", contenido)
	}
	// Un zip sin vito.exe debe fallar con mensaje claro.
	vacio := filepath.Join(carpeta, "vacio.zip")
	escribirZip(vacio, map[string]string{"otro.txt": "x"})
	if err := ExeDeZip(vacio, destino); err == nil {
		t.Error("extraccion de zip sin vito.exe deberia fallar")
	}
}
