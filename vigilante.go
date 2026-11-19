package main

// Vigilancia en tiempo real con ReadDirectoryChangesW (espeja watchdog_monitor.py).
// Un solo handle por unidad con modo recursivo, igual que el Observer de Watchdog.

import (
	"os"
	"path/filepath"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

type Vigilante struct {
	cola  *ColaEventos
	mu    sync.Mutex
	abrir map[string]windows.Handle // unidad -> handle
	parar chan struct{}
}

func NuevoVigilante(cola *ColaEventos) *Vigilante {
	return &Vigilante{cola: cola, abrir: map[string]windows.Handle{}, parar: make(chan struct{})}
}

// Iniciar pone un observador recursivo por cada unidad disponible.
func (v *Vigilante) Iniciar() {
	for _, unidad := range discosDisponibles() {
		v.vigilarUnidad(unidad)
	}
}

func (v *Vigilante) Detener() {
	close(v.parar)
	v.mu.Lock()
	for _, h := range v.abrir {
		windows.CancelIo(h)
		windows.CloseHandle(h)
	}
	v.abrir = map[string]windows.Handle{}
	v.mu.Unlock()
}

func (v *Vigilante) vigilarUnidad(unidad string) {
	rutaUTF16, err := windows.UTF16PtrFromString(unidad)
	if err != nil {
		return
	}
	handle, err := windows.CreateFile(
		rutaUTF16,
		windows.FILE_LIST_DIRECTORY,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS,
		0)
	if err != nil {
		escribirLog("no pude vigilar %s: %v", unidad, err)
		return
	}
	v.mu.Lock()
	v.abrir[unidad] = handle
	v.mu.Unlock()
	go v.leerCambios(unidad, handle)
}

const (
	accionCreado     = 1
	accionBorrado    = 2
	accionModificado = 3
	accionMovido     = 4
)

// leerCambios bloquea leyendo notificaciones y las convierte en eventos.
func (v *Vigilante) leerCambios(unidad string, handle windows.Handle) {
	buffer := make([]byte, 64*1024)
	var bytesLeidos uint32
	for {
		select {
		case <-v.parar:
			return
		default:
		}
		err := windows.ReadDirectoryChanges(handle,
			&buffer[0], uint32(len(buffer)), true, // recursivo
			windows.FILE_NOTIFY_CHANGE_FILE_NAME|
				windows.FILE_NOTIFY_CHANGE_DIR_NAME|
				windows.FILE_NOTIFY_CHANGE_LAST_WRITE|
				windows.FILE_NOTIFY_CHANGE_SIZE,
			&bytesLeidos, nil, 0)
		if err != nil {
			escribirLog("vigilancia de %s terminada: %v", unidad, err)
			return
		}
		if bytesLeidos == 0 {
			continue // buffer lleno: la reconciliacion reconstruye
		}
		offset := uintptr(0)
		for {
			informe := (*windows.FileNotifyInformation)(unsafe.Pointer(&buffer[offset]))
			nombreUTF16 := (*[1024]uint16)(unsafe.Pointer(
				&buffer[offset+unsafe.Sizeof(windows.FileNotifyInformation{})]))
			largo := 0
			for nombreUTF16[largo] != 0 && largo < 1024 {
				largo++
			}
			ruta := filepath.Join(unidad, windows.UTF16ToString(nombreUTF16[:largo]))
			v.procesarNotificacion(int(informe.Action), ruta)
			if informe.NextEntryOffset == 0 {
				break
			}
			offset += uintptr(informe.NextEntryOffset)
			if offset >= uintptr(bytesLeidos) {
				break
			}
		}
	}
}

func (v *Vigilante) procesarNotificacion(accion int, ruta string) {
	esCarpeta := false
	if info, err := osStat(ruta); err == nil {
		esCarpeta = info.IsDir()
	}
	switch accion {
	case accionCreado:
		if !IgnorarRuta(ruta, esCarpeta) {
			v.cola.Encolar(Evento{"creado", ruta, "", esCarpeta})
		}
	case accionBorrado:
		// El archivo ya no existe: usamos el nombre para saber si era carpeta.
		if !IgnorarRuta(ruta, esCarpeta) {
			v.cola.Encolar(Evento{"borrado", ruta, "", esCarpeta})
		}
	case accionModificado:
		if !esCarpeta && !IgnorarRuta(ruta, false) {
			v.cola.Encolar(Evento{"modificado", ruta, "", false})
		}
	case accionMovido:
		if !IgnorarRuta(ruta, esCarpeta) {
			v.cola.Encolar(Evento{"movido", ruta, "", esCarpeta})
		}
	}
	}

// osStat es un punto unico para comprobar el disco antes de tocar el indice.
func osStat(ruta string) (os.FileInfo, error) {
	return os.Stat(ruta)
}
