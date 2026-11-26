package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLeerPDFVito(t *testing.T) {
	carpeta := t.TempDir()
	pdf := filepath.Join(carpeta, "memo.pdf")
	if err := CrearPDF(pdf, "Memorando", "La reunion es el viernes.\nLlevar el informe."); err != nil {
		t.Fatalf("no pude crear el pdf: %v", err)
	}
	texto, err := leerPDF(pdf)
	if err != nil {
		t.Fatalf("no pude leer el pdf: %v", err)
	}
	if !strings.Contains(texto, "Memorando") || !strings.Contains(texto, "viernes") {
		t.Errorf("texto extraido incompleto: %q", texto)
	}
	// Idem a traves del lector general de documentos.
	viaLeer, err := LeerTextoDocumento(pdf)
	if err != nil || !strings.Contains(viaLeer, "informe") {
		t.Errorf("LeerTextoDocumento fallo: %q (err=%v)", viaLeer, err)
	}
}

func TestLeerPDFAcentos(t *testing.T) {
	carpeta := t.TempDir()
	pdf := filepath.Join(carpeta, "acentos.pdf")
	if err := CrearPDF(pdf, "", "Traducción y año 2026"); err != nil {
		t.Fatalf("no pude crear el pdf: %v", err)
	}
	texto, err := leerPDF(pdf)
	if err != nil {
		t.Fatalf("no pude leer: %v", err)
	}
	if !strings.Contains(texto, "ó") || !strings.Contains(texto, "Traducción") {
		t.Errorf("los acentos no sobrevivieron: %q", texto)
	}
}

func TestLeerPDFAcentosLimpios(t *testing.T) {
	// Los octales no deben dejar rastros de barras en el texto.
	carpeta := t.TempDir()
	pdf := filepath.Join(carpeta, "ninos.pdf")
	CrearPDF(pdf, "", "Niños jugando")
	texto, _ := leerPDF(pdf)
	if strings.Contains(texto, "\\3") {
		t.Errorf("quedaron escapes octales sin decodificar: %q", texto)
	}
	if _, err := os.Stat(pdf); err != nil {
		t.Fatal(err)
	}
}
