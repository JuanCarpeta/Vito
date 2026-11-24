package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func prepararDBHistorial(t *testing.T) {
	t.Helper()
	anteriorDB := rutaDB
	anteriorSesion := sesionActual
	anteriorTurnos := turnosDesdeResumen
	anteriorHistorial := historial
	rutaDB = filepath.Join(t.TempDir(), "archivos.db")
	if err := InicializarDB(); err != nil {
		t.Fatalf("db: %v", err)
	}
	sesionActual = "sesion-actual"
	turnosDesdeResumen = 0
	historial = nil
	t.Cleanup(func() {
		db.Close()
		db = nil
		rutaDB = anteriorDB
		sesionActual = anteriorSesion
		turnosDesdeResumen = anteriorTurnos
		historial = anteriorHistorial
	})
}

func insertarTurno(t *testing.T, sesion, rol, contenido, fecha string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO conversaciones(sesion, rol, contenido, fecha) VALUES(?,?,?,?)`,
		sesion, rol, contenido, fecha); err != nil {
		t.Fatalf("insertar turno: %v", err)
	}
}

func TestGuardarTurnoYResumen(t *testing.T) {
	prepararDBHistorial(t)
	guardarTurno("hola", "respondi")
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM conversaciones`).Scan(&n); err != nil {
		t.Fatalf("contar: %v", err)
	}
	if n != 2 {
		t.Errorf("esperaba 2 filas, hay %d", n)
	}
	if turnosDesdeResumen != 1 {
		t.Errorf("contador de resumen: %d", turnosDesdeResumen)
	}
	if err := guardarResumen("hablamos de Vito", 2); err != nil {
		t.Fatalf("guardar resumen: %v", err)
	}
	if resumen := leerResumen(); resumen != "hablamos de Vito" {
		t.Errorf("resumen inesperado: %q", resumen)
	}
	turnos, hasta := turnosSinResumir()
	if len(turnos) != 0 || hasta != 2 {
		t.Errorf("turnos sin resumir: %d, hasta %d", len(turnos), hasta)
	}
}

func TestResumenHistorialExcluyeSesionActual(t *testing.T) {
	prepararDBHistorial(t)
	ayer := time.Now().AddDate(0, 0, -1).Format(time.RFC3339)
	insertarTurno(t, "otra-sesion", "user", "hablamos del informe", ayer)
	insertarTurno(t, sesionActual, "user", "mensaje de la sesion actual", time.Now().Format(time.RFC3339))

	texto := ResumenHistorial()
	if !strings.Contains(texto, "hablamos del informe") {
		t.Errorf("no incluyo la frase anterior: %q", texto)
	}
	if !strings.Contains(texto, "ayer") {
		t.Errorf("no etiqueto el dia: %q", texto)
	}
	if strings.Contains(texto, "mensaje de la sesion actual") {
		t.Errorf("duplico la sesion actual: %q", texto)
	}
}

func TestOlvidarConversacion(t *testing.T) {
	prepararDBHistorial(t)
	ayer := time.Now().AddDate(0, 0, -1).Format(time.RFC3339)
	hoy := time.Now().Format(time.RFC3339)
	insertarTurno(t, "otra", "user", "frm_ayer", ayer)
	insertarTurno(t, "otra", "user", "frm_hoy", hoy)
	if err := guardarResumen("resumen viejo", 2); err != nil {
		t.Fatalf("guardar resumen: %v", err)
	}

	if respuesta := olvidarConversacion("ayer"); !strings.Contains(respuesta, "ayer") {
		t.Errorf("respuesta inesperada: %q", respuesta)
	}
	texto := ResumenHistorial()
	if strings.Contains(texto, "frm_ayer") || !strings.Contains(texto, "frm_hoy") {
		t.Errorf("borrado incorrecto: %q", texto)
	}

	if respuesta := olvidarConversacion(""); !strings.Contains(respuesta, "historial") {
		t.Errorf("respuesta inesperada: %q", respuesta)
	}
	var n int
	db.QueryRow(`SELECT COUNT(*) FROM conversaciones`).Scan(&n)
	if n != 0 {
		t.Errorf("quedaron %d filas", n)
	}
	if resumen := leerResumen(); resumen != "" {
		t.Errorf("quedo el resumen: %q", resumen)
	}
}

func TestOcultarSecretosYPodar(t *testing.T) {
	if secreto := ocultarSecretos("password: abc123"); strings.Contains(secreto, "abc123") || !strings.Contains(secreto, "[oculto]") {
		t.Errorf("no oculto la clave: %q", secreto)
	}
	if secreto := ocultarSecretos("Bearer abc.def"); strings.Contains(secreto, "abc.def") {
		t.Errorf("no oculto el token: %q", secreto)
	}

	prepararDBHistorial(t)
	viejo := time.Now().AddDate(0, 0, -40).Format(time.RFC3339)
	insertarTurno(t, "otra", "user", "frm_vieja", viejo)
	insertarTurno(t, "otra", "user", "frm_reciente", time.Now().Format(time.RFC3339))
	podarHistorial()
	texto := ResumenHistorial()
	if strings.Contains(texto, "frm_vieja") || !strings.Contains(texto, "frm_reciente") {
		t.Errorf("poda incorrecta: %q", texto)
	}
}
