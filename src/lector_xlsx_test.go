package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLeerXLSXVito(t *testing.T) {
	carpeta := t.TempDir()
	xlsx := filepath.Join(carpeta, "tabla.xlsx")
	if err := CrearXLSX(xlsx, "Nombre;Edad;Ciudad\nJuan;20;Bogota\nAna;22;Medellin"); err != nil {
		t.Fatalf("no pude crear el xlsx: %v", err)
	}
	texto, err := leerXLSX(xlsx)
	if err != nil {
		t.Fatalf("no pude leer: %v", err)
	}
	for _, esperado := range []string{"Nombre;Edad;Ciudad", "Juan;20;Bogota", "Ana;22;Medellin"} {
		if !strings.Contains(texto, esperado) {
			t.Errorf("falta %q en:\n%s", esperado, texto)
		}
	}
	// La segunda fila no debe heredar celdas de la primera.
	if strings.Contains(texto, "Juan;20;Medellin") {
		t.Errorf("columnas cruzadas entre filas:\n%s", texto)
	}
}

func TestLeerXLSXConCeldasVacias(t *testing.T) {
	carpeta := t.TempDir()
	xlsx := filepath.Join(carpeta, "huecos.xlsx")
	// Fila con hueco en medio: A y C tienen valor, B vacia.
	if err := CrearXLSX(xlsx, "uno;;tres"); err != nil {
		t.Fatalf("no pude crear: %v", err)
	}
	texto, err := leerXLSX(xlsx)
	if err != nil {
		t.Fatalf("no pude leer: %v", err)
	}
	if !strings.Contains(texto, "uno;;tres") {
		t.Errorf("el hueco del medio se perdio:\n%s", texto)
	}
}

func TestLeerXLSXViaLector(t *testing.T) {
	carpeta := t.TempDir()
	xlsx := filepath.Join(carpeta, "general.xlsx")
	CrearXLSX(xlsx, "dato de prueba")
	texto, err := LeerTextoDocumento(xlsx)
	if err != nil || !strings.Contains(texto, "dato de prueba") {
		t.Errorf("LeerTextoDocumento fallo: %q (err=%v)", texto, err)
	}
}
