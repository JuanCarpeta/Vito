package main

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

func modoSilencioso() bool {
	for _, a := range os.Args[1:] {
		if a == "--silencioso" || a == "--vigilante" {
			return true
		}
	}
	return os.Getenv("VITO_SILENCIOSO") == "1"
}

func ocultarConsola() {
	obtener := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow")
	mostrar := syscall.NewLazyDLL("user32.dll").NewProc("ShowWindow")
	if obtener.Find() != nil || mostrar.Find() != nil {
		return
	}
	hwnd, _, _ := obtener.Call()
	if hwnd != 0 {
		mostrar.Call(hwnd, 0)
	}
}

var manejadorVigilante windows.Handle

func adquirirVigilante() bool {
	nombre, err := windows.UTF16PtrFromString("Local\\VitoVigilante")
	if err != nil {
		return false
	}
	h, err := windows.CreateMutex(nil, false, nombre)
	if err != nil {
		return false
	}
	evento, err := windows.WaitForSingleObject(h, 0)
	if err == nil && evento == windows.WAIT_OBJECT_0 {
		manejadorVigilante = h
		return true
	}
	windows.CloseHandle(h)
	return false
}
