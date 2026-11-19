package main

// Registro de errores en vito.log y resumen del equipo (inventario).

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var rutaLog = filepath.Join(Base, "vito.log")

func escribirLog(formato string, args ...any) {
	entrada := fmt.Sprintf("%s %s\n", time.Now().Format("2006-01-02 15:04:05"),
		fmt.Sprintf(formato, args...))
	archivo, err := os.OpenFile(rutaLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer archivo.Close()
	archivo.WriteString(entrada)
}

type MEMORYSTATUSEX struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

// ResumenInventario devuelve un bloque de texto con el equipo para el prompt.
func ResumenInventario() string {
	var lineas []string
	nombreEquipo, _ := os.Hostname()
	usuario := os.Getenv("USERNAME")
	lineas = append(lineas, fmt.Sprintf("Equipo: %s, usuario: %s, sistema: Windows (%s)",
		nombreEquipo, usuario, runtime.GOARCH))

	var memoria MEMORYSTATUSEX
	memoria.Length = uint32(unsafe.Sizeof(memoria))
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	estadoMemoria := kernel32.NewProc("GlobalMemoryStatusEx")
	retorno, _, _ := estadoMemoria.Call(uintptr(unsafe.Pointer(&memoria)))
	if retorno != 0 {
		lineas = append(lineas, fmt.Sprintf("RAM: %.1f GB totales, %.1f GB libres",
			float64(memoria.TotalPhys)/1e9, float64(memoria.AvailPhys)/1e9))
	}
	lineas = append(lineas, fmt.Sprintf("CPU: %d nucleos logicos", runtime.NumCPU()))

	discos := []string{}
	for _, unidad := range discosDisponibles() {
		var libres, totales uint64
		rutaUTF16, err := windows.UTF16PtrFromString(unidad)
		if err != nil {
			continue
		}
		if err := windows.GetDiskFreeSpaceEx(rutaUTF16, nil, &totales, &libres); err == nil {
			discos = append(discos, fmt.Sprintf("%s %.0f/%.0f GB libres",
				unidad, float64(libres)/1e9, float64(totales)/1e9))
		}
	}
	if len(discos) > 0 {
		lineas = append(lineas, "Discos: "+strings.Join(discos, "; "))
	}
	return strings.Join(lineas, "\n")
}
