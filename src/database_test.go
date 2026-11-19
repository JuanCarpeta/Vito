package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBorradoSubarbol comprueba que borrar una carpeta quita tambien sus hijos.
func TestBorradoSubarbol(t *testing.T) {
	if err := InicializarDB(); err != nil {
		t.Fatalf("db: %v", err)
	}
	defer func() {
		db.Close()
		db = nil
		os.Remove(rutaDB)
	}()
	base := filepath.Join(Base, "prueba-patron")
	db.Exec(`INSERT OR REPLACE INTO archivos(ruta,nombre,normalizado,padre) VALUES(?,?,?,?)`,
		filepath.Join(base, "Sub"), "Sub", "sub", base)
	db.Exec(`INSERT OR REPLACE INTO archivos(ruta,nombre,normalizado,padre) VALUES(?,?,?,?)`,
		filepath.Join(base, "Sub", "foto.png"), "foto.png", "foto.png", filepath.Join(base, "Sub"))

	if err := BorrarArchivo(filepath.Join(base, "Sub")); err != nil {
		t.Fatalf("borrado fallo: %v", err)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM archivos WHERE ruta LIKE ?`,
		filepath.Join(base, "Sub")+"%").Scan(&n)
	if n != 0 {
		t.Errorf("quedaron %d filas del subarbol", n)
	}
}
