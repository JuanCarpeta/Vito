package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResumirDocumentoDirecto(t *testing.T) {
	if err := InicializarDB(); err != nil {
		t.Skip("sin base, no importa para este test")
	}
	defer func() { db.Close(); db = nil; os.Remove(rutaDB) }()

	carpeta := filepath.Join(Base, "prueba-resumen")
	os.MkdirAll(carpeta, 0o755)
	defer os.RemoveAll(carpeta)

	docx := filepath.Join(carpeta, "memo.docx")
	if err := CrearDOCX(docx, "Memo interno", "La reunion es el viernes a las 10.\nLlevar los avances del proyecto."); err != nil {
		t.Fatalf("no pude crear el docx: %v", err)
	}
	texto, err := LeerTextoDocumento(docx)
	if err != nil {
		t.Fatalf("no pude leer el docx: %v", err)
	}
	if !strings.Contains(texto, "viernes") {
		t.Errorf("el texto extraido no contiene el contenido: %q", texto)
	}
	t.Logf("texto extraido: %q", texto)

	resumen := ResumirDocumento(docx)
	t.Logf("resumen: %s", resumen)
	if strings.Contains(resumen, "No hay un documento") ||
		strings.Contains(resumen, "No pude") {
		t.Errorf("el resumen fallo: %s", resumen)
	}
}
