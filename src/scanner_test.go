package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIndiceEscaneaYBusca crea una carpeta con archivos, la escanea y
// comprueba que la busqueda por indice la encuentra.
func TestIndiceEscaneaYBusca(t *testing.T) {
	if err := InicializarDB(); err != nil {
		t.Fatalf("no pude inicializar la base: %v", err)
	}
	defer func() {
		if db != nil {
			db.Close()
			db = nil
		}
		os.Remove(rutaDB)
	}()

	carpeta := filepath.Join(Base, "prueba-indice-vito")
	if err := os.MkdirAll(filepath.Join(carpeta, "Sub"), 0o755); err != nil {
		t.Fatalf("no pude crear la carpeta de prueba: %v", err)
	}
	defer os.RemoveAll(carpeta)
	archivos := []string{"informe-quimica.txt", "Sub\\foto-vacaciones.png"}
	for _, relativa := range archivos {
		if err := os.WriteFile(filepath.Join(carpeta, relativa), []byte("x"), 0o644); err != nil {
			t.Fatalf("no pude crear %s: %v", relativa, err)
		}
	}

	if _, err := escanearCarpeta(carpeta); err != nil {
		t.Fatalf("no pude escanear la carpeta: %v", err)
	}
	escanearSubarbol(carpeta, time.Now().Add(time.Minute))

	for _, consulta := range []string{"informe quimica", "foto vacaciones"} {
		resultados := BuscarIndice(consulta, 5)
		if len(resultados) == 0 {
			t.Errorf("la busqueda '%s' no encontro nada", consulta)
			continue
		}
		if !strings.Contains(resultados[0], carpeta) {
			t.Errorf("resultado inesperado para '%s': %s", consulta, resultados[0])
		}
	}

	// Borrar una carpeta debe quitarla del indice junto con su subarbol.
	BorrarArchivo(filepath.Join(carpeta, "Sub"))
	var quedan int
	err := db.QueryRow(`SELECT COUNT(*) FROM archivos WHERE ruta LIKE ?`,
		filepath.Join(carpeta, "Sub")+"%").Scan(&quedan)
	if err != nil {
		t.Fatalf("no pude verificar el borrado: %v", err)
	}
	if quedan != 0 {
		// Diagnostico: mostrar que fila quedo.
		filas, _ := db.Query(`SELECT ruta FROM archivos WHERE ruta LIKE ?`,
			filepath.Join(carpeta, "Sub")+"%")
		for filas.Next() {
			var r string
			filas.Scan(&r)
			t.Logf("fila restante: %q", r)
		}
		filas.Close()
		t.Errorf("quedaron %d filas del subarbol borrado", quedan)
	}
}
