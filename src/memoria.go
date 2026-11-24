package main

// Memoria persistente en memoria.json: datos que Jarvis recuerda entre sesiones.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const (
	maxDatos = 200
	maxTexto = 500
)

var bloqueoMemoria sync.Mutex

// reClaveSecreta detecta claves que por nombre son secretos: no se guardan
// en claro aunque el valor no contenga la palabra.
var reClaveSecreta = regexp.MustCompile(`(?i)(password|contrase|token|secret|api[_-]?key|credencial|pin\b)`)

func rutaMemoria() string {
	ejecutable, err := os.Executable()
	if err != nil {
		return "memoria.json"
	}
	return filepath.Join(filepath.Dir(ejecutable), "memoria.json")
}

func leerMemoria() map[string]string {
	contenido, err := os.ReadFile(rutaMemoria())
	if err != nil {
		return map[string]string{}
	}
	var envoltura struct {
		Datos map[string]string `json:"datos"`
	}
	if json.Unmarshal(contenido, &envoltura) != nil || envoltura.Datos == nil {
		return map[string]string{}
	}
	return envoltura.Datos
}

func escribirMemoria(datos map[string]string) error {
	envoltura := struct {
		Datos map[string]string `json:"datos"`
	}{datos}
	contenido, err := json.MarshalIndent(envoltura, "", "  ")
	if err != nil {
		return err
	}
	temporal := rutaMemoria() + ".tmp"
	if err := os.WriteFile(temporal, contenido, 0o644); err != nil {
		return err
	}
	return os.Rename(temporal, rutaMemoria())
}

func GuardarMemoria(clave, valor string) string {
	bloqueoMemoria.Lock()
	defer bloqueoMemoria.Unlock()
	clave = strings.ToLower(strings.TrimSpace(clave))
	valor = strings.TrimSpace(valor)
	if clave == "" || valor == "" {
		return "Necesito una clave y un valor para guardar en la memoria."
	}
	// No guardar secretos en claro (misma regla que el historial de chat).
	valor = ocultarSecretos(valor)
	if reClaveSecreta.MatchString(clave) {
		valor = "[oculto por seguridad]"
	}
	if len(valor) > maxTexto {
		valor = valor[:maxTexto]
	}
	datos := leerMemoria()
	datos[clave] = valor
	if len(datos) > maxDatos {
		// Los mapas de Go no conservan orden; si se llena, se pide olvidar a mano.
		return "La memoria esta llena. Pide olvidar algo antes de guardar mas."
	}
	if err := escribirMemoria(datos); err != nil {
		return fmt.Sprintf("No pude escribir la memoria: %v", err)
	}
	return fmt.Sprintf("Guardado en la memoria: %s = %s", clave, valor)
}

func ConsultarMemoria(clave string) string {
	bloqueoMemoria.Lock()
	defer bloqueoMemoria.Unlock()
	datos := leerMemoria()
	if clave = strings.ToLower(strings.TrimSpace(clave)); clave != "" {
		if valor, ok := datos[clave]; ok {
			return fmt.Sprintf("%s: %s", clave, valor)
		}
		for c, v := range datos {
			if strings.Contains(c, clave) || strings.Contains(clave, c) {
				return fmt.Sprintf("%s: %s", c, v)
			}
		}
		return fmt.Sprintf("No tengo nada guardado sobre %s.", clave)
	}
	if len(datos) == 0 {
		return "La memoria esta vacia."
	}
	lineas := make([]string, 0, len(datos))
	for c, v := range datos {
		lineas = append(lineas, fmt.Sprintf("%s: %s", c, v))
	}
	return strings.Join(lineas, "\n")
}

func OlvidarMemoria(clave string) string {
	bloqueoMemoria.Lock()
	defer bloqueoMemoria.Unlock()
	clave = strings.ToLower(strings.TrimSpace(clave))
	if clave == "" {
		return "Dime que dato quieres que olvide."
	}
	datos := leerMemoria()
	borradas := []string{}
	for c := range datos {
		if c == clave || strings.Contains(c, clave) || strings.Contains(clave, c) {
			delete(datos, c)
			borradas = append(borradas, c)
		}
	}
	if len(borradas) == 0 {
		return fmt.Sprintf("No tengo nada guardado sobre %s.", clave)
	}
	if err := escribirMemoria(datos); err != nil {
		return fmt.Sprintf("No pude escribir la memoria: %v", err)
	}
	return "Olvide: " + strings.Join(borradas, ", ")
}

func ResumenMemoria() string {
	datos := leerMemoria()
	if len(datos) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Datos que recuerdas del usuario (memoria persistente):\n")
	for c, v := range datos {
		fmt.Fprintf(&b, "%s: %s\n", c, v)
	}
	return b.String()
}
