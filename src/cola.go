package main

// Cola de eventos con debounce y coalescencia (espeja core/event_queue.py).

import (
	"strings"
	"sync"
	"time"
)

// Evento es un cambio detectado en el disco.
type Evento struct {
	Tipo      string // creado | modificado | borrado | movido
	Ruta      string
	Destino   string
	EsCarpeta bool
}

func claveEvento(e Evento) string {
	if e.Tipo == "movido" && e.Destino != "" {
		return e.Destino
	}
	return e.Ruta
}

// combinar reduce varios eventos del mismo archivo al estado final probable.
func combinar(previo, nuevo Evento) Evento {
	if previo.Ruta == "" && previo.Tipo == "" {
		return nuevo
	}
	carpeta := previo.EsCarpeta || nuevo.EsCarpeta
	if nuevo.Tipo == "movido" {
		if previo.Tipo == "movido" && previo.Destino == nuevo.Ruta {
			return Evento{"movido", previo.Ruta, nuevo.Destino, carpeta}
		}
		return Evento{"movido", nuevo.Ruta, nuevo.Destino, carpeta}
	}
	if nuevo.Tipo == "borrado" {
		// La fila del indice sigue en el origen: borrar el origen es correcto.
		return Evento{"borrado", previo.Ruta, "", carpeta}
	}
	if previo.Tipo == "borrado" {
		return Evento{"modificado", previo.Ruta, "", carpeta}
	}
	if previo.Tipo == "movido" {
		return Evento{"movido", previo.Ruta, previo.Destino, carpeta}
	}
	tipo := "modificado"
	if previo.Tipo == "creado" || nuevo.Tipo == "creado" {
		tipo = "creado"
	}
	return Evento{tipo, previo.Ruta, "", carpeta}
}

// ColaEventos agrupa eventos en lotes y los entrega al procesador.
type ColaEventos struct {
	procesar      func([]Evento)
	reconciliar   func()
	pendientes    map[string]Evento
	orden         []string
	ultimo        time.Time
	primero       time.Time
	reconciliarYa bool
	mu            sync.Mutex
	aviso         *sync.Cond
	parar         chan struct{}
}

func NuevaColaEventos(procesar func([]Evento), reconciliar func()) *ColaEventos {
	c := &ColaEventos{
		procesar:    procesar,
		reconciliar: reconciliar,
		pendientes:  map[string]Evento{},
		parar:       make(chan struct{}),
	}
	c.aviso = sync.NewCond(&c.mu)
	return c
}

func (c *ColaEventos) Encolar(e Evento) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ahora := time.Now()
	if len(c.pendientes) >= MaxEventosPendientes {
		// Demasiados eventos: la reconciliacion reconstruye desde el disco.
		c.pendientes = map[string]Evento{}
		c.orden = nil
		c.reconciliarYa = true
	}
	if len(c.pendientes) == 0 {
		c.primero = ahora
	}
	clave := claveEvento(e)
	if previo, ok := c.pendientes[clave]; ok && e.Tipo == "movido" {
		e = combinar(previo, e)
	}
	e = combinar(c.pendientes[clave], e)
	if _, exists := c.pendientes[clave]; !exists {
		c.orden = append(c.orden, clave)
	}
	c.pendientes[clave] = e
	c.ultimo = ahora
	c.aviso.Signal()
}

// Iniciar lanza el hilo de procesamiento.
func (c *ColaEventos) Iniciar() {
	go c.bucle()
}

// Detener apaga el hilo y procesa lo pendiente.
func (c *ColaEventos) Detener() {
	close(c.parar)
	c.aviso.Broadcast()
	time.Sleep(300 * time.Millisecond)
	c.mu.Lock()
	resto := make([]Evento, 0, len(c.pendientes))
	for _, clave := range c.orden {
		if e, ok := c.pendientes[clave]; ok {
			resto = append(resto, e)
		}
	}
	c.pendientes = map[string]Evento{}
	c.orden = nil
	c.mu.Unlock()
	if len(resto) > 0 {
		c.procesar(resto)
	}
}

func (c *ColaEventos) bucle() {
	proximaReconciliacion := time.Now().Add(ReconciliacionMin * time.Minute)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-c.parar:
			return
		case <-ticker.C:
		}
		lote := []Evento{}
		c.mu.Lock()
		if len(c.pendientes) > 0 {
			quieto := time.Since(c.ultimo) >= DebounceSeg*time.Second
			maduro := !c.primero.IsZero() &&
				time.Since(c.primero) >= MaxEsperaLote*time.Second
			lleno := len(c.pendientes) >= MaxLoteEventos
			if quieto || maduro || lleno {
				for _, clave := range c.orden {
					lote = append(lote, c.pendientes[clave])
				}
				c.pendientes = map[string]Evento{}
				c.orden = nil
				c.primero = time.Time{}
			}
		}
		reconciliarAhora := c.reconciliarYa || time.Now().After(proximaReconciliacion)
		c.reconciliarYa = false
		c.mu.Unlock()

		if len(lote) > 0 {
			func() {
				defer recuperarError()
				c.procesar(lote)
			}()
		}
		if reconciliarAhora {
			func() {
				defer recuperarError()
				c.reconciliar()
			}()
			proximaReconciliacion = time.Now().Add(ReconciliacionMin * time.Minute)
		}
	}
}

func recuperarError() {
	if r := recover(); r != nil {
		escribirLog("error en el vigilante: %v", r)
	}
}

var _ = strings.ToLower // reservado
