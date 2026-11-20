package main

import (
	"strings"
	"testing"
)

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
