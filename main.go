package main

// Vito: asistente de terminal en Go con Ollama e indexacion del disco.

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	fmt.Println("Vito listo. Escribe una orden o 'apagar' para salir.")

	// Indice de archivos: arranque rapido, el escaneo completo va en segundo plano.
	if err := InicializarDB(); err != nil {
		fmt.Printf("Aviso: no pude abrir el indice de archivos (%v); la busqueda sera lenta.\n", err)
	} else {
		go func() {
			IndexarUnidades(false)
			escribirLog("escaneo inicial listo: %d filas en el indice", ConteoFilas())
		}()
	}

	// Vigilancia en tiempo real con cola de eventos y reconciliacion periodica.
	if VigilarTiempoReal {
		cola := NuevaColaEventos(AplicarLote, Reconciliar)
		cola.Iniciar()
		vigilante := NuevoVigilante(cola)
		vigilante.Iniciar()
		defer vigilante.Detener()
		defer cola.Detener()
	}

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
		if frase == "apagar" || frase == "salir" || frase == "apagar vito" || frase == "cerrar vito" {
			fmt.Println("Hasta luego.")
			return
		}
		fmt.Println(Preguntar(texto))
	}
}
