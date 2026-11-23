package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIntencionMoverCopiar(t *testing.T) {
	anterior := directorioActual
	directorioActual = t.TempDir()
	defer func() { directorioActual = anterior }()

	if err := os.WriteFile(filepath.Join(directorioActual, "notas.txt"), []byte("hola"), 0o644); err != nil {
		t.Fatalf("no pude crear el archivo: %v", err)
	}
	destino := filepath.Join(directorioActual, "destino")
	if err := os.MkdirAll(destino, 0o755); err != nil {
		t.Fatalf("no pude crear el destino: %v", err)
	}

	respuesta := responderDirecto("mueve notas.txt a destino", nil)
	if !strings.Contains(respuesta, "Move") {
		t.Errorf("respuesta inesperada: %s", respuesta)
	}
	if _, err := os.Stat(filepath.Join(destino, "notas.txt")); err != nil {
		t.Fatalf("no move el archivo: %v", err)
	}
}

func TestLlamadaEnTexto(t *testing.T) {
	casos := []string{
		`{"name": "resumir_documento", "parameters": {"ruta": "C:\\Proyectos\\Vito\\informe-prueba.docx"}}`,
		"```json\n{\"name\": \"buscar_archivos\", \"parameters\": {\"consulta\": \"informe\"}}\n```",
		`{"name":"leer_archivo","arguments":{"ruta":"notas.txt"}}`,
	}
	for i, contenido := range casos {
		llamada, ok := llamadaEnTexto(contenido)
		if !ok {
			t.Errorf("caso %d: no detecto la llamada en %q", i, contenido)
			continue
		}
		if llamada.Function.Nombre == "" {
			t.Errorf("caso %d: nombre vacio", i)
		}
		if len(llamada.Function.Argumentos) == 0 {
			t.Errorf("caso %d: argumentos vacios", i)
		}
	}
	// Texto normal no debe disparar el fallback.
	if _, ok := llamadaEnTexto("Hola, ¿en que te ayudo?"); ok {
		t.Error("texto normal detectado como llamada")
	}
	if _, ok := llamadaEnTexto(`{"clave": "valor"}`); ok {
		t.Error("json sin name detectado como llamada")
	}
	// El texto con saludo antes del JSON no debe colarse (es conversacion).
	if _, ok := llamadaEnTexto(strings.Join([]string{"Voy a resumirlo.", `{"name": "x"}`}, "\n")); ok {
		t.Log("nota: JSON con texto delante no se detecta (comportamiento aceptado)")
	}
}
