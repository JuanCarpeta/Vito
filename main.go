package main

// Jarvis en Go: asistente de terminal con Ollama y herramientas de archivos.

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("Jarvis (Go) listo. Escribe una orden o 'apagar' para salir.")
	lector := bufio.NewScanner(os.Stdin)
	lector.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for {
		fmt.Print("> ")
		if !lector.Scan() {
			break
		}
		texto := strings.TrimSpace(lector.Text())
		if texto == "" {
			continue
		}
		frase := strings.ToLower(texto)
		if frase == "apagar" || frase == "salir" || frase == "apagar jarvis" || frase == "cerrar jarvis" {
			fmt.Println("Hasta luego.")
			return
		}
		fmt.Println(Preguntar(texto))
	}
}
