package main

// Indice en archivos.db (SQLite): esquema, transacciones y metadatos.
// Espeja core/database.py del proyecto Python.

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Base es la raiz del proyecto (una sola fuente de verdad para todas las rutas).
var Base = func() string {
	ejecutable, err := os.Executable()
	if err != nil {
		d, _ := os.Getwd()
		return d
	}
	return filepath.Dir(ejecutable)
}()

var rutaDB = filepath.Join(Base, "archivos.db")

// VersionIndice debe cambiar cuando cambia lo que se guarda.
const VersionIndice = "1"

var db *sql.DB

func ConectarDB() (*sql.DB, error) {
	con, err := sql.Open("sqlite", rutaDB+"?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	con.SetMaxOpenConns(1)
	return con, nil
}

// InicializarDB crea el esquema y valida la version del indice.
func InicializarDB() error {
	var err error
	db, err = ConectarDB()
	if err != nil {
		return err
	}
	instrucciones := []string{
		`CREATE TABLE IF NOT EXISTS archivos (
			ruta TEXT PRIMARY KEY,
			nombre TEXT NOT NULL,
			extension TEXT NOT NULL DEFAULT '',
			tamano INTEGER NOT NULL DEFAULT 0,
			mtime INTEGER NOT NULL DEFAULT 0,
			es_carpeta INTEGER NOT NULL DEFAULT 0,
			normalizado TEXT NOT NULL DEFAULT '',
			padre TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_padre ON archivos(padre)`,
		`CREATE INDEX IF NOT EXISTS idx_nombre ON archivos(normalizado)`,
		`CREATE TABLE IF NOT EXISTS firmas (
			ruta TEXT PRIMARY KEY,
			firma TEXT NOT NULL,
			escaneado INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS meta (
			clave TEXT PRIMARY KEY,
			valor TEXT NOT NULL
		)`,
	}
	for _, instruccion := range instrucciones {
		if _, err := db.Exec(instruccion); err != nil {
			return err
		}
	}
	var version string
	err = db.QueryRow(`SELECT valor FROM meta WHERE clave='version_indice'`).Scan(&version)
	if err == sql.ErrNoRows {
		_, err = db.Exec(`INSERT INTO meta(clave, valor) VALUES('version_indice', ?)`, VersionIndice)
		return err
	}
	if err != nil {
		return err
	}
	if version != VersionIndice {
		// Esquema distinto: reconstruir desde cero.
		for _, tabla := range []string{"archivos", "firmas"} {
			if _, err := db.Exec(`DROP TABLE IF EXISTS ` + tabla); err != nil {
				return err
			}
		}
		_, err = db.Exec(`UPDATE meta SET valor=? WHERE clave='version_indice'`, VersionIndice)
		if err != nil {
			return err
		}
		return InicializarDB()
	}
	return nil
}

// GuardarArchivo inserta o actualiza una fila del indice.
func GuardarArchivo(e Entrada) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO archivos
		(ruta, nombre, extension, tamano, mtime, es_carpeta, normalizado, padre)
		VALUES(?,?,?,?,?,?,?,?)`,
		e.Ruta, e.Nombre, filepath.Ext(e.Nombre), e.Tamano,
		e.Mtime.Unix(), boolAInt(e.EsCarpeta), strings.ToLower(e.Nombre), e.Padre)
	return err
}

func boolAInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// BorrarArchivo elimina una ruta y (si es carpeta) todo su subarbol,
// recorriendo la jerarquia por padre (sin LIKE, que no usa indices).
func BorrarArchivo(ruta string) error {
	if _, err := db.Exec(`DELETE FROM archivos WHERE ruta=?`, ruta); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM archivos WHERE ruta IN (
		WITH RECURSIVE sub(r) AS (
			SELECT ruta FROM archivos WHERE padre = ?
			UNION ALL
			SELECT a.ruta FROM archivos a JOIN sub s ON a.padre = s.r
		) SELECT r FROM sub)`, ruta)
	return err
}

// MoverSubarbol reescribe el prefijo de una carpeta movida.
func MoverSubarbol(origen, destino string) error {
	rutas, err := subarbolDe(origen)
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, vieja := range rutas {
		nueva := destino + vieja[len(origen):]
		if _, err := tx.Exec(`DELETE FROM archivos WHERE ruta=?`, nueva); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`UPDATE archivos SET ruta=?, padre=? WHERE ruta=?`,
			nueva, padreDe(nueva), vieja); err != nil {
			tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`UPDATE firmas SET ruta=? WHERE ruta=?`, nueva, vieja); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func padreDe(ruta string) string {
	d := filepath.Dir(ruta)
	if d == "." {
		return ""
	}
	return d
}

// GuardarFirma guarda la firma de una carpeta escaneada.
func GuardarFirma(ruta, firma string, cuando time.Time) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO firmas(ruta, firma, escaneado)
		VALUES(?,?,?)`, ruta, firma, cuando.Unix())
	return err
}

// LeerFirma devuelve la firma guardada de una carpeta ("" si no existe).
func LeerFirma(ruta string) string {
	var firma string
	err := db.QueryRow(`SELECT firma FROM firmas WHERE ruta=?`, ruta).Scan(&firma)
	if err != nil {
		return ""
	}
	return firma
}

// MarcarEscaneo guarda la hora del ultimo escaneo completo.
func MarcarEscaneo(cuando time.Time) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO meta(clave, valor) VALUES('ultimo_escaneo', ?)`,
		cuando.Format(time.RFC3339))
	return err
}

// ConteoFilas devuelve cuantas filas tiene el indice.
func ConteoFilas() int {
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM archivos`).Scan(&n)
	return n
}

// ConteoFirmas devuelve cuantas carpetas tiene registradas el indice.
func ConteoFirmas() int {
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM firmas`).Scan(&n)
	return n
}

// BorrarTodoLimpia el indice (para reconstrucciones).
func BorrarTodo() error {
	_, err := db.Exec(`DELETE FROM archivos`)
	return err
}

// subarbolDe devuelve todas las rutas dentro de una carpeta (inclusive),
// recorriendo la jerarquia por padre.
func subarbolDe(ruta string) ([]string, error) {
	filas, err := db.Query(`WITH RECURSIVE sub(r) AS (
		SELECT ruta FROM archivos WHERE ruta = ?
		UNION ALL
		SELECT a.ruta FROM archivos a JOIN sub s ON a.padre = s.r
	) SELECT r FROM sub`, ruta)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	rutas := []string{}
	for filas.Next() {
		var r string
		if filas.Scan(&r) == nil {
			rutas = append(rutas, r)
		}
	}
	return rutas, filas.Err()
}
