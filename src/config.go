package main

// Configuracion central de Vito (equivalente a config.py del proyecto Python).

import "time"

const (
	Modelo      = "llama3.1:8b"
	MaxPasos    = 4
	Temperatura = 0.0

	// Vigilancia en tiempo real
	VigilarTiempoReal    = true
	DebounceSeg          = 2.0
	ReconciliacionMin    = 30
	IntervaloEscaneos    = 15 * time.Minute
	MaxEventosPendientes = 50000
	MaxLoteEventos       = 1000
	MaxEsperaLote        = 10.0

	// Busqueda
	MaxResultadosBusqueda = 5
	SegundosBusqueda      = 20
)
