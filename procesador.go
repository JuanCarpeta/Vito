package main

// Aplica eventos del disco al indice: el disco es la verdad.

import (
	"os"
	"path/filepath"
	"time"
)

// AplicarLote procesa un lote de eventos ya consolidados.
func AplicarLote(lote []Evento) {
	for _, e := range lote {
		switch e.Tipo {
		case "creado", "modificado":
			if _, err := os.Stat(e.Ruta); err != nil {
				continue // el disco manda: si ya no existe, no se indexa
			}
			if e.EsCarpeta {
				escanearSubarbol(e.Ruta, time.Now().Add(2*time.Minute))
			} else {
				indexarArchivoSuelto(e.Ruta)
			}
		case "borrado":
			BorrarArchivo(e.Ruta)
		case "movido":
			if destino := e.Destino; destino != "" {
				if _, err := os.Stat(destino); err == nil {
					MoverSubarbol(e.Ruta, destino)
					if e.EsCarpeta {
						escanearSubarbol(destino, time.Now().Add(2*time.Minute))
					}
				}
			} else if _, err := os.Stat(e.Ruta); err == nil {
				// Renombrado sin destino claro: se re-escanea el padre.
				escanearCarpeta(filepath.Dir(e.Ruta))
			} else {
				BorrarArchivo(e.Ruta)
			}
		}
	}
}

// indexarArchivoSuelto guarda un archivo con su mtime real.
func indexarArchivoSuelto(ruta string) {
	info, err := os.Stat(ruta)
	if err != nil {
		return
	}
	GuardarArchivo(Entrada{
		Ruta: ruta, Nombre: filepath.Base(ruta), Tamano: info.Size(),
		Mtime: info.ModTime(), EsCarpeta: false, Padre: filepath.Dir(ruta),
	})
}

// Reconciliar re-escanea las unidades (sin forzar) para corregir el indice.
func Reconciliar() {
	IndexarUnidades(false)
}
