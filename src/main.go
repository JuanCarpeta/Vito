package main

// Vito: asistente de terminal en Go con Ollama e indexacion del disco.

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	// Indice de archivos: arranque rapido, el escaneo completo va en segundo plano.
	indiceOK := InicializarDB() == nil
	if !indiceOK {
		fmt.Println("Aviso: no pude abrir el indice de archivos; la busqueda sera lenta.")
	}
	podarHistorial()

	// Vigilancia en tiempo real con cola de eventos y reconciliacion periodica.
	if VigilarTiempoReal {
		cola := NuevaColaEventos(AplicarLote, Reconciliar)
		cola.Iniciar()
		vigilante := NuevoVigilante(cola)
		vigilante.Iniciar()
		defer vigilante.Detener()
		defer cola.Detener()
	}

	if indiceOK {
		AnimarArranque(func() {
			IndexarUnidades(false)
			escribirLog("escaneo inicial listo: %d filas en el indice", ConteoFilas())
		})
	} else {
		AnimarArranque(nil)
	}

	lector := bufio.NewScanner(os.Stdin)
	lector.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	confirmarAccion := func(pregunta string) bool {
		fmt.Print(pregunta)
		if !lector.Scan() {
			return false
		}
		respuesta := strings.ToLower(strings.TrimSpace(lector.Text()))
		return respuesta == "s" || respuesta == "si" || respuesta == "sí"
	}
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
			// El resumen va en segundo plano con espera corta: no congelar la salida.
			listo := make(chan struct{})
			go func() {
				actualizarResumen()
				close(listo)
			}()
			select {
			case <-listo:
			case <-time.After(4 * time.Second):
			}
			fmt.Println("Hasta luego.")
			return
		}
		respuesta := Preguntar(texto, confirmarAccion)
		fmt.Println(respuesta)
		guardarTurno(texto, respuesta)
	}
}
